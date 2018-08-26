package main

import (
	"context"
	"io"

	"github.com/pkg/errors"
	gitplumbing "gopkg.in/src-d/go-git.v4/plumbing"
)

func recurseCommit(hash gitplumbing.Hash) error {
	err := fetchAndWriteObject(gitplumbing.CommitObject, hash)
	if err != nil {
		return err
	}

	commit, err := Repo.CommitObject(hash)
	if err != nil {
		return errors.WithStack(err)
	}

	if commit.NumParents() > 0 {
		for _, phash := range commit.ParentHashes {
			err := recurseCommit(phash)
			if err != nil {
				return err
			}
		}
	}

	return fetchTree(commit.TreeHash)
}

func fetchTree(hash gitplumbing.Hash) error {
	err := fetchAndWriteObject(gitplumbing.TreeObject, hash)
	if err != nil {
		return err
	}

	tIter, err := Repo.TreeObject(hash)
	if err != nil {
		return errors.WithStack(err)
	}

	for _, entry := range tIter.Entries {
		err := fetchAndWriteObject(gitplumbing.BlobObject, entry.Hash)
		if err != nil {
			return err
		}
	}

	return nil
}

func fetchAndWriteObject(objType gitplumbing.ObjectType, hash gitplumbing.Hash) error {
	_, err := Repo.Object(objType, hash)
	if err == nil {
		// already downloaded
		return nil
	}
	// @@TODO: give context a timeout and make it configurable
	objectStream, err := client.GetObject(context.Background(), repoID, hash[:])
	if err != nil {
		return errors.WithStack(err)
	}
	defer objectStream.Close()

	obj := Repo.Storer.NewEncodedObject() // returns a &plumbing.MemoryObject{}
	obj.SetType(objectStream.Type())

	w, err := obj.Writer()
	if err != nil {
		return errors.WithStack(err)
	}

	copied, err := io.Copy(w, objectStream)
	if err != nil {
		return errors.WithStack(err)
	} else if uint64(copied) != objectStream.Len() {
		return errors.Errorf("object stream bad length (copied: %v, object length: %v)", copied, objectStream.Len())
	}

	err = w.Close()
	if err != nil {
		return errors.WithStack(err)
	}

	// Check the checksum
	if hash != obj.Hash() {
		return errors.Errorf("bad checksum for piece %v", hash.String())
	}

	// Write the object to disk
	_, err = Repo.Storer.SetEncodedObject(obj)
	if err != nil {
		return errors.WithStack(err)
	}
	return nil
}
