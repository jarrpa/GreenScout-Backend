package userDB

// Utility for the nighly push to github of the databases

import (
	"GreenScoutBackend/constants"
	greenlogger "GreenScoutBackend/greenLogger"
	"os/exec"
	"strings"
)

// Makes a commit to the databases, and then pushes them to the upstream. If there is an upstream mismatch, the push will simply fail.
func CommitAndPushDBs() {
	commitCommand := exec.Command("git", "commit", "-am", "Daily database sync")
	pushCommand := exec.Command("git", "push")

	commitCommand.Dir = "./" + constants.CachedConfigs.PathToDatabases
	pushCommand.Dir = "./" + constants.CachedConfigs.PathToDatabases

	commit, commitErr := commitCommand.Output()
	greenlogger.ELogMessage("Response to committing daily DB sync: " + string(commit))

	if commitErr != nil && !strings.Contains(commitErr.Error(), "exit status 1") {
		greenlogger.LogErrorf(commitErr, "Error Committing daily databases sync")
	} else {
		push, pushErr := pushCommand.Output()
		greenlogger.ELogMessage("Response to pushing daily DB sync: " + string(push))

		if pushErr != nil && !strings.Contains(pushErr.Error(), "exit status 1") {
			greenlogger.LogErrorf(pushErr, "Error pushing daily databases sync")
		}
	}

}
