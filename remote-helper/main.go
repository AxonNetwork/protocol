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

	"../config"
	"../swarm"
)

var (
	GIT_DIR  string = os.Getenv("GIT_DIR")
	client   *swarm.RPCClient
	repo     *git.Repository
	repoUser string
	repoID   string
)

func main() {
	var err error

	remoteURL := os.Args[2]
	remoteURLParts := strings.Split(
		strings.Replace(remoteURL, "conscience://", "", -1),
		"/",
	)
	if len(remoteURLParts) != 2 {
		panic("malformed remote URL")
	}

	repoUser, repoID = remoteURLParts[0], remoteURLParts[1]

	cfg, err := config.ReadConfig()
	if err != nil {
		panic(err)
	}

	client, err = swarm.NewRPCClient(cfg.RPCClient.Network, cfg.RPCClient.Host)
	if err != nil {
		panic(err)
	}

	if GIT_DIR == "" {
		panic("empty GIT_DIR")
	}

	repo, err = git.PlainOpen(filepath.Dir(GIT_DIR))
	if err != nil {
		panic(err)
	}

	err = speakGit(os.Stdin, os.Stdout)
	check(err)
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
			// forPush := strings.Contains(text, "for-push")
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
			err := checkConfig()
			if err != nil {
				return err
			}
			fetchArgs := strings.Split(text, " ")
			objHash := fetchArgs[1]
			err = recurseCommit(gitplumbing.NewHash(objHash))
			if err != nil {
				return err
			}
			fmt.Fprintln(w)

		case strings.HasPrefix(text, "push"):
			for scanner.Scan() {
				pushSplit := strings.Split(text, " ")
				if len(pushSplit) < 2 {
					return fmt.Errorf("malformed 'push' command. %q", text)
				}
				srcDstSplit := strings.Split(pushSplit[1], ":")
				if len(srcDstSplit) < 2 {
					return fmt.Errorf("malformed 'push' command. %q", text)
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
	return nil

}

func check(err error) {
	if err != nil {
		log.Errorf("%+v", err)
		panic("die")
	}
}
