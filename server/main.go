package main

import (
	"log"
	"openvpn-proxy/actions"
	"openvpn-proxy/core"
	"openvpn-proxy/modules/http_proxy"
	"openvpn-proxy/modules/openvpn"
	"openvpn-proxy/modules/socks_proxy"
	"openvpn-proxy/utils"
	"openvpn-proxy/webserver"
	"os"
	"os/exec"
	"path/filepath"
)

func oneTimeSetup(dataDir string) {
	markerFile := "/.initialized"

	if _, err := os.Stat(markerFile); os.IsNotExist(err) {
		if _, err := os.Stat(core.AppScript); err == nil {
			log.Printf("Running one-time setup for apps script %s", core.AppScript)
			cmd := exec.Command(core.AppScript, "setup")
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
			actions.SaveOpenVPNSpec()
			utils.SignalRunning(core.PidFile, core.VPN_UP)
		case "down":
			utils.SignalRunning(core.PidFile, core.VPN_DOWN)
		}
		os.Exit(0)
	}

	utils.AddSignalHandler([]os.Signal{core.VPN_UP, core.VPN_DOWN}, func(sig os.Signal) {
		switch sig {
		case core.VPN_UP:
			actions.VpnUp(nil)
		case core.VPN_DOWN:
			actions.VpnDown()
		}
	})

	oneTimeSetup(dataDir)

	// Disable all connectivity
	actions.VpnDown()

	// Register modules
	openvpn.InitModule()
	http_proxy.InitModule()
	socks_proxy.InitModule()

	// Launch webserver
	webserver.WebServer(params["--port"].GetValue())
}
