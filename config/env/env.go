package env

import (
	"os"

	"github.com/mitchellh/go-homedir"
)

const AppVersion = "0.0.1"

var (
	HOME = func() string {
		dir, err := homedir.Dir()
		if err != nil {
			panic(err)
		}
		return dir
	}()

	BugsnagEnabled = getenv("BUGSNAG_ENABLED", "") != ""
	BugsnagAPIKey  = getenv("BUGSNAG_API_KEY", "5e15c785447c2580f572c382234ecdb1")
	ReleaseStage   = getenv("RELEASE_STAGE", "dev")

	// This controls whether or not log messages are displayed in the console.
	ConsoleLoggingEnabled = getenv("CONSOLE_LOGGING", "") != ""

	// This determines whether stdout and stderr for child processes spawned by the node (such as
	// git) will be logged to files for debugging purposes.
	ChildProcLoggingEnabled = getenv("LOG_CHILD_PROCS", "") != ""

	// This controls the output of the git-remote-conscience helper.  If enabled, the helper will
	// output progress information in a format more easily readable by other programs.  Otherwise,
	// it will print human-friendly output.
	MachineOutputEnabled = getenv("MACHINE_OUTPUT", "") != ""
)

func getenv(envvar string, defaultVal string) string {
	x := os.Getenv(envvar)
	if x == "" {
		return defaultVal
	}
	return x
}
