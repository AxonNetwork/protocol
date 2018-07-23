package main

import (
	"bufio"
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/src-d/go-git.v4"

	"../../swarm"
)

var GIT_DIR string = os.Getenv("GIT_DIR")

func main() {
	repo, err := git.PlainOpen(filepath.Dir(GIT_DIR))
	check(err)

	cfg, err := repo.Config()
	section := cfg.Raw.Section("conscience")
	repoID := section.Option("repoid")

	client, err := swarm.NewRPCClient("tcp", "127.0.0.1:1338")
	check(err)

	stdinCopy := &bytes.Buffer{}
	reader := io.TeeReader(os.Stdin, stdinCopy)
	scanner := bufio.NewScanner(reader)
	// First, make sure we have all of the chunks.  Any missing chunks are downloaded by the Node
	// in parallel.
	// @@TODO: had to remove parallelism because requesting too many chunks at once caused the Node
	// to panic and die.  that's why there are two loops here and things look kind of weird.
	for scanner.Scan() {
		objectID := scanner.Text()
		objectID = strings.TrimSpace(objectID)

		// break on empty string
		if len(objectID) == 0 {
			break
		}

		filePath := filepath.Join(GIT_DIR, "data", objectID)

		_, err = os.Stat(filePath)
		if err != nil {
			// file doesn't exist

			err := downloadChunk(client, repoID, objectID)
			if err != nil {
				panic(err)
			}
		}
	}

	scanner = bufio.NewScanner(stdinCopy)
	// Second, loop through the objectIDs in stdin again, emitting each chunk's data serially.
	for scanner.Scan() {
		objectID := scanner.Text()
		objectID = strings.TrimSpace(objectID)

		// break on empty string
		if len(objectID) == 0 {
			break
		}

		filePath := filepath.Join(GIT_DIR, "data", objectID)

		f, err := os.Open(filePath)
		if err != nil {
			panic(err)
		}

		_, err = io.Copy(os.Stdout, f)
		if err != nil {
			f.Close()
			panic(err)
		}

		f.Close()
	}
	if err = scanner.Err(); err != nil {
		fmt.Fprintln(os.Stderr, "reading standard input:", err)
	}
}

// @@TODO: use a `chan chan []byte` to structure concurrency, not a sync.WaitGroup
// func chunkData(repoID string, objectID []byte) chan []byte {
// }

func downloadChunk(client *swarm.RPCClient, repoID string, objectIDStr string) error {
	objectID, err := hex.DecodeString(objectIDStr)
	if err != nil {
		return err
	}

	fmt.Fprintf(os.Stderr, "Downloading chunk %v...\n", objectIDStr)

	objectStream, _, objectLen, err := client.GetObject(repoID, objectID)
	if err != nil {
		return err
	}
	defer objectStream.Close()

	dataDir := filepath.Join(GIT_DIR, "data")
	err = os.MkdirAll(dataDir, 0777)
	if err != nil {
		return err
	}

	chunkPath := filepath.Join(dataDir, objectIDStr)
	f, err := os.Create(chunkPath)
	if err != nil {
		return err
	}

	hasher := sha256.New()
	reader := io.TeeReader(objectStream, hasher)

	copied, err := io.Copy(f, reader)
	if err != nil {
		f.Close()
		os.Remove(chunkPath)
		return err
	} else if copied < objectLen {
		f.Close()
		os.Remove(chunkPath)
		return fmt.Errorf("copied (%v) < objectLen (%v)", copied, objectLen)
	} else if !bytes.Equal(objectID, hasher.Sum(nil)) {
		f.Close()
		os.Remove(chunkPath)
		return fmt.Errorf("checksum error (objectID: %v)", objectIDStr)
	}

	return f.Close()
}

func check(e error) {
	if e != nil {
		panic(e)
	}
}
