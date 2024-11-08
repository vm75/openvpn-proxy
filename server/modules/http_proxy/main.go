package http_proxy

import (
	"openvpn-proxy/core"
	"openvpn-proxy/utils"
	"path/filepath"
)

type HttpProxyModule struct {
	core.DefaultModule
}

var pidFile = ""
var configFile = ""

func InitModule() {
	var module = HttpProxyModule{
		DefaultModule: core.DefaultModule{
			Name: "http_proxy",
		},
	}

	core.RegisterModule(module.Name, &module)
	utils.RegisterListener("global-settings-changed", &module)
	utils.RegisterListener("vpn-up", &module)
	utils.RegisterListener("vpn-down", &module)

	configFile = filepath.Join(core.VarDir, "tinyproxy.conf")
	pidFile = filepath.Join(core.VarDir, "tinyproxy.pid")

	var err error
	module.Settings, err = core.GetSettings(module.Name)
	if err != nil {
		module.Settings["enabled"] = false
		core.SaveSettings(module.Name, module.Settings)
	}

	updateConfig()
}

// GetStatus implements core.Module.
func (h *HttpProxyModule) GetStatus() (core.ModuleStatus, error) {
	return core.ModuleStatus{Running: isRunning()}, nil
}

// HandleEvent implements utils.EventListener.
func (h *HttpProxyModule) HandleEvent(event utils.Event) {
	switch event.Name {
	case "global-settings-changed":
		updateConfig()
	case "vpn-up":
		if h.Settings["enabled"].(bool) {
			go startProxy()
		}
	case "vpn-down":
		stopProxy()
	}
}
