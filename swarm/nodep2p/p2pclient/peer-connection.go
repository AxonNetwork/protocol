package p2pclient

import (
	"context"
	"io"

	netp2p "github.com/libp2p/go-libp2p-net"
	"github.com/libp2p/go-libp2p-peer"

	"github.com/Conscience/protocol/swarm/nodep2p"
	. "github.com/Conscience/protocol/swarm/wire"
	"github.com/pkg/errors"
)

type PeerConnection struct {
	peerID peer.ID
	repoID string
	node   nodep2p.INode
	stream netp2p.Stream
}

func NewPeerConnection(node nodep2p.INode, peerID peer.ID, repoID string) *PeerConnection {
	return &PeerConnection{
		peerID: peerID,
		repoID: repoID,
		node:   node,
		stream: nil,
	}
}

// Caller has the duty to close the stream
func (pc *PeerConnection) OpenStream(ctx context.Context) error {
	var err error
	var stream netp2p.Stream
	defer func() {
		if err != nil && stream != nil {
			stream.Close()
		}
	}()
	stream, err = pc.node.NewStream(ctx, pc.peerID, nodep2p.HANDSHAKE_PROTO)
	if err != nil {
		return err
	}

	sig, err := pc.getSignature()
	if err != nil {
		return err
	}

	err = WriteStructPacket(stream, &HandshakeRequest{
		RepoID:    pc.repoID,
		Signature: sig,
	})
	if err != nil {
		return err
	}

	resp := HandshakeResponse{}
	err = ReadStructPacket(stream, &resp)
	if err != nil {
		return err
	} else if resp.ErrUnauthorized {
		return errors.Wrapf(ErrUnauthorized, "%v", pc.repoID)
	}

	return nil
}

func (pc *PeerConnection) IsStreamOpen() bool {
	return pc.stream != nil
}

func (pc *PeerConnection) Close() {
	if pc.IsStreamOpen() {
		pc.stream.Close()
	}
}

func (pc *PeerConnection) RequestPackfile(ctx context.Context, objectIDs [][]byte) ([][]byte, io.ReadCloser, error) {
	stream, err := pc.node.NewStream(ctx, pc.peerID, nodep2p.PACKFILE_PROTO)
	if err != nil {
		return nil, nil, err
	}

	sig, err := pc.getSignature()
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

func (pc *PeerConnection) getSignature() ([]byte, error) {
	return pc.node.SignHash([]byte(pc.repoID))
}
