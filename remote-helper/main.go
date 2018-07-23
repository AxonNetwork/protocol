package main

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	log "github.com/sirupsen/logrus"
	"gopkg.in/src-d/go-git.v4"
	gitplumbing "gopkg.in/src-d/go-git.v4/plumbing"

	"../swarm"
)

var (
	client  *swarm.RPCClient
	repoID  string = "testing"
	head    string
	repo    *git.Repository
	GIT_DIR string = os.Getenv("GIT_DIR")
)

func main() {
	var err error
	client, err = swarm.NewRPCClient("tcp", "127.0.0.1:1338")
	// client, err = swarm.NewRPCClient("unix", "/tmp/conscience-socket")
	check(err)

	if GIT_DIR == "" {
		panic("empty GIT_DIR")
	}

	log.Errorln("GIT_DIR ~>", GIT_DIR)
	log.Errorln("dir ~>", filepath.Dir(GIT_DIR))
	_, err = os.Stat(GIT_DIR)
	if err != nil {
		log.Errorln("STAT ~>", err)
	}

	repo, err = git.PlainOpen(filepath.Dir(GIT_DIR))
	if err != nil {
		panic(err)
	}

	head, err = readHead()
	check(err)

	createRefs()

	err = speakGit(os.Stdin, os.Stdout)
	check(err)
}

func speakGit(r io.Reader, w io.Writer) error {
	scanner := bufio.NewScanner(r)
	for scanner.Scan() {
		text := scanner.Text()
		log.Println("msg", text)

		switch {

		case text == "capabilities":
			fmt.Fprintln(w, "list")
			fmt.Fprintln(w, "fetch")
			fmt.Fprintln(w, "push")
			fmt.Fprintln(w)

		case strings.HasPrefix(text, "list"):
			log.Println("msg", head)
			fmt.Fprintf(w, "%s refs/heads/master\n", head)
			fmt.Fprintln(w, "@refs/heads/master HEAD")
			fmt.Fprintln(w)

		case strings.HasPrefix(text, "fetch"):
			fetchArgs := strings.Split(text, " ")
			objHash := fetchArgs[1]
			log.Println("msg", objHash)
			err := recurseCommit(gitplumbing.NewHash(objHash))
			if err != nil {
				return err
			}
			fmt.Fprintln(w)

		case text == "":
			break

		default:
			return fmt.Errorf("Error: default git speak: %q", text)

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
	p := filepath.Join(GIT_DIR, "HEAD")
	f, err := os.Create(p)
	if err != nil {
		return err
	}
	_, err = f.WriteString("ref: refs/heads/master")
	if err != nil {
		return err
	}
	p = filepath.Join(GIT_DIR, "refs", "heads", "master")
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

func check(err error) {
	if err != nil {
		log.Errorf("%+v", err)
		panic("die")
	}
}
