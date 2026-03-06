package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/joaogustavo/tb/internal/model"
)

const (
	DefaultConfigDir = ".config/tb"
	DefaultDataDir   = ".local/share/tb"
	ConfigFileName   = "config.json"
	StateFileName    = "state.json"
)

// InvoiceSettings controls how invoice numbers are generated and defaults.
type InvoiceSettings struct {
	Prefix          string         `json:"prefix"`
	StartNumber     int            `json:"start_number"`
	Padding         int            `json:"padding"`
	Separator       string         `json:"separator"`
	IncludeYear     bool           `json:"include_year"`
	PerClientPrefix bool           `json:"per_client_prefix"`
	DefaultDueDays  int            `json:"default_due_days"`
	DefaultTaxRate  float64        `json:"default_tax_rate"`
	DefaultCurrency model.Currency `json:"default_currency"`
	DefaultNotes    string         `json:"default_notes,omitempty"`
}

// Config is the main user configuration file (~/.config/tb/config.json).
type Config struct {
	Issuer  model.Issuer    `json:"issuer"`
	Invoice InvoiceSettings `json:"invoice"`
	DataDir string          `json:"data_dir,omitempty"`
	Locale  string          `json:"locale,omitempty"`
}

// State holds mutable runtime counters stored separately from config.
type State struct {
	NextInvoiceNumber int `json:"next_invoice_number"`
}

func DefaultConfig() Config {
	return Config{
		Issuer: model.Issuer{
			Name:  "Your Name",
			Email: "you@example.com",
		},
		Invoice: InvoiceSettings{
			Prefix:          "INV",
			StartNumber:     1,
			Padding:         3,
			Separator:       "-",
			IncludeYear:     true,
			PerClientPrefix: false,
			DefaultDueDays:  15,
			DefaultTaxRate:  0,
			DefaultCurrency: model.CurrencyBRL,
		},
		Locale: "pt-BR",
	}
}

type Paths struct {
	ConfigDir  string
	DataDir    string
	ConfigFile string
	StateFile  string
	DBFile     string
	InvoiceDir string
}

func ResolvePaths(cfg *Config) Paths {
	home, _ := os.UserHomeDir()

	configDir := filepath.Join(home, DefaultConfigDir)
	if dir := os.Getenv("TB_CONFIG_DIR"); dir != "" {
		configDir = dir
	}

	dataDir := filepath.Join(home, DefaultDataDir)
	if cfg != nil && cfg.DataDir != "" {
		dataDir = cfg.DataDir
	}
	if dir := os.Getenv("TB_DATA_DIR"); dir != "" {
		dataDir = dir
	}

	return Paths{
		ConfigDir:  configDir,
		DataDir:    dataDir,
		ConfigFile: filepath.Join(configDir, ConfigFileName),
		StateFile:  filepath.Join(dataDir, StateFileName),
		DBFile:     filepath.Join(dataDir, "timebill.db"),
		InvoiceDir: filepath.Join(dataDir, "invoices"),
	}
}

func Load() (*Config, error) {
	paths := ResolvePaths(nil)

	if err := os.MkdirAll(paths.ConfigDir, 0755); err != nil {
		return nil, fmt.Errorf("creating config dir: %w", err)
	}

	data, err := os.ReadFile(paths.ConfigFile)
	if os.IsNotExist(err) {
		cfg := DefaultConfig()
		if saveErr := Save(&cfg); saveErr != nil {
			return nil, fmt.Errorf("writing default config: %w", saveErr)
		}
		fmt.Printf("Created default config at %s\n", paths.ConfigFile)
		fmt.Println("  Edit it with your info: tb config edit")
		return &cfg, nil
	}
	if err != nil {
		return nil, fmt.Errorf("reading config: %w", err)
	}

	cfg := DefaultConfig()
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("parsing config: %w", err)
	}

	return &cfg, nil
}

func Save(cfg *Config) error {
	paths := ResolvePaths(cfg)

	if err := os.MkdirAll(paths.ConfigDir, 0755); err != nil {
		return err
	}

	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(paths.ConfigFile, data, 0644)
}

func LoadState(cfg *Config) (*State, error) {
	paths := ResolvePaths(cfg)

	if err := os.MkdirAll(paths.DataDir, 0755); err != nil {
		return nil, fmt.Errorf("creating data dir: %w", err)
	}

	data, err := os.ReadFile(paths.StateFile)
	if os.IsNotExist(err) {
		return &State{NextInvoiceNumber: cfg.Invoice.StartNumber}, nil
	}
	if err != nil {
		return nil, fmt.Errorf("reading state: %w", err)
	}

	var state State
	if err := json.Unmarshal(data, &state); err != nil {
		return nil, fmt.Errorf("parsing state: %w", err)
	}

	if state.NextInvoiceNumber < cfg.Invoice.StartNumber {
		state.NextInvoiceNumber = cfg.Invoice.StartNumber
	}

	return &state, nil
}

func SaveState(cfg *Config, state *State) error {
	paths := ResolvePaths(cfg)

	if err := os.MkdirAll(paths.DataDir, 0755); err != nil {
		return err
	}

	data, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(paths.StateFile, data, 0644)
}
