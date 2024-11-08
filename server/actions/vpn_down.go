package actions

import (
	"openvpn-proxy/core"
	"openvpn-proxy/utils"
	"os"
	"path/filepath"
	"strings"
)

func getHostGateway() string {
	if _, err := os.Stat("/etc/resolv.conf"); !os.IsNotExist(err) {
		fileContent, err := os.ReadFile("/etc/resolv.conf.ovpnsave")
		if err == nil {
			// extract first nameserver as host gateway
			lines := strings.Split(string(fileContent), "\n")
			for _, line := range lines {
				if strings.HasPrefix(line, "nameserver") {
					return strings.Split(line, " ")[1]
				}
			}
		}
	}

	return ""
}

func VpnDown() {
	utils.InitLog(filepath.Join(core.VarDir, "vpn-down.log"))
	utils.Log("vpn down")

	// restore resolv.conf
	utils.RestoreResolvConf()

	// get host gateway from resolv.conf
	hostGateway := getHostGateway()
	utils.Log("host gateway: " + hostGateway)

	// remove resolv.conf.ovpnsave
	if _, err := os.Stat("/etc/resolv.conf.ovpnsave"); !os.IsNotExist(err) {
		if err := os.Remove("/etc/resolv.conf.ovpnsave"); err != nil {
			utils.LogError("Error removing /etc/resolv.conf.ovpnsave", err)
		}
	}

	// Set routes
	// Remove all existing default routes
	utils.RunCommand("/sbin/ip", "route", "del", "default")

	// Add default gateway
	utils.RunCommand("/sbin/ip", "route", "add", "default", "via", hostGateway, "dev", "eth0")

	// Set firewall rules
	// Flush existing rules to start fresh
	utils.RunCommand("/sbin/iptables", "-F")

	// Allow related and established connections (for existing sessions to work)
	utils.RunCommand("/sbin/iptables", "-A", "INPUT", "-m", "conntrack", "--ctstate", "RELATED,ESTABLISHED", "-j", "ACCEPT")

	// Allow incoming connections only on port 80
	utils.RunCommand("/sbin/iptables", "-A", "INPUT", "-p", "tcp", "--dport", "80", "-j", "ACCEPT")

	// Drop all other incoming connections
	utils.RunCommand("/sbin/iptables", "-A", "INPUT", "-j", "DROP")

	// Trigger vpn down actions
	utils.RunCommand(core.AppScript, "down")
}
