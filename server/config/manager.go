package config

import (
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"nginx-ui/common"
)

const (
	templateExt = ".tpl"
	configExt   = ".cfg"
)

var dataDir = "./data"
var templatesDir = dataDir
var configsDir = dataDir

var Manager = common.ConfigManager{
	Init:           initAll,
	GetTemplates:   getTemplates,
	GetTemplate:    getTemplate,
	SaveTemplate:   saveTemplate,
	DeleteTemplate: deleteTemplate,
	GetConfigs:     getConfigs,
	GetConfig:      getConfig,
	SaveConfig:     saveConfig,
	DeleteConfig:   deleteConfig,
	ResolveConfig:  resolveConfig,
}

func initAll(path string) {
	// ensure folders
	os.MkdirAll(templatesDir, 0755)
	os.MkdirAll(configsDir, 0755)
}

// Get all templates
func getTemplates() []common.Template {
	var templates []common.Template
	files, _ := os.ReadDir(templatesDir)
	for _, f := range files {
		if !f.IsDir() && filepath.Ext(f.Name()) == templateExt {
			content, _ := os.ReadFile(filepath.Join(templatesDir, f.Name()))
			name := f.Name()[:len(f.Name())-4]
			templates = append(templates, common.Template{Name: name, Content: string(content)})
		}
	}
	return templates
}

// Extract variables {{name:type}} from the template content
func extractVariables(content string) map[string]string {
	var variables map[string]string = make(map[string]string)
	lines := strings.Split(content, "\n")

	for _, line := range lines {
		// Continue looking for {{...}} as long as there are more variables in the line
		for {
			start := strings.Index(line, "{{")
			end := strings.Index(line, "}}")

			// Break the loop if no more variables are found
			if start == -1 || end == -1 {
				break
			}

			// Extract the variable between {{ and }}
			variable := line[start+2 : end]
			parts := strings.Split(variable, ":")

			// Ensure the variable contains both a key and value
			if len(parts) == 2 {
				name := strings.TrimSpace(parts[0])
				valueType := strings.TrimSpace(parts[1])
				variables[name] = valueType
			} else {
				variables[variable] = "string"
			}

			// Remove the processed part from the line and continue checking
			line = line[end+2:]
		}
	}

	return variables
}

// Get a specific template
func getTemplate(name string) *common.Template {
	file, err := os.ReadFile(filepath.Join(templatesDir, name+templateExt))
	if err != nil {
		return nil
	}

	variables := extractVariables(string(file))
	return &common.Template{Name: name, Content: string(file), Variables: variables}
}

// Create or update a template
func saveTemplate(template common.Template) error {
	templatePath := filepath.Join(templatesDir, template.Name+templateExt)
	return os.WriteFile(templatePath, []byte(template.Content), fs.ModePerm)
}

// Delete a template
func deleteTemplate(name string) error {
	templatePath := filepath.Join(templatesDir, name+templateExt)
	return os.Remove(templatePath)
}

// Get all configs
func getConfigs() []common.Config {
	var configs []common.Config
	files, _ := os.ReadDir(configsDir)
	for _, f := range files {
		if !f.IsDir() && filepath.Ext(f.Name()) == configExt {
			configPath := filepath.Join(configsDir, f.Name())
			configFile, _ := os.ReadFile(configPath)
			var config common.Config
			json.Unmarshal(configFile, &config)
			configs = append(configs, config)
		}
	}
	return configs
}

// Get a specific config
func getConfig(name string) *common.Config {
	file, err := os.ReadFile(filepath.Join(configsDir, name+configExt))
	if err != nil {
		return nil
	}
	var config common.Config
	json.Unmarshal(file, &config)
	return &config
}

// Create or update a config
func saveConfig(config common.Config) error {
	configPath := filepath.Join(configsDir, config.Name+configExt)
	configFile, err := os.Create(configPath)
	if err != nil {
		return err
	}

	defer configFile.Close()
	json.NewEncoder(configFile).Encode(config)
	return nil
}

// Delete a config
func deleteConfig(name string) error {
	return os.Remove(filepath.Join(configsDir, name+configExt))
}

func resolveConfig(config common.Config) string {
	if config.TemplateName == "custom" {
		return config.Data
	}

	template := getTemplate(config.TemplateName)
	if template == nil {
		return ""
	}

	content := template.Content
	for key, value := range config.Fields {
		content = strings.ReplaceAll(content, "{{"+key+"}}", fmt.Sprintf("%v", value))
	}
	return content
}
