package main

import (
	"bytes"
	"crypto/sha1"
	"encoding/hex"
	"fmt"
	"io"
	"os"

	"github.com/aclements/go-rabin/rabin"
	"github.com/dustin/go-humanize"
)

func main() {
	p := os.Args[1]
	f, err := os.Open(p)
	check(err)
	defer f.Close()

	stat, err := f.Stat()
	check(err)
	size := humanize.Bytes(uint64(stat.Size()))
	chunks := getChunks(f)

	fmt.Printf("%s\n", size)
	fmt.Printf("%d chunks\n", len(chunks))
	for _, c := range chunks {
		fmt.Printf("%s\n", c)
	}
}

func getChunks(file io.Reader) (chunks []string) {
	const WINDOW_SIZE, MIN, AVG, MAX = 64, 512, 2048, 4096

	copy := new(bytes.Buffer)
	reader := io.TeeReader(file, copy)
	table := rabin.NewTable(rabin.Poly64, WINDOW_SIZE)
	chunker := rabin.NewChunker(table, reader, MIN, AVG, MAX)
	chunks = make([]string, 0)
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

		hash := sha1.Sum(bs.Bytes())
		hexHash := hex.EncodeToString(hash[:])

		chunks = append(chunks, hexHash)
	}
	return chunks
}

func check(e error) {
	if e != nil {
		panic(e)
	}
}
