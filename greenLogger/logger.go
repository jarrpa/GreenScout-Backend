package greenlogger

import (
	"GreenScoutBackend/constants"
	filemanager "GreenScoutBackend/fileManager"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

var logDirPath = "Logs"
var logFile *os.File

func InitLogFile() {
	filemanager.MkDirWithPermissions("Logs")
	logFilePath := filepath.Join(logDirPath, "GSLog_"+time.Now().String())
	file, err := filemanager.OpenWithPermissions(logFilePath)
	if err != nil {
		panic("ERR: Could not create log file! " + err.Error())
	}

	logFile = file
}

func LogErrorf(err error, message string, args ...any) {
	formatted := fmt.Sprintf(message, args...)
	LogError(err, formatted)
}

func LogError(err error, message string) {
	fmt.Println("ERR: " + message + ": " + err.Error())
	if constants.CachedConfigs.SlackConfigs.UsingSlack && slackAlive {
		NotifyError(err, message)
	}
	ElogError(err, message)
}

func LogMessage(message string) {
	fmt.Println(message)
	ELogMessage(message)
}

func LogMessagef(message string, args ...any) {
	formatted := fmt.Sprintf(message, args...)
	fmt.Println(formatted)
	ELogMessage(formatted)
}

func ELogMessage(message string) {
	logFile.Write([]byte(time.Now().String() + ": " + message + "\n"))
}

func ELogMessagef(message string, args ...any) {
	formatted := fmt.Sprintf(message, args...)
	logFile.Write([]byte(time.Now().String() + ": " + formatted + "\n"))
}

func ElogError(err error, message string) {
	logFile.Write([]byte("ERR at " + time.Now().String() + ": " + message + ": " + err.Error() + "\n"))
}

func FatalLogMessage(message string) {
	LogMessage("FATAL: " + message)
	logFile.Close()
	os.Exit(1)
}

func FatalError(err error, message string) {
	LogError(err, message)
	logFile.Close()
	os.Exit(1)
}

func HandleMkdirAll(filepath string) {
	mkDirErr := filemanager.MkDirWithPermissions(filepath)

	if mkDirErr != nil {
		LogErrorf(mkDirErr, "Problem making directory %v", filepath)
	}
}
