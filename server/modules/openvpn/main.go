package openvpn

import (
	"openvpn-proxy/core"
	"openvpn-proxy/utils"
	"os"
	"path/filepath"

	"github.com/gorilla/mux"
)

type OpenVPNModule struct {
	Enabled        bool   `json:"enabled"`
	ServerName     string `json:"serverName"`
	ServerEndpoint string `json:"serverEndpoint"`
	LogLevel       int    `json:"logLevel"`
	RetryInterval  int    `json:"retryInterval"`
}

var openvpnSettings = OpenVPNModule{
	Enabled:        false,
	ServerName:     "",
	ServerEndpoint: "",
	LogLevel:       0,
	RetryInterval:  3600,
}
var configFile = ""
var authFile = ""
var pidFile = ""
var logFile = ""
var statusFile = ""

func InitOpenVPNModule() {
	initDb()

	configFile = filepath.Join(core.VarDir, "vpn.ovpn")
	authFile = filepath.Join(core.VarDir, "vpn.auth")
	pidFile = filepath.Join(core.VarDir, "openvpn.pid")
	logFile = filepath.Join(core.VarDir, "openvpn.log")
	statusFile = filepath.Join(core.VarDir, "openvpn.status")

	savedSettings, err := core.GetSettings("openvpn")
	if err == nil {
		utils.MapToObject(savedSettings, &openvpnSettings)
	} else {
		utils.ObjectToMap(openvpnSettings, &savedSettings)
		core.SaveSettings("openvpn", savedSettings)
	}

	core.RegisterModule("openvpn", &openvpnSettings)

	if openvpnSettings.Enabled {
		go runOpenVPN()
	}
}

// RegisterRoutes implements core.Module.
func (o *OpenVPNModule) RegisterRoutes(r *mux.Router) {
	// Template-related routes
	r.HandleFunc("/api/openvpn/servers", listServers).Methods("GET")
	r.HandleFunc("/api/openvpn/servers/{name}", getServer).Methods("GET")
	r.HandleFunc("/api/openvpn/servers/save", saveServer).Methods("POST")
	r.HandleFunc("/api/openvpn/servers/delete/{name}", deleteServer).Methods("DELETE")
}

// GetStatus implements core.Module.
func (o *OpenVPNModule) GetStatus() (core.ModuleStatus, error) {
	if openvpnCmd == nil || openvpnCmd.Process == nil || (openvpnCmd.ProcessState != nil && openvpnCmd.ProcessState.Exited()) {
		return core.ModuleStatus{Running: false}, nil
	}
	return core.ModuleStatus{Running: true}, nil
}

// Enable implements core.Module.
func (o *OpenVPNModule) Enable(startNow bool) error {
	o.Enabled = true
	settings := map[string]interface{}{}
	utils.ObjectToMap(o, &settings)
	core.SaveSettings("openvpn", settings)
	if startNow {
		go runOpenVPN()
	}
	return nil
}

// Disable implements core.Module.
func (o *OpenVPNModule) Disable(stopNow bool) error {
	o.Enabled = false
	settings := map[string]interface{}{}
	utils.ObjectToMap(o, &settings)
	core.SaveSettings("openvpn", settings)
	if stopNow {
		killOpenVPN()
	}
	return nil
}

// Start implements core.Module.
func (o *OpenVPNModule) Start() error {
	if o.Enabled {
		go runOpenVPN()
	} else {
		o.Enable(true)
	}
	return nil
}

// Stop implements core.Module.
func (o *OpenVPNModule) Stop() error {
	if o.Enabled {
		o.Disable(true)
	} else {
		killOpenVPN()
	}
	return nil
}

// Restart implements core.Module.
func (o *OpenVPNModule) Restart() error {
	killOpenVPN()
	return nil
}

// GetSettings implements core.Module.
func (o *OpenVPNModule) GetSettings(params map[string]string) (map[string]interface{}, error) {
	var settings map[string]interface{}
	utils.ObjectToMap(openvpnSettings, &settings)
	return settings, nil
}

// SaveSettings implements core.Module.
func (o *OpenVPNModule) SaveSettings(params map[string]string, settings map[string]interface{}) error {
	if !settingsChanged(o, settings) {
		return nil
	}
	utils.MapToObject(settings, o)
	err := core.SaveSettings("openvpn", settings)
	if err != nil {
		return err
	}
	saveOvpnConfig()

	killOpenVPN()
	go runOpenVPN()

	return nil
}

func settingsChanged(o *OpenVPNModule, settings map[string]interface{}) bool {
	var currentSettings map[string]interface{}
	utils.ObjectToMap(o, &currentSettings)
	return !utils.AreEqual(currentSettings, settings)
}

// SignalReceived implements core.Module.
func (o *OpenVPNModule) SignalReceived() {
	o.Enabled = false
	killOpenVPN()
	os.Exit(0)
}
