package nodep2p

import (
	"context"
	"sync"
	"time"

	peer "github.com/libp2p/go-libp2p-peer"
	"github.com/pkg/errors"

	"github.com/Conscience/protocol/log"
)

type Chunk struct {
	ObjectID []byte
	Data     []byte
	End      bool
}

type MaybeChunk struct {
	Chunk *Chunk
	Error error
}

func (h *Host) FetchChunks(ctx context.Context, repoID string, chunkObjects []ManifestObject) <-chan MaybeFetchFromCommitPacket {
	// chOut := make(chan MaybeChunk)
	// wg := &sync.WaitGroup{}

	// // Load the job queue up with everything in the manifest
	// jobQueue := make(chan job, len(chunkObjects))
	// for _, obj := range chunkObjects {
	// 	wg.Add(1)
	// 	jobQueue <- job{
	// 		size:        0,
	// 		objectID:    obj,
	// 		failedPeers: make(map[peer.ID]bool),
	// 	}
	// }

	// go func() {
	// 	wg.Wait()
	// 	close(chOut)
	// 	close(jobQueue)
	// }()

	// go func() {
	// 	pool, err := newPeerPool(ctx, h, repoID, h.Config.Node.MaxConcurrentPeers, CHUNK_PROTO, true)
	// 	if err != nil {
	// 		chOut <- MaybeChunk{Error: err}
	// 		return
	// 	}
	// 	defer pool.Close()

	// 	for chunk := range jobQueue {
	// 		conn, err := pool.GetConn()
	// 		if err != nil {
	// 			log.Errorln("[packfile client] error obtaining peer connection:", err)
	// 			return
	// 		} else if conn == nil {
	// 			log.Errorln("[packfile client] nil PeerConnection, operation canceled?")
	// 			return
	// 		}

	// 		go func(chunk job) {
	// 			err := h.fetchDataChunk(ctx, conn, repoID, chunk, chOut, jobQueue, wg)
	// 			if err != nil {
	// 				log.Errorln("[chunk client] fetchObject:", err)
	// 				if errors.Cause(err) == ErrFetchingFromPeer {
	// 					// @@TODO: mark failed peer on job{}
	// 					// @@TODO: maybe call ReturnConn with true if the peer should be discarded
	// 				}
	// 				pool.ReturnConn(conn, true)

	// 			} else {
	// 				pool.ReturnConn(conn, false)
	// 			}
	// 		}(chunk)
	// 	}
	// }()

	// return chOut
	chOut := make(chan MaybeFetchFromCommitPacket)

	// Load the job queue up with everything in the manifest
	jobs := make([]job, len(chunkObjects))
	for i, obj := range chunkObjects {
		jobs[i] = job{
			size:        obj.UncompressedSize,
			objectID:    obj.Hash,
			failedPeers: make(map[peer.ID]bool),
		}
	}

	var (
		maxPeers     = h.Config.Node.MaxConcurrentPeers
		batchSize    = uint(1)
		batchTimeout = 3 * time.Second
		jobQueue     = newJobQueue(ctx, jobs, batchSize, batchTimeout)
	)

	// Consume the job queue with connections managed by a peerPool{}
	go func() {
		defer close(chOut)
		defer jobQueue.Close()

		pool, err := newPeerPool(ctx, h, repoID, maxPeers, CHUNK_PROTO, true)
		if err != nil {
			chOut <- MaybeFetchFromCommitPacket{Error: err}
			return
		}
		defer pool.Close()

		// seenPeers := make(map[peer.ID]bool)
		wg := &sync.WaitGroup{}

	Loop:
		for {
			select {
			case <-ctx.Done():
				break Loop
			default:
			}

			conn, err := pool.GetConn()
			if err != nil {
				log.Errorf("[chunk client] error obtaining peer connection: %+v", err)
				continue
			} else if conn == nil {
				log.Errorln("[chunk client] nil PeerConnection, operation canceled?")
				return
			}

			// if seenPeers[conn.peerID] {
			//     jobQueue.UncapBatchSize()
			// }
			// seenPeers[conn.peerID] = true

			batch := jobQueue.GetBatch()
			if batch == nil || len(batch) == 0 {
				pool.ReturnConn(conn, false)
				break
			}

			j := batch[0]

			wg.Add(1)
			go func(j job) {
				defer wg.Done()

				var strike bool
				defer func() { pool.ReturnConn(conn, strike) }()

				err := h.fetchDataChunk(ctx, conn, repoID, j, chOut, jobQueue)
				if err != nil {
					if errors.Cause(err) == ErrFetchingFromPeer {
						// @@TODO: mark failed peer on job{}
						// @@TODO: maybe call ReturnConn with true if the peer should be discarded
					}
					strike = true
				}
			}(j)
		}

		wg.Wait()
	}()

	return chOut
}

func (h *Host) fetchDataChunk(ctx context.Context, conn *peerConn, repoID string, j job, chOut chan MaybeFetchFromCommitPacket, jobQueue *jobQueue) error {
	log.Infof("[chunk client] requesting data chunk %0x", j.objectID)

	var totalBytes int64
	var readBytes int64
	{
		sig, err := h.ethClient.SignHash([]byte(repoID))
		if err != nil {
			return errors.WithStack(err)
		}

		err = WriteMsg(conn, &GetChunkRequest{
			RepoID:    repoID,
			ChunkID:   j.objectID,
			Signature: sig,
		})
		if err != nil {
			return errors.WithStack(err)
		}

		var resp GetChunkResponseHeader
		err = ReadMsg(conn, &resp)
		if err != nil {
			return err
		} else if resp.ErrObjectNotFound {
			return errors.Wrapf(ErrObjectNotFound, "%v", conn.repoID)
		} else if resp.ErrUnauthorized {
			return errors.Wrapf(ErrUnauthorized, "%v", conn.repoID)
		}

		totalBytes = resp.Length
	}

	for {
		var pkt GetChunkResponsePacket
		err := ReadMsg(conn, &pkt)
		if err != nil {
			return errors.WithStack(err)
		} else if pkt.End {
			break
		}

		chOut <- MaybeFetchFromCommitPacket{
			Chunk: &Chunk{
				ObjectID: j.objectID,
				Data:     pkt.Data,
			},
		}

		readBytes += int64(len(pkt.Data))
	}

	if totalBytes > readBytes {
		// @@TODO: need to be able to signal an error on a single chunk without erroring the entire multi-peer stream
		err := errors.Errorf("did not receive full chunk (%v)", j.objectID)
		chOut <- MaybeFetchFromCommitPacket{
			Chunk: &Chunk{ObjectID: j.objectID},
			Error: err,
		}
		return err
	}

	chOut <- MaybeFetchFromCommitPacket{
		Chunk: &Chunk{
			ObjectID: j.objectID,
			End:      true,
		},
	}

	return nil
}
