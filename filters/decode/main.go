package main

import (
	"bufio"
	"bytes"
	"context"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/Conscience/protocol/config"
	"github.com/Conscience/protocol/log"
	"github.com/Conscience/protocol/repo"
	"github.com/Conscience/protocol/swarm/noderpc"
	"github.com/pkg/errors"
)

func main() {
	r := bufio.NewReader(os.Stdin)

	// check first line to determine if the file is chunked
	header, err := r.Peek(18)
	if err != nil && err != io.EOF {
		die(err)
	}
	if bytes.Compare(header, []byte("CONSCIENCE_ENCODED")) != 0 {
		_, err = io.Copy(os.Stdout, r)
		if err != nil {
			die(err)
		}
		return
	}

	cwd, err := os.Getwd()
	if err != nil {
		die(err)
	}

	// ignore the first header line
	_, _, err = r.ReadLine()
	if err != nil {
		die(err)
	}

	chunks := make([]string, 0)
	toFetch := make([][]byte, 0)

	for {
		line, _, err := r.ReadLine()
		if err == io.EOF {
			break
		} else if err != nil {
			die(err)
		}

		p := filepath.Join(cwd, ".git", repo.CONSCIENCE_DATA_SUBDIR, string(line))
		chunks = append(chunks, p)
		_, err = os.Stat(p)
		if err != nil {
			hash, err := hex.DecodeString(string(line))
			if err != nil {
				die(err)
			}
			toFetch = append(toFetch, hash)
		}
	}

	if len(toFetch) > 0 {
		err = fetchChunks(toFetch)
		if err != nil {
			die(err)
		}
	}

	for _, chunk := range chunks {
		f, err := os.Open(chunk)
		if err != nil {
			die(err)
		}
		defer f.Close()

		_, err = io.Copy(os.Stdout, f)
	}
}

func fetchChunks(chunks [][]byte) error {
	GIT_DIR := os.Getenv("GIT_DIR")
	repoPath := filepath.Dir(GIT_DIR)

	r, err := repo.Open(repoPath)
	if err != nil {
		return err
	}

	repoID, err := r.RepoID()
	if err != nil {
		return err
	}

	cfg, err := config.ReadConfig()
	if err != nil {
		return err
	}

	client, err := noderpc.NewClient(cfg.RPCClient.Host)
	if err != nil {
		return err
	}
	defer client.Close()

	ch, err := client.FetchChunks(context.Background(), repoID, repoPath, chunks)
	if err != nil {
		return err
	}

	dataDir := filepath.Join(GIT_DIR, repo.CONSCIENCE_DATA_SUBDIR)
	err = os.MkdirAll(dataDir, 0777)
	if err != nil {
		return err
	}

	for pkt := range ch {
		if pkt.Error != nil {
			return pkt.Error
		}

		objectID := hex.EncodeToString(pkt.Chunk.ObjectID)
		objectPath := filepath.Join(dataDir, objectID)
		f, err := os.Create(objectPath)
		if err != nil {
			return err
		}

		n, err := f.Write(pkt.Chunk.Data)
		if err != nil {
			return errors.WithStack(err)
		} else if n != len(pkt.Chunk.Data) {
			return errors.New("remote helper: did not fully write chunk")
		}

		err = f.Close()
		if err != nil {
			return errors.WithStack(err)
		}

		os.Stderr.Write([]byte(fmt.Sprintln("Writing chunk: ", objectID)))
	}

	return nil
}

func die(err error) {
	log.Errorf("%+v\n", err)
	os.Exit(1)
}
