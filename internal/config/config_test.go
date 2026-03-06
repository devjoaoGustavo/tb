package config

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/devjoaoGustavo/tb/internal/model"
)

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()
	if cfg.Invoice.StartNumber != 1 {
		t.Errorf("expected StartNumber=1, got %d", cfg.Invoice.StartNumber)
	}
	if cfg.Invoice.Prefix != "INV" {
		t.Errorf("expected Prefix=INV, got %s", cfg.Invoice.Prefix)
	}
	if cfg.Invoice.DefaultCurrency != model.CurrencyBRL {
		t.Errorf("expected BRL, got %s", cfg.Invoice.DefaultCurrency)
	}
}

func TestSaveAndLoad(t *testing.T) {
	tmpHome := t.TempDir()
	t.Setenv("HOME", tmpHome)

	original := DefaultConfig()
	original.Issuer.Name = "Test User"
	original.Invoice.StartNumber = 42

	if err := Save(&original); err != nil {
		t.Fatalf("Save: %v", err)
	}

	path := filepath.Join(tmpHome, DefaultConfigDir, ConfigFileName)
	data, _ := os.ReadFile(path)
	var raw map[string]interface{}
	if err := json.Unmarshal(data, &raw); err != nil {
		t.Fatalf("not valid JSON: %v", err)
	}

	loaded, err := Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if loaded.Issuer.Name != "Test User" {
		t.Errorf("Name = %q, want Test User", loaded.Issuer.Name)
	}
	if loaded.Invoice.StartNumber != 42 {
		t.Errorf("StartNumber = %d, want 42", loaded.Invoice.StartNumber)
	}
}

func TestLoad_CreatesDefault(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.Issuer.Name != "Your Name" {
		t.Errorf("got %q", cfg.Issuer.Name)
	}
}

func TestLoadState_FirstRun(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	cfg := DefaultConfig()
	cfg.Invoice.StartNumber = 100
	state, err := LoadState(&cfg)
	if err != nil {
		t.Fatalf("LoadState: %v", err)
	}
	if state.NextInvoiceNumber != 100 {
		t.Errorf("got %d, want 100", state.NextInvoiceNumber)
	}
}

func TestLoadState_BumpStartNumber(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	cfg := DefaultConfig()
	cfg.Invoice.StartNumber = 1
	state := &State{NextInvoiceNumber: 5}
	SaveState(&cfg, state)

	cfg.Invoice.StartNumber = 50
	loaded, _ := LoadState(&cfg)
	if loaded.NextInvoiceNumber != 50 {
		t.Errorf("got %d, want 50", loaded.NextInvoiceNumber)
	}
}

func TestLoad_CorruptJSON(t *testing.T) {
	tmpHome := t.TempDir()
	t.Setenv("HOME", tmpHome)

	// Write corrupt JSON to the config file location.
	cfgDir := filepath.Join(tmpHome, DefaultConfigDir)
	if err := os.MkdirAll(cfgDir, 0755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	cfgPath := filepath.Join(cfgDir, ConfigFileName)
	if err := os.WriteFile(cfgPath, []byte("{bad json"), 0644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	_, err := Load()
	if err == nil {
		t.Error("expected error for corrupt JSON, got nil")
	}
}

func TestLoadState_CorruptJSON(t *testing.T) {
	tmpHome := t.TempDir()
	t.Setenv("HOME", tmpHome)

	cfg := DefaultConfig()
	paths := ResolvePaths(&cfg)

	if err := os.MkdirAll(paths.DataDir, 0755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(paths.StateFile, []byte("{bad json"), 0644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	_, err := LoadState(&cfg)
	if err == nil {
		t.Error("expected error for corrupt state JSON, got nil")
	}
}

func TestResolvePaths_EnvOverrides(t *testing.T) {
	tmpConfig := t.TempDir()
	tmpData := t.TempDir()
	t.Setenv("TB_CONFIG_DIR", tmpConfig)
	t.Setenv("TB_DATA_DIR", tmpData)

	paths := ResolvePaths(nil)
	if paths.ConfigDir != tmpConfig {
		t.Errorf("ConfigDir = %q, want %q", paths.ConfigDir, tmpConfig)
	}
	if paths.DataDir != tmpData {
		t.Errorf("DataDir = %q, want %q", paths.DataDir, tmpData)
	}
}
