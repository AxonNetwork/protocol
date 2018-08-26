package main

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	gitplumbing "gopkg.in/src-d/go-git.v4/plumbing"

	"../config"
	"../repo"
	"../swarm/noderpc2"
)

var (
	GIT_DIR = os.Getenv("GIT_DIR")
	client  *noderpc.Client
	Repo    *repo.Repo
	repoID  string
)

func main() {
	var err error

	repoID = strings.Replace(os.Args[2], "conscience://", "", -1)

	cfg, err := config.ReadConfig()
	if err != nil {
		die(err)
	}

	client, err = noderpc.NewClient(cfg.RPCClient.Host)
	if err != nil {
		die(err)
	}
	defer client.Close()

	if GIT_DIR == "" {
		fmt.Printf("error: empty GIT_DIR\n")
		os.Exit(1)
	}

	Repo, err = repo.Open(filepath.Dir(GIT_DIR))
	if err != nil {
		die(err)
	}

	err = speakGit(os.Stdin, os.Stdout)
	if err != nil {
		die(err)
	}
}

func speakGit(r io.Reader, w io.Writer) error {
	scanner := bufio.NewScanner(r)
	for scanner.Scan() {
		text := scanner.Text()
		log.Println(text)

		switch {

		case text == "capabilities":
			fmt.Fprintln(w, "list")
			fmt.Fprintln(w, "fetch")
			fmt.Fprintln(w, "push")
			fmt.Fprintln(w)

		case strings.HasPrefix(text, "list"):
			forPush := strings.Contains(text, "for-push")
			// @TODO: find a better spot for this?
			if !forPush {
				err := Repo.SetupConfig(repoID)
				if err != nil {
					return err
				}
			}

			refs, err := getRefs()
			if err != nil {
				return err
			}
			for _, ref := range refs {
				log.Println(ref)
				fmt.Fprintln(w, ref)
			}
			fmt.Fprintln(w)

		case strings.HasPrefix(text, "fetch"):
			fetchArgs := strings.Split(text, " ")
			objHash := fetchArgs[1]
			err := fetch(gitplumbing.NewHash(objHash))
			if err != nil {
				return err
			}

			// Tell the node to track this repo
			fullpath, err := filepath.Abs(filepath.Dir(GIT_DIR))
			if err != nil {
				return err
			}
			// @@TODO: give context a timeout and make it configurable
			err = client.TrackLocalRepo(context.Background(), fullpath)
			if err != nil {
				return err
			}
			fmt.Fprintln(w)

		case strings.HasPrefix(text, "push"):
			for scanner.Scan() {
				pushSplit := strings.Split(text, " ")
				if len(pushSplit) < 2 {
					return errors.Errorf("malformed 'push' command. %q", text)
				}
				srcDstSplit := strings.Split(pushSplit[1], ":")
				if len(srcDstSplit) < 2 {
					return errors.Errorf("malformed 'push' command. %q", text)
				}
				src, dst := srcDstSplit[0], srcDstSplit[1]
				err := push(src, dst)
				if err != nil {
					return err
				}
				text = scanner.Text()
				if text == "" {
					break
				}
			}
			fmt.Fprintln(w)

		case text == "":
			break

		default:
			return fmt.Errorf("Error: default git speak: %q", text)

		}
	}
	return scanner.Err()
}

func die(err error) {
	log.Errorf("%+v\n", err)
	os.Exit(1)
}
