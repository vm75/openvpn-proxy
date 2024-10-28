package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"net/http"
	"os"
	"path/filepath"

	"config-server/ovpn"

	"github.com/gorilla/mux"
)

var staticDir = "./static"

func errorLog(err error, exitCode int) {
	fmt.Fprintln(os.Stderr, err)
	os.Exit(exitCode)
}

// List all servers
func listServers(w http.ResponseWriter, r *http.Request) {
	var servers = ovpn.GetServers()
	json.NewEncoder(w).Encode(servers)
}

// Get a single server
func getServer(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	name := vars["name"]
	server := ovpn.GetServer(name)
	if server == nil {
		http.Error(w, "Server not found", http.StatusNotFound)
		return
	}
	json.NewEncoder(w).Encode(server)
}

// Create or update a server
func saveServer(w http.ResponseWriter, r *http.Request) {
	var svr ovpn.Server
	_ = json.NewDecoder(r.Body).Decode(&svr)

	err := ovpn.SaveServer(svr)
	if err != nil {
		http.Error(w, "Failed to save server", http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)
}

// Delete a template
func deleteServer(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	name := vars["name"]
	err := ovpn.DeleteServer(name)
	if err != nil {
		http.Error(w, "Failed to delete template", http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)
}

// Get a specific config
func getConfig(w http.ResponseWriter, r *http.Request) {
	config := ovpn.GetProxySettings()
	if config == nil {
		http.Error(w, "Config not found", http.StatusNotFound)
		return
	}

	json.NewEncoder(w).Encode(config)
}

// Save a config (new or existing)
func saveConfig(w http.ResponseWriter, r *http.Request) {
	var settings ovpn.ProxySettings
	err := json.NewDecoder(r.Body).Decode(&settings)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	err = ovpn.SaveProxySettings(&settings)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
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
		errorLog(err, 1)
	}
	err = os.Chdir(filepath.Dir(ex))
	if err != nil {
		errorLog(err, 1)
	}

	// Command-line flag for port
	portPtr := flag.String("port", "8080", "Port to run the server on")
	dataPtr := flag.String("data", "", "Directory to store data")
	staticPtr := flag.String("static", "./static", "Directory of static files")
	flag.Parse()

	// Check command-line
	port := *portPtr
	dataDir := *dataPtr
	staticDir = *staticPtr

	ovpn.Init(dataDir)

	// Create a new Gorilla Mux router
	r := mux.NewRouter()

	// Template-related routes
	r.HandleFunc("/api/servers", listServers).Methods("GET")
	r.HandleFunc("/api/servers/{name}", getServer).Methods("GET")
	r.HandleFunc("/api/servers/save", saveServer).Methods("POST")
	r.HandleFunc("/api/servers/delete/{name}", deleteServer).Methods("DELETE")

	// Config-related routes
	r.HandleFunc("/api/settings", getConfig).Methods("GET")
	r.HandleFunc("/api/settings/save", saveConfig).Methods("POST")

	// Serve static files
	handleStaticFiles(r)

	// Start the server
	fmt.Printf("Server starting on port %s\n", port)
	err = http.ListenAndServe(":"+port, r)
	if err != nil {
		errorLog(err, 1)
	}
}
