package daemon

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

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
	ServerName     string   `json:"serverName"`
	ServerEndpoint string   `json:"serverEndpoint"`
	HttpProxy      string   `json:"httpProxy"`
	SocksProxy     string   `json:"socksProxy"`
	Subnets        []string `json:"subnets"`
	ProxyUsername  string   `json:"proxyUsername"`
	ProxyPassword  string   `json:"proxyPassword"`
	VpnLogLevel    int      `json:"vpnLogLevel"`
	RetryInterval  int      `json:"retryInterval"`
}

var db *sql.DB = nil
var ConfigDir = ""
var VarDir = ""
var ConfigFile = ""
var AuthFile = ""
var SettingsFile = ""
var PidFile = ""
var OpenVpnLogFile = ""
var OpenVpnStatusFile = ""
var DefaultGateway = ""

func NewProxySettings() *ProxySettings {
	return &ProxySettings{
		ServerName:     "",
		ServerEndpoint: "",
		HttpProxy:      "",
		SocksProxy:     "",
		Subnets:        []string{},
		ProxyUsername:  "",
		ProxyPassword:  "",
		VpnLogLevel:    0,
		RetryInterval:  3600,
	}
}

func Init(dataDir string) error {
	ConfigDir = filepath.Join(dataDir, "config")
	VarDir = filepath.Join(dataDir, "var")
	ConfigFile = filepath.Join(VarDir, "vpn.ovpn")
	AuthFile = filepath.Join(VarDir, "vpn.auth")
	SettingsFile = filepath.Join(ConfigDir, "settings.json")
	PidFile = filepath.Join(VarDir, "openvpn.pid")
	OpenVpnLogFile = filepath.Join(VarDir, "openvpn.log")
	OpenVpnStatusFile = filepath.Join(VarDir, "openvpn.status")

	var dbPath = filepath.Join(ConfigDir, "settings.db")

	var err error
	err = os.MkdirAll(ConfigDir, 0755)
	if err != nil {
		log.Fatal(err)
		return err
	}

	cmd := exec.Command("ip", "r")

	// Capture standard output and standard error
	output, err := cmd.CombinedOutput()
	if err == nil {
		// get line starting with default via and get the following ip in that line
		DefaultGateway = strings.Split(strings.Split(string(output), "default via ")[1], " ")[0]
	}

	var settings = GetProxySettings()
	if settings == nil {
		settings = &ProxySettings{}
	}
	err = saveEnv(*settings)
	if err != nil {
		log.Fatal(err)
		return err
	}

	db, err = sql.Open("sqlite3", dbPath)
	if err != nil {
		log.Fatal(err)
		return err
	}

	createServersTable := `CREATE TABLE IF NOT EXISTS servers (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		name TEXT NOT NULL UNIQUE,
		template TEXT NOT NULL,
		username TEXT NOT NULL,
		password TEXT NOT NULL,
		endpoints JSON NOT NULL
	);`

	_, err = db.Exec(createServersTable)
	if err != nil {
		log.Fatal(err)
		return err
	}
	return nil
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
	row := db.QueryRow(
		"SELECT name, template, username, password, endpoints "+
			"FROM servers "+
			"WHERE name = ?", name)
	var endpointsStr []byte
	err := row.Scan(
		&config.Name,
		&config.Template,
		&config.Username,
		&config.Password,
		&endpointsStr)
	if err != nil {
		log.Fatal(err)
		return nil
	}
	json.Unmarshal(endpointsStr, &config.Endpoints)
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
		"INSERT OR REPLACE INTO servers "+
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
	file, err := os.ReadFile(SettingsFile)
	if err != nil {
		return NewProxySettings()
	}
	var settings ProxySettings
	json.Unmarshal(file, &settings)
	return &settings
}

func saveEnv(settings ProxySettings) error {
	var sb strings.Builder

	sb.WriteString(fmt.Sprintf("SUBNETS=%s\n", strings.Join(settings.Subnets, ",")))
	sb.WriteString(fmt.Sprintf("SOCKS_PROXY=%s\n", settings.SocksProxy))
	sb.WriteString(fmt.Sprintf("VPN_LOG_LEVEL=%d\n", settings.VpnLogLevel))
	sb.WriteString(fmt.Sprintf("HTTP_PROXY=%s\n", settings.HttpProxy))
	sb.WriteString(fmt.Sprintf("DEFAULT_GATEWAY=%s\n", DefaultGateway))

	return os.WriteFile(filepath.Join(VarDir, "env"), []byte(sb.String()), 0755)
}

func updateContent(content string, file string) (bool, error) {
	fileContent, err := os.ReadFile(file)
	if err != nil {
		return true, nil
	}
	if string(fileContent) == content {
		return false, nil
	}
	err = os.WriteFile(file, []byte(content), 0644)
	if err != nil {
		return false, err
	}
	return true, nil
}

func saveOvpnConfig(settings ProxySettings) error {
	var server = GetServer(settings.ServerName)

	if server == nil {
		return errors.New("server not found")
	}

	var ovpn = server.Template
	var endpoint map[string]string = nil

	for _, entry := range server.Endpoints {
		if entry["name"] == settings.ServerEndpoint {
			endpoint = entry
			break
		}
	}

	if endpoint == nil {
		return errors.New("endpoint not found")
	}

	for key, value := range endpoint {
		ovpn = strings.ReplaceAll(ovpn, "{{"+key+"}}", value)
	}

	auth := fmt.Sprintf("%s\n%s\n", server.Username, server.Password)

	configUpdated, configErr := updateContent(ovpn, ConfigFile)
	if configErr != nil {
		return configErr
	}
	authUpdated, authErr := updateContent(auth, AuthFile)
	if authErr != nil {
		return authErr
	}

	if configUpdated || authUpdated {
		log.Println("Configuration updated, restarting OpenVPN")
		RestartVPN()
		StartVPN()
	}

	return nil
}

func SaveProxySettings(settings *ProxySettings) error {
	file, err := json.MarshalIndent(settings, "", "  ")
	if err != nil {
		log.Fatal(err)
	}

	err = os.WriteFile(SettingsFile, file, 0644)
	if err != nil {
		log.Fatal(err)
		return err
	}

	err = saveOvpnConfig(*settings)
	if err != nil {
		log.Fatal(err)
		return err
	}

	err = saveEnv(*settings)
	if err != nil {
		log.Fatal(err)
	}

	return err
}
