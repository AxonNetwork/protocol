package main

import (
	"context"
	"io"

	"github.com/pkg/errors"
	gitplumbing "gopkg.in/src-d/go-git.v4/plumbing"
)

func recurseCommit(hash gitplumbing.Hash) error {
	err := fetchAndWriteObject(hash)
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

	return fetchAndWriteObject(commit.TreeHash)
}

func fetchAndWriteObject(hash gitplumbing.Hash) error {
	obj, err := Repo.Object(gitplumbing.AnyObject, hash)
	// The object has already been downloaded
	if err == nil {
		// If the object is a tree, make sure we have its children
		if obj.Type() == gitplumbing.TreeObject {
			return fetchTreeChildren(hash)
		}
		return nil
	}

	// Fetch an object stream from the node via RPC
	// @@TODO: give context a timeout and make it configurable
	objectStream, err := client.GetObject(context.Background(), repoID, hash[:])
	if err != nil {
		return errors.WithStack(err)
	}
	defer objectStream.Close()

	// Write the object to disk
	{
		newobj := Repo.Storer.NewEncodedObject() // returns a &plumbing.MemoryObject{}
		newobj.SetType(objectStream.Type())

		w, err := newobj.Writer()
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
		if hash != newobj.Hash() {
			return errors.Errorf("bad checksum for piece %v", hash.String())
		}

		// Write the object to disk
		_, err = Repo.Storer.SetEncodedObject(newobj)
		if err != nil {
			return errors.WithStack(err)
		}
	}

	// If the object is a tree, fetch its children as well
	if objectStream.Type() == gitplumbing.TreeObject {
		return fetchTreeChildren(hash)
	}
	return nil
}

func fetchTreeChildren(hash gitplumbing.Hash) error {
	tree, err := Repo.TreeObject(hash)
	if err != nil {
		return errors.WithStack(err)
	}

	for _, entry := range tree.Entries {
		err := fetchAndWriteObject(entry.Hash)
		if err != nil {
			return err
		}
	}
	return nil
}
