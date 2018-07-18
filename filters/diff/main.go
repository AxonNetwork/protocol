package main

import (
	"bytes"
	"crypto/sha1"
	"encoding/hex"
	"fmt"
	"github.com/aclements/go-rabin/rabin"
	"github.com/dustin/go-humanize"
	. "github.com/logrusorgru/aurora"
	"io"
	"os"
	"strconv"
)

func main() {
	filename := os.Args[1]

	file1, err := os.Open(os.Args[2])
	check(err)
	defer file1.Close()
	file2, err := os.Open(os.Args[5])
	check(err)
	defer file2.Close()

	chunks1 := getChunks(file1)
	chunks2 := getChunks(file2)
	stat1, err := file1.Stat()
	check(err)
	stat2, err := file2.Stat()
	check(err)

	size1 := humanize.Bytes(uint64(stat1.Size()))
	size2 := humanize.Bytes(uint64(stat2.Size()))
	added, deleted := getChunksDiff(chunks1, chunks2)
	addedLen := strconv.FormatInt(int64(len(added)), 10)
	deletedLen := strconv.FormatInt(int64(len(deleted)), 10)
	chunks1Len := strconv.FormatInt(int64(len(chunks1)), 10)
	chunks2Len := strconv.FormatInt(int64(len(chunks2)), 10)
	fmt.Println("")
	fmt.Println(Bold(filename))
	fmt.Println("_________________")

	fmt.Println(Red(concat("Before: ", size1)))
	fmt.Println(Green(concat("Green: ", size2)))
	fmt.Println(Red(concat("Deleted: ", deletedLen, " out of ", chunks1Len, " chunks")))
	fmt.Println(Green(concat("Added: ", addedLen, " out of ", chunks2Len, " chunks")))
}

func concat(strings ...string) string {
	var b bytes.Buffer
	for i := 0; i < len(strings); i++ {
		b.WriteString(strings[i])
	}
	return b.String()
}

func getChunks(file *os.File) (chunks []string) {
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

func getChunksDiff(chunks1 []string, chunks2 []string) (added []string, deleted []string) {
	added = make([]string, 0)
	deleted = make([]string, 0)
	for _, chunk := range chunks1 {
		if !stringInSlice(chunk, chunks2) {
			deleted = append(deleted, chunk)
		}

	}
	for _, chunk := range chunks2 {
		if !stringInSlice(chunk, chunks1) {
			added = append(added, chunk)
		}

	}
	return added, deleted
}

func stringInSlice(a string, slice []string) bool {
	for _, b := range slice {
		if a == b {
			return true
		}
	}
	return false
}

func check(e error) {
	if e != nil {
		panic(e)
	}
}
