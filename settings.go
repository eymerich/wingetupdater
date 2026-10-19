package main

import (
	"encoding/json"
	"os"
	"path/filepath"
)

// Settings rappresenta la configurazione persistita dell'applicazione.
// Viene salvata in %APPDATA%\github.com_eymerich_wingetupdater\settings.json
type Settings struct {
	// Whitelist contiene gli ID dei pacchetti che "Seleziona tutto"
	// deve considerare nella selezione/deselezione globale.
	Whitelist []string `json:"whitelist"`
}

// settingsDir restituisce la cartella di configurazione dell'applicazione.
func settingsDir() (string, error) {
	base, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(base, "github.com_eymerich_wingetupdater"), nil
}

// settingsPath restituisce il percorso completo del file settings.json.
func settingsPath() (string, error) {
	dir, err := settingsDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "settings.json"), nil
}

// loadSettings legge il file di configurazione; se mancante restituisce
// una Settings vuota senza errore.
func loadSettings() (*Settings, error) {
	path, err := settingsPath()
	if err != nil {
		return &Settings{}, err
	}

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return &Settings{}, nil
		}
		return &Settings{}, err
	}

	s := &Settings{}
	if err := json.Unmarshal(data, s); err != nil {
		// File corrotto: riparti da una configurazione vuota.
		return &Settings{}, nil
	}
	return s, nil
}

// saveSettings serializza e scrive il file di configurazione,
// creando la cartella se necessario.
func saveSettings(s *Settings) error {
	dir, err := settingsDir()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}

	data, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return err
	}

	path := filepath.Join(dir, "settings.json")
	return os.WriteFile(path, data, 0o644)
}
