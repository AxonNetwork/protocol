package nodep2p

import (
	"context"
	"sync"

	peer "github.com/libp2p/go-libp2p-peer"
	"github.com/pkg/errors"

	"github.com/Conscience/protocol/log"
	"github.com/Conscience/protocol/swarm/wire"
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

func (sc *Client) FetchChunks(ctx context.Context, chunkObjects [][]byte) <-chan MaybeChunk {
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
		pool, err := newPeerPool(ctx, sc.node, sc.repoID, maxPeers, CHUNK_PROTO, true)
		if err != nil {
			chOut <- MaybeChunk{Error: err}
			return
		}
		defer pool.Close()

		for chunk := range jobQueue {
			conn, err := pool.GetConn()
			if err != nil {
				log.Errorln("[packfile client] error obtaining peer connection:", err)
				return
			} else if conn == nil {
				log.Errorln("[packfile client] nil PeerConnection, operation canceled?")
				return
			}

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

func (sc *Client) fetchDataChunk(ctx context.Context, conn *peerConn, j job, chOut chan MaybeChunk, jobQueue chan job, wg *sync.WaitGroup) error {
	defer wg.Done()

	log.Infof("[chunk client] requesting data chunk %0x", j.objectID)

	var totalBytes int64
	var readBytes int64
	{
		sig, err := sc.node.SignHash([]byte(sc.repoID))
		if err != nil {
			return errors.WithStack(err)
		}

		err = wire.WriteStructPacket(conn, &wire.GetChunkRequest{
			RepoID:    sc.repoID,
			ChunkID:   j.objectID,
			Signature: sig,
		})
		if err != nil {
			return errors.WithStack(err)
		}

		var resp wire.GetChunkResponseHeader
		err = wire.ReadStructPacket(conn, &resp)
		if err != nil {
			return err
		} else if resp.ErrObjectNotFound {
			return errors.Wrapf(wire.ErrObjectNotFound, "%v", conn.repoID)
		} else if resp.ErrUnauthorized {
			return errors.Wrapf(wire.ErrUnauthorized, "%v", conn.repoID)
		}

		totalBytes = resp.Length
	}

	for {
		var pkt wire.GetChunkResponsePacket
		err := wire.ReadStructPacket(conn, &pkt)
		if err != nil {
			return errors.WithStack(err)
		} else if pkt.End {
			break
		}

		chOut <- MaybeChunk{
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
		chOut <- MaybeChunk{
			Chunk: &Chunk{ObjectID: j.objectID},
			Error: err,
		}
		return err
	}

	chOut <- MaybeChunk{
		Chunk: &Chunk{
			ObjectID: j.objectID,
			End:      true,
		},
	}

	return nil
}
