package main

import (
	"fmt"
	"io"
	"strings"
	"path/filepath"
	"os"

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
		return err
	}

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
	objectStream, err := client.GetObject(repoID, hash[:])
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
	} else if copied != objectStream.Len() {
		return errors.WithStack(fmt.Errorf("object stream bad length"))
	}

	err = w.Close()
	if err != nil {
		return errors.WithStack(err)
	}

	// Check the checksum
	if hash != obj.Hash() {
		return errors.WithStack(fmt.Errorf("bad checksum for piece %v", hash.String()))
	}

	// Write the object to disk
	_, err = Repo.Storer.SetEncodedObject(obj)
	if err != nil {
		return errors.WithStack(err)
	}
	return nil
}

func addRepoToNode() error {
	cwd, err := os.Getwd()
	if err != nil{
		return err
	}
	repoName := strings.Split(repoID, "/")[1]
	repoFolder := filepath.Join(cwd, repoName)
	return client.AddRepo(repoFolder)
}
