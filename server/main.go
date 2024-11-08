package main

import (
	"log"
	"openvpn-proxy/core"
	"openvpn-proxy/execmodes/vpn_action"
	"openvpn-proxy/execmodes/webserver"
	"openvpn-proxy/modules/http_proxy"
	"openvpn-proxy/modules/openvpn"
	"openvpn-proxy/modules/socks_proxy"
	"openvpn-proxy/utils"
	"os"
	"os/exec"
	"path/filepath"
)

func oneTimeSetup(dataDir string) {
	markerFile := "/.initialized"
	appScript := filepath.Join(dataDir, "apps.sh")

	if _, err := os.Stat(markerFile); os.IsNotExist(err) {
		if _, err := os.Stat(appScript); err == nil {
			log.Printf("Running one-time setup for apps script %s", appScript)
			cmd := exec.Command(appScript, "setup")
			cmd.Stdout = os.Stdout
			cmd.Stderr = os.Stderr
			err := cmd.Run()
			if err != nil {
				log.Println(err)
			}
		} else {
			log.Println("No apps script found")
		}
		os.Create(markerFile)
	}
}

func main() {
	ex, err := os.Executable()
	if err != nil {
		log.Fatal(err)
	}
	err = os.Chdir(filepath.Dir(ex))
	if err != nil {
		log.Fatal(err)
	}

	params, args := utils.SmartArgs("--data|-d=/data:,--port|-p=80:", os.Args[1:])
	dataDir := params["--data"].GetValue()

	// detect if this is an openvpn action
	scriptType := os.Getenv("script_type")
	appMode := core.WebServer
	if scriptType != "" && len(args) > 0 && args[0][:3] == "tun" {
		appMode = core.OpenVPNAction
	}

	err = core.Init(dataDir, appMode)
	if err != nil {
		log.Fatal(err)
	}

	if appMode == core.OpenVPNAction {
		switch scriptType {
		case "up":
			utils.SignalRunning(core.PidFile, core.VPN_UP)
		case "down":
			utils.SignalRunning(core.PidFile, core.VPN_DOWN)
		}
		os.Exit(0)
	}

	utils.AddSignalHandler([]os.Signal{core.VPN_UP, core.VPN_DOWN}, func(sig os.Signal) {
		switch sig {
		case core.VPN_UP:
			vpn_action.VpnUp(nil)
		case core.VPN_DOWN:
			vpn_action.VpnDown()
		}
	})

	oneTimeSetup(dataDir)

	// Disable all connectivity
	vpn_action.VpnDown()

	// Register modules
	openvpn.InitModule()
	http_proxy.InitModule()
	socks_proxy.InitModule()

	// Launch webserver
	webserver.WebServer(params["--port"].GetValue())
}
