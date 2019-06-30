package nodep2p

import (
	"context"
	"crypto/sha256"
	"sync"
	"time"

	netp2p "github.com/libp2p/go-libp2p-net"
	peer "github.com/libp2p/go-libp2p-peer"
	"github.com/pkg/errors"

	"github.com/Conscience/protocol/log"
)

func (h *Host) FetchGitPackfiles(ctx context.Context, repoID string, gitObjects []ManifestObject) <-chan MaybeFetchFromCommitPacket {
	chOut := make(chan MaybeFetchFromCommitPacket)

	// Load the job queue up with everything in the manifest
	jobs := make([]job, len(gitObjects))
	for i, obj := range gitObjects {
		jobs[i] = job{
			size:        obj.UncompressedSize,
			objectID:    obj.Hash,
			failedPeers: make(map[peer.ID]bool),
		}
	}

	var (
		maxPeers     = h.Config.Node.MaxConcurrentPeers
		batchSize    = uint(len(gitObjects)) / maxPeers
		batchTimeout = 3 * time.Second
		jobQueue     = newJobQueue(ctx, jobs, batchSize, batchTimeout)
	)

	// Consume the job queue with connections managed by a peerPool{}
	go func() {
		defer close(chOut)
		defer jobQueue.Close()

		pool, err := newPeerPool(ctx, h, repoID, maxPeers, PACKFILE_PROTO, true)
		if err != nil {
			chOut <- MaybeFetchFromCommitPacket{Error: err}
			return
		}
		defer pool.Close()

		seenPeers := make(map[peer.ID]bool)
		wg := &sync.WaitGroup{}

		for {
			conn, err := pool.GetConn()
			if err != nil {
				log.Errorln("[packfile client] error obtaining peer connection:", err)
				continue
			} else if conn == nil {
				log.Errorln("[packfile client] nil PeerConnection, operation canceled?")
				return
			}

			if seenPeers[conn.peerID] {
				jobQueue.UncapBatchSize()
			}
			seenPeers[conn.peerID] = true

			batch := jobQueue.GetBatch()
			if batch == nil {
				pool.ReturnConn(conn, false)
				break
			}

			wg.Add(1)
			go func(batch []job) {
				defer wg.Done()

				var strike bool
				defer func() { pool.ReturnConn(conn, strike) }()

				err := h.fetchPackfile(ctx, conn, repoID, batch, chOut, jobQueue)
				if err != nil {
					if errors.Cause(err) == ErrFetchingFromPeer {
						// @@TODO: mark failed peer on job{}
						// @@TODO: maybe call ReturnConn with true if the peer should be discarded
					}
					strike = true
				}
			}(batch)
		}

		wg.Wait()
	}()

	return chOut
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

func (h *Host) packfileHandshake(conn *peerConn, repoID string, objectIDs [][]byte) ([][]byte, netp2p.Stream, error) {
	sig, err := h.ethClient.SignHash([]byte(repoID))
	if err != nil {
		return nil, nil, err
	}

	// Write the request packet to the stream
	err = WriteMsg(conn, &GetPackfileRequest{
		RepoID:    repoID,
		Signature: sig,
		ObjectIDs: FlattenObjectIDs(objectIDs),
	})
	if err != nil {
		return nil, nil, err
	}
	log.Debugf("[packfile client] sent packfile request: %v (%v objects)", repoID, len(objectIDs))

	var resp GetPackfileResponseHeader
	err = ReadMsg(conn, &resp)
	if err != nil {
		return nil, nil, err
	} else if resp.ErrUnauthorized {
		return nil, nil, errors.Wrapf(ErrUnauthorized, "%v", repoID)
	} else if len(resp.ObjectIDs) == 0 {
		return nil, nil, errors.Wrapf(ErrObjectNotFound, "%v", repoID)
	}
	log.Debugf("[packfile client] got packfile response header: %v objects", len(UnflattenObjectIDs(resp.ObjectIDs)))
	return UnflattenObjectIDs(resp.ObjectIDs), conn, nil
}

func (h *Host) fetchPackfile(ctx context.Context, conn *peerConn, repoID string, batch []job, chOut chan MaybeFetchFromCommitPacket, jobQueue *jobQueue) error {
	log.Infof("[packfile client] requesting packfile with %v objects", len(batch))

	desiredObjectIDs := make([][]byte, len(batch))
	jobMap := make(map[string]job, len(batch))
	for i := range batch {
		desiredObjectIDs[i] = batch[i].objectID
		jobMap[string(batch[i].objectID)] = batch[i]
	}

	availableObjectIDs, packfileStream, err := h.packfileHandshake(conn, repoID, desiredObjectIDs)
	if err != nil {
		err = errors.Wrapf(ErrFetchingFromPeer, "tried requesting packfile from peer %v: %v", conn.peerID, err)
		log.Errorf("[packfile client]", err)
		go jobQueue.ReturnFailed(batch)
		return err
	}

	// Determine which objects the peer can't send us and re-add those to the job queue.
	missingObjectIDs := determineMissingIDs(desiredObjectIDs, availableObjectIDs)
	if len(missingObjectIDs) > 0 {
		jobsToReturn := make([]job, len(missingObjectIDs))
		for i, oid := range missingObjectIDs {
			jobsToReturn[i] = jobMap[string(oid)]
		}
		go jobQueue.ReturnFailed(jobsToReturn)
	}

	if len(availableObjectIDs) == 0 {
		return nil
	}

	// Calculate the total uncompressed size of the objects in the packfile.
	var uncompressedSize int64
	for _, objectID := range availableObjectIDs {
		uncompressedSize += jobMap[string(objectID)].size
	}

	packfileTempID := makePackfileTempID(availableObjectIDs)

	chOut <- MaybeFetchFromCommitPacket{
		PackfileHeader: &PackfileHeader{
			PackfileID:       packfileTempID,
			UncompressedSize: uncompressedSize,
		},
	}

	for {
		var packet GetPackfileResponsePacket
		err = ReadMsg(packfileStream, &packet)
		if err != nil {
			log.Errorln("[packfile client] error reading GetPackfileResponsePacket:", err)
			break
		} else if packet.End {
			break
		}

		chOut <- MaybeFetchFromCommitPacket{
			PackfileData: &PackfileData{
				PackfileID: packfileTempID,
				Data:       packet.Data,
			},
		}
	}

	if err != nil {
		failedJobs := make([]job, len(availableObjectIDs))
		for i, oid := range availableObjectIDs {
			failedJobs[i] = jobMap[string(oid)]
		}
		go jobQueue.ReturnFailed(failedJobs)
		return err
	}

	chOut <- MaybeFetchFromCommitPacket{
		PackfileData: &PackfileData{
			PackfileID: packfileTempID,
			End:        true,
		},
	}

	go jobQueue.MarkDone(len(availableObjectIDs))

	return nil
}
