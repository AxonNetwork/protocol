package nodegit

import (
	"context"

	"github.com/pkg/errors"

	"github.com/Conscience/protocol/util"
)

func CloneRepo(ctx context.Context, repoRoot string, repoID string) chan MaybeProgress {
	ch := make(chan MaybeProgress)

	go func() {
		var err error

		defer func() {
			if err != nil {
				ch <- MaybeProgress{Error: errors.WithStack(err)}
			}
			close(ch)
		}()

		cloneEnv := []string{"MACHINE_OUTPUT=true"}
		_, stderr, closeCmd, err := util.ExecCmdWithEnv(ctx, []string{"git", "clone", "conscience://" + repoID}, repoRoot, cloneEnv)
		if err != nil {
			return
		}

		err = ParseProgress(stderr, ch)
		if err != nil {
			closeCmd()
			return
		}

		err = closeCmd()
		if err != nil {
			return
		}
	}()

	return ch
}