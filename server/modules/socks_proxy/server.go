package socks_proxy

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
	proxyCmd = exec.Command("/usr/local/sbin/sockd", "-f", configFile)

	proxyCmd.Stdout = os.Stdout
	proxyCmd.Stderr = os.Stderr

	err := proxyCmd.Start()
	if err != nil {
		log.Println(err)
	} else {
		log.Println("Socks Proxy started with pid", proxyCmd.Process.Pid)
		os.WriteFile(pidFile, []byte(strconv.Itoa(proxyCmd.Process.Pid)), 0644)
		status := proxyCmd.Wait()
		os.Remove(pidFile)
		log.Printf("Socks Proxy exited with status: %v\n", status)
	}
}

func stopProxy() {
	utils.SignalCmd(proxyCmd, syscall.SIGTERM)
	proxyCmd.Wait()
}
