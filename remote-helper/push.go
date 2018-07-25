package main

import (
	gitplumbing "gopkg.in/src-d/go-git.v4/plumbing"
	"os"
	"strings"
)

func addRepo() error {
	cwd, err := os.Getwd()
	if err != nil {
		return err
	}
	err = client.AddRepo(cwd)
	return nil
}

func push(src string, dst string) error {
	force := strings.HasPrefix(src, "+")
	if force {
		src = src[1:]
	}
	cwd, err := os.Getwd()
	if err != nil {
		return err
	}
	err = client.AddRepo(cwd)
	if err != nil {
		return err
	}
	ref, err := repo.Reference(gitplumbing.ReferenceName(src), false)
	if err != nil {
		return err
	}
	_, err = client.AddRef(repoID, ref.Strings()[1], dst)
	return err
}
