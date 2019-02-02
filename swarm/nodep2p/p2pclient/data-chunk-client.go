package p2pclient

import (
	"context"
	"sync"

	peer "github.com/libp2p/go-libp2p-peer"

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

		// for chunk := range jobQueue {
		// 	conn := pool.GetConn()

		// }

	}()

	return chOut
}
