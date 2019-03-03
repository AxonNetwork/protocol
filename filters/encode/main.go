package main

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/aclements/go-rabin/rabin"
)

var (
	GIT_DIR  = os.Getenv("GIT_DIR")
	RepoRoot = filepath.Dir(GIT_DIR)
	DataRoot = filepath.Join(GIT_DIR, repo.CONSCIENCE_DATA_SUBDIR)
)

const (
	KB          = 1024
	MB          = 1024 * KB
	WINDOW_SIZE = KB
	MIN         = MB
	AVG         = 2 * MB
	MAX         = 4 * MB
)

func main() {
	copy := &bytes.Buffer{}
	reader := io.TeeReader(os.Stdin, copy)
	table := rabin.NewTable(rabin.Poly64, WINDOW_SIZE)
	chunker := rabin.NewChunker(table, reader, MIN, AVG, MAX)
	for {
		len, err := chunker.Next()
		if err == io.EOF {
			break
		} else {
			check(err)
		}

		bs := &bytes.Buffer{}
		_, err = io.CopyN(bs, copy, int64(len))
		check(err)

		hash := sha256.Sum256(bs.Bytes())
		hexHash := hex.EncodeToString(hash[:])
		filePath := filepath.Join(DataRoot, hexHash[:])

		err = os.MkdirAll(folderPath, 0777)
		check(err)

		if !fileExists(filePath) {
			err = ioutil.WriteFile(filePath, bs.Bytes(), 0666)
			check(err)
		}

		os.Stdout.Write([]byte(hexHash + "\n"))
	}
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

func check(e error) {
	if e != nil {
		panic(e)
	}
}
