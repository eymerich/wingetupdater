package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestSettingsRoundtrip(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("AppData", dir)

	p, err := settingsPath()
	if err != nil {
		t.Fatalf("settingsPath: %v", err)
	}
	want := filepath.Join(dir, "github.com_eymerich_wingetupdater", "settings.json")
	if p != want {
		t.Fatalf("path = %q, want %q", p, want)
	}

	in := &Settings{Whitelist: []string{"A.B", "C.D"}}
	if err := saveSettings(in); err != nil {
		t.Fatalf("saveSettings: %v", err)
	}
	if _, err := os.Stat(p); err != nil {
		t.Fatalf("file non creato: %v", err)
	}

	out, err := loadSettings()
	if err != nil {
		t.Fatalf("loadSettings: %v", err)
	}
	if len(out.Whitelist) != 2 || out.Whitelist[0] != "A.B" || out.Whitelist[1] != "C.D" {
		t.Fatalf("whitelist = %v", out.Whitelist)
	}
}
