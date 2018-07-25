package main

import (
	log "github.com/sirupsen/logrus"
	gitplumbing "gopkg.in/src-d/go-git.v4/plumbing"
	"os"
	"strings"
)

func addRepo() error {
	cwd, err := os.Getwd()
	if err != nil {
		return err
	}
	added, err := client.AddRepo(cwd)
	log.Printf("added: %v", added)
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
	_, err = client.AddRepo(cwd)
	ref, err := repo.Reference(gitplumbing.ReferenceName(src), false)
	_, err = client.AddRef(repoID, ref.Strings()[1], dst)
	return err
}
