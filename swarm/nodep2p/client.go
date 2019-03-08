package nodep2p

import (
	"context"
	"sync"

	"github.com/libgit2/git2go"
	peer "github.com/libp2p/go-libp2p-peer"
	"github.com/pkg/errors"

	"github.com/Conscience/protocol/config"
	"github.com/Conscience/protocol/repo"
	. "github.com/Conscience/protocol/swarm/wire"
)

type Client struct {
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
	*Chunk
	Error error
}

type PackfileHeader struct {
	PackfileID       []byte
	UncompressedSize int64
}

type PackfileData struct {
	PackfileID []byte
	Data       []byte
	End        bool
}

var ErrFetchingFromPeer = errors.New("fetching from peer")

func NewClient(node INode, repoID string, repoPath string, config *config.Config) *Client {
	r, _ := node.RepoAtPathOrID(repoPath, repoID)

	sc := &Client{
		node:   node,
		config: config,
		repo:   r,
		repoID: repoID,
	}
	return sc
}

func (sc *Client) FetchFromCommit(ctx context.Context, commitID git.Oid, checkoutType CheckoutType) (<-chan MaybeFetchFromCommitPacket, int64, int64) {
	chOut := make(chan MaybeFetchFromCommitPacket)

	gitObjects, chunkObjects, uncompressedSize, err := sc.GetManifest(ctx, commitID, checkoutType)
	if err != nil {
		go func() {
			defer close(chOut)
			chOut <- MaybeFetchFromCommitPacket{Error: err}
		}()
		return chOut, 0, 0
	}

	wg := &sync.WaitGroup{}

	wg.Add(1)
	go func() {
		defer wg.Done()
		gitCh := sc.FetchGitPackfiles(ctx, gitObjects)
		for packet := range gitCh {
			chOut <- packet
		}
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		chunks := ManifestObjectsToHashes(chunkObjects)
		chunkCh := sc.FetchChunks(ctx, chunks)
		for packet := range chunkCh {
			if packet.Error != nil {
				chOut <- MaybeFetchFromCommitPacket{Error: err}
			} else {
				chOut <- MaybeFetchFromCommitPacket{Chunk: packet.Chunk}
			}
		}
	}()

	go func() {
		wg.Wait()
		close(chOut)
	}()

	return chOut, uncompressedSize, int64(len(chunkObjects))

}

func ManifestObjectsToHashes(objects []ManifestObject) [][]byte {
	hashes := make([][]byte, 0)
	for _, obj := range objects {
		hashes = append(hashes, obj.Hash)
	}
	return hashes
}

func (sc *Client) FetchChunksFromCommit(ctx context.Context, commitID git.Oid, checkoutType CheckoutType) (<-chan MaybeChunk, int64) {
	chOut := make(chan MaybeChunk)

	_, chunkObjects, _, err := sc.GetManifest(ctx, commitID, checkoutType)
	if err != nil {
		go func() {
			defer close(chOut)
			chOut <- MaybeChunk{Error: err}
		}()
		return chOut, 0
	}

	chunks := ManifestObjectsToHashes(chunkObjects)
	go func() {
		chunkCh := sc.FetchChunks(ctx, chunks)
		for packet := range chunkCh {
			chOut <- packet
		}
		close(chOut)
	}()

	return chOut, int64(len(chunkObjects))
}
