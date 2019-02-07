package p2pclient

import (
	"context"
	"encoding/hex"
	"sync"

	peer "github.com/libp2p/go-libp2p-peer"
	"github.com/pkg/errors"

	"github.com/Conscience/protocol/log"
	. "github.com/Conscience/protocol/swarm/wire"
)

func (sc *SmartClient) FetchChunks(ctx context.Context, chunkObjects []ManifestObject) <-chan MaybeFetchFromCommitPacket {
	chOut := make(chan MaybeFetchFromCommitPacket)
	wg := &sync.WaitGroup{}

	// Load the job queue up with everything in the manifest
	jobQueue := make(chan job, len(chunkObjects))
	for _, obj := range chunkObjects {
		wg.Add(1)
		jobQueue <- job{
			size:        obj.UncompressedSize,
			objectID:    obj.Hash,
			failedPeers: make(map[peer.ID]bool),
		}
	}

	go func() {
		wg.Wait()
		close(chOut)
		close(jobQueue)
	}()

	maxPeers := sc.config.Node.MaxConcurrentPeers

	go func() {
		pool, err := newPeerPool(ctx, sc.node, sc.repoID, maxPeers, true)
		if err != nil {
			chOut <- MaybeFetchFromCommitPacket{Error: err}
			return
		}
		defer pool.Close()

		for chunk := range jobQueue {
			conn := pool.GetConn()

			go func(chunk job) {
				err := sc.fetchDataChunk(ctx, conn, chunk, chOut, jobQueue, wg)
				if err != nil {
					log.Errorln("[chunk client] fetchObject:", err)
					if errors.Cause(err) == ErrFetchingFromPeer {
						// @@TODO: mark failed peer on job{}
						// @@TODO: maybe call ReturnConn with true if the peer should be discarded
					}
					pool.ReturnConn(conn, true)

				} else {
					pool.ReturnConn(conn, false)
				}
			}(chunk)
		}
	}()

	return chOut
}

func (sc *SmartClient) fetchDataChunk(ctx context.Context, conn *PeerConnection, j job, chOut chan MaybeFetchFromCommitPacket, jobQueue chan job, wg *sync.WaitGroup) error {
	defer wg.Done()

	chunkStr := hex.EncodeToString(j.objectID)
	log.Infof("[chunk client] requesting data chunk %v", chunkStr)

	data, err := conn.RequestChunk(ctx, j.objectID)
	if err != nil {
		err = errors.Wrapf(ErrFetchingFromPeer, "tried requesting chunk %v from peer %v: %v", chunkStr, conn.peerID, err)
		log.Errorf("[chunk client]", err)
		jobQueue <- j
		return err
	}

	chOut <- MaybeFetchFromCommitPacket{
		Chunk: &Chunk{
			ObjectID: j.objectID,
			Data:     data,
		},
	}

	return nil
}
