package util

import (
	"bufio"
	"context"
	"io/ioutil"
	"os"
	"os/exec"
	"strings"

	"github.com/pkg/errors"
)

func EnvToMap(env []string) map[string]string {
	m := make(map[string]string, len(env))
	for _, line := range env {
		pair := strings.Split(line, "=")
		if len(pair) < 2 {
			continue
		}
		m[pair[0]] = m[pair[1]]
	}
	return m
}

func MapToEnv(m map[string]string) []string {
	env := make([]string, len(m))
	i := 0
	for k, v := range m {
		env[i] = k + "=" + v
		i++
	}
	return env
}

func CopyEnv() []string {
	envMap := EnvToMap(os.Environ())
	if envMap["PATH"] == "" {
		envMap["PATH"] = "/usr/local/bin:/usr/bin:/bin:/usr/sbin:/sbin"
	} else {
		envMap["PATH"] = envMap["PATH"] + ":" + "/usr/local/bin:/usr/bin:/bin:/usr/sbin:/sbin"
	}
	return MapToEnv(envMap)
}

func ExecAndScanStdout(ctx context.Context, cmdAndArgs []string, cwd string, fn func(string) error) (err error) {
	defer func() {
		err = errors.Wrapf(err, "error running %v", strings.Join(cmdAndArgs, " "))
	}()

	var args []string
	if len(cmdAndArgs) == 1 {
		args = []string{}
	} else {
		args = cmdAndArgs[1:]
	}

	cmd := exec.CommandContext(ctx, cmdAndArgs[0], args...)
	cmd.Dir = cwd
	cmd.Env = CopyEnv()

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

	err = cmd.Start()
	if err != nil {
		return err
	}

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
