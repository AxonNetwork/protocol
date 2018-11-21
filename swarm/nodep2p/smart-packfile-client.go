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
	node        INode
	repo        *repo.Repo
	config      *config.Config
	flatHead    []byte
	flatHistory []byte
}

type job struct {
	objectID    []byte
	failedPeers map[peer.ID]bool
}

var ErrFetchingFromPeer = errors.New("fetching from peer")

func NewSmartPackfileClient(node INode, repo *repo.Repo, config *config.Config) *SmartPackfileClient {
	sc := &SmartPackfileClient{
		node:   node,
		repo:   repo,
		config: config,
	}
	return sc
}

func (sc *SmartPackfileClient) FetchFromCommit(ctx context.Context, repoID string, commit string) <-chan MaybeChunk {
	ch := make(chan MaybeChunk)
	wg := &sync.WaitGroup{}

	flatHead, flatHistory, err := sc.requestManifestFromSwarm(ctx, repoID, commit)
	if err != nil {
		go func() {
			defer close(ch)
			ch <- MaybeChunk{Error: err}
		}()
		return ch
	}

	allObjects := UnflattenObjectIDs(append(flatHead, flatHistory...))
	jobQueue := make(chan job, len(allObjects))

	numPeers := 4

	// Load the job queue up with everything in the manifest
	go func() {
		defer close(ch)
		defer close(jobQueue)

		for i := 0; i < len(allObjects); i++ {
			wg.Add(1)
			jobQueue <- job{allObjects[i], make(map[peer.ID]bool)}
		}

		wg.Wait()
	}()

	// Consume the job queue with connections managed by a peerPool{}
	go func() {
		pool, err := newPeerPool(ctx, sc.node, repoID, numPeers)
		if err != nil {
			ch <- MaybeChunk{Error: err}
			return
		}
		defer pool.Close()

		for batch := range aggregateWork(ctx, jobQueue, len(allObjects)/numPeers) {
			conn := pool.GetConn()
			if conn == nil {
				log.Errorln("[smart client] nil PeerConnection, operation canceled?")
				return
			}

			go func(batch []job) {
				err := sc.fetchPackfile(ctx, conn, batch, ch, jobQueue, wg)
				if err != nil {
					log.Errorln("[smart client] fetchObject:", err)
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

	return ch
}

func aggregateWork(ctx context.Context, ch chan job, batchSize int) chan []job {
	chBatch := make(chan []job)
	go func() {
		defer close(chBatch)

	Outer:
		for {
			timeout := time.After(5 * time.Second)
			current := make([]job, 0)

			for {
				select {
				case j, open := <-ch:
					if !open {
						chBatch <- current
						return
					}

					current = append(current, j)
					if len(current) >= batchSize {
						chBatch <- current
						continue Outer
					}

				case <-timeout:
					chBatch <- current
					continue Outer

				case <-ctx.Done():
					return
				}
			}
		}
	}()
	return chBatch
}

func (sc *SmartPackfileClient) requestManifestFromSwarm(ctx context.Context, repoID string, commit string) ([]byte, []byte, error) {
	c, err := util.CidForString(repoID)
	if err != nil {
		return nil, nil, err
	}

	ctxTimeout, cancel := context.WithTimeout(ctx, time.Duration(sc.config.Node.FindProviderTimeout))
	defer cancel()

	for provider := range sc.node.FindProvidersAsync(ctxTimeout, c, 10) {
		if provider.ID != sc.node.ID() {
			// We found a peer with the object
			head, history, err := sc.requestManifestFromPeer(ctx, provider.ID, repoID, commit)
			if err != nil {
				log.Errorln("[smart client] requestManifestFromPeer:", err)
				continue
			}
			return head, history, nil
		}
	}
	return nil, nil, errors.Errorf("could not find provider for repo '%v'", repoID)
}

func (sc *SmartPackfileClient) requestManifestFromPeer(ctx context.Context, peerID peer.ID, repoID string, commit string) ([]byte, []byte, error) {
	log.Debugf("[p2p object client] requesting manifest %v/%v from peer %v", repoID, commit, peerID.Pretty())

	// Open the stream
	stream, err := sc.node.NewStream(ctx, peerID, MANIFEST_PROTO)
	if err != nil {
		return nil, nil, err
	}

	sig, err := sc.node.SignHash([]byte(commit))
	if err != nil {
		return nil, nil, err
	}

	// Write the request packet to the stream
	err = WriteStructPacket(stream, &GetManifestRequest{RepoID: repoID, Commit: commit, Signature: sig})
	if err != nil {
		return nil, nil, err
	}

	// // Read the response
	resp := GetManifestResponse{}
	err = ReadStructPacket(stream, &resp)
	if err != nil {
		return nil, nil, err
	} else if !resp.Authorized {
		return nil, nil, errors.Wrapf(ErrUnauthorized, "%v:%0x", repoID, commit)
	} else if !resp.HasCommit {
		return nil, nil, errors.Wrapf(ErrObjectNotFound, "%v:%0x", repoID, commit)
	}

	log.Debugf("[p2p object client] got manifest metadata %+v", resp)

	flatHead := make([]byte, resp.HeadLen)
	_, err = io.ReadFull(stream, flatHead)
	if err != nil {
		return nil, nil, err
	}

	flatHistory := make([]byte, resp.HistoryLen)
	_, err = io.ReadFull(stream, flatHistory)
	if err != nil {
		return nil, nil, err
	}

	return flatHead, flatHistory, nil
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

func (sc *SmartPackfileClient) fetchPackfile(ctx context.Context, conn *PeerConnection, batch []job, ch chan MaybeChunk, jobQueue chan job, wg *sync.WaitGroup) error {
	desiredObjectIDs := make([][]byte, len(batch))
	jobMap := map[string]job{}
	for i := range batch {
		desiredObjectIDs[i] = batch[i].objectID
		jobMap[string(batch[i].objectID)] = batch[i]
	}

	availableObjectIDs, packfileReader, err := conn.RequestPackfile(ctx, desiredObjectIDs)
	if err != nil {
		err = errors.Wrapf(ErrFetchingFromPeer, "tried requesting packfile from peer %v: %v", conn.peerID, err)
		log.Errorf("[smart client]", err)
		go sc.returnJobsToQueue(ctx, batch, jobQueue)
		return err
	}
	defer packfileReader.Close()

	missingObjectIDs := determineMissingIDs(desiredObjectIDs, availableObjectIDs)
	if len(missingObjectIDs) > 0 {
		failedJobs := make([]job, len(missingObjectIDs))
		for _, oid := range missingObjectIDs {
			failedJobs = append(failedJobs, jobMap[string(oid)])
		}
		go sc.returnJobsToQueue(ctx, failedJobs, jobQueue)
	}

	var packfileTempID gitplumbing.Hash
	copy(packfileTempID[:], makePackfileTempID(availableObjectIDs))

	data := make([]byte, OBJ_CHUNK_SIZE)
	for {
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
		ch <- MaybeChunk{
			ObjHash: packfileTempID,
			ObjType: -1,
			ObjLen:  0,
			Data:    data,
		}
	}

	ch <- MaybeChunk{
		ObjHash: packfileTempID,
		ObjType: -1,
		ObjLen:  0,
		End:     true,
	}

	for range availableObjectIDs {
		wg.Done()
	}

	return nil
}
