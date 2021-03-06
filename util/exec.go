package util

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"syscall"
	"time"

	"github.com/pkg/errors"

	"github.com/Conscience/protocol/config/env"
	"github.com/Conscience/protocol/log"
)

var (
	logChildProcessesToFile = os.Getenv("LOG_CHILD_PROCS") != ""
)

func EnvToMap(env []string) map[string]string {
	m := make(map[string]string, len(env))
	for _, line := range env {
		pair := strings.Split(line, "=")
		if len(pair) < 2 {
			continue
		}
		m[pair[0]] = pair[1]
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

// Accepts a PATH string directly from the environment (of the format "path1:path2:path3" on POSIX or
// "path1;path2;path3" on Windows), prepends the paths in `newPaths` to it, filters any empty
// paths, and returns the result as a joined string.
func PrependToPathList(PATH string, newPaths ...string) string {
	pathList := strings.Split(PATH, string(os.PathListSeparator))
	pathList = append(newPaths, pathList...)
	pathListFiltered := []string{}
	for _, x := range pathList {
		if x != "" {
			pathListFiltered = append(pathListFiltered, x)
		}
	}
	return strings.Join(pathListFiltered, string(os.PathListSeparator))
}

// Copies the current environment (from `os.Environ()`), prepends the path in CONSCIENCE_BINARIES_PATH,
// and returns the result.
func CopyEnv() []string {
	envMap := EnvToMap(os.Environ())
	envMap["PATH"] = PrependToPathList(envMap["PATH"], envMap["CONSCIENCE_BINARIES_PATH"])
	return MapToEnv(envMap)
}

var logfilePrefixSanitizer = regexp.MustCompile("[^a-zA-Z0-9]+")

func getLogfilePrefix(cmdAndArgs []string) string {
	sanitized := make([]string, len(cmdAndArgs))
	for i := range cmdAndArgs {
		sanitized[i] = logfilePrefixSanitizer.ReplaceAllString(cmdAndArgs[i], "")
	}
	return fmt.Sprintf("cmd--%v--%v", strings.Join(sanitized, "-"), time.Now().UTC().UnixNano())
}

func logEnvironment(logfilePrefix string, vars []string) {
	logfile_env, err := os.Create(filepath.Join(env.HOME, fmt.Sprintf("%v--env.txt", logfilePrefix)))
	if err != nil {
		log.Errorln("ExecCmd error (writing environment):", err)
		return
	}
	defer logfile_env.Close()

	for _, line := range vars {
		logfile_env.WriteString(fmt.Sprintf("%v\r\n", line))
	}
}

func logStdoutStderr(logfilePrefix string, stdout, stderr io.Reader) (io.Reader, io.Reader, func()) {
	logfile_stdout, err := os.Create(filepath.Join(env.HOME, fmt.Sprintf("%v--stdout.txt", logfilePrefix)))
	if err != nil {
		log.Errorln("ExecCmd error (writing stdout):", err)
		return stdout, stderr, func() {}
	}

	logfile_stderr, err := os.Create(filepath.Join(env.HOME, fmt.Sprintf("%v--stderr.txt", logfilePrefix)))
	if err != nil {
		log.Errorln("ExecCmd error (writing stderr):", err)
		logfile_stdout.Close()
		return stdout, stderr, func() {}
	}

	closeFn := func() {
		ioutil.ReadAll(stdout)
		ioutil.ReadAll(stderr)
		logfile_stdout.WriteString("\r\n(end)\r\n")
		logfile_stderr.WriteString("\r\n(end)\r\n")
		logfile_stdout.Close()
		logfile_stderr.Close()
	}

	return io.TeeReader(stdout, logfile_stdout), io.TeeReader(stderr, logfile_stderr), closeFn
}

func ExecCmdWithEnv(ctx context.Context, cmdAndArgs []string, cwd string, withEnv []string) (stdout io.Reader, stderr io.Reader, closeProc func() error, err error) {
	defer func() {
		if err != nil {
			err = errors.Wrapf(err, "error running '%v'", strings.Join(cmdAndArgs, " "))
		}
	}()

	var args []string
	if len(cmdAndArgs) == 1 {
		args = []string{}
	} else {
		args = cmdAndArgs[1:]
	}

	cmd := exec.CommandContext(ctx, cmdAndArgs[0], args...)
	cmd.Dir = cwd
	cmd.Env = append(CopyEnv(), withEnv...)

	stdout, err = cmd.StdoutPipe()
	if err != nil {
		return nil, nil, nil, err
	}

	stderr, err = cmd.StderrPipe()
	if err != nil {
		return nil, nil, nil, err
	}

	var closeLogfiles func()
	if env.ChildProcLoggingEnabled {
		logfilePrefix := getLogfilePrefix(cmdAndArgs)

		logEnvironment(logfilePrefix, CopyEnv())
		stdout, stderr, closeLogfiles = logStdoutStderr(logfilePrefix, stdout, stderr)
	}

	err = cmd.Start()
	if err != nil {
		if closeLogfiles != nil {
			closeLogfiles()
		}
		return nil, nil, nil, err
	}

	// caller's responsibility to call close
	// closeProc also closes stdout/stderr readers
	closeProc = func() error {
		if closeLogfiles != nil {
			closeLogfiles()
		}
		return cmd.Wait()
	}

	return stdout, stderr, closeProc, nil
}

func ExecCmd(ctx context.Context, cmdAndArgs []string, cwd string) (stdout io.Reader, stderr io.Reader, closeProc func() error, err error) {
	return ExecCmdWithEnv(ctx, cmdAndArgs, cwd, []string{})
}

func ExecAndScanStdout(ctx context.Context, cmdAndArgs []string, cwd string, fn func(string) error) (err error) {
	stdout, stderr, closeProc, err := ExecCmd(ctx, cmdAndArgs, cwd)
	if err != nil {
		return
	}
	defer CheckCloseFunc(closeProc, &err)

	defer func() {
		stderrBytes, _ := ioutil.ReadAll(stderr)
		if err != nil {
			err = errors.Wrapf(err, "stderr: %v", string(stderrBytes))
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

	return
}

func ExitCodeForError(err error) int {
	if msg, ok := err.(*exec.ExitError); ok { // there is error code
		return msg.Sys().(syscall.WaitStatus).ExitStatus()
	} else {
		return 0
	}
	// windows code
	// https://groups.google.com/forum/#!msg/golang-nuts/8XIlxWgpdJw/Z8s2N-SoWHsJ
	// if err != nil {
	//         if e2, ok := err.(*exec.ExitError); ok {
	//                 if s, ok := e2.Sys().(syscall.WaitStatus); ok {
	//                         return int(s.ExitCode), nil
	//                 }
	//         }
	//         return 0, err
	// }
	// return 0, nil
}
