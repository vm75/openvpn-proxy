package daemon

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"

	"github.com/gorilla/mux"
)

var StaticDir *string

// List all servers
func listServers(w http.ResponseWriter, r *http.Request) {
	var servers = GetServers()
	json.NewEncoder(w).Encode(servers)
}

// Get a single server
func getServer(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	name := vars["name"]
	server := GetServer(name)
	if server == nil {
		http.Error(w, "Server not found", http.StatusNotFound)
		return
	}
	json.NewEncoder(w).Encode(server)
}

// Create or update a server
func saveServer(w http.ResponseWriter, r *http.Request) {
	var svr Server
	_ = json.NewDecoder(r.Body).Decode(&svr)

	err := SaveServer(svr)
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
	err := DeleteServer(name)
	if err != nil {
		http.Error(w, "Failed to delete template", http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)
}

// Get a specific config
func getSettings(w http.ResponseWriter, r *http.Request) {
	config := GetProxySettings()
	if config == nil {
		config = &ProxySettings{}
	}

	json.NewEncoder(w).Encode(config)
}

// Save a config (new or existing)
func saveSettings(w http.ResponseWriter, r *http.Request) {
	var settings ProxySettings
	err := json.NewDecoder(r.Body).Decode(&settings)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	err = SaveProxySettings(&settings)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}

// Separate function to handle static files
func handleStaticFiles(r *mux.Router) {
	// Serve static files from /static and root (/)
	fs := http.FileServer(http.Dir(*StaticDir))
	r.PathPrefix("/static/").Handler(http.StripPrefix("/static/", fs))
	r.PathPrefix("/").Handler(http.StripPrefix("/", fs)) // Serve "/" from staticDir
}

func WebServer(port string) {
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
