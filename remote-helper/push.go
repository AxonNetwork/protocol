package main

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/Conscience/protocol/repo"
	"github.com/Conscience/protocol/swarm/nodep2p"
	"github.com/Conscience/protocol/util"
)

func push(srcRefName string, destRefName string) error {
	force := strings.HasPrefix(srcRefName, "+")
	if force {
		srcRefName = srcRefName[1:]
	}

	// @@TODO: give context a timeout and make it configurable
	ctx, cancel := context.WithTimeout(context.TODO(), 15*time.Second)
	defer cancel()

	chProgress, err := client.PushRepo(ctx, Repo.Path(), srcRefName, force)
	if err != nil {
		return err
	}

	progressWriter := util.NewSingleLineWriter(os.Stderr)

	for progress := range chProgress {
		switch {
		case progress.Error == nodep2p.ErrRepoIDNotRegistered,
			progress.Error == repo.ErrNoRepoID,
			progress.Error == repo.ErrNotTracked:
			return progress.Error

		case progress.Error == nodep2p.ErrNoReplicatorsAvailable:
			fmt.Fprintln(os.Stderr, "axon: ref successfully pushed, but no replicators are currently available.")

		case progress.Error == nodep2p.ErrAllReplicatorsFailed:
			fmt.Fprintln(os.Stderr, "axon: ref successfully pushed, but all available replicators failed to pull.")

		case progress.Error != nil:
			fmt.Fprintln(os.Stderr, "axon: error:", progress.Error)

		case progress.Done:
			progressWriter.Printf("Done.")

		default:
			progressWriter.Printf("Progress: %d%%", (progress.Current*100)/progress.Total)
		}
	}

	return nil
}
