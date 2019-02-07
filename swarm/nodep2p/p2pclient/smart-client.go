package p2pclient

import (
	"context"
	"sync"

	peer "github.com/libp2p/go-libp2p-peer"
	"github.com/pkg/errors"
	gitplumbing "gopkg.in/src-d/go-git.v4/plumbing"

	"github.com/Conscience/protocol/config"
	"github.com/Conscience/protocol/repo"
	"github.com/Conscience/protocol/swarm/nodep2p"
	. "github.com/Conscience/protocol/swarm/wire"
)

type SmartClient struct {
	node   nodep2p.INode
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

type Chunk struct {
	ObjectID []byte
	Data     []byte
}

var ErrFetchingFromPeer = errors.New("fetching from peer")

func NewSmartClient(node nodep2p.INode, repoID string, repoPath string, config *config.Config) *SmartClient {
	r, _ := node.RepoAtPathOrID(repoPath, repoID)

	sc := &SmartClient{
		node:   node,
		config: config,
		repo:   r,
		repoID: repoID,
	}
	return sc
}

func (sc *SmartClient) FetchFromCommit(ctx context.Context, commit gitplumbing.Hash, checkoutType CheckoutType) (<-chan MaybeFetchFromCommitPacket, int64, int64) {
	chOut := make(chan MaybeFetchFromCommitPacket)

	gitObjects, chunkObjects, uncompressedSize, err := sc.GetManifest(ctx, commit, checkoutType)
	if err != nil {
		go func() {
			defer close(chOut)
			chOut <- MaybeFetchFromCommitPacket{Error: err}
		}()
		return chOut, 0, 0
	}

	wg := &sync.WaitGroup{}

	go func() {
		wg.Add(1)
		defer wg.Done()
		gitCh := sc.FetchGitPackfiles(ctx, gitObjects)
		for packet := range gitCh {
			chOut <- packet
		}
	}()

	go func() {
		wg.Add(1)
		defer wg.Done()
		chunkCh := sc.FetchChunks(ctx, chunkObjects)
		for packet := range chunkCh {
			chOut <- packet
		}
	}()

	go func() {
		wg.Wait()
		close(chOut)
	}()

	return chOut, uncompressedSize, int64(len(chunkObjects))

}
