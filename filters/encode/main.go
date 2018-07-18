package main

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"github.com/aclements/go-rabin/rabin"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
)

func main() {
	const WINDOW_SIZE, MIN, AVG, MAX = 64, 512, 2048, 4096

	cwd, err := os.Getwd()
	check(err)

	copy := new(bytes.Buffer)
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

		bs := new(bytes.Buffer)
		_, err = io.CopyN(bs, copy, int64(len))
		check(err)

		hash := sha256.Sum256(bs.Bytes())
		hexHash := hex.EncodeToString(hash[:])
		folderPath := filepath.Join(cwd, ".git", "data", hexHash[:2])
		filePath := filepath.Join(folderPath, hexHash[2:])

		err = os.MkdirAll(folderPath, os.FileMode(0777))
		check(err)

		if !fileExists(filePath) {
			err = ioutil.WriteFile(filePath, bs.Bytes(), os.FileMode(0666))
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
