package main

import (
	"context"
	"log"
	"os"
	"strings"
	"time"

	"github.com/Conscience/protocol/util"
)

func push(srcRefName string, destRefName string) error {
	force := strings.HasPrefix(srcRefName, "+")
	if force {
		srcRefName = srcRefName[1:]
	}

	// @@TODO: give context a timeout and make it configurable
	ctx, cancel1 := context.WithTimeout(context.TODO(), 15*time.Second)
	defer cancel1()

	chProgress, err := client.PushRepo(ctx, Repo.Path(), srcRefName, force)
	if err != nil {
		return err
	}

	progressWriter := util.NewSingleLineWriter(os.Stderr)

	for progress := range chProgress {
		if progress.Error != nil {
			log.Printf("Could not find replicator for repo")
			return nil

		} else if progress.Done {
			progressWriter.Printf("Done.")
		} else {
			progressWriter.Printf("Progress: %d%%", (progress.Current*100)/progress.Total)
		}
	}

	return nil
}
