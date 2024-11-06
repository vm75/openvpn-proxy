package http_proxy

import (
	"log"
	"openvpn-proxy/core"
	"openvpn-proxy/utils"

	"github.com/gorilla/mux"
)

const moduleName = "http_proxy"

type HttpProxyModule struct {
	Enabled bool `json:"enabled"`
}

var proxySettings = HttpProxyModule{
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
func (h *HttpProxyModule) RegisterRoutes(r *mux.Router) {
}

// GetStatus implements core.Module.
func (h *HttpProxyModule) GetStatus() (core.ModuleStatus, error) {
	return core.ModuleStatus{Running: proxySettings.Enabled}, nil
}

// Enable implements core.Module.
func (h *HttpProxyModule) Enable(startNow bool) error {
	h.Enabled = true
	settings := map[string]interface{}{}
	utils.ObjectToMap(h, &settings)
	core.SaveSettings(moduleName, settings)
	if startNow {
		// TODO
	}
	return nil
}

// Disable implements core.Module.
func (h *HttpProxyModule) Disable(stopNow bool) error {
	h.Enabled = false
	settings := map[string]interface{}{}
	utils.ObjectToMap(h, &settings)
	core.SaveSettings(moduleName, settings)
	if stopNow {
		// TODO
	}
	return nil
}

// Start implements core.Module.
func (h *HttpProxyModule) Start() error {
	log.Println("unimplemented")

	return nil
}

// Stop implements core.Module.
func (h *HttpProxyModule) Stop() error {
	log.Println("unimplemented")

	return nil
}

// Restart implements core.Module.
func (h *HttpProxyModule) Restart() error {
	log.Println("unimplemented")

	return nil
}

// GetSettings implements core.Module.
func (h *HttpProxyModule) GetSettings(params map[string]string) (map[string]interface{}, error) {
	var settings map[string]interface{}
	utils.ObjectToMap(proxySettings, &settings)
	return settings, nil
}

// SaveSettings implements core.Module.
func (h *HttpProxyModule) SaveSettings(params map[string]string, settings map[string]interface{}) error {
	if !settingsChanged(h, settings) {
		return nil
	}
	utils.MapToObject(settings, h)
	err := core.SaveSettings(moduleName, settings)
	if err != nil {
		return err
	}

	return nil
}

func settingsChanged(h *HttpProxyModule, settings map[string]interface{}) bool {
	var currentSettings map[string]interface{}
	utils.ObjectToMap(h, &currentSettings)
	return !utils.AreEqual(currentSettings, settings)
}
