package swarm

import (
	"context"
	"io"

	"github.com/pkg/errors"

	peer "gx/ipfs/QmdVrMn1LhB4ybb8hMVaMLXnA8XRSewMnK6YqXKXoTcRvN/go-libp2p-peer"

	"github.com/Conscience/protocol/log"
	. "github.com/Conscience/protocol/swarm/wire"
	"github.com/Conscience/protocol/util"
)

func (n *Node) handshake(ctx context.Context, peerID peer.ID, repoID string) (*util.ObjectReader, error) {

	return nil, nil
}

func (n *Node) requestManifest(ctx context.Context, peerID peer.ID, repoID string, commit string) (*util.ObjectReader, error) {
	log.Debugf("[p2p object client] requesting manifest %v/%v from peer %v", repoID, commit, peerID.Pretty())

	// Open the stream
	stream, err := n.host.NewStream(ctx, peerID, MANIFEST_PROTO)
	if err != nil {
		return nil, err
	}

	sig, err := n.eth.SignHash([]byte(commit))
	if err != nil {
		return nil, err
	}

	// Write the request packet to the stream
	err = WriteStructPacket(stream, &GetManifestRequest{RepoID: repoID, Commit: commit, Signature: sig})
	if err != nil {
		return nil, err
	}

	// // Read the response
	resp := GetManifestResponse{}
	err = ReadStructPacket(stream, &resp)
	if err != nil {
		return nil, err
	} else if !resp.Authorized {
		return nil, errors.Wrapf(ErrUnauthorized, "%v:%0x", repoID, commit)
	} else if !resp.HasCommit {
		return nil, errors.Wrapf(ErrObjectNotFound, "%v:%0x", repoID, commit)
	}

	log.Debugf("[p2p object client] got manifest metadata %+v", resp)

	flatHead := make([]byte, resp.HeadLen)
	_, err = io.ReadFull(stream, flatHead)
	if err != nil {
		return nil, err
	}

	flatHistory := make([]byte, resp.HistoryLen)
	_, err = io.ReadFull(stream, flatHistory)
	if err != nil {
		return nil, err
	}

	return nil, nil

	// or := &util.ObjectReader{
	// 	Reader:     stream,
	// 	Closer:     stream,
	// 	ObjectType: resp.ObjectType,
	// 	ObjectLen:  resp.ObjectLen,
	// }
	// return or, nil

}
