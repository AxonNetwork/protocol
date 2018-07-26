package main

import (
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/pkg/errors"
	gitplumbing "gopkg.in/src-d/go-git.v4/plumbing"
	gitconfig "gopkg.in/src-d/go-git.v4/plumbing/format/config"
	gitobject "gopkg.in/src-d/go-git.v4/plumbing/object"
)

func recurseCommit(hash gitplumbing.Hash) error {
	obj, err := fetchAndWriteObject(hash)
	if err != nil {
		return err
	}

	commit := &gitobject.Commit{}
	err = commit.Decode(obj)
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
	objectStream, err := client.GetObject(repoID, hash[:])
	if err != nil {
		return nil, errors.WithStack(err)
	}
	defer objectStream.Close()

	obj := repo.Storer.NewEncodedObject() // returns a &plumbing.MemoryObject{}
	obj.SetType(objectStream.Type())

	w, err := obj.Writer()
	if err != nil {
		return nil, errors.WithStack(err)
	}

	copied, err := io.Copy(w, objectStream)
	if err != nil {
		return nil, errors.WithStack(err)
	} else if copied != objectStream.Len() {
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

func checkConfig() error {
	cfg, err := repo.Config()
	if err != nil {
		return err
	}

	raw := cfg.Raw

	section := raw.Section("conscience")
	if section == nil {
		raw.AddOption("conscience", "", "", "")
	}

	raw.AddOption("conscience", "", "reponame", repoName)

	username := section.Option("username")
	if username == "" {
		raw.AddOption("conscience", "", "username", repoUser)
	}

	cfg.Raw.AddOption("filter", "conscience", "clean", "conscience_encode")
	cfg.Raw.AddOption("filter", "conscience", "smudge", "conscience_decode")

	p := filepath.Join(GIT_DIR, "config")
	f, err := os.OpenFile(p, os.O_WRONLY, os.ModeAppend)
	if err != nil {
		return err
	}
	w := io.Writer(f)

	enc := gitconfig.NewEncoder(w)
	err = enc.Encode(cfg.Raw)
	if err != nil {
		return err
	}

	return nil
}
