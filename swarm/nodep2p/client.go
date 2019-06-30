package nodep2p

import (
	"context"
	"sync"

	"github.com/libgit2/git2go"
	"github.com/pkg/errors"

	"github.com/Conscience/protocol/repo"
)

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

func (h *Host) FetchFromCommit(ctx context.Context, r *repo.Repo, commitID git.Oid, checkoutType CheckoutType, remote *git.Remote) (<-chan MaybeFetchFromCommitPacket, *Manifest, error) {
	chOut := make(chan MaybeFetchFromCommitPacket)

	repoID, err := r.RepoID()
	if err != nil {
		return nil, nil, err
	}

	manifest, err := h.requestManifestFromSwarm(ctx, repoID, commitID, checkoutType)
	if err != nil {
		return nil, nil, err
	}

	// Filter objects we already have out of the manifest
	var missingGitObjects ManifestObjectSet
	var missingChunkObjects ManifestObjectSet
	if r != nil {
		for i := range manifest.GitObjects {
			if !r.HasObject(manifest.GitObjects[i].Hash) {
				missingGitObjects = append(missingGitObjects, manifest.GitObjects[i])
			}
		}
		for i := range manifest.ChunkObjects {
			if !r.HasObject(manifest.ChunkObjects[i].Hash) {
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
		if cb := getCheckManifestCallback(remote); cb != nil {
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
		for pkt := range h.FetchGitPackfiles(ctx, repoID, manifest.GitObjects) {
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
		for pkt := range h.FetchChunks(ctx, repoID, manifest.ChunkObjects.ToHashes()) {
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

func (h *Host) FetchChunksFromCommit(ctx context.Context, r *repo.Repo, commitID git.Oid, checkoutType CheckoutType, remote *git.Remote) (<-chan MaybeChunk, *Manifest, error) {
	chOut := make(chan MaybeChunk)

	repoID, err := r.RepoID()
	if err != nil {
		return nil, nil, err
	}

	manifest, err := h.requestManifestFromSwarm(ctx, repoID, commitID, checkoutType)
	if err != nil {
		return nil, nil, err
	}

	// Filter objects we already have out of the manifest
	var missingChunkObjects ManifestObjectSet
	if r != nil {
		for i := range manifest.ChunkObjects {
			if !r.HasObject(manifest.ChunkObjects[i].Hash) {
				missingChunkObjects = append(missingChunkObjects, manifest.ChunkObjects[i])
			}
		}
	} else {
		missingChunkObjects = manifest.ChunkObjects
	}
	manifest.ChunkObjects = missingChunkObjects

	// Allow the caller to respond to the contents of the manifest prior to initiating the download.
	// This is how we implement replication policies.
	if cb := getCheckManifestCallback(remote); cb != nil {
		err = cb(manifest)
		if err != nil {
			return nil, nil, err
		}
	}

	go func() {
		defer close(chOut)

		for packet := range h.FetchChunks(ctx, repoID, manifest.ChunkObjects.ToHashes()) {
			select {
			case chOut <- packet:
			case <-ctx.Done():
				return
			}
		}
	}()

	return chOut, manifest, nil
}
