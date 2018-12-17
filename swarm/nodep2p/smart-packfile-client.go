package nodep2p

import (
	"context"
	"crypto/sha256"
	"io"
	"sync"
	"time"

	peer "github.com/libp2p/go-libp2p-peer"
	"github.com/pkg/errors"
	gitplumbing "gopkg.in/src-d/go-git.v4/plumbing"

	"github.com/Conscience/protocol/config"
	"github.com/Conscience/protocol/log"
	"github.com/Conscience/protocol/repo"
	. "github.com/Conscience/protocol/swarm/wire"
	"github.com/Conscience/protocol/util"
)

type SmartPackfileClient struct {
	node   INode
	config *config.Config
	repo   *repo.Repo
	repoID string
}

type job struct {
	objectID    []byte
	size        int64
	failedPeers map[peer.ID]bool
}

type MaybeFetchFromCommitPacket struct {
	*PackfileHeader
	*PackfileData
	Error error
}

type PackfileHeader struct {
	PackfileID       []byte
	UncompressedSize int64
}

type PackfileData struct {
	ObjHash gitplumbing.Hash
	ObjType gitplumbing.ObjectType
	ObjLen  uint64
	Data    []byte
	End     bool
}

var ErrFetchingFromPeer = errors.New("fetching from peer")

func NewSmartPackfileClient(node INode, repoID string, repoPath string, config *config.Config) *SmartPackfileClient {
	r, _ := node.RepoAtPathOrID(repoPath, repoID)

	sc := &SmartPackfileClient{
		node:   node,
		config: config,
		repo:   r,
		repoID: repoID,
	}
	return sc
}

func (sc *SmartPackfileClient) FetchFromCommit(ctx context.Context, commit gitplumbing.Hash) (<-chan MaybeFetchFromCommitPacket, int64) {
	chOut := make(chan MaybeFetchFromCommitPacket)
	wg := &sync.WaitGroup{}

	manifest, err := sc.requestManifestFromSwarm(ctx, sc.repoID, commit)
	if err != nil {
		go func() {
			defer close(chOut)
			chOut <- MaybeFetchFromCommitPacket{Error: err}
		}()
		return chOut, 0
	}

	// If we're pulling (instead of cloning), filter objects we already have
	if sc.repo != nil {
		filteredManifest := []ManifestObject{}
		for i := range manifest {
			if !sc.repo.HasObject(manifest[i].Hash) {
				filteredManifest = append(filteredManifest, manifest[i])
			}
		}
		manifest = filteredManifest
	}

	// Calculate the uncompressed size of the entire tree of commits that will be transferred.
	var uncompressedSize int64
	for _, obj := range manifest {
		uncompressedSize += obj.UncompressedSize
	}

	// Load the job queue up with everything in the manifest
	jobQueue := make(chan job, len(manifest))
	go func() {
		defer close(chOut)
		defer close(jobQueue)

		for _, obj := range manifest {
			wg.Add(1)
			jobQueue <- job{
				size:        obj.UncompressedSize,
				objectID:    obj.Hash,
				failedPeers: make(map[peer.ID]bool),
			}
		}

		wg.Wait()
	}()

	maxPeers := sc.config.Node.MaxConcurrentPeers

	// Consume the job queue with connections managed by a peerPool{}
	go func() {
		pool, err := newPeerPool(ctx, sc.node, sc.repoID, maxPeers)
		if err != nil {
			chOut <- MaybeFetchFromCommitPacket{Error: err}
			return
		}
		defer pool.Close()

		for batch := range aggregateWork(ctx, jobQueue, uint(len(manifest))/maxPeers, 5*time.Second) {
			conn := pool.GetConn()
			if conn == nil {
				log.Errorln("[packfile client] nil PeerConnection, operation canceled?")
				return
			}

			go func(batch []job) {
				err := sc.fetchPackfile(ctx, conn, batch, chOut, jobQueue, wg)
				if err != nil {
					log.Errorln("[packfile client] fetchObject:", err)
					if errors.Cause(err) == ErrFetchingFromPeer {
						// @@TODO: mark failed peer on job{}
						// @@TODO: maybe call ReturnConn with true if the peer should be discarded
					}
					pool.ReturnConn(conn, false)

				} else {
					pool.ReturnConn(conn, false)
				}
			}(batch)
		}
	}()

	return chOut, uncompressedSize
}

// Takes a job queue and batches received jobs up to `batchSize`.  Batches are also time-constrained.
// If `batchSize` jobs aren't received within `batchTimeout`, the batch is sent anyway.
func aggregateWork(ctx context.Context, jobQueue chan job, batchSize uint, batchTimeout time.Duration) chan []job {
	chBatch := make(chan []job)
	go func() {
		defer close(chBatch)

	Outer:
		for {
			// We don't wait more than this amount of time
			timeout := time.After(batchTimeout)
			current := make([]job, 0)

			for {
				select {
				case j, open := <-jobQueue:
					// If the channel is open, add the received job to the current batch.
					// If it's closed, send whatever we have and close the batch channel.
					if open {
						current = append(current, j)
						if uint(len(current)) >= batchSize {
							chBatch <- current
							continue Outer
						}

					} else {
						if len(current) > 0 {
							chBatch <- current
						}
						return
					}

				case <-timeout:
					if len(current) > 0 {
						chBatch <- current
					}
					continue Outer

				case <-ctx.Done():
					return
				}
			}
		}
	}()
	return chBatch
}

func (sc *SmartPackfileClient) requestManifestFromSwarm(ctx context.Context, repoID string, commit gitplumbing.Hash) ([]ManifestObject, error) {
	c, err := util.CidForString(repoID)
	if err != nil {
		return nil, err
	}

	ctxTimeout, cancel := context.WithTimeout(ctx, time.Duration(sc.config.Node.FindProviderTimeout))
	defer cancel()

	for provider := range sc.node.FindProvidersAsync(ctxTimeout, c, 10) {
		if provider.ID != sc.node.ID() {
			// We found a peer with the object
			manifest, err := sc.requestManifestFromPeer(ctx, provider.ID, repoID, commit)
			if err != nil {
				log.Errorln("[packfile client] requestManifestFromPeer:", err)
				continue
			}
			return manifest, nil
		}
	}
	return nil, errors.Errorf("could not find provider for repo '%v'", repoID)
}

func (sc *SmartPackfileClient) requestManifestFromPeer(ctx context.Context, peerID peer.ID, repoID string, commit gitplumbing.Hash) ([]ManifestObject, error) {
	log.Debugf("[p2p object client] requesting manifest %v/%v from peer %v", repoID, commit, peerID.Pretty())

	// Open the stream
	stream, err := sc.node.NewStream(ctx, peerID, MANIFEST_PROTO)
	if err != nil {
		return nil, err
	}

	sig, err := sc.node.SignHash(commit[:])
	if err != nil {
		return nil, err
	}

	// Write the request packet to the stream
	err = WriteStructPacket(stream, &GetManifestRequest{RepoID: repoID, Commit: commit, Signature: sig})
	if err != nil {
		return nil, err
	}

	// Read the response
	resp := GetManifestResponse{}
	err = ReadStructPacket(stream, &resp)
	if err != nil {
		return nil, err
	} else if resp.ErrUnauthorized {
		return nil, errors.Wrapf(ErrUnauthorized, "%v:%0x", repoID, commit)
	} else if resp.ErrMissingCommit {
		return nil, errors.Wrapf(ErrObjectNotFound, "%v:%0x", repoID, commit)
	}

	log.Debugf("[p2p object client] got manifest metadata %+v", resp)

	manifest := make([]ManifestObject, resp.ManifestLen)
	for i := range manifest {
		err = ReadStructPacket(stream, &manifest[i])
		if err == io.EOF {
			break
		} else if err != nil {
			return nil, err
		}
	}

	return manifest, nil
}

func makePackfileTempID(objectIDs [][]byte) []byte {
	h := sha256.New()
	for i := range objectIDs {
		h.Write(objectIDs[i])
	}
	return h.Sum(nil)
}

func determineMissingIDs(desired, available [][]byte) [][]byte {
	m := map[string]struct{}{}
	for _, bs := range available {
		m[string(bs)] = struct{}{}
	}

	missing := [][]byte{}
	for _, bs := range desired {
		if _, exists := m[string(bs)]; !exists {
			missing = append(missing, []byte(bs))
		}
	}
	return missing
}

func (sc *SmartPackfileClient) returnJobsToQueue(ctx context.Context, jobs []job, jobQueue chan job) {
	for _, j := range jobs {
		select {
		case jobQueue <- j:
		case <-ctx.Done():
			return
		}
	}
}

func (sc *SmartPackfileClient) fetchPackfile(ctx context.Context, conn *PeerConnection, batch []job, chOut chan MaybeFetchFromCommitPacket, jobQueue chan job, wg *sync.WaitGroup) error {
	log.Infof("[packfile client] requesting packfile with %v objects", len(batch))

	desiredObjectIDs := make([][]byte, len(batch))
	jobMap := make(map[string]job, len(batch))
	for i := range batch {
		desiredObjectIDs[i] = batch[i].objectID
		jobMap[string(batch[i].objectID)] = batch[i]
	}

	availableObjectIDs, packfileReader, err := conn.RequestPackfile(ctx, desiredObjectIDs)
	if err != nil {
		err = errors.Wrapf(ErrFetchingFromPeer, "tried requesting packfile from peer %v: %v", conn.peerID, err)
		log.Errorf("[packfile client]", err)
		go sc.returnJobsToQueue(ctx, batch, jobQueue)
		return err
	}
	defer packfileReader.Close()

	// Determine which objects the peer can't send us and re-add those to the job queue.
	missingObjectIDs := determineMissingIDs(desiredObjectIDs, availableObjectIDs)
	if len(missingObjectIDs) > 0 {
		jobsToReturn := make([]job, len(missingObjectIDs))
		for _, oid := range missingObjectIDs {
			jobsToReturn = append(jobsToReturn, jobMap[string(oid)])
		}
		go sc.returnJobsToQueue(ctx, jobsToReturn, jobQueue)
	}

	// Calculate the total uncompressed size of the objects in the packfile.
	var uncompressedSize int64
	for _, objectID := range availableObjectIDs {
		uncompressedSize += jobMap[string(objectID)].size
	}

	var packfileTempID gitplumbing.Hash
	copy(packfileTempID[:], makePackfileTempID(availableObjectIDs))

	chOut <- MaybeFetchFromCommitPacket{
		PackfileHeader: &PackfileHeader{
			PackfileID:       packfileTempID[:],
			UncompressedSize: uncompressedSize,
		},
	}

	for {
		data := make([]byte, OBJ_CHUNK_SIZE)
		n, err := io.ReadFull(packfileReader, data)
		if err == io.EOF {
			// read no bytes
			break

		} else if err == io.ErrUnexpectedEOF {
			data = data[:n]

		} else if err != nil {
			failedJobs := make([]job, len(availableObjectIDs))
			for _, oid := range availableObjectIDs {
				failedJobs = append(failedJobs, jobMap[string(oid)])
			}
			go sc.returnJobsToQueue(ctx, failedJobs, jobQueue)
			return err
		}
		chOut <- MaybeFetchFromCommitPacket{
			PackfileData: &PackfileData{
				ObjHash: packfileTempID,
				ObjType: -1,
				ObjLen:  0,
				Data:    data,
			},
		}
	}

	chOut <- MaybeFetchFromCommitPacket{
		PackfileData: &PackfileData{
			ObjHash: packfileTempID,
			ObjType: -1,
			ObjLen:  0,
			End:     true,
		},
	}

	for range availableObjectIDs {
		wg.Done()
	}

	return nil
}
