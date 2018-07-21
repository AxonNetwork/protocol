package main

import (
	"bufio"
	// "encoding/hex"
	"fmt"
	"github.com/cryptix/go/logging"
	"io"
	"net/rpc"
	"os"
	"path/filepath"
	"strings"
)

var (
	log      logging.Interface
	client   *rpc.Client
	repoID   string
	repoPath string
	head     string
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

	head, err = readHead()
	check(err)

	createRefs()

	speakGit(os.Stdin, os.Stdout)
}

func speakGit(r io.Reader, w io.Writer) error {
	scanner := bufio.NewScanner(r)
	for scanner.Scan() {
		text := scanner.Text()
		log.Log("msg", text)

		switch {

		case text == "capabilities":
			fmt.Fprintln(w, "list")
			fmt.Fprintln(w, "fetch")
			fmt.Fprintln(w, "push")
			fmt.Fprintln(w)

		case strings.HasPrefix(text, "list"):
			log.Log("msg", head)
			fmt.Fprintf(w, "%s refs/heads/masters\n", head)
			fmt.Fprintln(w, "@refs/heads/masters HEAD")
			fmt.Fprintln(w)

		case strings.HasPrefix(text, "fetch"):
			fetchArgs := strings.Split(text, " ")
			objHash := fetchArgs[1]
			log.Log("msg", objHash)
			err := recurseCommit(objHash)
			check(err)
			fmt.Fprintln(w)

		}
	}
	return nil

}

func readHead() (string, error) {
	cwd, err := os.Getwd()
	if err != nil {
		return "", err
	}
	p := filepath.Join(cwd, "../", "../", "remote-helper", "head")
	f, err := os.Open(p)
	if err != nil {
		return "", err
	}
	r := bufio.NewReader(f)
	head, _, err := r.ReadLine()
	if err != nil {
		return "", err
	}
	return string(head), nil
}

func createRefs() error {
	p := filepath.Join(repoPath, ".git", "HEAD")
	f, err := os.Create(p)
	if err != nil {
		return err
	}
	_, err = f.WriteString("ref: refs/heads/master")
	if err != nil {
		return err
	}
	p = filepath.Join(repoPath, ".git", "refs", "heads", "master")
	f, err = os.Create(p)
	if err != nil {
		return err
	}
	_, err = f.WriteString(head)
	if err != nil {
		return err
	}
	return nil
}

func check(e error) {
	if e != nil {
		panic(e)
	}
}
