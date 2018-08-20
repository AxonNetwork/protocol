package main

import (
	"strings"

	gitplumbing "gopkg.in/src-d/go-git.v4/plumbing"
)

func push(srcRefName string, destRefName string) error {
	force := strings.HasPrefix(srcRefName, "+")
	if force {
		srcRefName = srcRefName[1:]
	}

	// Tell the node to announce the new content so that replicator nodes can find and pull it.
	err := client.AnnounceRepoContent(repoID)
	if err != nil {
		return err
	}

	srcRef, err := Repo.Reference(gitplumbing.ReferenceName(srcRefName), false)
	if err != nil {
		return err
	}

	commitHash := srcRef.Hash().String()

	err = client.UpdateRef(repoID, destRefName, commitHash)
	if err != nil {
		return err
	}

	err = client.RequestReplication(repoID)
	return err
}
