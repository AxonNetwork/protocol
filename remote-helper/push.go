package main

import (
	"strings"

	log "github.com/sirupsen/logrus"
	gitplumbing "gopkg.in/src-d/go-git.v4/plumbing"
)

func push(srcRefName string, destRefName string) error {
	force := strings.HasPrefix(srcRefName, "+")
	if force {
		srcRefName = srcRefName[1:]
	}

	log.Printf("Why is this being called?")
	err := client.AnnounceRepoContent(repoID)
	if err != nil {
		return err
	}

	srcRef, err := Repo.Reference(gitplumbing.ReferenceName(srcRefName), false)
	if err != nil {
		return err
	}

	commitHash := srcRef.Hash().String()

	log.Printf("Updating Ref On-chain")
	err = client.UpdateRef(repoID, destRefName, commitHash)
	if err != nil {
		return err
	}

	err = client.RequestPull(repoID)
	return err
}
