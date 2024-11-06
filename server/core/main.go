package core

import (
	"log"
	"openvpn-proxy/utils"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
)

var DataDir string
var ConfigDir string
var VarDir string

type GlobalSettings struct {
	VPNTypes      []string `json:"vpnTypes"`
	VPNType       string   `json:"vpnType"`
	Subnets       []string `json:"subnets"`
	ProxyUsername string   `json:"proxyUsername"`
	ProxyPassword string   `json:"proxyPassword"`
}

var globalSettings = GlobalSettings{
	VPNTypes:      []string{"openvpn", "wireguard"},
	VPNType:       "openvpn",
	Subnets:       []string{},
	ProxyUsername: "",
	ProxyPassword: "",
}

func Init(dataDir string, dbNeeed bool) error {
	// Set up termination signal handler
	sigChannel := make(chan os.Signal, 1)
	signal.Notify(sigChannel, syscall.SIGTERM)
	go func() {
		<-sigChannel
		log.Println("SIGTERM received, shutting down...")

		for _, module := range GetModules() {
			module.SignalReceived()
		}

		os.Exit(0)
	}()

	DataDir = dataDir
	ConfigDir = filepath.Join(dataDir, "config")
	VarDir = filepath.Join(dataDir, "var")

	err := os.MkdirAll(ConfigDir, 0755)
	if err != nil {
		return err
	}
	err = os.MkdirAll(VarDir, 0755)
	if err != nil {
		return err
	}

	if dbNeeed {
		err = initDb()
		if err != nil {
			return err
		}

		var savedSettings map[string]interface{}
		savedSettings, err = GetSettings("global")
		if err == nil {
			utils.MapToObject(savedSettings, &globalSettings)
		} else {
			utils.ObjectToMap(globalSettings, &savedSettings)
			SaveSettings("global", savedSettings)
		}
	}

	return nil
}

func GetGlobalSettings() (map[string]interface{}, error) {
	var settings map[string]interface{}
	utils.ObjectToMap(globalSettings, &settings)
	return settings, nil
}

func SaveGlobalSettings(settings map[string]interface{}) error {
	utils.MapToObject(settings, &globalSettings)
	return SaveSettings("global", settings)
}
