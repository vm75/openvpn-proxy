package openvpn

import (
	"log"
	"os"
	"os/exec"
	"strconv"
	"syscall"
	"time"
	"vpn-sandbox/core"
	"vpn-sandbox/utils"
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
	if isRunning || !openvpnConfig.Enabled {
		return
	}

	// check if config files exist
	if !fileExists(configFile) || !fileExists(authFile) {
		log.Println("VPN config/auth file(s) not found")
		return
	}

	execPath, _ := os.Executable()

	isRunning = true
	for openvpnConfig.Enabled {
		retryInterval := strconv.Itoa(openvpnConfig.RetryInterval)

		log.Println("Starting OpenVPN")
		openvpnCmd = exec.Command("openvpn",
			"--client",
			"--cd", core.VarDir,
			"--config", configFile,
			"--auth-user-pass", authFile,
			"--auth-nocache",
			"--verb", strconv.Itoa(openvpnConfig.LogLevel),
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
			"--writepid", pidFile,
		)

		openvpnCmd.Stdout = os.Stdout
		openvpnCmd.Stderr = os.Stderr

		err := openvpnCmd.Start()
		if err != nil {
			log.Println(err)
			sleepFor := max(openvpnConfig.RetryInterval, 60)
			time.Sleep(time.Duration(sleepFor) * time.Second)
		} else {
			log.Println("OpenVPN started with pid", openvpnCmd.Process.Pid)
			status := openvpnCmd.Wait()
			log.Printf("OpenVPN exited with status: %v\n", status)
			sleepFor := max(openvpnConfig.RetryInterval, 60)
			time.Sleep(time.Duration(sleepFor) * time.Second)
		}

		if !openvpnConfig.Enabled {
			break
		}
	}

	isRunning = false
}

func killOpenVPN() {
	utils.SignalCmd(openvpnCmd, syscall.SIGTERM)
	// openvpnCmd.Wait()
}
