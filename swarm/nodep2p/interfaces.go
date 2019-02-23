package nodep2p

import (
	"context"

	cid "github.com/ipfs/go-cid"
	netp2p "github.com/libp2p/go-libp2p-net"
	peer "github.com/libp2p/go-libp2p-peer"
	peerstore "github.com/libp2p/go-libp2p-peerstore"
	protocol "github.com/libp2p/go-libp2p-protocol"

	"github.com/Conscience/protocol/config"
	"github.com/Conscience/protocol/repo"
	"github.com/Conscience/protocol/swarm/nodeeth"
)

type INode interface {
	ID() peer.ID
	FindProvidersAsync(ctx context.Context, key cid.Cid, count int) <-chan peerstore.PeerInfo
	NewStream(ctx context.Context, peerID peer.ID, pids ...protocol.ID) (netp2p.Stream, error)
	SignHash(data []byte) ([]byte, error)
	AddrFromSignedHash(data, sig []byte) (nodeeth.Address, error)
	AddressHasPullAccess(ctx context.Context, user nodeeth.Address, repoID string) (bool, error)
	Repo(repoID string) *repo.Repo
	RepoAtPathOrID(path string, repoID string) (*repo.Repo, error)
	GetConfig() config.Config
	SetReplicationPolicy(repoID string, shouldReplicate bool) error
}

type MaybeReplProgress struct {
	Percent int
	Error   error
}
