package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// Account represents a single email account
type Account struct {
	ID        string    `json:"id"`   // Unique identifier (email-based)
	Name      string    `json:"name"` // User-friendly name (defaults to email)
	Email     string    `json:"email"`
	Server    string    `json:"server"`
	Password  string    `json:"password"` // encrypted
	CreatedAt time.Time `json:"created_at"`
}

// Config stores all accounts and the currently selected one
type Config struct {
	Accounts   []Account `json:"accounts"`
	SelectedID string    `json:"selected_id"` // ID of currently selected account
}

// Legacy Config for backward compatibility
type LegacyConfig struct {
	Email    string `json:"email"`
	Server   string `json:"server"`
	Password string `json:"password"` // encrypted
}

func ConfigDir() (string, error) {
	dir, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	path := filepath.Join(dir, "newsletter-cli")
	if _, err := os.Stat(path); os.IsNotExist(err) {
		os.MkdirAll(path, 0700)
	}
	return path, nil
}

func ConfigPath() (string, error) {
	dir, err := ConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "config.json"), nil
}

// Save saves the config with all accounts
func Save(cfg Config) error {
	path, err := ConfigPath()
	if err != nil {
		return err
	}
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}
	err = os.WriteFile(path, data, 0600)
	if err != nil {
		return err
	}

	// Auto-sync to cloud if premium is enabled
	// Import here to avoid circular dependency
	go func() {
		// Use a separate import to avoid circular dependency
		// This will be handled by the UI layer calling AutoSync after Save
	}()

	return nil
}

// Load loads the config, handling both new and legacy formats
func Load() (*Config, error) {
	path, err := ConfigPath()
	if err != nil {
		return nil, err
	}

	// Return empty config if file doesn't exist
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return &Config{Accounts: []Account{}}, nil
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	// Try to parse as new format first
	var cfg Config
	if err := json.Unmarshal(data, &cfg); err == nil {
		// Successfully parsed as new format
		return &cfg, nil
	}

	// Try legacy format
	var legacyCfg LegacyConfig
	if err := json.Unmarshal(data, &legacyCfg); err == nil && legacyCfg.Email != "" {
		// Migrate legacy config to new format
		account := Account{
			ID:        legacyCfg.Email,
			Name:      legacyCfg.Email,
			Email:     legacyCfg.Email,
			Server:    legacyCfg.Server,
			Password:  legacyCfg.Password,
			CreatedAt: time.Now(),
		}
		cfg = Config{
			Accounts:   []Account{account},
			SelectedID: account.ID,
		}
		// Save migrated config
		Save(cfg)
		return &cfg, nil
	}

	// If neither format works, return empty config
	return &Config{Accounts: []Account{}}, nil
}

// GetAccount returns an account by ID
func GetAccount(id string) (*Account, error) {
	cfg, err := Load()
	if err != nil {
		return nil, err
	}

	for _, acc := range cfg.Accounts {
		if acc.ID == id {
			return &acc, nil
		}
	}

	return nil, fmt.Errorf("account not found: %s", id)
}

// GetSelectedAccount returns the currently selected account
func GetSelectedAccount() (*Account, error) {
	cfg, err := Load()
	if err != nil {
		return nil, err
	}

	if cfg.SelectedID == "" {
		if len(cfg.Accounts) > 0 {
			// Auto-select first account if none selected
			cfg.SelectedID = cfg.Accounts[0].ID
			Save(*cfg)
			return &cfg.Accounts[0], nil
		}
		return nil, fmt.Errorf("no accounts available")
	}

	return GetAccount(cfg.SelectedID)
}

// AddAccount adds a new account
func AddAccount(email, server, password, name string) (*Account, error) {
	cfg, err := Load()
	if err != nil {
		return nil, err
	}

	// Encrypt password
	encryptedPassword, err := Encrypt(password)
	if err != nil {
		return nil, fmt.Errorf("failed to encrypt password: %w", err)
	}

	// Use email as ID
	id := email
	if name == "" {
		name = email
	}

	// Check if account already exists
	for i, acc := range cfg.Accounts {
		if acc.ID == id {
			// Update existing account
			cfg.Accounts[i].Name = name
			cfg.Accounts[i].Server = server
			cfg.Accounts[i].Password = encryptedPassword
			// Don't change SelectedID when updating existing account - preserve user's selection
			// Only set SelectedID if no account is currently selected
			if cfg.SelectedID == "" {
				cfg.SelectedID = id
			}
			if err := Save(*cfg); err != nil {
				return nil, err
			}
			return &cfg.Accounts[i], nil
		}
	}

	// Create new account
	account := Account{
		ID:        id,
		Name:      name,
		Email:     email,
		Server:    server,
		Password:  encryptedPassword,
		CreatedAt: time.Now(),
	}

	cfg.Accounts = append(cfg.Accounts, account)
	// Only auto-select new account if no account is currently selected
	if cfg.SelectedID == "" {
		cfg.SelectedID = account.ID
	}

	if err := Save(*cfg); err != nil {
		return nil, err
	}

	return &account, nil
}

// DeleteAccount removes an account by ID
func DeleteAccount(id string) error {
	cfg, err := Load()
	if err != nil {
		return err
	}

	var newAccounts []Account
	for _, acc := range cfg.Accounts {
		if acc.ID != id {
			newAccounts = append(newAccounts, acc)
		}
	}

	cfg.Accounts = newAccounts

	// Clear selection if deleted account was selected
	if cfg.SelectedID == id {
		if len(cfg.Accounts) > 0 {
			cfg.SelectedID = cfg.Accounts[0].ID
		} else {
			cfg.SelectedID = ""
		}
	}

	return Save(*cfg)
}

// SetSelectedAccount sets the currently selected account
func SetSelectedAccount(id string) error {
	cfg, err := Load()
	if err != nil {
		return err
	}

	// Verify account exists
	found := false
	for _, acc := range cfg.Accounts {
		if acc.ID == id {
			found = true
			break
		}
	}

	if !found {
		return fmt.Errorf("account not found: %s", id)
	}

	cfg.SelectedID = id
	return Save(*cfg)
}

// GetAllAccounts returns all accounts
func GetAllAccounts() ([]Account, error) {
	cfg, err := Load()
	if err != nil {
		return nil, err
	}
	return cfg.Accounts, nil
}
