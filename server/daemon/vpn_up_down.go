package daemon

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
	"syscall"
)

var logFile *os.File

func vpnLog(msg string) {
	fmt.Fprintln(logFile, msg)
}

func vpnLogError(msg string, err error) {
	fmt.Fprintln(logFile, msg, err)
}

func run(command string, args ...string) {
	vpnLog(fmt.Sprintf("Executing: %s %s", command, strings.Join(args, " ")))

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

func VpnUp() {
	vpnLog("vpn up")

	domains := []string{}
	dns := []string{}

	// iterate over foreign_option_N env vars
	i := 1
	for ; os.Getenv(fmt.Sprintf("foreign_option_%d", i)) != ""; i++ {
		fopt := os.Getenv(fmt.Sprintf("foreign_option_%d", i))
		if fopt == "" {
			break
		}

		if strings.HasPrefix(fopt, "dhcp-option DOMAIN ") {
			domains = append(domains, fopt[len("dhcp-option DOMAIN "):])
			continue
		}
		if strings.HasPrefix(fopt, "dhcp-option DNS ") {
			dns = append(dns, fopt[len("dhcp-option DNS "):])
			continue
		}
	}

	var sb strings.Builder

	if len(domains) == 1 {
		sb.WriteString(fmt.Sprintf("domain %s\n", domains[0]))
	} else if len(domains) > 1 {
		sb.WriteString(fmt.Sprintf("search %s\n", strings.Join(domains, " ")))
	}
	for _, nameserver := range dns {
		sb.WriteString(fmt.Sprintf("nameserver %s\n", nameserver))
	}

	// copy exising resolv.conf to resolv.conf.ovpnsave
	if _, err := os.Stat("/etc/resolv.conf.ovpnsave"); os.IsNotExist(err) {
		run("/bin/cp", "/etc/resolv.conf", "/etc/resolv.conf.ovpnsave")
	}

	// write resolv.conf
	if err := os.WriteFile("/etc/resolv.conf", []byte(sb.String()), 0644); err != nil {
		vpnLogError("Error updating /etc/resolv.conf", err)
	}

	dev := os.Getenv("dev")

	// Set routes
	// Remove all existing default routes
	run("/sbin/ip", "route", "del", "default")

	// Default route for all traffic through the VPN tunnel
	vpnGateway := os.Getenv("route_vpn_gateway")
	run("/sbin/ip", "route", "add", "default", "via", vpnGateway, "dev", dev)

	// Set firewall rules
	// Flush existing rules to start fresh
	run("/sbin/iptables", "-F")

	// Allow incoming ESTABLISHED and RELATED connections on the VPN interface
	run("/sbin/iptables", "-A", "INPUT", "-i", dev, "-m", "conntrack", "--ctstate", "ESTABLISHED,RELATED", "-j", "ACCEPT")

	// Drop all other incoming connections on the VPN interface
	run("/sbin/iptables", "-A", "INPUT", "-i", dev, "-j", "DROP")

	// Run apps
	run("/usr/local/bin/vpn-up")
}

func VpnDown() {
	vpnLog("vpn down")

	// copy exising resolv.conf.ovpnsave to resolv.conf. don't use cp, read content from resolv.conf.ovpnsave
	host_gateway := ""
	if _, err := os.Stat("/etc/resolv.conf.ovpnsave"); !os.IsNotExist(err) {
		fileContent, err := os.ReadFile("/etc/resolv.conf.ovpnsave")
		if err != nil {
			vpnLogError("Error reading /etc/resolv.conf.ovpnsave", err)
		} else {
			if err := os.WriteFile("/etc/resolv.conf", fileContent, 0644); err != nil {
				vpnLogError("Error updating /etc/resolv.conf", err)
			}
			// extract first nameserver as host gateway
			lines := strings.Split(string(fileContent), "\n")
			for _, line := range lines {
				if strings.HasPrefix(line, "nameserver") {
					host_gateway = strings.Split(line, " ")[1]
					break
				}
			}
		}
	}

	vpnLog("host gateway: " + host_gateway)

	// remove resolv.conf.ovpnsave
	if _, err := os.Stat("/etc/resolv.conf.ovpnsave"); !os.IsNotExist(err) {
		if err := os.Remove("/etc/resolv.conf.ovpnsave"); err != nil {
			vpnLogError("Error removing /etc/resolv.conf.ovpnsave", err)
		}
	}

	// Set routes
	// Remove all existing default routes
	run("/sbin/ip", "route", "del", "default")
	vpnLog("1")

	// Add default gateway
	run("/sbin/ip", "route", "add", "default", "via", host_gateway, "dev", "eth0")
	vpnLog("2")

	// Set firewall rules
	// Flush existing rules to start fresh
	run("/sbin/iptables", "-F")

	// Allow related and established connections (for existing sessions to work)
	run("/sbin/iptables", "-A", "INPUT", "-m", "conntrack", "--ctstate", "RELATED,ESTABLISHED", "-j", "ACCEPT")

	// Allow incoming connections only on port 80
	run("/sbin/iptables", "-A", "INPUT", "-p", "tcp", "--dport", "80", "-j", "ACCEPT")

	// Drop all other incoming connections
	run("/sbin/iptables", "-A", "INPUT", "-j", "DROP")

	// Stop apps
	run("/usr/local/bin/vpn-down")
}

func VpnUpDown() {
	// read script_type env
	scriptType := os.Getenv("script_type")

	if scriptType == "up" {
		logFile, _ = os.OpenFile("/data/var/vpn-up.log", os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666)
		VpnUp()
	} else if scriptType == "down" {
		logFile, _ = os.OpenFile("/data/var/vpn-down.log", os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666)
		VpnDown()
	}
}
