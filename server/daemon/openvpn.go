package daemon

import (
	"encoding/json"
	"fmt"
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

var reconnect = true
var openVpnCmd *exec.Cmd = nil
var loopRunning = false
var IpInfo map[string]string = map[string]string{}

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
	for reconnect {
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
			sleepFor := max(settings.RetryInterval, 60)
			time.Sleep(time.Duration(sleepFor) * time.Second)
		} else {
			log.Println("OpenVPN started with pid", openVpnCmd.Process.Pid)
			os.WriteFile(PidFile, []byte(strconv.Itoa(openVpnCmd.Process.Pid)), 0644)
			status := openVpnCmd.Wait()
			IpInfo = map[string]string{}
			os.Remove(PidFile)
			log.Printf("OpenVPN exited with status: %v\n", status)
		}

		if !reconnect {
			break
		}
	}

	loopRunning = false
}

func StartVPN() {
	reconnect = true
	go startOpenVPNLoop()
}

func RestartVPN() {
	if openVpnCmd != nil {
		log.Printf("Stopping OpenVPN with pid %d\n", openVpnCmd.Process.Pid)
		openVpnCmd.Process.Signal(syscall.SIGTERM)
	}
}

func StopVPN() {
	reconnect = false
	RestartVPN()
}

type VPNStatus struct {
	Running bool              `json:"running"`
	IpInfo  map[string]string `json:"ipInfo"`
}

func getIpInfo() {
	// https://worldtimeapi.org/api/ip
	cmd := exec.Command("/usr/bin/wget", "-q", "-O", "-", "https://ipinfo.io/json")
	out, err := cmd.CombinedOutput()
	if err != nil {
		fmt.Fprintln(logFile, string(out))
		IpInfo = map[string]string{}
		return
	}

	err = json.Unmarshal(out, &IpInfo)
	if err != nil {
		fmt.Fprintln(logFile, err)
	}
}

func GetVPNStatus() VPNStatus {
	running := false
	if openVpnCmd == nil || openVpnCmd.Process == nil || (openVpnCmd.ProcessState != nil && openVpnCmd.ProcessState.Exited()) {
		IpInfo = map[string]string{}
		return VPNStatus{
			Running: false,
			IpInfo:  IpInfo,
		}
	}

	running = true
	if len(IpInfo) == 0 {
		getIpInfo()
	}

	return VPNStatus{
		Running: running,
		IpInfo:  IpInfo,
	}
}
