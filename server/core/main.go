package core

import (
	"fmt"
	"openvpn-proxy/utils"
	"os"
	"path/filepath"
	"syscall"
)

var DataDir string
var ConfigDir string
var VarDir string
var PidFile string

type GlobalSettings struct {
	VPNTypes      []string `json:"vpnTypes"`
	VPNType       string   `json:"vpnType"`
	Subnets       []string `json:"subnets"`
	ProxyUsername string   `json:"proxyUsername"`
	ProxyPassword string   `json:"proxyPassword"`
}

var GlobalConfig = GlobalSettings{
	VPNTypes:      []string{"openvpn", "wireguard"},
	VPNType:       "openvpn",
	Subnets:       []string{},
	ProxyUsername: "",
	ProxyPassword: "",
}

// enum for app mode (1 = webserver, 2 = vpn-action)
type AppMode int

const (
	WebServer AppMode = iota + 1
	OpenVPNAction
)

var (
	SHUTDOWN = syscall.SIGTERM
	VPN_UP   = utils.RealTimeSignal(1)
	VPN_DOWN = utils.RealTimeSignal(2)
)

func Init(dataDir string, appMode AppMode) error {
	utils.InitSignals([]os.Signal{SHUTDOWN, VPN_UP, VPN_DOWN})

	DataDir = dataDir
	ConfigDir = filepath.Join(dataDir, "config")
	VarDir = filepath.Join(dataDir, "var")
	PidFile = filepath.Join(VarDir, "openvpn-proxy.pid")

	err := os.MkdirAll(ConfigDir, 0755)
	if err != nil {
		return err
	}
	err = os.MkdirAll(VarDir, 0755)
	if err != nil {
		return err
	}

	if appMode == OpenVPNAction {
		return nil
	}

	// if pid file exists, and process is still running, return
	if utils.SignalRunning(PidFile, syscall.SIGCONT) {
		os.Exit(0)
	}
	err = os.WriteFile(PidFile, []byte(fmt.Sprintf("%d", os.Getpid())), 0644)
	if err != nil {
		return err
	}

	// appMode == WebServer
	err = initDb()
	if err != nil {
		return err
	}

	var savedSettings map[string]interface{}
	savedSettings, err = GetSettings("global")
	if err == nil {
		utils.MapToObject(savedSettings, &GlobalConfig)
	} else {
		utils.ObjectToMap(GlobalConfig, &savedSettings)
		SaveSettings("global", savedSettings)
	}

	return nil
}

func GetGlobalSettings() (map[string]interface{}, error) {
	var settings map[string]interface{}
	utils.ObjectToMap(GlobalConfig, &settings)
	return settings, nil
}

func SaveGlobalSettings(settings map[string]interface{}) error {
	if !utils.HasChanged(&GlobalConfig, settings) {
		return nil
	}
	utils.MapToObject(settings, &GlobalConfig)
	err := SaveSettings("global", settings)
	if err != nil {
		return err
	}

	utils.PublishEvent(utils.Event{Name: "global-settings-changed", Context: settings})

	return nil
}
