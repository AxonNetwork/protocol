package strategy

import (
	"context"

	netp2p "gx/ipfs/QmPjvxTpVH8qJyQDnxnsxF9kv9jezKD1kozz1hs3fCGsNh/go-libp2p-net"
	cid "gx/ipfs/QmYVNvtQkeZ6AKSwDrjQTs432QtL6umrrK41EBq3cu7iSP/go-cid"
	protocol "gx/ipfs/QmZNkThpqfVXs9GNbexPrfBbXSLNYeKrE7jwFM2oqHbyqN/go-libp2p-protocol"
	peerstore "gx/ipfs/QmZR2XWVVBCtbgBWnQhWk2xcQfaR3W8faQPriAiaaj7rsr/go-libp2p-peerstore"
	peer "gx/ipfs/QmdVrMn1LhB4ybb8hMVaMLXnA8XRSewMnK6YqXKXoTcRvN/go-libp2p-peer"

	gitplumbing "gopkg.in/src-d/go-git.v4/plumbing"

	"github.com/Conscience/protocol/repo"
	"github.com/Conscience/protocol/swarm/nodeeth"
	"github.com/Conscience/protocol/util"
)

type (
	INode interface {
		ID() peer.ID
		FindProvidersAsync(ctx context.Context, key *cid.Cid, count int) <-chan peerstore.PeerInfo
		NewStream(ctx context.Context, peerID peer.ID, pids ...protocol.ID) (netp2p.Stream, error)
		SignHash(data []byte) ([]byte, error)
		AddrFromSignedHash(data, sig []byte) (nodeeth.Address, error)
		AddressHasPullAccess(ctx context.Context, user nodeeth.Address, repoID string) (bool, error)
		Repo(repoID string) *repo.Repo
	}

	IStrategy interface {
		FetchFromCommit(ctx context.Context, repoID string, commit string) <-chan MaybeChunk
	}

	MaybeChunk struct {
		ObjHash gitplumbing.Hash
		ObjType gitplumbing.ObjectType
		ObjLen  uint64
		Data    []byte
		Error   error
	}

	IPeerConnection interface {
		RequestObject(ctx context.Context, objectID []byte) (*util.ObjectReader, error)
	}
)
