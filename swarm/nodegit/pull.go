package nodegit

import (
	"context"

	"github.com/pkg/errors"

	"github.com/Conscience/protocol/log"
	"github.com/Conscience/protocol/util"
)

func PullRepo(ctx context.Context, path string) chan MaybeProgress {
	ch := make(chan MaybeProgress)

	go func() {
		var err error

		defer func() {
			if err != nil {
				ch <- MaybeProgress{Error: errors.WithStack(err)}
			}
			close(ch)
		}()

		// first stash any local changes
		didStash := false
		// @@TODO: should just use CombinedOutput instead of scanning lines
		err = util.ExecAndScanStdout(ctx, []string{"git", "stash"}, path, func(line string) error {
			// @@TODO: this line might vary across different git versions
			if line != "No local changes to save" {
				didStash = true
			}
			return nil
		})
		if err != nil {
			log.Errorf("[pull repo]", errors.WithStack(err))
		}

		// then pull changes from network and send progress updates
		_, stderr, closeCmd, err := util.ExecCmd(ctx, []string{"git", "pull", "origin", "master"}, path)
		if err != nil {
			return
		}

		// parse stderr for progress output
		ParseProgress(stderr, ch)

		if err = closeCmd(); err != nil {
			return
		}

		// finally pop any stashed chnages
		if didStash {
			// @@TODO: handle merge conflict on stash pop
			err = util.ExecAndScanStdout(ctx, []string{"git", "stash", "apply"}, path, func(line string) error { return nil })
			if err != nil {
				return
			}
			// @@TODO: git stash drop
		}
	}()

	return ch
}
