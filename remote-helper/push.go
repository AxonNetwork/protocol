package main

import (
	"context"
	"log"
	"strings"
	"time"

	"github.com/pkg/errors"
	gitplumbing "gopkg.in/src-d/go-git.v4/plumbing"
)

func push(srcRefName string, destRefName string) error {
	force := strings.HasPrefix(srcRefName, "+")
	if force {
		srcRefName = srcRefName[1:]
	}

	// Tell the node to announce the new content so that replicator nodes can find and pull it.
	// @@TODO: give context a timeout and make it configurable
	ctx, cancel1 := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel1()
	err := client.AnnounceRepoContent(ctx, repoID)
	if err != nil {
		return err
	}

	srcRef, err := Repo.Reference(gitplumbing.ReferenceName(srcRefName), false)
	if err != nil {
		return errors.WithStack(err)
	}

	commitHash := srcRef.Hash().String()

	// @@TODO: give context a timeout and make it configurable
	ctx, cancel2 := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel2()
	err = client.UpdateRef(ctx, repoID, destRefName, commitHash)
	if err != nil {
		return err
	}

	// @@TODO: give context a timeout and make it configurable
	// ctx, cancel3 := context.WithTimeout(context.Background(), 15*time.Second)
	// defer cancel3()
	log.Println("Contacting peers for replication...")
	ch := client.RequestReplication(context.Background(), repoID)
	for progress := range ch {
		if progress.Error != nil {
			log.Printf("Could not find replicator for repo")
			return nil
		}
		log.Printf("Progress: %d%%", progress.Percent)
	}

	return nil
}
