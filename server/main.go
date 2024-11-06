package main

import (
	"log"
	"openvpn-proxy/core"
	"openvpn-proxy/execmodes/vpn_action"
	"openvpn-proxy/execmodes/webserver"
	"openvpn-proxy/modules/openvpn"
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
	openvpnAction := false
	if scriptType != "" && len(args) > 0 && args[0][:3] == "tun" {
		openvpnAction = true
	}

	err = core.Init(dataDir, !openvpnAction)
	if err != nil {
		log.Fatal(err)
	}

	if openvpnAction {
		switch scriptType {
		case "up":
			vpn_action.VpnUp(nil)
		case "down":
			vpn_action.VpnDown()
		}
		os.Exit(0)
	}

	oneTimeSetup(dataDir)

	// Disable all connectivity
	vpn_action.VpnDown()

	// Register modules
	openvpn.InitOpenVPNModule()

	// Launch webserver
	webserver.WebServer(params["--port"].GetValue())
}
