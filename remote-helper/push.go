package main

import (
	"context"
	"strings"

	gitplumbing "gopkg.in/src-d/go-git.v4/plumbing"
)

func push(srcRefName string, destRefName string) error {
	force := strings.HasPrefix(srcRefName, "+")
	if force {
		srcRefName = srcRefName[1:]
	}

	// Tell the node to announce the new content so that replicator nodes can find and pull it.
	// @@TODO: give context a timeout and make it configurable
	err := client.AnnounceRepoContent(context.Background(), repoID)
	if err != nil {
		return err
	}

	srcRef, err := Repo.Reference(gitplumbing.ReferenceName(srcRefName), false)
	if err != nil {
		return err
	}

	commitHash := srcRef.Hash().String()

	// @@TODO: give context a timeout and make it configurable
	err = client.UpdateRef(context.Background(), repoID, destRefName, commitHash)
	if err != nil {
		return err
	}

	// @@TODO: give context a timeout and make it configurable
	err = client.RequestReplication(context.Background(), repoID)
	return err
}
