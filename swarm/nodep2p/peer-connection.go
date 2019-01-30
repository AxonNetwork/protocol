package nodep2p

import (
	"context"
	"io"

	"github.com/libp2p/go-libp2p-peer"

	. "github.com/Conscience/protocol/swarm/wire"
	"github.com/pkg/errors"
)

type PeerConnection struct {
	peerID peer.ID
	repoID string
	node   INode
}

func NewPeerConnection(node INode, peerID peer.ID, repoID string) *PeerConnection {
	return &PeerConnection{
		peerID: peerID,
		repoID: repoID,
		node:   node,
	}
}

func (pc *PeerConnection) RequestPackfile(ctx context.Context, objectIDs [][]byte) ([][]byte, io.ReadCloser, error) {
	stream, err := pc.node.NewStream(ctx, pc.peerID, PACKFILE_PROTO)
	if err != nil {
		return nil, nil, err
	}

	sig, err := pc.node.SignHash([]byte(pc.repoID))
	if err != nil {
		return nil, nil, err
	}

	// Write the request packet to the stream
	err = WriteStructPacket(stream, &GetPackfileRequest{
		RepoID:    pc.repoID,
		Signature: sig,
		ObjectIDs: FlattenObjectIDs(objectIDs),
	})
	if err != nil {
		return nil, nil, err
	}

	resp := GetPackfileResponse{}
	err = ReadStructPacket(stream, &resp)
	if err != nil {
		stream.Close()
		return nil, nil, err
	} else if resp.ErrUnauthorized {
		stream.Close()
		return nil, nil, errors.Wrapf(ErrUnauthorized, "%v", pc.repoID)
	} else if len(resp.ObjectIDs) == 0 {
		stream.Close()
		return nil, nil, errors.Wrapf(ErrObjectNotFound, "%v", pc.repoID)
	}
	return UnflattenObjectIDs(resp.ObjectIDs), stream, nil
}
