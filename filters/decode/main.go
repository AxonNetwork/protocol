package main

import (
	"bufio"
	"bytes"
	"encoding/hex"
	"fmt"
	"io"
	"io/ioutil"
	"net/rpc"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/src-d/go-git.v4"
)

func main() {
	ioutil.WriteFile("/tmp/did-it.txt", []byte("did it"), 0777)

	cwd, err := os.Getwd()
	check(err)

	repo, err := git.PlainOpen(cwd)
	check(err)

	cfg, err := repo.Config()
	section := cfg.Raw.Section("conscience")
	repoID := section.Option("repoid")

	client, err := rpc.Dial("tcp", fmt.Sprintf("127.0.0.1:1338"))
	check(err)

	buf := &bytes.Buffer{}
	reader := io.TeeReader(os.Stdin, buf)
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

		filePath := filepath.Join(cwd, ".git", "data", objectID)

		_, err = os.Stat(filePath)
		if err != nil {
			// file doesn't exist

			err := downloadChunk(client, repoID, objectID)
			if err != nil {
				panic(err)
			}
		}
	}

	scanner = bufio.NewScanner(buf)
	// Second, loop through the objectIDs in stdin again, emitting each chunk's data serially.
	for scanner.Scan() {
		objectID := scanner.Text()
		objectID = strings.TrimSpace(objectID)

		// break on empty string
		if len(objectID) == 0 {
			break
		}

		filePath := filepath.Join(cwd, ".git", "data", objectID)

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

//
// @@TODO: move these structs into a swarmrpctypes package
//

type GetObjectInput struct {
	RepoID   string
	ObjectID []byte
}

type GetObjectOutput struct{}

func downloadChunk(client *rpc.Client, repoID string, objectIDStr string) error {
	objectID, err := hex.DecodeString(objectIDStr)
	if err != nil {
		return err
	}

	fmt.Fprintf(os.Stderr, "Downloading chunk %v...\n", objectIDStr)

	in := GetObjectInput{
		repoID,
		objectID,
	}
	out := GetObjectOutput{}
	err = client.Call("Node.GetObject", in, &out)
	return err
}

func check(e error) {
	if e != nil {
		panic(e)
	}
}
