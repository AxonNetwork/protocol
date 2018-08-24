package swarm

import (
	"context"

	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"

	peer "gx/ipfs/QmdVrMn1LhB4ybb8hMVaMLXnA8XRSewMnK6YqXKXoTcRvN/go-libp2p-peer"

	"../util"
	. "./wire"
)

// Opens an outgoing request to another Node for the given object.
func (n *Node) requestObject(ctx context.Context, peerID peer.ID, repoID string, objectID []byte) (*util.ObjectReader, error) {
	log.Debugf("[p2p object client] requesting object %v/%0x from peer %v", repoID, objectID, peerID.Pretty())

	// Open the stream
	stream, err := n.host.NewStream(ctx, peerID, OBJECT_PROTO)
	if err != nil {
		return nil, err
	}

	sig, err := n.eth.SignHash(objectID)
	if err != nil {
		return nil, err
	}

	// Write the request packet to the stream
	err = WriteStructPacket(stream, &GetObjectRequestSigned{RepoID: repoID, ObjectID: objectID, Signature: sig})
	if err != nil {
		return nil, err
	}

	// Read the response
	resp := GetObjectResponse{}
	err = ReadStructPacket(stream, &resp)
	if err != nil {
		return nil, err
	} else if resp.Unauthorized {
		return nil, errors.Wrapf(ErrUnauthorized, "%v:%0x", repoID, objectID)
	} else if !resp.HasObject {
		return nil, errors.Wrapf(ErrObjectNotFound, "%v:%0x", repoID, objectID)
	}

	log.Debugf("[p2p object client] got object metadata %+v", resp)

	or := &util.ObjectReader{
		Reader:     stream,
		Closer:     stream,
		ObjectType: resp.ObjectType,
		ObjectLen:  resp.ObjectLen,
	}
	return or, nil
}
