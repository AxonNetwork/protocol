package util

import (
	"bufio"
	"context"
	"io/ioutil"
	"os/exec"
	"strings"

	"github.com/pkg/errors"
)

func ExecAndScanStdout(ctx context.Context, cmdAndArgs []string, cwd string, fn func(string) error) (err error) {
	defer func() {
		err = errors.Wrapf(err, "error running %v", strings.Join(cmdAndArgs, " "))
	}()

	cmd := exec.CommandContext(ctx, cmdAndArgs[0], cmdAndArgs[1:]...)
	cmd.Dir = cwd

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		return
	}

	defer func() {
		if err != nil {
			stderr, _ := ioutil.ReadAll(stderr)
			err = errors.Wrapf(err, "stderr: %v", string(stderr))
		}
	}()

	scanner := bufio.NewScanner(stdout)
	for scanner.Scan() {
		line := scanner.Text()
		err = fn(line)
		if err != nil {
			return
		}
	}
	if err = scanner.Err(); err != nil {
		return
	}

	err = cmd.Wait()
	return
}
