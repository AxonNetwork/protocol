package constants

import (
	"os"
)

const BugsnagAPIKey = "5e15c785447c2580f572c382234ecdb1"
const AppVersion = "0.0.1"

var ReleaseStage = func() string {
	stage := os.Getenv("RELEASE_STAGE")
	if stage == "" {
		stage = "dev"
	}
	return stage
}()
