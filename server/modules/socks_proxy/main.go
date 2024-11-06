package socks_proxy

import (
	"log"
	"openvpn-proxy/core"
	"openvpn-proxy/utils"

	"github.com/gorilla/mux"
)

const moduleName = "socks_proxy"

type SocksProxyModule struct {
	Enabled bool `json:"enabled"`
}

var proxySettings = SocksProxyModule{
	Enabled: false,
}

func InitModule() {
	core.RegisterModule(moduleName, &proxySettings)

	savedSettings, err := core.GetSettings(moduleName)
	if err == nil {
		utils.MapToObject(savedSettings, &proxySettings)
	} else {
		utils.ObjectToMap(proxySettings, &savedSettings)
		core.SaveSettings(moduleName, savedSettings)
	}
}

// RegisterRoutes implements core.Module.
func (s *SocksProxyModule) RegisterRoutes(r *mux.Router) {
}

// GetStatus implements core.Module.
func (s *SocksProxyModule) GetStatus() (core.ModuleStatus, error) {
	return core.ModuleStatus{Running: proxySettings.Enabled}, nil
}

// Enable implements core.Module.
func (s *SocksProxyModule) Enable(startNow bool) error {
	s.Enabled = true
	settings := map[string]interface{}{}
	utils.ObjectToMap(s, &settings)
	core.SaveSettings(moduleName, settings)
	if startNow {
		// TODO
	}
	return nil
}

// Disable implements core.Module.
func (s *SocksProxyModule) Disable(stopNow bool) error {
	s.Enabled = false
	settings := map[string]interface{}{}
	utils.ObjectToMap(s, &settings)
	core.SaveSettings(moduleName, settings)
	if stopNow {
		// TODO
	}
	return nil
}

// Start implements core.Module.
func (s *SocksProxyModule) Start() error {
	log.Println("unimplemented")

	return nil
}

// Stop implements core.Module.
func (s *SocksProxyModule) Stop() error {
	log.Println("unimplemented")

	return nil
}

// Restart implements core.Module.
func (s *SocksProxyModule) Restart() error {
	log.Println("unimplemented")

	return nil
}

// GetSettings implements core.Module.
func (s *SocksProxyModule) GetSettings(params map[string]string) (map[string]interface{}, error) {
	var settings map[string]interface{}
	utils.ObjectToMap(proxySettings, &settings)
	return settings, nil
}

// SaveSettings implements core.Module.
func (s *SocksProxyModule) SaveSettings(params map[string]string, settings map[string]interface{}) error {
	if !settingsChanged(s, settings) {
		return nil
	}
	utils.MapToObject(settings, s)
	err := core.SaveSettings(moduleName, settings)
	if err != nil {
		return err
	}

	return nil
}

func settingsChanged(s *SocksProxyModule, settings map[string]interface{}) bool {
	var currentSettings map[string]interface{}
	utils.ObjectToMap(s, &currentSettings)
	return !utils.AreEqual(currentSettings, settings)
}
