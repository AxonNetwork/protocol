package main

import (
	"bufio"
	"encoding/hex"
	"fmt"
	"gopkg.in/src-d/go-git.v4"
	gitobject "gopkg.in/src-d/go-git.v4/plumbing/object"
	"net/rpc"
	"os"
	"path/filepath"
	"strings"
	"sync"
)

func main() {
	cwd, err := os.Getwd()
	check(err)

	repo, err := git.PlainOpen(cwd)
	check(err)

	head, err := repo.Head()
	check(err)

	commit, err := repo.CommitObject(head.Hash())
	check(err)

	tree, err := commit.Tree()
	check(err)

	entries := tree.Entries
	cfg, err := repo.Config()
	section := cfg.Raw.Section("conscience")
	repoID := section.Option("repoid")

	client, err := rpc.Dial("tcp", fmt.Sprintf("127.0.01:1338"))

	for _, entry := range entries {
		if filepath.Ext(entry.Name) == ".JPG" {
			blob, err := repo.BlobObject(entry.Hash)
			check(err)
			downloadChunks(client, repoID, blob)
		}
	}
	fmt.Println("Downloaded all chunks")
}

func downloadChunks(client *rpc.Client, repoID string, b *gitobject.Blob) {
	r, err := b.Reader()
	check(err)

	wg := sync.WaitGroup{}
	scanner := bufio.NewScanner(r)
	for scanner.Scan() {
		hexHash := scanner.Text()
		// break on empty string
		if len(strings.TrimSpace(hexHash)) == 0 {
			break
		}

		objectID, err := hex.DecodeString(hexHash)
		check(err)
		fmt.Println("Getting: ")
		fmt.Println("repoID: ", repoID)
		fmt.Println("hexHash: ", hexHash)

		wg.Add(1)
		go func(client *rpc.Client, repoID string, objectID []byte) {
			defer wg.Done()
			downloadChunk(client, repoID, objectID)
		}(client, repoID, objectID)
	}
	wg.Wait()
}

func downloadChunk(client *rpc.Client, repoID string, objectID []byte) {
	in := GetObjectInput{
		repoID,
		objectID,
	}
	out := GetObjectOutput{}
	err := client.Call("Node.GetObject", in, &out)
	check(err)
}

func check(e error) {
	if e != nil {
		panic(e)
	}
}

type Node interface {
	GetObject(GetObjectInput, *GetObjectOutput) error
}

type GetObjectInput struct {
	RepoID   string
	ObjectID []byte
}

type GetObjectOutput struct{}
