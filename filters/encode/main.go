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

const KB = 1024
const MB = KB * KB
const THRESHOLD = 4 * MB // 100kB
const WINDOW_SIZE, MIN, AVG, MAX = KB, MB, 2 * MB, 3.5 * MB

func main() {

	if len(os.Args) < 2 || !shouldEncode(os.Args[1]) {
		_, err := io.Copy(os.Stdout, os.Stdin)
		check(err)
		return
	}

	cwd, err := os.Getwd()
	check(err)

	os.Stdout.Write([]byte("CONSCIENCE_ENCODED\n"))

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
		folderPath := filepath.Join(cwd, ".git", "data")
		filePath := filepath.Join(folderPath, hexHash[:])

		err = os.MkdirAll(folderPath, os.FileMode(0777))
		check(err)

		if !fileExists(filePath) {
			err = ioutil.WriteFile(filePath, bs.Bytes(), os.FileMode(0666))
			check(err)
		}

		os.Stdout.Write([]byte(hexHash + "\n"))
	}
}

func shouldEncode(filename string) bool {
	return getFileSize(filename) >= THRESHOLD
}

// func shouldEncode(filename string) bool {
// 	TO_ENCODE := map[string]bool{
// 		".png": true,
// 	}
// 	ext := filepath.Ext(filename)
// 	should, ok := TO_ENCODE[ext]
// 	return should && ok
// }

func getFileSize(filename string) int64 {
	cwd, err := os.Getwd()
	check(err)

	p := filepath.Join(cwd, filename)
	f, err := os.Open(p)
	check(err)
	defer f.Close()

	s, err := f.Stat()
	check(err)
	return s.Size()
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
