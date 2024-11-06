package core

import (
	"fmt"
	"sync"

	"github.com/gorilla/mux"
)

type ModuleStatus struct {
	Running bool                   `json:"running"`
	Info    map[string]interface{} `json:"ipInfo"`
}

type Module interface {
	RegisterRoutes(r *mux.Router)
	GetStatus() (ModuleStatus, error)
	Enable(startNow bool) error
	Disable(stopNow bool) error
	Start() error
	Stop() error
	Restart() error
	GetSettings(params map[string]string) (map[string]interface{}, error)
	SaveSettings(params map[string]string, settings map[string]interface{}) error
}

var modules = make(map[string]Module)
var modulesMutex sync.RWMutex

func RegisterModule(name string, module Module) {
	modulesMutex.Lock()
	defer modulesMutex.Unlock()
	modules[name] = module
}

func GetModules() []Module {
	modulesMutex.RLock()
	defer modulesMutex.RUnlock()
	var moduleList []Module
	for _, module := range modules {
		moduleList = append(moduleList, module)
	}
	return moduleList
}

func GetModule(name string) Module {
	modulesMutex.RLock()
	defer modulesMutex.RUnlock()
	return modules[name]
}

func GetModuleStatus(name string) (ModuleStatus, error) {
	modulesMutex.RLock()
	defer modulesMutex.RUnlock()
	if module, exists := modules[name]; exists {
		return module.GetStatus()
	}
	return ModuleStatus{}, fmt.Errorf("module %s not found", name)
}

func EnableModule(name string, startNow bool) error {
	modulesMutex.RLock()
	defer modulesMutex.RUnlock()
	if module, exists := modules[name]; exists {
		return module.Enable(startNow)
	}
	return fmt.Errorf("module %s not found", name)
}

func DisableModule(name string, stopNow bool) error {
	modulesMutex.RLock()
	defer modulesMutex.RUnlock()
	if module, exists := modules[name]; exists {
		return module.Disable(stopNow)
	}
	return fmt.Errorf("module %s not found", name)
}

func StartModule(name string) error {
	modulesMutex.RLock()
	defer modulesMutex.RUnlock()
	if module, exists := modules[name]; exists {
		return module.Start()
	}
	return fmt.Errorf("module %s not found", name)
}

func StopModule(name string) error {
	modulesMutex.RLock()
	defer modulesMutex.RUnlock()
	if module, exists := modules[name]; exists {
		return module.Stop()
	}
	return fmt.Errorf("module %s not found", name)
}

func RestartModule(name string) error {
	modulesMutex.RLock()
	defer modulesMutex.RUnlock()
	if module, exists := modules[name]; exists {
		return module.Restart()
	}
	return fmt.Errorf("module %s not found", name)
}

func GetModuleSettings(name string, params map[string]string) (map[string]interface{}, error) {
	modulesMutex.RLock()
	defer modulesMutex.RUnlock()
	if module, exists := modules[name]; exists {
		return module.GetSettings(params)
	}
	return nil, fmt.Errorf("module %s not found", name)
}

func SaveModuleSettings(name string, params map[string]string, settings map[string]interface{}) error {
	modulesMutex.RLock()
	defer modulesMutex.RUnlock()
	if module, exists := modules[name]; exists {
		module.SaveSettings(params, settings)
		return nil
	}

	return fmt.Errorf("module %s not found", name)
}
