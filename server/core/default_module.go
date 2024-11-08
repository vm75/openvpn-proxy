package core

import (
	"github.com/gorilla/mux"
)

type DefaultModule struct {
	Name string

	Settings map[string]interface{}
}

func (d *DefaultModule) RegisterRoutes(r *mux.Router) {}

func (d *DefaultModule) GetStatus() (ModuleStatus, error) {
	return ModuleStatus{}, nil
}

func (d *DefaultModule) Enable(startNow bool) error {
	d.Settings["enabled"] = true
	return SaveSettings(d.Name, d.Settings)
}

func (d *DefaultModule) Disable(stopNow bool) error {
	d.Settings["enabled"] = false
	return SaveSettings(d.Name, d.Settings)
}

func (d *DefaultModule) Start() error {
	return nil
}

func (d *DefaultModule) Stop() error {
	return nil
}

func (d *DefaultModule) Restart() error {
	return nil
}

func (d *DefaultModule) GetSettings(params map[string]string) (map[string]interface{}, error) {
	return d.Settings, nil
}

func (d *DefaultModule) SaveSettings(params map[string]string, settings map[string]interface{}) error {
	d.Settings["enabled"] = settings["enabled"] == true
	return SaveSettings(d.Name, d.Settings)
}
