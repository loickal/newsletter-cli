package config

import (
	"bytes"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"os"
	"os/user"
	"strings"

	"filippo.io/age"
)

// deriveKey derives an encryption key from system-specific information
// This ensures the key is unique to the user's machine
func deriveKey() ([]byte, error) {
	// Collect system-specific information
	var parts []string

	// User home directory
	usr, err := user.Current()
	if err == nil {
		parts = append(parts, usr.HomeDir)
		parts = append(parts, usr.Username)
	}

	// Config directory
	if configDir, err := ConfigDir(); err == nil {
		parts = append(parts, configDir)
	}

	// If we have no parts, use a fallback (less secure but better than nothing)
	if len(parts) == 0 {
		hostname, _ := os.Hostname()
		parts = []string{hostname, "newsletter-cli"}
	}

	// Create a deterministic key from the parts
	input := strings.Join(parts, ":")
	hash := sha256.Sum256([]byte(input))
	return hash[:], nil
}

// getPassphrase derives a passphrase from system-specific information
// This ensures the passphrase is unique to the user's machine
func getPassphrase() (string, error) {
	key, err := deriveKey()
	if err != nil {
		return "", err
	}
	// Convert key to a string passphrase (base64 encoded for readability)
	return base64.StdEncoding.EncodeToString(key), nil
}

// getRecipient creates an age recipient from the derived passphrase
func getRecipient() (age.Recipient, error) {
	passphrase, err := getPassphrase()
	if err != nil {
		return nil, err
	}
	// Create a Scrypt recipient using the passphrase
	return age.NewScryptRecipient(passphrase)
}

// getIdentity creates an age identity from the derived passphrase
func getIdentity() (age.Identity, error) {
	passphrase, err := getPassphrase()
	if err != nil {
		return nil, err
	}
	// Create a Scrypt identity using the passphrase
	return age.NewScryptIdentity(passphrase)
}

// Encrypt encrypts a string using age encryption
// The encryption key is derived from system-specific information
func Encrypt(input string) (string, error) {
	if input == "" {
		return "", nil
	}

	recipient, err := getRecipient()
	if err != nil {
		return "", fmt.Errorf("failed to create recipient: %w", err)
	}

	var buf bytes.Buffer
	w, err := age.Encrypt(&buf, recipient)
	if err != nil {
		return "", fmt.Errorf("failed to create encrypt writer: %w", err)
	}

	if _, err := w.Write([]byte(input)); err != nil {
		return "", fmt.Errorf("failed to write data: %w", err)
	}

	if err := w.Close(); err != nil {
		return "", fmt.Errorf("failed to close encrypt writer: %w", err)
	}

	// Return base64-encoded encrypted data
	return base64.StdEncoding.EncodeToString(buf.Bytes()), nil
}

// Decrypt decrypts an age-encrypted string
func Decrypt(encrypted string) (string, error) {
	if encrypted == "" {
		return "", nil
	}

	// Handle legacy XOR-encrypted data (for backward compatibility)
	if isLegacyFormat(encrypted) {
		return decryptLegacy(encrypted), nil
	}

	// Decode base64
	data, err := base64.StdEncoding.DecodeString(encrypted)
	if err != nil {
		// If base64 decode fails, try legacy decryption
		return decryptLegacy(encrypted), nil
	}

	identity, err := getIdentity()
	if err != nil {
		return "", fmt.Errorf("failed to create identity: %w", err)
	}

	r, err := age.Decrypt(bytes.NewReader(data), identity)
	if err != nil {
		// If age decryption fails, try legacy (might be old format)
		return decryptLegacy(encrypted), nil
	}

	var buf bytes.Buffer
	if _, err := buf.ReadFrom(r); err != nil {
		return "", fmt.Errorf("failed to read decrypted data: %w", err)
	}

	return buf.String(), nil
}

// isLegacyFormat checks if the encrypted string is in the old XOR format
// Legacy format doesn't use base64 and contains non-ASCII characters
func isLegacyFormat(encrypted string) bool {
	if len(encrypted) == 0 {
		return false
	}

	// Try to decode as base64 - if it fails, likely legacy
	_, err := base64.StdEncoding.DecodeString(encrypted)
	if err != nil {
		return true
	}

	// Check if decoded data looks like age format (starts with age-1)
	decoded, err := base64.StdEncoding.DecodeString(encrypted)
	if err == nil && len(decoded) > 0 {
		// Age format typically starts with specific magic bytes
		// If it doesn't start with age format, might be legacy
		if len(decoded) < 16 {
			return true
		}
		// Check first few bytes - legacy XOR would produce random-looking bytes
		// Age format has more structure
		return false
	}

	return true
}

// decryptLegacy decrypts using the old XOR method (for backward compatibility)
func decryptLegacy(input string) string {
	const key = "newslettercli"
	if input == "" {
		return ""
	}
	out := make([]rune, len(input))
	for i, r := range input {
		out[i] = r ^ rune(key[i%len(key)])
	}
	return string(out)
}

// Mask masks a string for display
func Mask(s string) string {
	if len(s) == 0 {
		return "(empty)"
	}
	return strings.Repeat("*", len(s))
}
