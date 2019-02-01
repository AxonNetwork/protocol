package nodep2p

import (
	"context"
	"io"
	"time"

	peer "github.com/libp2p/go-libp2p-peer"
	"github.com/pkg/errors"
	gitplumbing "gopkg.in/src-d/go-git.v4/plumbing"

	"github.com/Conscience/protocol/log"
	"github.com/Conscience/protocol/repo"
	. "github.com/Conscience/protocol/swarm/wire"
	"github.com/Conscience/protocol/util"
)

func (sc *SmartClient) GetManifest(ctx context.Context, commit gitplumbing.Hash, checkoutType CheckoutType) ([]ManifestObject, []ManifestObject, int64, error) {
	manifest, err := sc.requestManifestFromSwarm(ctx, commit, checkoutType)
	if err != nil {
		return nil, nil, 0, err
	}

	// If we're pulling (instead of cloning), filter objects we already have
	if sc.repo != nil {
		filteredManifest := []ManifestObject{}
		for i := range manifest {
			if !sc.repo.HasObject(manifest[i].Hash) {
				filteredManifest = append(filteredManifest, manifest[i])
			}
		}
		manifest = filteredManifest
	}

	// Split the manifest into git objects and conscience chunks
	gitObjects := []ManifestObject{}
	chunkObjects := []ManifestObject{}
	// Calculate the uncompressed size of the entire tree of commits & chunks that will be transferred.
	var uncompressedSize int64
	for _, obj := range manifest {
		uncompressedSize += obj.UncompressedSize
		if len(obj.Hash) == repo.GIT_HASH_LENGTH {
			gitObjects = append(gitObjects, obj)
		} else if len(obj.Hash) == repo.CONSCIENCE_HASH_LENGTH {
			chunkObjects = append(chunkObjects, obj)
		} else {
			log.Errorln("[manifest clients] received an oddly sized hash from peer")
		}
	}

	return gitObjects, chunkObjects, uncompressedSize, nil
}

func (sc *SmartClient) requestManifestFromSwarm(ctx context.Context, commit gitplumbing.Hash, checkoutType CheckoutType) ([]ManifestObject, error) {
	c, err := util.CidForString(sc.repoID)
	if err != nil {
		return nil, err
	}

	ctxTimeout, cancel := context.WithTimeout(ctx, time.Duration(sc.config.Node.FindProviderTimeout))
	defer cancel()

	for provider := range sc.node.FindProvidersAsync(ctxTimeout, c, 10) {
		if provider.ID != sc.node.ID() {
			// We found a peer with the object
			manifest, err := sc.requestManifestFromPeer(ctx, provider.ID, commit, checkoutType)
			if err != nil {
				log.Errorln("[packfile client] requestManifestFromPeer:", err)
				continue
			}
			return manifest, nil
		}
	}
	return nil, errors.Errorf("could not find provider for repo '%v'", sc.repoID)
}

func (sc *SmartClient) requestManifestFromPeer(ctx context.Context, peerID peer.ID, commit gitplumbing.Hash, checkoutType CheckoutType) ([]ManifestObject, error) {
	log.Debugf("[p2p object client] requesting manifest %v/%v from peer %v", sc.repoID, commit, peerID.Pretty())

	// Open the stream
	stream, err := sc.node.NewStream(ctx, peerID, MANIFEST_PROTO)
	if err != nil {
		return nil, err
	}

	sig, err := sc.node.SignHash(commit[:])
	if err != nil {
		return nil, err
	}

	// Write the request packet to the stream
	err = WriteStructPacket(stream, &GetManifestRequest{
		RepoID:       sc.repoID,
		Commit:       commit,
		Signature:    sig,
		CheckoutType: int(checkoutType),
	})
	if err != nil {
		return nil, err
	}

	// Read the response
	resp := GetManifestResponse{}
	err = ReadStructPacket(stream, &resp)
	if err != nil {
		return nil, err
	} else if resp.ErrUnauthorized {
		return nil, errors.Wrapf(ErrUnauthorized, "%v:%0x", sc.repoID, commit)
	} else if resp.ErrMissingCommit {
		return nil, errors.Wrapf(ErrObjectNotFound, "%v:%0x", sc.repoID, commit)
	}

	log.Debugf("[p2p object client] got manifest metadata %+v", resp)

	manifest := make([]ManifestObject, 0)
	for {
		obj := ManifestObject{}
		err = ReadStructPacket(stream, &obj)
		if err == io.EOF {
			break
		} else if err != nil {
			return nil, err
		}
		if obj.End == true {
			break
		}
		manifest = append(manifest, obj)
	}

	return manifest, nil
}
