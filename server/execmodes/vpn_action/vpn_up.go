package vpn_action

import (
	"fmt"
	"openvpn-proxy/core"
	"openvpn-proxy/utils"
	"os"
	"path/filepath"
	"strings"
)

type NetSpec struct {
	Dev        string
	Domains    []string
	DNS        []string
	VPNGateway string
}

func VpnUp(netSpec *NetSpec) {
	utils.InitLog(filepath.Join(core.VarDir, "vpn-up.log"))

	utils.Log("vpn up")

	dev := os.Getenv("dev")
	domains := []string{}
	dns := []string{}
	vpnGateway := os.Getenv("route_vpn_gateway")

	if netSpec != nil {
		dev = netSpec.Dev
		domains = netSpec.Domains
		dns = netSpec.DNS
		vpnGateway = netSpec.VPNGateway
	} else {
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
	}

	utils.BackupResolvConf()

	var sb strings.Builder
	if len(domains) == 1 {
		sb.WriteString(fmt.Sprintf("domain %s\n", domains[0]))
	} else if len(domains) > 1 {
		sb.WriteString(fmt.Sprintf("search %s\n", strings.Join(domains, " ")))
	}
	for _, nameserver := range dns {
		sb.WriteString(fmt.Sprintf("nameserver %s\n", nameserver))
	}
	// write resolv.conf
	if err := os.WriteFile("/etc/resolv.conf", []byte(sb.String()), 0644); err != nil {
		utils.LogError("Error updating /etc/resolv.conf", err)
	}

	// Set routes
	// Remove all existing default routes
	utils.RunCommand("/sbin/ip", "route", "del", "default")

	// Default route for all traffic through the VPN tunnel
	utils.RunCommand("/sbin/ip", "route", "add", "default", "via", vpnGateway, "dev", dev)

	// Set firewall rules
	// Flush existing rules to start fresh
	utils.RunCommand("/sbin/iptables", "-F")

	// Allow incoming ESTABLISHED and RELATED connections on the VPN interface
	utils.RunCommand("/sbin/iptables", "-A", "INPUT", "-i", dev, "-m", "conntrack", "--ctstate", "ESTABLISHED,RELATED", "-j", "ACCEPT")

	// Drop all other incoming connections on the VPN interface
	utils.RunCommand("/sbin/iptables", "-A", "INPUT", "-i", dev, "-j", "DROP")

	// Run apps
	utils.RunCommand("/usr/local/bin/vpn-up")
}
