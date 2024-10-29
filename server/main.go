package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"

	"config-server/ovpn"

	"github.com/gorilla/mux"
)

var staticDir *string

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
func getSettings(w http.ResponseWriter, r *http.Request) {
	config := ovpn.GetProxySettings()
	if config == nil {
		config = &ovpn.ProxySettings{}
	}

	json.NewEncoder(w).Encode(config)
}

// Save a config (new or existing)
func saveSettings(w http.ResponseWriter, r *http.Request) {
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
	fs := http.FileServer(http.Dir(*staticDir))
	r.PathPrefix("/static/").Handler(http.StripPrefix("/static/", fs))
	r.PathPrefix("/").Handler(http.StripPrefix("/", fs)) // Serve "/" from staticDir
}

func server(port string) {
	// Create a new Gorilla Mux router
	r := mux.NewRouter()

	// Template-related routes
	r.HandleFunc("/api/servers", listServers).Methods("GET")
	r.HandleFunc("/api/servers/{name}", getServer).Methods("GET")
	r.HandleFunc("/api/servers/save", saveServer).Methods("POST")
	r.HandleFunc("/api/servers/delete/{name}", deleteServer).Methods("DELETE")

	// Config-related routes
	r.HandleFunc("/api/settings", getSettings).Methods("GET")
	r.HandleFunc("/api/settings/save", saveSettings).Methods("POST")

	// Serve static files
	handleStaticFiles(r)

	// Start the server
	fmt.Printf("Server starting on port %s\n", port)
	err := http.ListenAndServe(":"+port, r)
	if err != nil {
		log.Fatal(err)
	}
}

func main() {
	ex, err := os.Executable()
	if err != nil {
		log.Fatal(err)
	}
	err = os.Chdir(filepath.Dir(ex))
	if err != nil {
		log.Fatal(err)
	}

	// Command-line flag for port
	port := flag.String("port", "8080", "Port to run the server on")
	dataDir := flag.String("data", "", "Directory to store data")
	staticDir = flag.String("static", "./static", "Directory of static files")
	isDaemon := flag.Bool("daemon", false, "Run in daemon mode")
	flag.Parse()

	if !*isDaemon {
		err = ovpn.Init(*dataDir, *isDaemon)
		if err != nil {
			log.Fatal(err)
		}

	}

	ovpn.Init(*dataDir, *isDaemon)
	fmt.Println("Running in daemon mode")
	server(*port)
	os.Exit(0)
}
