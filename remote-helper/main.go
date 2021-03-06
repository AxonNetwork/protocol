package main

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/pkg/errors"

	"github.com/Conscience/protocol/config"
	"github.com/Conscience/protocol/log"
	"github.com/Conscience/protocol/repo"
	"github.com/Conscience/protocol/swarm/noderpc"
)

var (
	GIT_DIR = os.Getenv("GIT_DIR")
	client  *noderpc.Client
	Repo    *repo.Repo
	repoID  string
)

func main() {
	log.SetField("App", "git-remote-axon")

	cfg, err := config.ReadConfig()
	if err != nil {
		die(err)
	}
	config.AttachToLogger(cfg)

	if GIT_DIR == "" {
		die(errors.New("empty GIT_DIR"))
	}

	repoID = strings.Replace(os.Args[2], "axon://", "", -1)

	client, err = noderpc.NewClient(cfg.RPCClient.Host)
	if err != nil {
		die(err)
	}
	defer client.Close()

	gitDir, err := filepath.Abs(filepath.Dir(GIT_DIR))
	if err != nil {
		die(err)
	}

	Repo, err = repo.Open(gitDir)
	if err != nil {
		die(err)
	}
	defer Repo.Free()

	err = speakGit(os.Stdin, os.Stdout)
	if err != nil {
		die(err)
	}
}

func speakGit(r io.Reader, w io.Writer) error {
	scanner := bufio.NewScanner(r)
	for scanner.Scan() {
		text := scanner.Text()
		text = strings.TrimSpace(text)
		log.Println("[git]", text)

		switch {

		case strings.HasPrefix(text, "capabilities"):
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
			commitHash := fetchArgs[1]
			err := fetchFromCommit_packfile(commitHash)
			if err != nil {
				return err
			}

			err = trackRepo(true)
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
			// The blank line is the stream terminator.  We return when we see this.
			err := trackRepo(false)
			if err != nil {
				return err
			}
			if runtime.GOOS == "windows" {
				return nil
			}

		default:
			return fmt.Errorf("unknown git speak: %v", text)
		}
	}
	return scanner.Err()
}

func die(err error) {
	fmt.Fprintf(os.Stderr, "error: %+v\n", err)
	os.Exit(1)
}

func trackRepo(forceReload bool) error {
	// Tell the node to track this repo
	fullpath, err := filepath.Abs(filepath.Dir(GIT_DIR))
	if err != nil {
		return err
	}
	// @@TODO: give context a timeout and make it configurable
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	err = client.TrackLocalRepo(ctx, fullpath, forceReload)
	if err != nil {
		return err
	}
	return nil
}
