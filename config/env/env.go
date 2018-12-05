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

	BugsnagAPIKey           = getenv("BUGSNAG_API_KEY", "5e15c785447c2580f572c382234ecdb1")
	ReleaseStage            = getenv("RELEASE_STAGE", "dev")
	BugsnagEnabled          = getenv("BUGSNAG_ENABLED", "") != ""
	ConsoleLoggingEnabled   = getenv("CONSOLE_LOGGING", "") != ""
	ChildProcLoggingEnabled = getenv("LOG_CHILD_PROCS", "") != ""
)

func getenv(envvar string, defaultVal string) string {
	x := os.Getenv(envvar)
	if x == "" {
		return defaultVal
	}
	return x
}
