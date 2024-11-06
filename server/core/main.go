package core

import (
	"fmt"
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
var PidFile string

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

// enum for app mode (1 = webserver, 2 = vpn-action)
type AppMode int

const (
	WebServer AppMode = iota + 1
	VPNAction
)

func SignalRunning(signal syscall.Signal) bool {
	isRunning := false
	if _, err := os.Stat(PidFile); err == nil {
		file, err := os.Open(PidFile)
		if err != nil {
			return isRunning
		}
		defer file.Close()
		var pid int
		_, err = fmt.Fscanf(file, "%d", &pid)
		if err != nil {
			return isRunning
		}
		proc, err := os.FindProcess(pid)
		if err == nil {
			err = proc.Signal(signal)
			if err != nil {
				return isRunning
			}
			isRunning = true
		}
	}

	return isRunning
}

func Init(dataDir string, appMode AppMode) error {
	// Set up termination signal handler
	sigChannel := make(chan os.Signal, 1)
	signal.Notify(sigChannel, syscall.SIGTERM, syscall.SIGUSR1, syscall.SIGUSR2)
	go func() {
		for {
			sig := <-sigChannel
			switch sig {
			case syscall.SIGTERM:
				log.Println("Received SIGTERM")
				utils.PublishEvent(utils.Event{Name: "shutdown"})
			case syscall.SIGUSR1:
				log.Println("Received SIGUSR1")
				utils.PublishEvent(utils.Event{Name: "vpn-up"})
			case syscall.SIGUSR2:
				log.Println("Received SIGUSR2")
				utils.PublishEvent(utils.Event{Name: "vpn-down"})
			}
		}
	}()

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

	if appMode == VPNAction {
		return nil
	}

	// if pid file exists, and process is still running, return
	if SignalRunning(syscall.SIGCONT) {
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
		utils.MapToObject(savedSettings, &globalSettings)
	} else {
		utils.ObjectToMap(globalSettings, &savedSettings)
		SaveSettings("global", savedSettings)
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
