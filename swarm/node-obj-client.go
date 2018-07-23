package swarm

import (
	"context"
	"encoding/binary"
	"encoding/hex"
	"io"

	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	gitplumbing "gopkg.in/src-d/go-git.v4/plumbing"

	peer "gx/ipfs/QmdVrMn1LhB4ybb8hMVaMLXnA8XRSewMnK6YqXKXoTcRvN/go-libp2p-peer"
)

// Opens an outgoing request to another Node for the given object.
func (n *Node) openPeerObjectReader(ctx context.Context, peerID peer.ID, repoID string, objectID []byte) (io.ReadCloser, gitplumbing.ObjectType, int64, error) {
	log.Printf("[stream] requesting object...")

	// Open the stream
	stream, err := n.Host.NewStream(ctx, peerID, OBJECT_STREAM_PROTO)
	if err != nil {
		return nil, 0, 0, err
	}

	//
	// 1. Write the repo name and object ID to the stream.
	//

	repoIDLen := make([]byte, 8)
	objectIDLen := make([]byte, 8)
	binary.LittleEndian.PutUint64(repoIDLen, uint64(len(repoID)))
	binary.LittleEndian.PutUint64(objectIDLen, uint64(len(objectID)))

	msg := append(repoIDLen, []byte(repoID)...)
	msg = append(msg, objectIDLen...)
	msg = append(msg, objectID...)
	// msg = append(msg, 0x0)
	_, err = stream.Write(msg)
	if err != nil {
		return nil, 0, 0, err
	}

	//
	// 2. If the reply is 0x0, the peer doesn't have the object.  If the reply is 0x1, it does.
	//
	reply := make([]byte, 1)
	recvd, err := stream.Read(reply)
	if err != nil {
		return nil, 0, 0, err
	} else if recvd < 1 {
		return nil, 0, 0, errors.WithStack(ErrProtocol)
	}

	if reply[0] == 0x0 {
		return nil, 0, 0, errors.New("peer doesn't have object " + repoID + ":" + hex.EncodeToString(objectID))
	} else if reply[0] != 0x1 {
		return nil, 0, 0, errors.WithStack(ErrProtocol)
	}

	//
	// 3. Read the object type.  This only matters for Git objects.  Conscience objects use 0x0.
	//
	recvd, err = stream.Read(reply)
	if err != nil {
		return nil, 0, 0, err
	} else if recvd < 1 {
		return nil, 0, 0, errors.WithStack(ErrProtocol)
	}

	objectType := gitplumbing.ObjectType(reply[0])
	if objectType < 0 || objectType > 7 {
		return nil, 0, 0, errors.WithStack(ErrProtocol)
	}

	objectLen, err := readUint64(stream)
	if err != nil {
		return nil, 0, 0, err
	}

	return stream, objectType, int64(objectLen), nil
}
