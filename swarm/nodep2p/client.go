package nodep2p

import (
	"context"
	"sync"

	"github.com/libgit2/git2go"
	"github.com/pkg/errors"

	"github.com/Conscience/protocol/config"
	"github.com/Conscience/protocol/repo"
	. "github.com/Conscience/protocol/swarm/wire"
)

type Client struct {
	node   INode
	config *config.Config
	repo   *repo.Repo
	repoID string
}

type MaybeFetchFromCommitPacket struct {
	*PackfileHeader
	*PackfileData
	*Chunk
	Error error
}

type PackfileHeader struct {
	PackfileID       []byte
	UncompressedSize int64
}

type PackfileData struct {
	PackfileID []byte
	Data       []byte
	End        bool
}

var ErrFetchingFromPeer = errors.New("fetching from peer")

func NewClient(node INode, repoID string, repoPath string, config *config.Config) *Client {
	r, _ := node.RepoAtPathOrID(repoPath, repoID)

	sc := &Client{
		node:   node,
		config: config,
		repo:   r,
		repoID: repoID,
	}
	return sc
}

func (sc *Client) FetchFromCommit(ctx context.Context, commitID git.Oid, checkoutType CheckoutType, remote *git.Remote) (<-chan MaybeFetchFromCommitPacket, *Manifest, error) {
	chOut := make(chan MaybeFetchFromCommitPacket)

	manifest, err := sc.requestManifestFromSwarm(ctx, commitID, checkoutType)
	if err != nil {
		return nil, nil, err
	}

	// Filter objects we already have out of the manifest
	var missingGitObjects ManifestObjectSet
	var missingChunkObjects ManifestObjectSet
	if sc.repo != nil {
		for i := range manifest.GitObjects {
			if !sc.repo.HasObject(manifest.GitObjects[i].Hash) {
				missingGitObjects = append(missingGitObjects, manifest.GitObjects[i])
			}
		}
		for i := range manifest.ChunkObjects {
			if !sc.repo.HasObject(manifest.ChunkObjects[i].Hash) {
				missingChunkObjects = append(missingChunkObjects, manifest.ChunkObjects[i])
			}
		}
	} else {
		missingGitObjects = manifest.GitObjects
		missingChunkObjects = manifest.ChunkObjects
	}
	manifest.GitObjects = missingGitObjects
	manifest.ChunkObjects = missingChunkObjects

	// Allow the caller to respond to the contents of the manifest prior to initiating the download.
	// This is how we implement replication policies.
	if remote != nil {
		if cb := GetCheckManifestCallback(remote); cb != nil {
			err = cb(manifest)
			if err != nil {
				return nil, nil, err
			}
		}
	}

	wg := &sync.WaitGroup{}

	wg.Add(1)
	go func() {
		defer wg.Done()
		for pkt := range sc.FetchGitPackfiles(ctx, manifest.GitObjects) {
			select {
			case chOut <- pkt:
			case <-ctx.Done():
				return
			}
		}
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		chunks := ManifestObjectsToHashes(manifest.ChunkObjects)
		for pkt := range sc.FetchChunks(ctx, chunks) {
			var toSend MaybeFetchFromCommitPacket
			if pkt.Error != nil {
				toSend.Error = pkt.Error
			} else {
				toSend.Chunk = pkt.Chunk
			}

			select {
			case chOut <- toSend:
			case <-ctx.Done():
				return
			}
		}
	}()

	go func() {
		wg.Wait()
		close(chOut)
	}()

	return chOut, manifest, nil
}

func ManifestObjectsToHashes(objects ManifestObjectSet) [][]byte {
	hashes := make([][]byte, 0)
	for _, obj := range objects {
		hashes = append(hashes, obj.Hash)
	}
	return hashes
}

func (sc *Client) FetchChunksFromCommit(ctx context.Context, commitID git.Oid, checkoutType CheckoutType, remote *git.Remote) (<-chan MaybeChunk, *Manifest, error) {
	chOut := make(chan MaybeChunk)

	manifest, err := sc.requestManifestFromSwarm(ctx, commitID, checkoutType)
	if err != nil {
		return nil, nil, err
	}

	// Filter objects we already have out of the manifest
	var missingChunkObjects ManifestObjectSet
	if sc.repo != nil {
		for i := range manifest.ChunkObjects {
			if !sc.repo.HasObject(manifest.ChunkObjects[i].Hash) {
				missingChunkObjects = append(missingChunkObjects, manifest.ChunkObjects[i])
			}
		}
	} else {
		missingChunkObjects = manifest.ChunkObjects
	}
	manifest.ChunkObjects = missingChunkObjects

	// Allow the caller to respond to the contents of the manifest prior to initiating the download.
	// This is how we implement replication policies.
	if cb := GetCheckManifestCallback(remote); cb != nil {
		err = cb(manifest)
		if err != nil {
			return nil, nil, err
		}
	}

	go func() {
		defer close(chOut)

		chunks := ManifestObjectsToHashes(manifest.ChunkObjects)
		for packet := range sc.FetchChunks(ctx, chunks) {
			select {
			case chOut <- packet:
			case <-ctx.Done():
				return
			}
		}
	}()

	return chOut, manifest, nil
}
