package main

import (
	"fmt"
	"io"

	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	gitplumbing "gopkg.in/src-d/go-git.v4/plumbing"
	gitobject "gopkg.in/src-d/go-git.v4/plumbing/object"
)

func recurseCommit(hash gitplumbing.Hash) error {
	log.Printf("recurseCommit %v", hash.String())
	obj, err := fetchAndWriteObject(hash)
	if err != nil {
		return err
	}

	// commit, err := repo.CommitObject(hash)
	// if err != nil {
	//  return errors.WithStack(err)
	// }
	commit := &gitobject.Commit{}
	commit.Decode(obj)

	if commit.NumParents() > 0 {
		for _, phash := range commit.ParentHashes {
			err := recurseCommit(phash)
			if err != nil {
				return errors.WithStack(err)
			}
		}
	}

	return fetchTree(commit.TreeHash)
}

func fetchTree(hash gitplumbing.Hash) error {
	log.Printf("fetchTree %v", hash.String())
	_, err := fetchAndWriteObject(hash)
	if err != nil {
		return err
	}

	tIter, err := repo.TreeObject(hash)
	if err != nil {
		return errors.WithStack(err)
	}

	for _, entry := range tIter.Entries {
		_, err := fetchAndWriteObject(entry.Hash)
		if err != nil {
			return err
		}
	}

	return nil
}

func fetchAndWriteObject(hash gitplumbing.Hash) (gitplumbing.EncodedObject, error) {
	log.Printf("fetchAndWriteObject %v", hash.String())
	objectStream, objectType, objectLen, err := client.GetObject(repoID, hash[:])
	if err != nil {
		return nil, errors.WithStack(err)
	}
	defer objectStream.Close()

	obj := repo.Storer.NewEncodedObject() // returns a &plumbing.MemoryObject{}
	obj.SetType(objectType)

	w, err := obj.Writer()
	if err != nil {
		return nil, errors.WithStack(err)
	}

	copied, err := io.Copy(w, objectStream)
	if err != nil {
		return nil, errors.WithStack(err)
	} else if copied != objectLen {
		return nil, errors.WithStack(fmt.Errorf("object stream bad length"))
	}

	err = w.Close()
	if err != nil {
		return nil, errors.WithStack(err)
	}

	// Check the checksum
	if hash != obj.Hash() {
		return nil, errors.WithStack(fmt.Errorf("bad checksum for piece %v", hash.String()))
	}

	// Write the object to disk
	_, err = repo.Storer.SetEncodedObject(obj)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	return obj, nil
}
