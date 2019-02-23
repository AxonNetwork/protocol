package p2pclient

import (
	"context"
	"encoding/hex"
	"io"
	"sync"

	peer "github.com/libp2p/go-libp2p-peer"
	"github.com/pkg/errors"

	"github.com/Conscience/protocol/log"
	"github.com/Conscience/protocol/swarm/nodep2p"
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

func (sc *SmartClient) FetchChunks(ctx context.Context, chunkObjects [][]byte) <-chan MaybeChunk {
	chOut := make(chan MaybeChunk)
	wg := &sync.WaitGroup{}

	// Load the job queue up with everything in the manifest
	jobQueue := make(chan job, len(chunkObjects))
	for _, obj := range chunkObjects {
		wg.Add(1)
		jobQueue <- job{
			size:        0,
			objectID:    obj,
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
		pool, err := newPeerPool(ctx, sc.node, sc.repoID, maxPeers, nodep2p.CHUNK_PROTO)
		if err != nil {
			chOut <- MaybeChunk{Error: err}
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

func (sc *SmartClient) fetchDataChunk(ctx context.Context, conn *PeerConnection, j job, chOut chan MaybeChunk, jobQueue chan job, wg *sync.WaitGroup) error {
	defer wg.Done()

	chunkStr := hex.EncodeToString(j.objectID)
	log.Infof("[chunk client] requesting data chunk %v", chunkStr)

	stream, err := conn.RequestChunk(ctx, j.objectID)
	if err != nil {
		err = errors.Wrapf(ErrFetchingFromPeer, "tried requesting chunk %v from peer %v: %v", chunkStr, conn.peerID, err)
		log.Errorf("[chunk client]", err)
		jobQueue <- j
		return err
	}

	for {
		data := make([]byte, nodep2p.OBJ_CHUNK_SIZE)
		end := false
		n, err := io.ReadFull(stream, data)
		if err == io.EOF {
			end = true
		} else if err == io.ErrUnexpectedEOF {
			data = data[:n]
		} else if err != nil {
			chOut <- MaybeChunk{Error: err}
		}

		if end == true {
			chOut <- MaybeChunk{
				Chunk: &Chunk{
					ObjectID: j.objectID,
					End:      true,
				},
			}
			return nil
		}

		chOut <- MaybeChunk{
			Chunk: &Chunk{
				ObjectID: j.objectID,
				Data:     data,
			},
		}
	}

	return nil
}
