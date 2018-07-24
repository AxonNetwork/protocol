package swarm

import (
	"context"
	"encoding/hex"

	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"

	peer "gx/ipfs/QmdVrMn1LhB4ybb8hMVaMLXnA8XRSewMnK6YqXKXoTcRvN/go-libp2p-peer"
)

// Opens an outgoing request to another Node for the given object.
func (n *Node) openPeerObjectReader(ctx context.Context, peerID peer.ID, repoID string, objectID []byte) (ObjectReader, error) {
	log.Printf("[stream] requesting object...")

	// Open the stream
	stream, err := n.Host.NewStream(ctx, peerID, OBJECT_STREAM_PROTO)
	if err != nil {
		return nil, err
	}

	// Write the request packet to the stream
	err = writeStructPacket(stream, &GetObjectRequest{RepoID: repoID, ObjectID: objectID})
	if err != nil {
		return nil, err
	}

	// Read the response
	resp := GetObjectResponse{}
	err = readStructPacket(stream, &resp)
	if err != nil {
		return nil, err
	} else if !resp.HasObject {
		return nil, errors.New("peer doesn't have object " + repoID + ":" + hex.EncodeToString(objectID))
	}

	or := objectReader{
		Reader:     stream,
		Closer:     stream,
		objectType: resp.ObjectType,
		objectLen:  resp.ObjectLen,
	}
	return or, nil
}
