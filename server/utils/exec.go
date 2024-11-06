package utils

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
	"syscall"
)

func RunCommand(command string, args ...string) {
	Log(fmt.Sprintf("Executing: %s %s", command, strings.Join(args, " ")))

	cmd := exec.Command(command, args...)
	cmd.Env = append(os.Environ(), "PATH=/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin")
	cmd.Stdout = logFile
	cmd.Stderr = logFile
	cmd.SysProcAttr = &syscall.SysProcAttr{
		Setpgid: true, // Create a new process group
	}
	err := cmd.Run()
	if err != nil {
		fmt.Fprintln(logFile, err)
	}
}
