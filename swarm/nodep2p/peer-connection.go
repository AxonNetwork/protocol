package nodep2p

import (
	"context"
	"io"
	"time"

	netp2p "github.com/libp2p/go-libp2p-net"
	peer "github.com/libp2p/go-libp2p-peer"

	"github.com/Conscience/protocol/log"
	. "github.com/Conscience/protocol/swarm/wire"
	"github.com/Conscience/protocol/util"
	"github.com/pkg/errors"
)

type PeerConnection struct {
	peerID        peer.ID
	repoID        string
	currentCommit string
	stream        netp2p.Stream
	// resChannels   map[string]chan MaybeRes
}

type MaybeRes struct {
	Res   *util.ObjectReader
	Error error
}

func NewPeerConnection(ctx context.Context, node INode, peerID peer.ID, repoID string) (*PeerConnection, error) {
	ctxHandshake, cancel := context.WithTimeout(ctx, time.Second*10)
	defer cancel()

	currentCommit, stream, err := handshakeRequest(ctxHandshake, node, peerID, repoID)
	if err != nil {
		return nil, err
	}

	pc := &PeerConnection{
		peerID:        peerID,
		repoID:        repoID,
		currentCommit: currentCommit,
		stream:        stream,
		// resChannels:   make(map[string]chan MaybeRes),
	}
	return pc, nil
}

func handshakeRequest(ctx context.Context, node INode, peerID peer.ID, repoID string) (string, netp2p.Stream, error) {
	log.Debugf("[p2p swarm client] requesting handshake %v from peer %v", repoID, peerID.Pretty())

	stream, err := node.NewStream(ctx, peerID, HANDSHAKE_PROTO)
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

func (pc *PeerConnection) RequestObject(ctx context.Context, objectID []byte) (*util.ObjectReader, error) {
	err := WriteStructPacket(pc.stream, &GetObjectRequest{ObjectID: objectID})
	if err != nil {
		return nil, err
	}

	resp := GetObjectResponse{}
	err = ReadStructPacket(pc.stream, &resp)
	if err != nil {
		return nil, err
	} else if !resp.HasObject {
		return nil, errors.Wrapf(ErrObjectNotFound, "%v:%0x", pc.repoID, objectID)
	}

	r := io.LimitReader(pc.stream, int64(resp.ObjectLen))
	rc := util.MakeReadAllCloser(r)
	or := &util.ObjectReader{
		Reader:     rc,
		Closer:     rc,
		ObjectType: resp.ObjectType,
		ObjectLen:  resp.ObjectLen,
	}
	return or, nil
}

func (pc *PeerConnection) Close() error {
	return pc.stream.Close()
}

// func (pc *PeerConnection) RequestObject(ctx context.Context, objectID []byte) (*util.ObjectReader, error) {
// 	err := WriteStructPacket(pc.stream, &GetObjectRequest{ObjectID: objectID})
// 	if err != nil {
// 		return nil, err
// 	}
// 	resCh := make(chan MaybeRes)
// 	pc.resChannels[string(objectID)] = resCh
// 	res := <-resCh
// 	log.Println("hey response here bb ", string(objectID))
// 	if res.Error != nil {
// 		return nil, err
// 	}
// 	return res.Res, nil
// }

// func (pc *PeerConnection) connectLoop() {
// 	for {
// 		resp := GetObjectResponse{}
// 		err := ReadStructPacket(pc.stream, &resp)
// 		if err != nil {
// 			continue
// 		}
// 		objectID := resp.ObjectID
// 		ch := pc.resChannels[string(objectID)]
// 		if !resp.HasObject {
// 			ch <- MaybeRes{Error: errors.Wrapf(ErrObjectNotFound, "%v:%0x", pc.repoID, objectID)}
// 			continue
// 		}
// 		or := &util.ObjectReader{
// 			Reader:     &io.LimitedReader{r, resp.ObjectLen},
// 			Closer:     r,
// 			ObjectType: resp.ObjectType,
// 			ObjectLen:  resp.ObjectLen,
// 		}
// 		ch <- MaybeRes{Res: or}
// 	}
// }
