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

var (
	GIT_DIR  = os.Getenv("GIT_DIR")
	RepoRoot = filepath.Dir(GIT_DIR)
	DataRoot = filepath.Join(GIT_DIR, repo.CONSCIENCE_DATA_SUBDIR)

	repository = func() *repo.Repo {
		r, err := repo.Open(RepoRoot)
		if err != nil {
			panic(err)
		}

		return r
	}()
)

func main() {
	chunks := make([]string, 0)
	toFetch := make([][]byte, 0)

	for {
		line, _, err := r.ReadLine()
		if err == io.EOF {
			break
		} else if err != nil {
			die(err)
		}

		p := filepath.Join(DataDir, string(line))
		chunks = append(chunks, p)
		_, err = os.Stat(p)
		if os.IsNotExist(err) {
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
		err = copyFileToStdout(chunk)
		if err != nil {
			die(err)
		}
	}
}

func copyFileToStdout(filename string) error {
	f, err := os.Open(filename)
	if err != nil {
		return err
	}
	defer f.Close()

	_, err = io.Copy(os.Stdout, f)
	if err != nil {
		return err
	}
	return nil
}

func fetchChunks(chunks [][]byte) error {
	repoID, err := repository.RepoID()
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

	ch, err := client.FetchChunks(context.TODO(), repoID, RepoRoot, chunks)
	if err != nil {
		return err
	}

	err = os.MkdirAll(DataDir, 0777)
	if err != nil {
		return err
	}

	chunkWriters := make(map[string]*os.File)

	for pkt := range ch {
		if pkt.Error != nil {
			return pkt.Error
		}

		objectID := hex.EncodeToString(pkt.Chunk.ObjectID)

		if pkt.Chunk.End {
			err = chunkWriters[objectID].Close()
			if err != nil {
				return errors.WithStack(err)
			}

			os.Stderr.Write([]byte(fmt.Sprintln("Wrote chunk: ", objectID)))
			chunkWriters[objectID] = nil

		} else {
			f := chunkWriters[objectID]

			if f == nil {
				f, err = os.Create(filepath.Join(DataDir, objectID))
				if err != nil {
					return errors.WithStack(err)
				}
				chunkWriters[objectID] = f
			}

			n, err := f.Write(pkt.Chunk.Data)
			if err != nil {
				return errors.WithStack(err)
			} else if n != len(pkt.Chunk.Data) {
				return errors.New("remote helper: did not fully write chunk")
			}

		}
	}

	return nil
}

func die(err error) {
	log.Errorf("%+v\n", err)
	os.Exit(1)
}
