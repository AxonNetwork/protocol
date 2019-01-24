package nodegit

import (
	"context"
	"io"

	gitplumbing "gopkg.in/src-d/go-git.v4/plumbing"

	"github.com/Conscience/protocol/util"
)

func GetDiff(ctx context.Context, path string, commitHash gitplumbing.Hash) (io.ReadCloser, error) {
	var err error

	// then pull changes from network and send progress updates
	stdout, _, closeCmd, err := util.ExecCmd(ctx, []string{"git", "show", commitHash.String()}, path)
	if err != nil {
		return nil, err
	}

	return readCloser{stdout, closeCmd}, nil
}

type readCloser struct {
	io.Reader
	closeCmd func() error
}

func (rc readCloser) Close() error {
	return rc.closeCmd()
}
