package nodep2p

import (
	"context"
	"io"

	netp2p "github.com/libp2p/go-libp2p-net"
	"github.com/libp2p/go-libp2p-peer"

	"github.com/Conscience/protocol/log"
	. "github.com/Conscience/protocol/swarm/wire"
	"github.com/Conscience/protocol/util"
	"github.com/pkg/errors"
)

type PeerConnection struct {
	peerID peer.ID
	repoID string
	node   INode
	// currentCommit string
	// stream        netp2p.Stream
	// resChannels   map[string]chan MaybeRes
}

type MaybeRes struct {
	Res   *util.ObjectReader
	Error error
}

func NewPeerConnection(ctx context.Context, node INode, peerID peer.ID, repoID string) (*PeerConnection, error) {
	// ctxHandshake, cancel := context.WithTimeout(ctx, time.Second*10)
	// defer cancel()

	// currentCommit, stream, err := initiatePackfileStream(ctxHandshake, node, peerID, repoID)
	// if err != nil {
	// 	return nil, err
	// }

	pc := &PeerConnection{
		peerID: peerID,
		repoID: repoID,
		node:   node,
		// currentCommit: currentCommit,
		// stream:        stream,
		// resChannels:   make(map[string]chan MaybeRes),
	}
	return pc, nil
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
	// return UnflattenObjectIDs(resp.ObjectIDs), newPackfileStreamReader(ctx, stream), nil
}

// type PackfileStreamReader struct {
// 	io.Reader
// 	cancel func()
// }

// func newPackfileStreamReader(ctx context.Context, stream io.Reader) *PackfileStreamReader {
// 	pr, pw := io.Pipe()
// 	ctxInner, cancel := context.WithCancel(ctx)

// 	go func() {
// 		var err error
// 		defer func() { pw.CloseWithError(err) }()

// 		for {
// 			pkt := PackfileStreamChunk{}

// 			select {
// 			case <-ctxInner.Done():
// 				// Drain the rest of this packfile's packets from the stream so that the next reader
// 				// starts at the right position.
// 				for {
// 					err = ReadStructPacket(stream, &pkt)
// 					if err != nil {
// 						return
// 					} else if pkt.End {
// 						return
// 					}
// 				}

// 				err = ctxInner.Err()
// 				return
// 			default:
// 			}

// 			err = ReadStructPacket(stream, &pkt)
// 			if err != nil {
// 				return
// 			} else if pkt.End {
// 				return
// 			}

// 			n, err := pw.Write(pkt.Data)
// 			if err != nil {
// 				return
// 			} else if n < len(pkt.Data) {
// 				err = errors.New("[peer connection] RequestPackfile: read less than expected")
// 				return
// 			}
// 		}
// 	}()

// 	return &PackfileStreamReader{
// 		Reader: pr,
// 		cancel: cancel,
// 	}
// }

// func (r *PackfileStreamReader) Close() error {
// 	r.cancel()
// 	return nil
// }

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

func (pc *PeerConnection) RequestObject(ctx context.Context, objectID []byte) (*util.ObjectReader, error) {
	// err := WriteStructPacket(pc.stream, &GetObjectRequest{ObjectID: objectID})
	// if err != nil {
	// 	return nil, err
	// }

	// resp := GetObjectResponse{}
	// err = ReadStructPacket(pc.stream, &resp)
	// if err != nil {
	// 	return nil, err
	// } else if !resp.HasObject {
	// 	return nil, errors.Wrapf(ErrObjectNotFound, "%v:%0x", pc.repoID, objectID)
	// }

	// r := io.LimitReader(pc.stream, int64(resp.ObjectLen))
	// rc := util.MakeReadAllCloser(r)
	// or := &util.ObjectReader{
	// 	Reader:     rc,
	// 	Closer:     rc,
	// 	ObjectType: resp.ObjectType,
	// 	ObjectLen:  resp.ObjectLen,
	// }
	// return or, nil
	return nil, nil
}

// func (pc *PeerConnection) Close() error {
// 	return pc.stream.Close()
// }

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
