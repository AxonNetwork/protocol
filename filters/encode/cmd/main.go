package main

import (
	"io"
	"os"

	"github.com/Conscience/protocol/filters/encode"
	"github.com/Conscience/protocol/log"
)

var (
	GIT_DIR = os.Getenv("GIT_DIR")
)

func main() {
	os.Stderr.Write([]byte("ENCODING"))
	log.Println("ENCODING")
	gitDir := os.Getenv("GIT_DIR")
	reader := encode.Encode(gitDir, os.Stdin)
	_, err := io.Copy(os.Stdout, reader)
	if err != nil {
		die(err)
	}
}

func die(err error) {
	log.Errorf("%+v\n", err)
	os.Exit(1)
}
