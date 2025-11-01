package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/loickal/newsletter-cli/internal/config"
)

type PremiumConfig struct {
	APIURL                   string    `json:"api_url"`
	Token                    string    `json:"token"`
	RefreshToken             string    `json:"refresh_token"`
	Email                    string    `json:"email"`
	Enabled                  bool      `json:"enabled"`
	LastSyncTime             time.Time `json:"last_sync_time,omitempty"`
	LastAccountsSync         time.Time `json:"last_accounts_sync,omitempty"`
	LastUnsubSync            time.Time `json:"last_unsub_sync,omitempty"`
	AccountsSynced           int       `json:"accounts_synced,omitempty"`
	UnsubscribedCount        int       `json:"unsubscribed_count,omitempty"`
	LocalAccountsVersion     int64     `json:"local_accounts_version,omitempty"`
	LocalUnsubscribedVersion int64     `json:"local_unsubscribed_version,omitempty"`

	// Sync settings
	AutoSyncOnStartup    bool `json:"auto_sync_on_startup,omitempty"`           // Default: true
	PeriodicSyncEnabled  bool `json:"periodic_sync_enabled,omitempty"`          // Default: true
	PeriodicSyncInterval int  `json:"periodic_sync_interval_minutes,omitempty"` // Default: 5
	SyncAccounts         bool `json:"sync_accounts,omitempty"`                  // Default: true
	SyncUnsubscribed     bool `json:"sync_unsubscribed,omitempty"`              // Default: true

	// Analytics settings
	// Note: We use omitempty, but when user explicitly toggles, we ensure it's written
	AnalyticsEnabled bool `json:"analytics_enabled,omitempty"` // Default: true for new premium users
	// Track if user has explicitly set analytics (to distinguish from default)
	AnalyticsExplicitlySet bool `json:"analytics_explicitly_set,omitempty"`

	// API Secret for HMAC signing (optional)
	APISecret string `json:"api_secret,omitempty"`
}

const PremiumConfigFile = "premium.json"

func GetPremiumConfig() (*PremiumConfig, error) {
	configDir, err := config.ConfigDir()
	if err != nil {
		return nil, err
	}

	configPath := filepath.Join(configDir, PremiumConfigFile)
	data, err := os.ReadFile(configPath)
	if os.IsNotExist(err) {
		// Return default disabled config
		return &PremiumConfig{
			Enabled:              false,
			APIURL:               "https://api.newsletter-cli.apps.paas-01.pulseflow.cloud", // Default API URL
			AutoSyncOnStartup:    true,
			PeriodicSyncEnabled:  true,
			PeriodicSyncInterval: 5,
			SyncAccounts:         true,
			SyncUnsubscribed:     true,
			AnalyticsEnabled:     true, // Default to enabled for new premium users
		}, nil
	}
	if err != nil {
		return nil, err
	}

	var premiumConfig PremiumConfig
	if err := json.Unmarshal(data, &premiumConfig); err != nil {
		return nil, err
	}

	// Set defaults for new fields (backward compatibility)
	if premiumConfig.AutoSyncOnStartup == false && premiumConfig.PeriodicSyncEnabled == false && premiumConfig.PeriodicSyncInterval == 0 && !premiumConfig.SyncAccounts && !premiumConfig.SyncUnsubscribed {
		// All fields are false/zero, likely old config - set defaults
		premiumConfig.AutoSyncOnStartup = true
		premiumConfig.PeriodicSyncEnabled = true
		premiumConfig.PeriodicSyncInterval = 5
		premiumConfig.SyncAccounts = true
		premiumConfig.SyncUnsubscribed = true
	}

	// Default analytics to enabled for premium users (backward compatibility)
	// For existing premium users, if analytics_enabled field doesn't exist in JSON,
	// the bool will be false. We enable it by default for premium users.
	if premiumConfig.Enabled && premiumConfig.Token != "" {
		// Check if analytics_enabled field exists in raw JSON
		// If it doesn't exist and premium is enabled, default to true
		var rawJSON map[string]interface{}
		if err := json.Unmarshal(data, &rawJSON); err == nil {
			_, analyticsFieldExists := rawJSON["analytics_enabled"]

			_, explicitlySetFieldExists := rawJSON["analytics_explicitly_set"]

			if !analyticsFieldExists {
				// Field doesn't exist in JSON - default to true for premium users
				premiumConfig.AnalyticsEnabled = true
				premiumConfig.AnalyticsExplicitlySet = false // Not explicitly set by user
				// Save the updated config with default value (this will write analytics_enabled: true)
				_ = SavePremiumConfig(&premiumConfig)
			} else if analyticsFieldExists {
				// Field exists in JSON - use the actual value from JSON
				// (premiumConfig.AnalyticsEnabled already has the correct value from unmarshal)
				if !explicitlySetFieldExists {
					// If field exists but explicitlySet doesn't, mark it as set (backward compat)
					premiumConfig.AnalyticsExplicitlySet = true
					// Update the config file to include the flag
					_ = SavePremiumConfig(&premiumConfig)
				}
			}
		}
	}

	return &premiumConfig, nil
}

func SavePremiumConfig(cfg *PremiumConfig) error {
	configDir, err := config.ConfigDir()
	if err != nil {
		return err
	}

	configPath := filepath.Join(configDir, PremiumConfigFile)

	// Marshal to JSON first
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}

	// Handle analytics_enabled persistence:
	// 1. If explicitly set to false, we need to write it (omitempty would skip false)
	// 2. If true (default or explicitly), it will be written normally
	// 3. We always write analytics_explicitly_set when user has toggled
	var jsonMap map[string]interface{}
	if err := json.Unmarshal(data, &jsonMap); err == nil {
		if cfg.AnalyticsExplicitlySet && !cfg.AnalyticsEnabled {
			// User explicitly disabled - ensure it's written even though it's false
			jsonMap["analytics_enabled"] = false
		} else if !cfg.AnalyticsExplicitlySet && cfg.AnalyticsEnabled {
			// Default enabled state - ensure it's written
			jsonMap["analytics_enabled"] = true
		}
		// Always write explicitly_set flag when user has interacted
		if cfg.AnalyticsExplicitlySet {
			jsonMap["analytics_explicitly_set"] = true
		}
		// Re-marshal with explicit fields
		data, _ = json.MarshalIndent(jsonMap, "", "  ")
	}

	return os.WriteFile(configPath, data, 0600)
}

func IsPremiumEnabled() bool {
	cfg, err := GetPremiumConfig()
	if err != nil {
		return false
	}
	return cfg.Enabled && cfg.Token != ""
}

func GetAPIClient() (*Client, error) {
	cfg, err := GetPremiumConfig()
	if err != nil || !cfg.Enabled {
		return nil, fmt.Errorf("premium features not enabled")
	}

	client := NewClient(cfg.APIURL)
	if cfg.Token != "" {
		client.SetToken(cfg.Token)
	}
	if cfg.RefreshToken != "" {
		client.RefreshToken = cfg.RefreshToken
	}
	if cfg.APISecret != "" {
		client.APISecret = cfg.APISecret
	}

	// Set callback to save refreshed tokens
	client.OnTokenRefresh = func(newToken, newRefreshToken string) error {
		cfg.Token = newToken
		if newRefreshToken != "" {
			cfg.RefreshToken = newRefreshToken
		}
		return SavePremiumConfig(cfg)
	}

	return client, nil
}

// GenerateAPISecret generates and saves an API secret for HMAC signing
func GenerateAPISecret() (string, error) {
	if !IsPremiumEnabled() {
		return "", fmt.Errorf("premium features not enabled")
	}

	client, err := GetAPIClient()
	if err != nil {
		return "", err
	}

	// Generate secret on backend
	secretResp, err := client.GenerateAPISecret()
	if err != nil {
		return "", fmt.Errorf("failed to generate API secret: %w", err)
	}

	// Save to local config
	cfg, err := GetPremiumConfig()
	if err != nil {
		return "", err
	}

	cfg.APISecret = secretResp.APISecret
	if err := SavePremiumConfig(cfg); err != nil {
		return "", fmt.Errorf("failed to save API secret: %w", err)
	}

	return secretResp.APISecret, nil
}

// GetAPISecretStatus checks if user has an API secret configured
func GetAPISecretStatus() (bool, error) {
	if !IsPremiumEnabled() {
		return false, nil
	}

	client, err := GetAPIClient()
	if err != nil {
		return false, err
	}

	statusResp, err := client.GetAPISecretStatus()
	if err != nil {
		return false, err
	}

	return statusResp.HasSecret, nil
}

// SyncAccountsToCloud syncs local accounts to cloud with retry logic
func SyncAccountsToCloud() error {
	if !IsPremiumEnabled() {
		return fmt.Errorf("premium features not enabled")
	}

	client, err := GetAPIClient()
	if err != nil {
		return err
	}

	// Load local accounts
	accounts, err := config.GetAllAccounts()
	if err != nil {
		return err
	}

	// Convert to JSON
	accountsJSON, err := json.Marshal(accounts)
	if err != nil {
		return err
	}

	// Sync to cloud with shorter timeout for faster failure
	// Create a client copy with 5s timeout for sync operations
	syncClient := &Client{
		BaseURL:      client.BaseURL,
		HTTPClient:   &http.Client{Timeout: 5 * time.Second}, // Short timeout for UI responsiveness
		Token:        client.Token,
		RefreshToken: client.RefreshToken,
	}

	var accountsData *AccountsData

	// Only 1 attempt - if it fails, queue immediately for background retry
	accountsData, err = syncClient.UpdateAccounts(accountsJSON)
	if err != nil {
		// Check if error is subscription-related - don't queue for retry in that case
		errStr := err.Error()
		if isSubscriptionError(errStr) {
			return fmt.Errorf("sync failed: %v", err)
		}
		// Queue immediately for background retry instead of blocking
		queue := GetSyncQueue()
		queue.QueueSync("accounts", accounts)
		return fmt.Errorf("sync failed: %v (queued for background retry)", err)
	}

	// Update sync timestamp and stats
	cfg, err := GetPremiumConfig()
	if err != nil {
		return err
	}
	now := time.Now()
	cfg.LastAccountsSync = now
	cfg.LastSyncTime = now
	cfg.AccountsSynced = len(accounts)
	// Update local version from cloud response
	if accountsData != nil {
		cfg.LocalAccountsVersion = accountsData.Version
	}
	return SavePremiumConfig(cfg)
}

// SyncAccountsFromCloud syncs cloud accounts to local
func SyncAccountsFromCloud() ([]config.Account, error) {
	if !IsPremiumEnabled() {
		return nil, fmt.Errorf("premium features not enabled")
	}

	client, err := GetAPIClient()
	if err != nil {
		return nil, err
	}

	// Get from cloud
	accountsData, err := client.GetAccounts()
	if err != nil {
		return nil, err
	}

	// Parse accounts
	var accounts []config.Account
	if err := json.Unmarshal(accountsData.Accounts, &accounts); err != nil {
		return nil, err
	}

	return accounts, nil
}

// SyncUnsubscribedToCloud syncs local unsubscribed newsletters to cloud with retry logic
func SyncUnsubscribedToCloud() error {
	if !IsPremiumEnabled() {
		return fmt.Errorf("premium features not enabled")
	}

	client, err := GetAPIClient()
	if err != nil {
		return err
	}

	// Load local unsubscribed
	store, err := config.LoadUnsubscribed()
	if err != nil {
		return err
	}

	// Marshal to JSON
	data, err := json.Marshal(store)
	if err != nil {
		return err
	}

	// Sync to cloud with shorter timeout for faster failure
	// Create a client copy with 5s timeout for sync operations
	syncClient := &Client{
		BaseURL:      client.BaseURL,
		HTTPClient:   &http.Client{Timeout: 5 * time.Second}, // Short timeout for UI responsiveness
		Token:        client.Token,
		RefreshToken: client.RefreshToken,
	}

	var unsubscribedData *UnsubscribedData

	unsubscribedData, err = syncClient.UpdateUnsubscribed(data)
	if err != nil {
		// Check if error is subscription-related - don't queue for retry in that case
		errStr := err.Error()
		if isSubscriptionError(errStr) {
			return fmt.Errorf("sync failed: %v", err)
		}
		// Queue immediately for background retry instead of blocking
		queue := GetSyncQueue()
		queue.QueueSync("unsubscribed", store)
		return fmt.Errorf("sync failed: %v (queued for background retry)", err)
	}

	// Update sync timestamp and stats
	cfg, err := GetPremiumConfig()
	if err != nil {
		return err
	}
	cfg.LastUnsubSync = time.Now()
	cfg.LastSyncTime = time.Now()
	cfg.UnsubscribedCount = len(store.Newsletters)
	// Update local version from cloud response
	if unsubscribedData != nil {
		cfg.LocalUnsubscribedVersion = unsubscribedData.Version
	}
	return SavePremiumConfig(cfg)
}

// SyncUnsubscribedFromCloud pulls unsubscribed newsletters from the cloud
func SyncUnsubscribedFromCloud() (*config.UnsubscribedStore, error) {
	if !IsPremiumEnabled() {
		return nil, fmt.Errorf("premium features not enabled")
	}

	client, err := GetAPIClient()
	if err != nil {
		return nil, err
	}

	cloudData, err := client.GetUnsubscribed()
	if err != nil {
		return nil, err
	}

	// Update local version from cloud
	cfg, err := GetPremiumConfig()
	if err == nil && cloudData != nil {
		cfg.LocalUnsubscribedVersion = cloudData.Version
		SavePremiumConfig(cfg) // Best effort, don't fail if this errors
	}

	var store config.UnsubscribedStore
	if err := json.Unmarshal(cloudData.Unsubscribed, &store); err != nil {
		return nil, err
	}

	return &store, nil
}

// CheckLicense validates license and returns tier/features
func CheckLicense(licenseKey string) (*LicenseResponse, error) {
	if !IsPremiumEnabled() {
		return nil, fmt.Errorf("premium features not enabled")
	}

	client, err := GetAPIClient()
	if err != nil {
		return nil, err
	}

	return client.ValidateLicense(licenseKey)
}

// GetLicenseFeatures returns available features for current user
func GetLicenseFeatures() (map[string]interface{}, error) {
	if !IsPremiumEnabled() {
		return nil, fmt.Errorf("premium features not enabled")
	}

	client, err := GetAPIClient()
	if err != nil {
		return nil, err
	}

	return client.GetLicenseFeatures()
}

// HasFeature checks if a specific premium feature is available
func HasFeature(featureName string) bool {
	features, err := GetLicenseFeatures()
	if err != nil {
		return false
	}

	// Check tier - must not be "free"
	tier, _ := features["tier"].(string)
	if tier == "" || tier == "free" {
		return false
	}

	featureList, ok := features["features"].([]interface{})
	if !ok {
		return false
	}

	for _, f := range featureList {
		if fStr, ok := f.(string); ok && fStr == featureName {
			return true
		}
	}

	return false
}

// HasActiveSubscription checks if user has an active subscription
// This is a convenience function that checks features and ensures tier is not free
func HasActiveSubscription() bool {
	features, err := GetLicenseFeatures()
	if err != nil {
		return false
	}

	tier, _ := features["tier"].(string)
	return tier != "" && tier != "free"
}

// GetMaxAccountsForTier returns the maximum number of accounts allowed for a subscription tier
func GetMaxAccountsForTier(tier string) int {
	switch tier {
	case "starter":
		return 3
	case "pro":
		return 10
	case "enterprise":
		return 50
	default:
		// Free tier: 1 account (first account only)
		return 1
	}
}

// CanAddAccount checks if user can add another account based on their subscription tier and current account count
// Returns (canAdd, reason) - reason is empty if canAdd is true
func CanAddAccount(currentAccountCount int) (bool, string) {
	// First account is always free (no subscription required)
	if currentAccountCount == 0 {
		return true, ""
	}

	// Get subscription tier
	features, err := GetLicenseFeatures()
	if err != nil {
		// If can't get features, assume no subscription - only first account allowed
		if currentAccountCount >= 1 {
			return false, "Premium subscription required for multiple accounts"
		}
		return true, ""
	}

	tier, _ := features["tier"].(string)
	if tier == "" || tier == "free" {
		// No active subscription - only first account allowed
		if currentAccountCount >= 1 {
			return false, "Premium subscription required for multiple accounts"
		}
		return true, ""
	}

	// Check account limit for tier
	maxAccounts := GetMaxAccountsForTier(tier)
	if currentAccountCount >= maxAccounts {
		return false, fmt.Sprintf("Account limit exceeded: Your %s plan allows up to %d accounts. Please upgrade your subscription.", tier, maxAccounts)
	}

	return true, ""
}

// DeleteAccountFromCloud deletes all user data from the cloud API (GDPR compliance)
func DeleteAccountFromCloud() error {
	if !IsPremiumEnabled() {
		return fmt.Errorf("premium features not enabled")
	}

	client, err := GetAPIClient()
	if err != nil {
		return err
	}

	return client.DeleteAccount()
}

var globalAnalyticsCollector *AnalyticsCollector
var analyticsCollectorMu sync.Mutex

// GetAnalyticsCollector returns the global analytics collector instance
func GetAnalyticsCollector() (*AnalyticsCollector, error) {
	analyticsCollectorMu.Lock()
	defer analyticsCollectorMu.Unlock()

	if globalAnalyticsCollector != nil {
		return globalAnalyticsCollector, nil
	}

	// Check if premium is enabled and analytics is enabled
	cfg, err := GetPremiumConfig()
	if err != nil {
		return NewAnalyticsCollector(nil, false), fmt.Errorf("failed to get premium config: %w", err)
	}

	if !cfg.Enabled {
		return NewAnalyticsCollector(nil, false), nil
	}

	if !cfg.AnalyticsEnabled {
		return NewAnalyticsCollector(nil, false), nil
	}

	client, err := GetAPIClient()
	if err != nil {
		// Return disabled collector if API client can't be created
		return NewAnalyticsCollector(nil, false), fmt.Errorf("failed to get API client: %w", err)
	}

	globalAnalyticsCollector = NewAnalyticsCollector(client, true)
	return globalAnalyticsCollector, nil
}

// ResetAnalyticsCollector resets the global collector (useful for testing or re-initialization)
func ResetAnalyticsCollector() {
	analyticsCollectorMu.Lock()
	defer analyticsCollectorMu.Unlock()
	if globalAnalyticsCollector != nil {
		globalAnalyticsCollector.Disable()
		globalAnalyticsCollector = nil
	}
}

// GetDashboardURL returns the analytics dashboard URL with authentication token
func GetDashboardURL() string {
	cfg, err := GetPremiumConfig()
	if err != nil || !cfg.Enabled || cfg.Token == "" {
		return ""
	}

	apiURL := cfg.APIURL
	if apiURL == "" {
		apiURL = "https://api.newsletter-cli.apps.paas-01.pulseflow.cloud"
	}

	// Remove trailing slash if present
	if strings.HasSuffix(apiURL, "/") {
		apiURL = strings.TrimSuffix(apiURL, "/")
	}

	// Replace /api/v1 with empty string if present, otherwise append /dashboard
	dashboardBase := apiURL
	if strings.Contains(apiURL, "/api/v1") {
		dashboardBase = strings.Replace(apiURL, "/api/v1", "", 1)
	}
	if !strings.HasSuffix(dashboardBase, "/") {
		dashboardBase += "/"
	}

	// URL encode the token to ensure it's properly formatted
	// This prevents issues with special characters and ensures the full token is preserved
	return fmt.Sprintf("%sdashboard?token=%s", dashboardBase, url.QueryEscape(cfg.Token))
}

// isSubscriptionError checks if an error is related to subscription requirements
// This includes account limit errors as they require subscription upgrades
func isSubscriptionError(errStr string) bool {
	errLower := strings.ToLower(errStr)
	return strings.Contains(errLower, "403") ||
		strings.Contains(errLower, "forbidden") ||
		strings.Contains(errLower, "subscription") ||
		strings.Contains(errLower, "active subscription required") ||
		strings.Contains(errLower, "account limit") ||
		strings.Contains(errLower, "upgrade your subscription")
}
