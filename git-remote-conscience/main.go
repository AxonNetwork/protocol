// TO TEST:
// 1) open 2 terminals and instantiate peer nodes, one on 1337 and one on 8080
// 2) call add-peer between them. 
// 3) in a testing folder, create a git repo
// 4) call "add-repo <route to git repo>" on the peer open at 8080
// 5) open a new terminal with root at git-remote conscience
// 6) run "go build *.go"
// 7) run mv fetch /usr/local/bin/git-remote-conscience
// 8) run git clone conscience://conscience/testing

package main

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"github.com/cryptix/go/logging"
	"github.com/pkg/errors"
)

var (
	ref2hash = make(map[string]string)
	conscienceRepoPath  string
	thisGitRepo   string
	thisGitRemote string
	errc          chan<- error
	log           logging.Interface
	check         = logging.CheckFatal
)

func logFatal(msg string) {
	log.Log("event", "fatal", "msg", msg)
	os.Exit(1)
}

func main() {
	// logging

	fmt.Printf("%v", os.Args)

	logging.SetupLogging(nil)
	log = logging.Logger("git-remote-conscience")

	// env var and arguments
	thisGitRepo = os.Getenv("GIT_DIR")
	if thisGitRepo == "" {
		logFatal("could not get GIT_DIR env var")
	}
	if thisGitRepo == ".git" {
		cwd, err := os.Getwd()
		logging.CheckFatal(err)
		thisGitRepo = filepath.Join(cwd, ".git")
	}

	var u string // repo url
	v := len(os.Args[1:])
	switch v {
	case 2:
		thisGitRemote = os.Args[1]
		u = os.Args[2]
	default:
		logFatal(fmt.Sprintf("usage: unknown # of args: %d\n%v", v, os.Args[1:]))
	}

	conscienceRepoPath = strings.SplitAfter(u, "conscience://conscience/")[1]
	log.Log("msg", conscienceRepoPath)

	// interrupt / error handling
	go func() {
		check(interrupt())
	}()

	check(speakGit(os.Stdin, os.Stdout))
}

func speakGit(r io.Reader, w io.Writer) error {
	scanner := bufio.NewScanner(r)
	for scanner.Scan() {
		text := scanner.Text()

		fmt.Printf(text)
		switch {

		case text == "capabilities":
			fmt.Fprintln(w, "fetch")
			fmt.Fprintln(w, "push")
			fmt.Fprintln(w, "")

		case strings.HasPrefix(text, "list"):
			listRefs(conscienceRepoPath)
			fmt.Fprintf(w, "%s %s\n", "089be761cc888ccc50b534d7604f9276f2eafc63", "refs/head/master")	
			fmt.Fprintf(w, "%s HEAD\n", "089be761cc888ccc50b534d7604f9276f2eafc63")
			fmt.Fprintln(w)

		case strings.HasPrefix(text, "fetch "):
			log.Log("msg", "fetching!")
			for scanner.Scan() {
				fetchSplit := strings.Split(text, " ")
				if len(fetchSplit) < 2 {
					return errors.Errorf("malformed 'fetch' command. %q", text)
				}
				err := fetchObject(fetchSplit[1])
				if err == nil {
					fmt.Fprintln(w)
					continue
				}
				// TODO isNotExist(err) would be nice here
				//log.Log("sha1", fetchSplit[1], "name", fetchSplit[2], "err", err, "msg", "fetchLooseObject failed, trying packed...")

				err = fetchPackedObject(fetchSplit[1])
				if err != nil {
					return errors.Wrap(err, "fetchPackedObject() failed")
				}
				text = scanner.Text()
				if text == "" {
					break
				}
			}
			fmt.Fprintln(w, "")

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
				f := []interface{}{
					"src", src,
					"dst", dst,
				}
				log.Log(append(f, "msg", "got push"))
				if src == "" {
					fmt.Fprintf(w, "error %s %s\n", dst, "delete remote dst: not supported yet - please open an issue on github")
				} else {
					if err := push(src, dst); err != nil {
						fmt.Fprintf(w, "error %s %s\n", dst, err)
						return err
					}
					fmt.Fprintln(w, "ok", dst)
				}
				text = scanner.Text()
				if text == "" {
					break
				}
			}
			fmt.Fprintln(w, "")

		case text == "":
			break

		default:
			return errors.Errorf("Error: default git speak: %q", text)
		}
	}
	if err := scanner.Err(); err != nil {
		return errors.Wrap(err, "scanner.Err()")
	}
	return nil
}
