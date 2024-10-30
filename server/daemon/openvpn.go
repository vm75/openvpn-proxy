package daemon

import (
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"syscall"
	"time"
)

const (
	binDir      = "/usr/local/bin"
	dataCiphers = "AES-256-GCM:AES-128-GCM:CHACHA20-POLY1305:AES-256-CBC:AES-128-CBC"
)

var shuttingDown = false
var openVpnCmd *exec.Cmd = nil
var loopRunning = false

func fileExists(filename string) bool {
	_, err := os.Stat(filename)
	return !os.IsNotExist(err)
}

func startOpenVPNLoop() {
	if loopRunning {
		return
	}

	// check if config files exist
	if !fileExists(ConfigFile) || !fileExists(AuthFile) {
		log.Println("VPN config/auth file(s) not found")
		return
	}

	execPath, _ := os.Executable()

	loopRunning = true
	for !shuttingDown {
		settings := GetProxySettings()
		retryInterval := strconv.Itoa(settings.RetryInterval)

		log.Println("Starting OpenVPN")
		openVpnCmd = exec.Command("openvpn",
			"--client",
			"--cd", VarDir,
			"--config", ConfigFile,
			"--auth-user-pass", AuthFile,
			"--auth-nocache",
			"--verb", strconv.Itoa(settings.VpnLogLevel),
			"--log", filepath.Join(VarDir, "openvpn.log"),
			"--status", filepath.Join(VarDir, "openvpn.status"), retryInterval,
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

		openVpnCmd.Stdout = os.Stdout
		openVpnCmd.Stderr = os.Stderr

		err := openVpnCmd.Start()
		if err != nil {
			log.Println(err)
			time.Sleep(time.Duration(settings.RetryInterval) * time.Second)
		} else {
			log.Println("OpenVPN started with pid", openVpnCmd.Process.Pid)
			os.WriteFile(PidFile, []byte(strconv.Itoa(openVpnCmd.Process.Pid)), 0644)
			status := openVpnCmd.Wait()
			os.Remove(PidFile)
			log.Printf("OpenVPN exited with status: %v\n", status)
		}

		if shuttingDown {
			break
		}
	}

	loopRunning = false
}

func StartOpenVPNLoop() {
	go startOpenVPNLoop()
}

func StopOpenVPN() {
	if openVpnCmd != nil {
		log.Printf("Stopping OpenVPN with pid %d\n", openVpnCmd.Process.Pid)
		openVpnCmd.Process.Signal(syscall.SIGTERM)
	}
}

func ShutdownOpenVPN() {
	shuttingDown = true
	StopOpenVPN()
}
