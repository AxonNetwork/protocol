package nodep2p

import (
	"bufio"
	"bytes"
	"io"
	"os"
	"path/filepath"

	gitplumbing "gopkg.in/src-d/go-git.v4/plumbing"
	gitobject "gopkg.in/src-d/go-git.v4/plumbing/object"

	"github.com/pkg/errors"

	"github.com/Conscience/protocol/log"
	"github.com/Conscience/protocol/repo"
	. "github.com/Conscience/protocol/swarm/wire"
	"github.com/Conscience/protocol/util"
)

func getManifest(r *repo.Repo, commitHash gitplumbing.Hash) ([]ManifestObject, error) {
	objectHashes := make(map[gitplumbing.Hash]bool)
	chunkHashes := make(map[string]bool)

	err := objectHashesForCommit(r, commitHash, objectHashes, chunkHashes)
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

		manifest = append(manifest, ManifestObject{Hash: h[:], UncompressedSize: size})
	}

	// for hash := range chunkHashes {
	// 	p := filepath.Join(r.Path, ".git", repo.CONSCIENCE_DATA_SUBDIR, hash)
	// 	stat, err := os.Stat(p)
	// 	size := stat.Size()

	// 	hex, err := hex.DecodeString(hash)
	// 	if err != nil {
	// 		return nil, err
	// 	}

	// 	var h [repo.CONSCIENCE_HASH_LENGTH]byte
	// 	copy(h[:], hex[:])

	// 	manifest = append(manifest, ManifestObject{Hash: h[:], UncompressedSize: size})
	// }

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

func objectHashesForCommit(r *repo.Repo, commitHash gitplumbing.Hash, seenObj map[gitplumbing.Hash]bool, seenChunks map[string]bool) error {
	stack := []gitplumbing.Hash{commitHash}

	for len(stack) > 0 {
		if seenObj[stack[0]] {
			stack = stack[1:]
			continue
		}

		commit, err := r.CommitObject(stack[0])
		if err != nil {
			return err
		}

		parentHashes := []gitplumbing.Hash{}
		for _, h := range commit.ParentHashes {
			if _, wasSeen := seenObj[h]; !wasSeen {
				parentHashes = append(parentHashes, h)
			}
		}

		stack = append(stack[1:], parentHashes...)

		// Walk the tree for this commit
		tree, err := r.TreeObject(commit.TreeHash)
		if err != nil {
			return err
		}

		walker := gitobject.NewTreeWalker(tree, true, seenObj)

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
			case gitplumbing.TreeObject:
				seenObj[entry.Hash] = true
			case gitplumbing.BlobObject:
				seenObj[entry.Hash] = true
				err := chunkHashesForObj(r, entry.Hash, seenChunks)
				if err != nil {
					log.Printf("[err] error on chunkHashesForObj: %v\n", err)
				}
			default:
				log.Printf("found weird object: %v (%v)\n", entry.Hash.String(), obj.Type())
			}
		}

		seenObj[commit.Hash] = true
		seenObj[commit.TreeHash] = true
	}

	return nil
}

func chunkHashesForObj(r *repo.Repo, hash gitplumbing.Hash, seenChunks map[string]bool) error {
	obj, err := r.Storer.EncodedObject(gitplumbing.BlobObject, hash)
	if err != nil {
		return err
	}

	reader, err := obj.Reader()
	if err != nil {
		return err
	}
	defer reader.Close()

	br := bufio.NewReader(reader)
	header, err := br.Peek(18)
	if err == io.EOF {
		// file is shorter than header
		return nil
	} else if err != nil {
		return err
	}
	if bytes.Compare(header, []byte("CONSCIENCE_ENCODED")) != 0 {
		// not a chunked file
		return nil
	}

	for {
		line, _, err := br.ReadLine()
		if err == io.EOF {
			break
		} else if err != nil {
			return err
		}
		seenChunks[string(line)] = true
	}

	return nil
}
