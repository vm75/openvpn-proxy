package common

type Template struct {
	Name      string            `json:"name"`
	Content   string            `json:"content"`
	Variables map[string]string `json:"variables"`
}

type Config struct {
	Name         string                 `json:"name"`         // Name of the config
	TemplateName string                 `json:"templateName"` // Name of the template (if any)
	Fields       map[string]interface{} `json:"fields"`       // Fields for template-based configs
	Data         string                 `json:"data"`         // Raw data for custom configs
}

type ConfigManager struct {
	Init func(path string)

	GetTemplates   func() []Template
	GetTemplate    func(name string) *Template
	SaveTemplate   func(template Template) error
	DeleteTemplate func(name string) error

	GetConfigs   func() []Config
	GetConfig    func(name string) *Config
	SaveConfig   func(config Config) error
	DeleteConfig func(name string) error

	ResolveConfig func(config Config) string
}
