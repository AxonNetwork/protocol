package main

import (
	"context"
	"os"
	"strings"
	"time"

	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	gitplumbing "gopkg.in/src-d/go-git.v4/plumbing"
)

func push(srcRefName string, destRefName string) error {
	force := strings.HasPrefix(srcRefName, "+")
	if force {
		srcRefName = srcRefName[1:]
	}

	f, err := os.OpenFile("/tmp/conscience-push", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	defer f.Close()

	for _, v := range os.Environ() {
		f.WriteString("env: " + v + "\n")
	}

	// Tell the node to announce the new content so that replicator nodes can find and pull it.
	// @@TODO: give context a timeout and make it configurable
	log.Infof("announcing repo content")
	f.WriteString("announcing repo content\n")
	ctx, cancel1 := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel1()
	err = client.AnnounceRepoContent(context.Background(), repoID)
	if err != nil {
		f.WriteString("announcing repo content / err: " + err.Error() + "\n")
		return err
	}

	f.WriteString("repo.reference\n")
	srcRef, err := Repo.Reference(gitplumbing.ReferenceName(srcRefName), false)
	if err != nil {
		f.WriteString("repo.reference / err: " + err.Error() + "\n")
		return errors.WithStack(err)
	}

	commitHash := srcRef.Hash().String()

	// @@TODO: give context a timeout and make it configurable
	ctx, cancel2 := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel2()
	log.Infof("updating ref")
	f.WriteString("updating ref\n")
	err = client.UpdateRef(ctx, repoID, destRefName, commitHash)
	if err != nil {
		f.WriteString("updating ref / err: " + err.Error() + "\n")
		return err
	}

	// @@TODO: give context a timeout and make it configurable
	ctx, cancel3 := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel3()
	log.Infof("requesting replication")
	f.WriteString("requesting replication\n")
	err = client.RequestReplication(ctx, repoID)
	if err != nil {
		f.WriteString("requesting replication / err: " + err.Error() + "\n")
	}
	log.Infof("done!")
	return err
}
