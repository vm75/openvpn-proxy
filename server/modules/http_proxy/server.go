package http_proxy

import (
	"log"
	"os"
	"os/exec"
	"strconv"
	"syscall"
	"vpn-sandbox/utils"
)

var proxyCmd *exec.Cmd = nil

func isRunning() bool {
	return utils.IsRunning(proxyCmd)
}

func startProxy() {
	if utils.IsRunning(proxyCmd) {
		return
	}
	proxyCmd = exec.Command("/usr/bin/tinyproxy", "-d", "-c", configFile)

	proxyCmd.Stdout = os.Stdout
	proxyCmd.Stderr = os.Stderr

	err := proxyCmd.Start()
	if err != nil {
		log.Println(err)
	} else {
		log.Println("Http Proxy started with pid", proxyCmd.Process.Pid)
		os.WriteFile(pidFile, []byte(strconv.Itoa(proxyCmd.Process.Pid)), 0644)
		status := proxyCmd.Wait()
		os.Remove(pidFile)
		log.Printf("Http Proxy exited with status: %v\n", status)
	}
}

func stopProxy() {
	utils.SignalCmd(proxyCmd, syscall.SIGTERM)
	proxyCmd.Wait()
}
