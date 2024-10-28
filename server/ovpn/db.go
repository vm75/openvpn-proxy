package ovpn

import (
	"database/sql"
	"encoding/json"
	"log"
	"os"
	"path/filepath"

	_ "github.com/mattn/go-sqlite3"
)

type Server struct {
	Name      string              `json:"name"`
	Template  string              `json:"template"`
	Username  string              `json:"username"`
	Password  string              `json:"password"`
	Endpoints []map[string]string `json:"endpoints"`
}

type ProxySettings struct {
	ServerName     string `json:"serverName"`
	ServerEndpoint string `json:"serverEndpoint"`
}

var configsDir = ""
var db *sql.DB = nil

func Init(dataDir string) {
	configsDir = dataDir
	var dbPath = filepath.Join(configsDir, "settings.db")
	var err error
	db, err = sql.Open("sqlite3", dbPath)
	if err != nil {
		log.Fatal(err)
	}

	createServersTable := `CREATE TABLE IF NOT EXISTS servers (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		name TEXT NOT NULL UNIQUE,
		template TEXT NOT NULL,
		username TEXT NOT NULL,
		password TEXT NOT NULL,
		endpoints JSON NOT NULL
	);`

	db.Exec(createServersTable)
}

func GetServers() []Server {
	var templates []Server = make([]Server, 0)
	rows, err := db.Query("SELECT name, template, username, password, endpoints FROM servers")
	if err != nil {
		log.Fatal(err)
	}
	defer rows.Close()
	var endpointsStr []byte
	for rows.Next() {
		var config Server
		err := rows.Scan(
			&config.Name,
			&config.Template,
			&config.Username,
			&config.Password,
			&endpointsStr)
		if err != nil {
			log.Fatal(err)
			return templates
		}
		json.Unmarshal(endpointsStr, &config.Endpoints)
		templates = append(templates, config)
	}
	return templates
}

func GetServer(name string) *Server {
	var config Server
	row := db.QueryRow("SELECT name, template, username, password, endpoints FROM servers WHERE name = ?", name)
	err := row.Scan(
		&config.Name,
		&config.Template,
		&config.Username,
		&config.Password,
		&config.Endpoints)
	if err != nil {
		log.Fatal(err)
		return nil
	}
	return &config
}

func SaveServer(serverConfig Server) error {
	// Remove empty endpoints
	var savedNames = make(map[string]bool)
	for i, endpoint := range serverConfig.Endpoints {
		if endpoint["name"] == "" || savedNames[endpoint["name"]] {
			serverConfig.Endpoints = append(serverConfig.Endpoints[:i], serverConfig.Endpoints[i+1:]...)
		}
		savedNames[endpoint["name"]] = true
	}

	endpointsStr, err := json.Marshal(serverConfig.Endpoints)
	if err != nil {
		log.Fatal(err)
		return err
	}
	_, err = db.Exec(
		"INSERT OR REPLACE INTO servers"+
			"(name, template, username, password, endpoints) "+
			"VALUES (?, ?, ?, ?, ?)",
		serverConfig.Name,
		serverConfig.Template,
		serverConfig.Username,
		serverConfig.Password,
		endpointsStr)
	if err != nil {
		log.Fatal(err)
	}
	return err
}

func DeleteServer(name string) error {
	_, err := db.Exec("DELETE FROM servers WHERE name = ?", name)
	if err != nil {
		log.Fatal(err)
	}
	return err
}

// Proxy config is saved as a json file in the data directory
func GetProxySettings() *ProxySettings {
	file, err := os.ReadFile(filepath.Join(configsDir, "settings.json"))
	if err != nil {
		return nil
	}
	var settings ProxySettings
	json.Unmarshal(file, &settings)
	return &settings
}

func SaveProxySettings(settings *ProxySettings) error {
	file, err := json.MarshalIndent(settings, "", "  ")
	if err != nil {
		log.Fatal(err)
	}
	err = os.WriteFile(filepath.Join(configsDir, "settings.json"), file, 0644)
	if err != nil {
		log.Fatal(err)
	}
	return err
}
