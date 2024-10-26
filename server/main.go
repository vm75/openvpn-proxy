package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"net/http"
	"os"
	"path/filepath"

	"nginx-ui/common"
	"nginx-ui/config"

	"github.com/gorilla/mux"
)

const (
	templatesDir = "./templates"
	configsDir   = "./configs"
	staticDir    = "./static"
)

var m = config.Manager

// List all templates
func listTemplates(w http.ResponseWriter, r *http.Request) {
	var templates = m.GetTemplates()
	json.NewEncoder(w).Encode(templates)
}

// Get a single template
func getTemplate(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	name := vars["name"]
	template := m.GetTemplate(name)
	if template == nil {
		http.Error(w, "Template not found", http.StatusNotFound)
		return
	}
	json.NewEncoder(w).Encode(template)
}

// Create or update a template
func saveTemplate(w http.ResponseWriter, r *http.Request) {
	var tpl common.Template
	_ = json.NewDecoder(r.Body).Decode(&tpl)

	err := m.SaveTemplate(tpl)
	if err != nil {
		http.Error(w, "Failed to save template", http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)
}

// Delete a template
func deleteTemplate(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	name := vars["name"]
	err := m.DeleteTemplate(name)
	if err != nil {
		http.Error(w, "Failed to delete template", http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)
}

// Get all configs
func getConfigs(w http.ResponseWriter, r *http.Request) {
	var configs = m.GetConfigs()
	json.NewEncoder(w).Encode(configs)
}

// Get a specific config
func getConfig(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	name := vars["name"]

	config := m.GetConfig(name)
	if config == nil {
		http.Error(w, "Config not found", http.StatusNotFound)
		return
	}

	json.NewEncoder(w).Encode(config)
}

// Save a config (new or existing)
func saveConfig(w http.ResponseWriter, r *http.Request) {
	var config common.Config
	err := json.NewDecoder(r.Body).Decode(&config)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	err = m.SaveConfig(config)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}

// Delete a config
func deleteConfig(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	name := vars["name"]

	err := m.DeleteConfig(name)
	if err != nil {
		http.Error(w, "Failed to delete config", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}

// Separate function to handle static files
func handleStaticFiles(r *mux.Router) {
	// Serve static files from /static and root (/)
	fs := http.FileServer(http.Dir(staticDir))
	r.PathPrefix("/static/").Handler(http.StripPrefix("/static/", fs))
	r.PathPrefix("/").Handler(http.StripPrefix("/", fs)) // Serve "/" from staticDir
}

func main() {
	ex, err := os.Executable()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	err = os.Chdir(filepath.Dir(ex))
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	// Command-line flag for port
	portPtr := flag.String("port", "", "Port to run the server on")
	flag.Parse()

	// Check command-line, then env, then default port
	port := *portPtr
	if port == "" {
		port = os.Getenv("PORT")
		if port == "" {
			port = "8080" // Default port if none is provided
		}
	}

	// Create the templates directory if it doesn't exist
	if _, err := os.Stat(templatesDir); os.IsNotExist(err) {
		os.Mkdir(templatesDir, 0755)
	}

	// Create the configs directory if it doesn't exist
	if _, err := os.Stat(configsDir); os.IsNotExist(err) {
		os.Mkdir(configsDir, 0755)
	}

	// Create a new Gorilla Mux router
	r := mux.NewRouter()

	// Template-related routes
	r.HandleFunc("/api/templates", listTemplates).Methods("GET")
	r.HandleFunc("/api/template/{name}", getTemplate).Methods("GET")
	r.HandleFunc("/api/template/save", saveTemplate).Methods("POST")
	r.HandleFunc("/api/template/delete/{name}", deleteTemplate).Methods("DELETE")

	// Config-related routes
	r.HandleFunc("/api/configs", getConfigs).Methods("GET")
	r.HandleFunc("/api/config/{name}", getConfig).Methods("GET")
	r.HandleFunc("/api/config/save", saveConfig).Methods("POST")
	r.HandleFunc("/api/config/delete/{name}", deleteConfig).Methods("DELETE")

	// Serve static files
	handleStaticFiles(r)

	// Start the server
	fmt.Printf("Server starting on port %s\n", port)
	err = http.ListenAndServe(":"+port, r)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
