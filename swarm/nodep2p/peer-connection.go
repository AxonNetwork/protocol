package nodep2p

import (
	"context"
	"io"

	netp2p "github.com/libp2p/go-libp2p-net"
	"github.com/libp2p/go-libp2p-peer"

	"github.com/Conscience/protocol/log"
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

	objectIDsCompacted := []byte{}
	for i := range objectIDs {
		objectIDsCompacted = append(objectIDsCompacted, objectIDs[i]...)
	}

	// Write the request packet to the stream
	err = WriteStructPacket(stream, &GetPackfileRequest{
		RepoID:    pc.repoID,
		Signature: sig,
		ObjectIDs: objectIDsCompacted,
	})
	if err != nil {
		return nil, nil, err
	}

	resp := GetPackfileResponse{}
	err = ReadStructPacket(stream, &resp)
	if err != nil {
		return nil, nil, err
	} else if !resp.Authorized {
		return nil, nil, errors.Wrapf(ErrUnauthorized, "%v", pc.repoID)
	}

	return UnflattenObjectIDs(resp.ObjectIDs), stream, nil
}

func initiateObjectStream(ctx context.Context, node INode, peerID peer.ID, repoID string) (string, netp2p.Stream, error) {
	log.Debugf("[p2p swarm client] requesting handshake %v from peer %v", repoID, peerID.Pretty())

	stream, err := node.NewStream(ctx, peerID, OBJECT_PROTO)
	if err != nil {
		return "", nil, err
	}

	sig, err := node.SignHash([]byte(repoID))
	if err != nil {
		return "", nil, err
	}

	// Write the request packet to the stream
	err = WriteStructPacket(stream, &HandshakeRequest{RepoID: repoID, Signature: sig})
	if err != nil {
		return "", nil, err
	}

	// Read the response
	resp := HandshakeResponse{}
	err = ReadStructPacket(stream, &resp)
	if err != nil {
		return "", nil, err
	} else if !resp.Authorized {
		return "", nil, errors.Wrapf(ErrUnauthorized, "%v", repoID)
	}

	return resp.Commit, stream, nil
}
