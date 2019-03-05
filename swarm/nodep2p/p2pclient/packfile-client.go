package p2pclient

import (
	"context"
	"crypto/sha256"
	"sync"
	"time"

	netp2p "github.com/libp2p/go-libp2p-net"
	peer "github.com/libp2p/go-libp2p-peer"
	"github.com/pkg/errors"

	"github.com/Conscience/protocol/log"
	"github.com/Conscience/protocol/swarm/nodep2p"
	"github.com/Conscience/protocol/swarm/wire"
)

func (sc *SmartClient) FetchGitPackfiles(ctx context.Context, gitObjects []wire.ManifestObject) <-chan MaybeFetchFromCommitPacket {
	chOut := make(chan MaybeFetchFromCommitPacket)
	wg := &sync.WaitGroup{}

	// Load the job queue up with everything in the manifest
	jobQueue := make(chan job, len(gitObjects))
	for _, obj := range gitObjects {
		wg.Add(1)
		jobQueue <- job{
			size:        obj.UncompressedSize,
			objectID:    obj.Hash,
			failedPeers: make(map[peer.ID]bool),
		}
	}

	go func() {
		wg.Wait()
		close(jobQueue)
		close(chOut)
	}()

	maxPeers := sc.config.Node.MaxConcurrentPeers

	// Consume the job queue with connections managed by a peerPool{}
	go func() {
		pool, err := newPeerPool(ctx, sc.node, sc.repoID, maxPeers, nodep2p.PACKFILE_PROTO, true)
		if err != nil {
			chOut <- MaybeFetchFromCommitPacket{Error: err}
			return
		}
		defer pool.Close()

		batchSize := uint(len(gitObjects)) / maxPeers
		batchTimeout := 3 * time.Second

		for batch := range aggregateWork(ctx, jobQueue, batchSize, batchTimeout) {
			conn, err := pool.GetConn()
			if err != nil {
				log.Errorln("[packfile client] error obtaining peer connection:", err)
				return
			} else if conn == nil {
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
					pool.ReturnConn(conn, true)

				} else {
					pool.ReturnConn(conn, false)
				}
			}(batch)
		}
	}()

	return chOut
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

func (sc *SmartClient) returnJobsToQueue(ctx context.Context, jobs []job, jobQueue chan job) {
	for _, j := range jobs {
		select {
		case jobQueue <- j:
		case <-ctx.Done():
			return
		}
	}
}

func (sc *SmartClient) packfileHandshake(conn *peerConn, objectIDs [][]byte) ([][]byte, netp2p.Stream, error) {
	sig, err := sc.node.SignHash([]byte(sc.repoID))
	if err != nil {
		return nil, nil, err
	}

	// Write the request packet to the stream
	err = wire.WriteStructPacket(conn, &wire.GetPackfileRequest{
		RepoID:    sc.repoID,
		Signature: sig,
		ObjectIDs: wire.FlattenObjectIDs(objectIDs),
	})
	if err != nil {
		return nil, nil, err
	}
	log.Debugf("[packfile client] sent packfile request: %v (%v objects)", sc.repoID, len(objectIDs))

	resp := wire.GetPackfileResponseHeader{}
	err = wire.ReadStructPacket(conn, &resp)
	if err != nil {
		return nil, nil, err
	} else if resp.ErrUnauthorized {
		return nil, nil, errors.Wrapf(wire.ErrUnauthorized, "%v", sc.repoID)
	} else if len(resp.ObjectIDs) == 0 {
		return nil, nil, errors.Wrapf(wire.ErrObjectNotFound, "%v", sc.repoID)
	}
	log.Debugf("[packfile client] got packfile response header: %v objects", len(wire.UnflattenObjectIDs(resp.ObjectIDs)))
	return wire.UnflattenObjectIDs(resp.ObjectIDs), conn, nil
}

func (sc *SmartClient) fetchPackfile(ctx context.Context, conn *peerConn, batch []job, chOut chan MaybeFetchFromCommitPacket, jobQueue chan job, wg *sync.WaitGroup) error {
	log.Infof("[packfile client] requesting packfile with %v objects", len(batch))

	desiredObjectIDs := make([][]byte, len(batch))
	jobMap := make(map[string]job, len(batch))
	for i := range batch {
		desiredObjectIDs[i] = batch[i].objectID
		jobMap[string(batch[i].objectID)] = batch[i]
	}

	availableObjectIDs, packfileStream, err := sc.packfileHandshake(conn, desiredObjectIDs)
	if err != nil {
		err = errors.Wrapf(ErrFetchingFromPeer, "tried requesting packfile from peer %v: %v", conn.peerID, err)
		log.Errorf("[packfile client]", err)
		go sc.returnJobsToQueue(ctx, batch, jobQueue)
		return err
	}
	// @@TODO: wrap this in a Closer that just reads to the end of the stream, or negotiates an early termination or something
	defer packfileStream.Close()

	// Determine which objects the peer can't send us and re-add those to the job queue.
	missingObjectIDs := determineMissingIDs(desiredObjectIDs, availableObjectIDs)
	if len(missingObjectIDs) > 0 {
		jobsToReturn := make([]job, len(missingObjectIDs))
		for i, oid := range missingObjectIDs {
			jobsToReturn[i] = jobMap[string(oid)]
		}
		go sc.returnJobsToQueue(ctx, jobsToReturn, jobQueue)
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
		var packet wire.GetPackfileResponsePacket
		err = wire.ReadStructPacket(packfileStream, &packet)
		if err != nil {
			log.Errorln("[packfile client] error reading GetPackfileResponsePacket:", err)
			break
		} else if packet.End {
			log.Debugln("[packfile client] got packet.End")
			break
		}

		log.Debugln("[packfile client] got packet", packet.DataLen, packet.End)
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
		go sc.returnJobsToQueue(ctx, failedJobs, jobQueue)
		return err
	}

	chOut <- MaybeFetchFromCommitPacket{
		PackfileData: &PackfileData{
			PackfileID: packfileTempID,
			End:        true,
		},
	}
	log.Debugln("[packfile client] done")

	for i := 0; i < len(availableObjectIDs); i++ {
		wg.Done()
	}

	return nil
}