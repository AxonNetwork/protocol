package swarm

import (
	"context"
	"io"
	"io/ioutil"
	"time"

	netp2p "gx/ipfs/QmPjvxTpVH8qJyQDnxnsxF9kv9jezKD1kozz1hs3fCGsNh/go-libp2p-net"
	peer "gx/ipfs/QmdVrMn1LhB4ybb8hMVaMLXnA8XRSewMnK6YqXKXoTcRvN/go-libp2p-peer"

	"github.com/Conscience/protocol/log"
	. "github.com/Conscience/protocol/swarm/wire"
	"github.com/Conscience/protocol/util"
	"github.com/pkg/errors"
)

type MaybeRes struct {
	Res   *util.ObjectReader
	Error error
}

type PeerConnection struct {
	peerID        peer.ID
	repoID        string
	currentCommit string
	stream        netp2p.Stream
	resChannels   map[string]chan MaybeRes
}

func (n *Node) NewPeerConnection(peerID peer.ID, repoID string) (*PeerConnection, error) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
	defer cancel()
	currentCommit, stream, err := n.handshakeRequest(ctx, peerID, repoID)
	if err != nil {
		return nil, err
	}

	pc := &PeerConnection{
		peerID:        peerID,
		repoID:        repoID,
		currentCommit: currentCommit,
		stream:        stream,
		resChannels:   map[string]chan MaybeRes{},
	}
	return pc, nil
}

func (n *Node) handshakeRequest(ctx context.Context, peerID peer.ID, repoID string) (string, netp2p.Stream, error) {
	log.Debugf("[p2p swarm client] requesting handshake %v from peer %v", repoID, peerID.Pretty())

	stream, err := n.NewStream(ctx, peerID, HANDSHAKE_PROTO)
	if err != nil {
		return "", nil, err
	}

	sig, err := n.SignHash([]byte(repoID))
	if err != nil {
		return "", nil, err
	}

	// Write the request packet to the stream
	err = WriteStructPacket(stream, &HandshakeRequest{RepoID: repoID, Signature: sig})
	if err != nil {
		return "", nil, err
	}

	// // Read the response
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
	}
	if !resp.HasObject {
		return nil, errors.Wrapf(ErrObjectNotFound, "%v:%0x", pc.repoID, objectID)
	}
	r := io.LimitReader(pc.stream, int64(resp.ObjectLen))
	rc := ioutil.NopCloser(r)
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
