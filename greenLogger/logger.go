package greenlogger

import (
	"fmt"
	"os"
	"path/filepath"
	"time"
)

var logDirPath = "Logs"
var logFile *os.File

func InitLogFile() {
	os.Mkdir("Logs", os.ModePerm)
	logFilePath := filepath.Join(logDirPath, "GSLog_"+time.Now().String())
	file, err := os.Create(logFilePath)
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

func ElogError(err error, message string) {
	logFile.Write([]byte("ERR at " + time.Now().String() + ": " + message + ": " + err.Error()))
}

func FatalLogMessage(message string) {
	LogMessage("FATAL: " + message)
	logFile.Close()
	os.Exit(1)
}
