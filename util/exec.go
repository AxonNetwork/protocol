package util

import (
	"bufio"
	"context"
	"fmt"
	"io/ioutil"
	"math/rand"
	"os"
	"os/exec"
	"strings"

	"github.com/pkg/errors"
)

func envToMap(env []string) map[string]string {
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

func mapToEnv(m map[string]string) []string {
	env := make([]string, len(m))
	i := 0
	for k, v := range m {
		env[i] = k + "=" + v
		i++
	}
	return env
}

func ExecAndScanStdout(ctx context.Context, cmdAndArgs []string, cwd string, fn func(string) error) (err error) {
	defer func() {
		err = errors.Wrapf(err, "error running %v", strings.Join(cmdAndArgs, " "))
	}()

	asdf := []string{}
	for _, x := range cmdAndArgs {
		asdf = append(asdf, strings.Replace(x, "/", "-", -1))
	}
	f, err := os.OpenFile(fmt.Sprintf("/tmp/conscience-proc-"+strings.Join(asdf, "-")+"-%v", rand.Int()), os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	defer f.Close()

	cmd := exec.CommandContext(ctx, cmdAndArgs[0], cmdAndArgs[1:]...)
	cmd.Dir = cwd
	envMap := envToMap(os.Environ())
	if envMap["PATH"] == "" {
		envMap["PATH"] = "/usr/local/bin:/usr/bin:/bin:/usr/sbin:/sbin"
	} else {
		envMap["PATH"] = envMap["PATH"] + ":" + "/usr/local/bin:/usr/bin:/bin:/usr/sbin:/sbin"
	}
	cmd.Env = mapToEnv(envMap)

	for _, v := range cmd.Env {
		f.WriteString(v)
	}

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
