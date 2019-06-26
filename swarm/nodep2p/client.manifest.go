package nodep2p

import (
	"context"
	"io"
	"time"

	"github.com/libgit2/git2go"
	peer "github.com/libp2p/go-libp2p-peer"
	"github.com/pkg/errors"

	"github.com/Conscience/protocol/log"
	"github.com/Conscience/protocol/repo"
	. "github.com/Conscience/protocol/swarm/wire"
	"github.com/Conscience/protocol/util"
)

type Manifest struct {
	GitObjects   ManifestObjectSet
	ChunkObjects ManifestObjectSet
}

type ManifestObjectSet []ManifestObject

func (s ManifestObjectSet) UncompressedSize() int64 {
	var size int64
	for i := range s {
		size += s[i].UncompressedSize
	}
	return size
}

func (sc *Client) requestManifestFromSwarm(ctx context.Context, commitID git.Oid, checkoutType CheckoutType) (*Manifest, error) {
	c, err := util.CidForString(sc.repoID)
	if err != nil {
		return nil, err
	}

	ctxTimeout, cancel := context.WithTimeout(ctx, time.Duration(sc.config.Node.FindProviderTimeout))
	defer cancel()

	for provider := range sc.node.FindProvidersAsync(ctxTimeout, c, 10) {
		if provider.ID != sc.node.ID() {
			// We found a peer with the object
			manifest, err := sc.requestManifestFromPeer(ctx, provider.ID, commitID, checkoutType)
			if err != nil {
				log.Errorln("[manifest client] requestManifestFromPeer:", err)
				continue
			}
			return manifest, nil
		}
	}
	return nil, errors.Errorf("could not find provider for repo '%v'", sc.repoID)
}

func (sc *Client) requestManifestFromPeer(ctx context.Context, peerID peer.ID, commitID git.Oid, checkoutType CheckoutType) (*Manifest, error) {
	log.Debugf("[manifest client] requesting manifest %v/%v from peer %v", sc.repoID, commitID.String(), peerID.Pretty())

	// Open the stream
	stream, err := sc.node.NewStream(ctx, peerID, MANIFEST_PROTO)
	if err != nil {
		return nil, err
	}

	sig, err := sc.node.SignHash(commitID[:])
	if err != nil {
		return nil, err
	}

	// Write the request packet to the stream
	err = WriteStructPacket(stream, &GetManifestRequest{
		RepoID:       sc.repoID,
		Commit:       commitID,
		Signature:    sig,
		CheckoutType: int(checkoutType),
	})
	if err != nil {
		return nil, err
	}

	// Read the response
	var resp GetManifestResponse
	err = ReadStructPacket(stream, &resp)
	if err != nil {
		return nil, err
	} else if resp.ErrUnauthorized {
		return nil, errors.Wrapf(ErrUnauthorized, "%v:%0x", sc.repoID, commitID)
	} else if resp.ErrMissingCommit {
		return nil, errors.Wrapf(ErrObjectNotFound, "%v:%0x", sc.repoID, commitID)
	}

	log.Debugf("[manifest client] got manifest header %+v", resp)

	manifest := &Manifest{}
	for {
		var obj ManifestObject
		err = ReadStructPacket(stream, &obj)
		if err == io.EOF {
			break
		} else if err != nil {
			return nil, err
		}

		if obj.End == true {
			break
		}

		if len(obj.Hash) == repo.GIT_HASH_LENGTH {
			manifest.GitObjects = append(manifest.GitObjects, obj)
		} else if len(obj.Hash) == repo.CONSCIENCE_HASH_LENGTH {
			manifest.ChunkObjects = append(manifest.ChunkObjects, obj)
		} else {
			log.Errorln("[manifest client] received an oddly sized hash from peer:", obj.Hash)
		}
	}

	return manifest, nil
}
