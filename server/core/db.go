package core

import (
	"database/sql"
	"encoding/json"
	"path/filepath"

	_ "github.com/mattn/go-sqlite3"
)

var Db *sql.DB = nil
var createSettingsQuery = `CREATE TABLE IF NOT EXISTS settings (
	id INTEGER PRIMARY KEY AUTOINCREMENT,
	name TEXT NOT NULL UNIQUE,
	settings JSON NOT NULL
);`
var getSettingsQuery = `SELECT settings
	FROM settings
	WHERE name = ?;`
var saveSettingsQuery = `INSERT OR REPLACE INTO settings
	(name, settings)
	VALUES (?, ?);`

func initDb() error {
	var dbPath = filepath.Join(ConfigDir, "settings.db")
	var err error
	Db, err = sql.Open("sqlite3", dbPath)
	if err != nil {
		return err
	}
	_, err = Db.Exec(createSettingsQuery)
	return err
}

func SaveSettings(name string, settings map[string]interface{}) error {
	settingsStr, _ := json.Marshal(settings)
	_, err := Db.Exec(saveSettingsQuery, name, settingsStr)
	return err
}

func GetSettings(name string) (map[string]interface{}, error) {
	var settingsStr []byte
	row := Db.QueryRow(getSettingsQuery, name)
	err := row.Scan(&settingsStr)
	if err != nil {
		return nil, err
	}
	var settings map[string]interface{}
	json.Unmarshal(settingsStr, &settings)
	return settings, nil
}
