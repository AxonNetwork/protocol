package nodep2p

import (
	"context"

	cid "github.com/ipfs/go-cid"
	netp2p "github.com/libp2p/go-libp2p-net"
	peer "github.com/libp2p/go-libp2p-peer"
	peerstore "github.com/libp2p/go-libp2p-peerstore"
	protocol "github.com/libp2p/go-libp2p-protocol"
	gitplumbing "gopkg.in/src-d/go-git.v4/plumbing"

	"github.com/Conscience/protocol/config"
	"github.com/Conscience/protocol/repo"
	"github.com/Conscience/protocol/swarm/nodeeth"
	"github.com/Conscience/protocol/util"
)

type (
	INode interface {
		ID() peer.ID
		FindProvidersAsync(ctx context.Context, key cid.Cid, count int) <-chan peerstore.PeerInfo
		NewStream(ctx context.Context, peerID peer.ID, pids ...protocol.ID) (netp2p.Stream, error)
		SignHash(data []byte) ([]byte, error)
		AddrFromSignedHash(data, sig []byte) (nodeeth.Address, error)
		AddressHasPullAccess(ctx context.Context, user nodeeth.Address, repoID string) (bool, error)
		Repo(repoID string) *repo.Repo
		GetConfig() config.Config
		PullRepo(repoID string) error
		SetReplicationPolicy(repoID string, shouldReplicate bool) error
	}

	IStrategy interface {
		FetchFromCommit(ctx context.Context, repoID string, commit string) (ch <-chan MaybeFetchFromCommitPacket, uncompressedSize int64)
	}

	MaybeFetchFromCommitPacket struct {
		*PackfileHeader
		*PackfileData
		Error error
	}

	PackfileHeader struct {
		PackfileID       []byte
		UncompressedSize int64
	}

	PackfileData struct {
		ObjHash gitplumbing.Hash
		ObjType gitplumbing.ObjectType
		ObjLen  uint64
		Data    []byte
		End     bool
	}

	IPeerConnection interface {
		RequestObject(ctx context.Context, objectID []byte) (*util.ObjectReader, error)
	}
)
