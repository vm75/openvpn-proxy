package openvpn

import (
	"log"
	"openvpn-proxy/core"
	"os"
	"os/exec"
	"strconv"
	"syscall"
	"time"
)

const (
	binDir      = "/usr/local/bin"
	dataCiphers = "AES-256-GCM:AES-128-GCM:CHACHA20-POLY1305:AES-256-CBC:AES-128-CBC"
)

var openvpnCmd *exec.Cmd = nil
var isRunning = false

func fileExists(filename string) bool {
	_, err := os.Stat(filename)
	return !os.IsNotExist(err)
}

func runOpenVPN() {
	if isRunning || !openvpnSettings.Enabled {
		return
	}

	// check if config files exist
	if !fileExists(configFile) || !fileExists(authFile) {
		log.Println("VPN config/auth file(s) not found")
		return
	}

	execPath, _ := os.Executable()

	isRunning = true
	for openvpnSettings.Enabled {
		retryInterval := strconv.Itoa(openvpnSettings.RetryInterval)

		log.Println("Starting OpenVPN")
		openvpnCmd = exec.Command("openvpn",
			"--client",
			"--cd", core.VarDir,
			"--config", configFile,
			"--auth-user-pass", authFile,
			"--auth-nocache",
			"--verb", strconv.Itoa(openvpnSettings.LogLevel),
			"--log", logFile,
			"--status", statusFile, retryInterval,
			"--ping-restart", retryInterval,
			"--connect-retry-max", "3",
			"--script-security", "2",
			"--up", execPath, "--up-delay",
			"--down", execPath,
			"--up-restart",
			"--pull-filter", "ignore", "route-ipv6",
			"--pull-filter", "ignore", "ifconfig-ipv6",
			"--pull-filter", "ignore", "block-outside-dns",
			"--redirect-gateway", "def1",
			"--remote-cert-tls", "server",
			"--data-ciphers", dataCiphers,
		)

		openvpnCmd.Stdout = os.Stdout
		openvpnCmd.Stderr = os.Stderr

		err := openvpnCmd.Start()
		if err != nil {
			log.Println(err)
			sleepFor := max(openvpnSettings.RetryInterval, 60)
			time.Sleep(time.Duration(sleepFor) * time.Second)
		} else {
			log.Println("OpenVPN started with pid", openvpnCmd.Process.Pid)
			os.WriteFile(pidFile, []byte(strconv.Itoa(openvpnCmd.Process.Pid)), 0644)
			status := openvpnCmd.Wait()
			os.Remove(pidFile)
			log.Printf("OpenVPN exited with status: %v\n", status)
		}

		if !openvpnSettings.Enabled {
			break
		}
	}

	isRunning = false
}

func killOpenVPN() {
	if openvpnCmd != nil {
		log.Printf("Stopping OpenVPN with pid %d\n", openvpnCmd.Process.Pid)
		openvpnCmd.Process.Signal(syscall.SIGTERM)
		// openvpnCmd.Wait()
	}
}
