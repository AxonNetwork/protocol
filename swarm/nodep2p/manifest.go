package nodep2p

import (
	"io"
	"os"
	"path/filepath"

	gitplumbing "gopkg.in/src-d/go-git.v4/plumbing"
	gitobject "gopkg.in/src-d/go-git.v4/plumbing/object"
	gitioutil "gopkg.in/src-d/go-git.v4/utils/ioutil"

	"github.com/Conscience/protocol/log"
	"github.com/Conscience/protocol/repo"
	. "github.com/Conscience/protocol/swarm/wire"
	"github.com/pkg/errors"
)

func getManifest(r *repo.Repo) ([]ManifestObject, error) {

	// Build the manifest

	head, err := r.Head()
	if err != nil {
		return nil, err
	}

	objectHashes := make(map[gitplumbing.Hash]bool)
	err = objectHashesForCommit(r, head.Hash(), objectHashes)
	if err != nil {
		return nil, err
	}

	manifest := []ManifestObject{}
	for hash := range objectHashes {
		size, err := r.Storer.EncodedObjectSize(hash)
		if err != nil {
			return nil, err
		}

		// If we don't copy the hash here, they all end up being the same
		var h gitplumbing.Hash
		copy(h[:], hash[:])

		manifest = append(manifest, ManifestObject{Hash: h[:], Size: size})
	}

	return manifest, nil
}

func getCachedManifest(repoID string) []ManifestObject {
	// return nil
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
	defer gitioutil.CheckClose(f, &err)

	for i := range manifest {
		err = WriteStructPacket(f, &manifest[i])
		if err != nil {
			return
		}
	}
	return
}

func objectHashesForCommit(r *repo.Repo, commitHash gitplumbing.Hash, seen map[gitplumbing.Hash]bool) error {
	stack := []gitplumbing.Hash{commitHash}

	for len(stack) > 0 {
		if seen[stack[0]] {
			stack = stack[1:]
			continue
		}

		commit, err := r.CommitObject(stack[0])
		if err != nil {
			return err
		}

		parentHashes := []gitplumbing.Hash{}
		for _, h := range commit.ParentHashes {
			if _, wasSeen := seen[h]; !wasSeen {
				parentHashes = append(parentHashes, h)
			}
		}

		stack = append(stack[1:], parentHashes...)

		// Walk the tree for this commit
		tree, err := r.TreeObject(commit.TreeHash)
		if err != nil {
			return err
		}

		walker := gitobject.NewTreeWalker(tree, true, seen)

		for {
			_, entry, err := walker.Next()
			if err == io.EOF {
				walker.Close()
				break
			} else if err != nil {
				walker.Close()
				return err
			}
			obj, err := r.Object(gitplumbing.AnyObject, entry.Hash)
			if err != nil {
				log.Printf("[err] error on r.Object: %v\n", err)
				continue
			}
			switch obj.Type() {
			case gitplumbing.TreeObject, gitplumbing.BlobObject:
				seen[entry.Hash] = true
			default:
				log.Printf("found weird object: %v (%v)\n", entry.Hash.String(), obj.Type())
			}
		}

		seen[commit.Hash] = true
		seen[commit.TreeHash] = true
	}

	return nil
}
