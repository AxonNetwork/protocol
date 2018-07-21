package main

import (
	"bufio"
	// "encoding/hex"
	"fmt"
	"github.com/cryptix/go/logging"
	"io"
	"net/rpc"
	"os"
	"strings"
)

var (
	log      logging.Interface
	client   *rpc.Client
	repoID   string
	repoPath string
)

func main() {
	logging.SetupLogging(nil)
	log = logging.Logger("git-remote-conscience")
	var err error
	client, err = rpc.Dial("tcp", fmt.Sprintf("127.0.0.1:1338"))
	check(err)

	repoPath, err = os.Getwd()
	check(err)

	repoID = "testing"

	speakGit(os.Stdin, os.Stdout)
}

func speakGit(r io.Reader, w io.Writer) error {
	scanner := bufio.NewScanner(r)
	for scanner.Scan() {
		text := scanner.Text()
		// log.Log("msg", text)

		switch {

		case text == "capabilities":
			fmt.Fprintln(w, "list")
			fmt.Fprintln(w, "fetch")
			fmt.Fprintln(w, "push")
			fmt.Fprintln(w)

		case strings.HasPrefix(text, "list"):
			fmt.Fprintln(w, "2f582d8a67dabe011cc9e200780078a2b98cce0d refs/heads/masters")
			fmt.Fprintln(w, "@refs/heads/masters HEAD")
			fmt.Fprintln(w)

		case strings.HasPrefix(text, "fetch"):
			fetchArgs := strings.Split(text, " ")
			objHash := fetchArgs[1]
			err := recurseCommit(objHash)
			check(err)
			fmt.Fprintln(w)

		}
	}
	return nil

}

func check(e error) {
	if e != nil {
		panic(e)
	}
}
