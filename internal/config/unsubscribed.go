package config

import (
	"encoding/json"
	"os"
	"path/filepath"
	"time"
)

// UnsubscribedNewsletter represents an unsubscribed newsletter
type UnsubscribedNewsletter struct {
	Sender         string    `json:"sender"`
	UnsubscribedAt time.Time `json:"unsubscribed_at"`
}

// UnsubscribedStore manages the list of unsubscribed newsletters
type UnsubscribedStore struct {
	Newsletters []UnsubscribedNewsletter `json:"newsletters"`
}

// UnsubscribedPath returns the path to the unsubscribed newsletters file
func UnsubscribedPath() (string, error) {
	dir, err := ConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "unsubscribed.json"), nil
}

// LoadUnsubscribed loads the list of unsubscribed newsletters
func LoadUnsubscribed() (*UnsubscribedStore, error) {
	path, err := UnsubscribedPath()
	if err != nil {
		return nil, err
	}

	// Return empty store if file doesn't exist
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return &UnsubscribedStore{Newsletters: []UnsubscribedNewsletter{}}, nil
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var store UnsubscribedStore
	if err := json.Unmarshal(data, &store); err != nil {
		return nil, err
	}

	return &store, nil
}

// SaveUnsubscribed saves the list of unsubscribed newsletters
func SaveUnsubscribed(store *UnsubscribedStore) error {
	path, err := UnsubscribedPath()
	if err != nil {
		return err
	}

	data, err := json.MarshalIndent(store, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(path, data, 0600)
}

// AddUnsubscribed adds a newsletter to the unsubscribed list
func AddUnsubscribed(sender string) error {
	store, err := LoadUnsubscribed()
	if err != nil {
		return err
	}

	// Check if already exists
	for _, n := range store.Newsletters {
		if n.Sender == sender {
			// Already exists, just update timestamp
			store.Newsletters = removeUnsubscribed(store.Newsletters, sender)
			break
		}
	}

	// Add new entry
	store.Newsletters = append(store.Newsletters, UnsubscribedNewsletter{
		Sender:         sender,
		UnsubscribedAt: time.Now(),
	})

	return SaveUnsubscribed(store)
}

// IsUnsubscribed checks if a newsletter is in the unsubscribed list
func IsUnsubscribed(sender string) (bool, error) {
	store, err := LoadUnsubscribed()
	if err != nil {
		return false, err
	}

	for _, n := range store.Newsletters {
		if n.Sender == sender {
			return true, nil
		}
	}

	return false, nil
}

// GetUnsubscribedList returns all unsubscribed newsletter senders
func GetUnsubscribedList() (map[string]bool, error) {
	store, err := LoadUnsubscribed()
	if err != nil {
		return nil, err
	}

	result := make(map[string]bool)
	for _, n := range store.Newsletters {
		result[n.Sender] = true
	}

	return result, nil
}

// removeUnsubscribed removes a newsletter from the list
func removeUnsubscribed(list []UnsubscribedNewsletter, sender string) []UnsubscribedNewsletter {
	result := []UnsubscribedNewsletter{}
	for _, n := range list {
		if n.Sender != sender {
			result = append(result, n)
		}
	}
	return result
}
