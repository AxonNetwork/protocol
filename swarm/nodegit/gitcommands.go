package nodegit

import (
	"bufio"
	"context"
	"io/ioutil"
	"strconv"
	"strings"

	"github.com/Conscience/protocol/log"
	"github.com/Conscience/protocol/util"
	"github.com/pkg/errors"
)

type MaybeProgress struct {
	Fetched int64
	ToFetch int64
	Error   error
}

func PullRepo(path string, ch chan MaybeProgress) {
	var err error
	defer func() {
		ch <- MaybeProgress{Error: err}
		close(ch)
	}()

	// first stash any local changes
	didStash := false
	ctx := context.Background()
	err = util.ExecAndScanStdout(ctx, []string{"git", "stash"}, path, func(line string) error {
		if line != "No local changes to save" {
			didStash = true
		}
		return nil
	})
	if err != nil {
		log.Errorf("[pull repo]", errors.WithStack(err))
	}

	// then pull changes from network and send progress udpates
	stdout, stderr, closeCmd, err := util.ExecCmd(context.Background(), []string{"git", "pull", "origin", "master"}, path)
	if err != nil {
		return
	}
	// we don't care about stdout
	_, err = ioutil.ReadAll(stdout)
	if err != nil {
		return
	}

	// parse stderr for progress output
	scanner := bufio.NewScanner(stderr)
	for scanner.Scan() {
		line := scanner.Text()
		log.Println("[git pull]", line)
		parts := strings.Split(line, " ")
		if len(parts) == 6 && parts[2] == "Progress:" {
			progress := strings.Split(parts[3], "/")
			Fetched, err := strconv.ParseInt(progress[0], 10, 64)
			if err != nil {
				return
			}
			ToFetch, err := strconv.ParseInt(progress[1], 10, 64)
			if err != nil {
				return
			}
			ch <- MaybeProgress{
				Fetched: Fetched,
				ToFetch: ToFetch,
			}
		}
	}
	if err = scanner.Err(); err != nil {
		return
	}
	if err = closeCmd(); err != nil {
		return
	}

	// finally pop any stashed chnages
	if didStash {
		// @@TODO: handle merge conflict on stash pop
		err = util.ExecAndScanStdout(ctx, []string{"git", "stash", "apply"}, path, func(line string) error {
			return nil
		})
		if err != nil {
			return
		}
		// @@TODO: git stash drop
	}
	return
}
