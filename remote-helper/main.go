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
	repoName string
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

	repoUser, repoName = remoteURLParts[0], remoteURLParts[1]
	repoID = fmt.Sprintf("%s/%s", repoUser, repoName)

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
	if err != nil {
		panic(err)
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
				err := setupConfig()
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
			err := recurseCommit(gitplumbing.NewHash(objHash))
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

func setupConfig() error {
	cfg, err := repo.Config()
	if err != nil {
		return err
	}

	raw := cfg.Raw
	changed := false
	section := raw.Section("conscience")
	if section.Option("username") != repoUser {
		raw.SetOption("conscience", "", "username", repoUser)
		changed = true
	}
	if section.Option("reponame") != repoName {
		raw.SetOption("conscience", "", "reponame", repoName)
		changed = true
	}

	filter := raw.Section("filter").Subsection("conscience")
	if filter.Option("clean") != "conscience_encode" {
		raw.SetOption("filter", "conscience", "clean", "conscience_encode")
		changed = true
	}
	if filter.Option("smudge") != "conscience_decode" {
		raw.SetOption("filter", "conscience", "smudge", "conscience_decode")
		changed = true
	}

	if changed {
		p := filepath.Join(GIT_DIR, "config")
		f, err := os.OpenFile(p, os.O_WRONLY, os.ModeAppend)
		if err != nil {
			return err
		}
		w := io.Writer(f)

		enc := gitconfig.NewEncoder(w)
		err = enc.Encode(raw)
		if err != nil {
			return err
		}
	}

	return nil
}
