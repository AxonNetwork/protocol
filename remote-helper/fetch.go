package main

import (
	"encoding/hex"
	// "fmt"
	"github.com/cryptix/exp/git"
	"os"
	"path/filepath"
)

func recurseCommit(hash string) error {
	obj, err := fetchAndWriteObject(hash)
	if err != nil {
		return err
	}
	commit, _ := obj.Commit()

	if commit.Parent != "" {
		recurseCommit(commit.Parent)
	}

	fetchTree(commit.Tree)
	return nil
}

func fetchTree(hash string) error {
	obj, err := fetchAndWriteObject(hash)
	if err != nil {
		return err
	}
	entries, _ := obj.Tree()

	for _, t := range entries {
		_, err := fetchAndWriteObject(t.SHA1Sum.String())
		if err != nil {
			return err
		}
	}

	return nil
}

func fetchAndWriteObject(hash string) (*git.Object, error) {
	objectID, err := hex.DecodeString(hash)
	if err != nil {
		return nil, err
	}

	in := GetObjectInput{
		repoID,
		objectID,
	}
	out := GetObjectOutput{}
	err = client.Call("Node.GetObject", in, &out)
	if err != nil {
		return nil, err
	}

	p := filepath.Join(repoPath, ".git", "objects", hash[:2], hash[2:])
	f, err := os.Open(p)
	if err != nil {
		return nil, err
	}

	obj, err := git.DecodeObject(f)
	if err != nil {
		return nil, err
	}

	return obj, nil
}

type GetObjectInput struct {
	RepoID   string
	ObjectID []byte
}

type GetObjectOutput struct{}
