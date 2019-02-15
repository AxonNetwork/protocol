package main

import (
	"context"
	"log"
	"os"
	"strings"
	"time"

	"github.com/pkg/errors"
    "github.com/libgit2/git2go"

	"github.com/Conscience/protocol/util"
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

    srcRef, err := Repo.References.Lookup(srcRefName)
    if err != nil {
        return err
    }

    srcRef, err = srcRef.Resolve()
    if err != nil {
        return err
    }

	commitHash := srcRef.Target().String()

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

	progressWriter := util.NewSingleLineWriter(os.Stderr)

	ch := client.RequestReplication(context.Background(), repoID)
	for progress := range ch {
		if progress.Error != nil {
			log.Printf("Could not find replicator for repo")
			return nil
		}
		progressWriter.Printf("Progress: %d%%", progress.Percent)
	}

	return nil
}
