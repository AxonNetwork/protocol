package main

import (
	"bufio"
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/pkg/errors"

	"github.com/Conscience/protocol/config"
	"github.com/Conscience/protocol/log"
	"github.com/Conscience/protocol/repo"
	"github.com/Conscience/protocol/swarm/noderpc"
)

var GIT_DIR string = os.Getenv("GIT_DIR")

func main() {
	r, err := repo.Open(filepath.Dir(GIT_DIR))
	if err != nil {
		die(err)
	}
	defer r.Free()

	repoID, err := r.RepoID()
	if err != nil {
		die(err)
	}

	cfg, err := config.ReadConfig()
	if err != nil {
		die(err)
	}

	client, err := noderpc.NewClient(cfg.RPCClient.Host)
	if err != nil {
		die(err)
	}
	defer client.Close()

	// First, make sure we have all of the chunks.  Any missing chunks are downloaded from the Node
	// in parallel.
	chch := make(chan chan string)
	chErr := make(chan error)
	chDone := make(chan struct{})

	go func() {
		scanner := bufio.NewScanner(os.Stdin)
		for scanner.Scan() {
			objectIDStr := scanner.Text()
			objectIDStr = strings.TrimSpace(objectIDStr)

			// break on empty string
			if len(objectIDStr) == 0 {
				break
			}

			ch := make(chan string)
			chch <- ch

			_, err = os.Stat(filepath.Join(GIT_DIR, repo.CONSCIENCE_DATA_SUBDIR, objectIDStr))
			if err != nil {
				// file doesn't exist

				err := downloadChunk(client, repoID, objectIDStr)
				if err != nil {
					chErr <- err
					return
				}
			}
			ch <- objectIDStr
		}
		if err = scanner.Err(); err != nil {
			chErr <- errors.Wrap(err, "error scanning stdin")
			return
		}

		close(chch)
	}()

	// Second, loop through the objectIDs in stdin again, emitting each chunk's data serially.
	go func() {
		for ch := range chch {
			objectIDStr := <-ch

			f, err := os.Open(filepath.Join(GIT_DIR, repo.CONSCIENCE_DATA_SUBDIR, objectIDStr))
			if err != nil {
				chErr <- errors.Wrapf(err, "could not open file to write object '%v'", objectIDStr)
				return
			}
			defer f.Close()

			_, err = io.Copy(os.Stdout, f)
			if err != nil {
				chErr <- errors.Wrap(err, "error while streaming file contents to stdout")
				return
			}

			// Try to close after each iteration to keep our resource footprint small
			f.Close()
		}

		close(chDone)
	}()

	select {
	case err := <-chErr:
		die(err)
	case <-chDone:
	}
}

func downloadChunk(client *noderpc.Client, repoID string, objectIDStr string) error {
	objectID, err := hex.DecodeString(objectIDStr)
	if err != nil {
		return errors.Wrapf(err, "error decoding objectID '%v'", objectIDStr)
	}

	fmt.Fprintf(os.Stderr, "Downloading chunk %v...\n", objectIDStr)

	// @@TODO: give context a timeout and make it configurable
	objectStream, err := client.GetObject(context.Background(), repoID, objectID)
	if err != nil {
		return errors.Wrap(err, "could not get object stream via RPC")
	}
	defer objectStream.Close()

	dataDir := filepath.Join(GIT_DIR, "data")
	err = os.MkdirAll(dataDir, 0777)
	if err != nil {
		return errors.Wrap(err, "could not mkdir")
	}

	chunkPath := filepath.Join(dataDir, objectIDStr)
	f, err := os.Create(chunkPath)
	if err != nil {
		return errors.Wrap(err, "could not create chunk on disk")
	}
	defer f.Close()

	hasher := sha256.New()
	reader := io.TeeReader(objectStream, hasher)

	copied, err := io.Copy(f, reader)
	if err != nil {
		os.Remove(chunkPath)
		return err
	} else if uint64(copied) != objectStream.Len() {
		os.Remove(chunkPath)
		return errors.Errorf("copied (%v) != objectLen (%v)", copied, objectStream.Len())
	} else if !bytes.Equal(objectID, hasher.Sum(nil)) {
		os.Remove(chunkPath)
		return errors.Errorf("checksum error (objectID: %v)", objectIDStr)
	}

	return nil
}

func die(err error) {
	log.Errorf("error: %+v\n", err)
	os.Exit(1)
}
