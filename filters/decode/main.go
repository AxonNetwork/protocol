package main

import (
	"bufio"
	"bytes"
	"io"
	"os"
	"path/filepath"
)

func main() {
	r := bufio.NewReader(os.Stdin)

	// check first line to determine if the file is chunked
	header, err := r.Peek(18)
	if err != nil && err != io.EOF {
		panic(err)
	}
	if bytes.Compare(header, []byte("CONSCIENCE_ENCODED")) != 0 {
		_, err = io.Copy(os.Stdout, r)
		check(err)
		return
	}

	cwd, err := os.Getwd()
	check(err)

	// ignore the first header line
	_, _, err = r.ReadLine()
	check(err)

	for {
		line, _, err := r.ReadLine()
		if err == io.EOF {
			break
		} else if err != nil {
			panic(err)
		}
		p := filepath.Join(cwd, ".git", "data", string(line))
		f, err := os.Open(p)
		check(err)
		defer f.Close()

		_, err = io.Copy(os.Stdout, f)
		check(err)

		f.Close()
	}
}

func check(e error) {
	if e != nil {
		panic(e)
	}
}
