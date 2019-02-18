package nodep2p

import (
	"io"
	"os"
	"path/filepath"

	"github.com/libgit2/git2go"

	"github.com/pkg/errors"

	"github.com/Conscience/protocol/log"
	"github.com/Conscience/protocol/repo"
	. "github.com/Conscience/protocol/swarm/wire"
	"github.com/Conscience/protocol/util"
)

func getManifest(r *repo.Repo, commitHash git.Oid) ([]ManifestObject, error) {
	objectHashes := make(map[git.Oid]bool)
	err := objectHashesForCommit(r, commitHash, objectHashes)
	if err != nil {
		return nil, err
	}

	odb, err := r.Odb()
	if err != nil {
		return nil, err
	}

	manifest := []ManifestObject{}
	for hash := range objectHashes {
		obj, err := odb.Read(&hash)
		if err != nil {
			return nil, err
		}

		// If we don't copy the hash here, they all end up being the same
		var h git.Oid
		copy(h[:], hash[:])

		manifest = append(manifest, ManifestObject{Hash: h[:], UncompressedSize: int64(obj.Len())})
	}

	return manifest, nil
}

func getCachedManifest(repoID string) []ManifestObject {
	cacheDir := filepath.Join(os.TempDir(), "conscience-manifest-cache")
	err := os.MkdirAll(cacheDir, os.ModePerm)
	if err != nil {
		log.Errorln("[repo] getCachedManifest:", err)
		return nil
	}

	f, err := os.Open(filepath.Join(cacheDir, repoID))
	if os.IsNotExist(err) {
		return nil
	} else if err != nil {
		log.Errorln("[repo] getCachedManifest:", err)
		return nil
	}

	manifest := []ManifestObject{}
	for {
		obj := ManifestObject{}
		err = ReadStructPacket(f, &obj)
		if errors.Cause(err) == io.EOF || errors.Cause(err) == io.ErrUnexpectedEOF {
			break

		} else if err != nil {
			log.Errorln("[repo] getCachedManifest:", err)
			return nil
		}
		manifest = append(manifest, obj)
	}

	log.Infoln("using cached manifest")
	return manifest
}

func createCachedManifest(repoID string, manifest []ManifestObject) (err error) {
	log.Infoln("creating cached manifest")

	cacheDir := filepath.Join(os.TempDir(), "conscience-manifest-cache")
	err = os.MkdirAll(cacheDir, os.ModePerm)
	if err != nil {
		return
	}

	f, err := os.Create(filepath.Join(cacheDir, repoID))
	if err != nil {
		return
	}
	defer util.CheckClose(f, &err)

	for i := range manifest {
		err = WriteStructPacket(f, &manifest[i])
		if err != nil {
			return
		}
	}
	return
}

func objectHashesForCommit(r *repo.Repo, commitHash git.Oid, seen map[git.Oid]bool) error {
	stack := []git.Oid{commitHash}

	for len(stack) > 0 {
		if seen[stack[0]] {
			stack = stack[1:]
			continue
		}

		commit, err := r.LookupCommit(&stack[0])
		if err != nil {
			return err
		}

		parentCount := commit.ParentCount()
		parentHashes := []git.Oid{}
		for i := uint(0); i < parentCount; i++ {
			hash := commit.ParentId(i)
			if _, wasSeen := seen[*hash]; !wasSeen {
				parentHashes = append(parentHashes, *hash)
			}
		}

		stack = append(stack[1:], parentHashes...)

		tree, err := commit.Tree()
		if err != nil {
			return err
		}

		var innerErr error
		err = tree.Walk(func(name string, entry *git.TreeEntry) int {
			obj, innerErr := r.Lookup(entry.Id)
			if innerErr != nil {
				return -1
			}

			switch obj.Type() {
			case git.ObjectTree, git.ObjectBlob:
				seen[*entry.Id] = true
			default:
				log.Printf("found weird object: %v (%v)\n", entry.Id.String(), obj.Type().String())
			}

			return 0
		})

		if err != nil {
			return err
		} else if innerErr != nil {
			return innerErr
		}

		seen[*commit.Id()] = true
		seen[*commit.TreeId()] = true
	}

	return nil
}
