package userDB

import (
	"GreenScoutBackend/constants"
	greenlogger "GreenScoutBackend/greenLogger"
	"os/exec"
	"strings"
)

func CommitAndPushDBs() {
	commitCommand := exec.Command("git", "commit", "-am", `"Daily database sync"`)
	pushCommand := exec.Command("git", "push")

	commitCommand.Dir = "./" + constants.CachedConfigs.PathToDatabases
	pushCommand.Dir = "./" + constants.CachedConfigs.PathToDatabases

	out, err := commitCommand.Output()

	if err != nil && !strings.Contains(err.Error(), "exit status 1") {
		greenlogger.LogErrorf(err, "Error executing command %v %v %v %v", "git", "commit", "-am", `"Daily database sync"`)
	}

	greenlogger.LogMessagef("%v", out)
}
