package main

import (
	"fmt"
	"io"
	"os"
	"time"

	gitplumbing "gopkg.in/src-d/go-git.v4/plumbing"
	gitobject "gopkg.in/src-d/go-git.v4/plumbing/object"

	"github.com/Conscience/protocol/repo"
	. "github.com/Conscience/protocol/swarm/wire"
)

func main() {
	r1, err := repo.Open(os.Args[1])
	if err != nil {
		panic(err)
	}

	start := time.Now()
	getManifest(r1)
	total1 := time.Now().Sub(start)
	fmt.Println(os.Args[1], "total =", total1)

	var total2 time.Duration
	if len(os.Args) > 2 {
		r2, err := repo.Open(os.Args[2])
		if err != nil {
			panic(err)
		}

		start = time.Now()
		getManifest(r2)
		total2 := time.Now().Sub(start)
		fmt.Println(os.Args[2], "total =", total2)
	}

	fmt.Println("packed =", total1)
	if len(os.Args) > 2 {
		fmt.Println("unpacked =", total2)
	}
}

func getManifest(r *repo.Repo) ([]ManifestObject, error) {
	// repoID, err := r.RepoID()
	// if err != nil {
	//  return nil, err
	// }

	// cached := getCachedManifest(repoID)
	// if cached != nil {
	//     return cached, nil
	// }

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
		if len(parentHashes) > 0 {
			if len(parentHashes) == 1 {
				// fmt.Printf("  - pushing %v new commits (len = %v)\n", len(parentHashes), len(stack))
			} else {
				// fmt.Printf("  - pushing %v new commits (len = %v)\n", len(parentHashes), len(stack))
			}
		}

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
				fmt.Printf("[err] error on r.Object: %v\n", err)
				continue
			}
			switch obj.Type() {
			case gitplumbing.TreeObject, gitplumbing.BlobObject:
				seen[entry.Hash] = true
			default:
				fmt.Printf("found weird object: %v (%v)\n", entry.Hash.String(), obj.Type())
			}
		}

		seen[commit.Hash] = true
		seen[commit.TreeHash] = true
	}

	return nil
}
