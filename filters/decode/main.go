package main

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

func main() {
	cwd, err := os.Getwd()
	if err != nil {
		panic(err)
	}

	scanner := bufio.NewScanner(os.Stdin)
	for scanner.Scan() {
		hexHash := scanner.Text()

		// break on empty string
		if len(strings.TrimSpace(hexHash)) == 0 {
			break
		}

		filePath := filepath.Join(cwd, ".git", "data", hexHash[:2], hexHash[2:])

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
