package utils

import (
	"fmt"
	"os"
)

var logFile *os.File

func InitLog(filePath string) {
	logFile, _ = os.OpenFile(filePath, os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666)
}

func Log(msg string) {
	fmt.Fprintln(logFile, msg)
}

func LogError(msg string, err error) {
	fmt.Fprintln(logFile, msg, err)
}

func GetLogFile() *os.File {
	return logFile
}
