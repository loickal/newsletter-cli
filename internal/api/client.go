package api

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"
)

type Client struct {
	BaseURL        string
	HTTPClient     *http.Client
	Token          string
	RefreshToken   string
	APISecret      string // Optional HMAC signing secret
	OnTokenRefresh func(newToken, newRefreshToken string) error // Callback to save new tokens
}

type AuthResponse struct {
	Token        string `json:"token"`
	RefreshToken string `json:"refresh_token"`
	ExpiresAt    int64  `json:"expires_at"`
}

type RegisterRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type LoginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type ConfigData struct {
	Config  json.RawMessage `json:"config"`
	Version int64           `json:"version"`
}

type AccountsData struct {
	Accounts json.RawMessage `json:"accounts"`
	Version  int64           `json:"version"`
}

type LicenseResponse struct {
	Valid     bool     `json:"valid"`
	Tier      string   `json:"tier"`
	Features  []string `json:"features"`
	ExpiresAt *int64   `json:"expires_at,omitempty"`
}

type UnsubscribedData struct {
	Unsubscribed json.RawMessage `json:"unsubscribed"`
	Version      int64           `json:"version"`
}

type Plan struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	Amount   int64  `json:"amount"`   // in cents
	Interval string `json:"interval"` // "month" or "year"
}

type CheckoutSessionResponse struct {
	CheckoutURL string `json:"checkout_url"`
	SessionID   string `json:"session_id"`
}

type PortalSessionResponse struct {
	URL string `json:"url"`
}

type Subscription struct {
	Tier                 string     `json:"tier"`
	Status               string     `json:"status"`
	CurrentPeriodEnd     *time.Time `json:"current_period_end,omitempty"`
	CanceledAt           *time.Time `json:"canceled_at,omitempty"` // When subscription was canceled
	StripeCustomerID     string     `json:"stripe_customer_id,omitempty"`
	StripeSubscriptionID string     `json:"stripe_subscription_id,omitempty"`
}

type APIError struct {
	Message string
	Code    int
}

func (e *APIError) Error() string {
	return fmt.Sprintf("API error (%d): %s", e.Code, e.Message)
}

func NewClient(baseURL string) *Client {
	if baseURL == "" {
		baseURL = "https://api.newsletter-cli.apps.paas-01.pulseflow.cloud"
	}

	return &Client{
		BaseURL: baseURL,
		HTTPClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

func (c *Client) SetToken(token string) {
	c.Token = token
}

func (c *Client) doRequest(method, path string, body interface{}) (*http.Response, error) {
	var bodyBytes []byte
	var reqBody io.Reader
	
	if body != nil {
		var err error
		bodyBytes, err = json.Marshal(body)
		if err != nil {
			return nil, err
		}
		reqBody = bytes.NewBuffer(bodyBytes)
	}

	req, err := http.NewRequest(method, c.BaseURL+path, reqBody)
	if err != nil {
		return nil, err
	}

	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	// Use HMAC signing if API secret is set, otherwise use JWT
	if c.APISecret != "" {
		// Generate timestamp
		timestamp := time.Now().UTC().Format(time.RFC3339)
		
		// Build message to sign: method + path + timestamp + body
		message := fmt.Sprintf("%s\n%s\n%s\n%s", method, path, timestamp, string(bodyBytes))
		
		// Calculate HMAC signature
		mac := hmac.New(sha256.New, []byte(c.APISecret))
		mac.Write([]byte(message))
		signature := hex.EncodeToString(mac.Sum(nil))
		
		// Set HMAC headers
		req.Header.Set("X-API-Key", c.APISecret)
		req.Header.Set("X-API-Timestamp", timestamp)
		req.Header.Set("X-API-Signature", signature)
	} else if c.Token != "" {
		// Fall back to JWT if no API secret
		req.Header.Set("Authorization", "Bearer "+c.Token)
	}

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, err
	}

	return resp, nil
}

func (c *Client) Register(email, password string) (*AuthResponse, error) {
	resp, err := c.doRequest("POST", "/api/v1/auth/register", RegisterRequest{
		Email:    email,
		Password: password,
	})
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, &APIError{
			Message: string(body),
			Code:    resp.StatusCode,
		}
	}

	var authResp AuthResponse
	if err := json.NewDecoder(resp.Body).Decode(&authResp); err != nil {
		return nil, err
	}

	c.Token = authResp.Token
	return &authResp, nil
}

func (c *Client) Login(email, password string) (*AuthResponse, error) {
	resp, err := c.doRequest("POST", "/api/v1/auth/login", LoginRequest{
		Email:    email,
		Password: password,
	})
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, &APIError{
			Message: string(body),
			Code:    resp.StatusCode,
		}
	}

	var authResp AuthResponse
	if err := json.NewDecoder(resp.Body).Decode(&authResp); err != nil {
		return nil, err
	}

	c.Token = authResp.Token
	return &authResp, nil
}

func (c *Client) GetConfig() (*ConfigData, error) {
	resp, err := c.doRequestWithRefresh("GET", "/api/v1/sync/config", nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, &APIError{
			Message: string(body),
			Code:    resp.StatusCode,
		}
	}

	var configData ConfigData
	if err := json.NewDecoder(resp.Body).Decode(&configData); err != nil {
		return nil, err
	}

	return &configData, nil
}

func (c *Client) UpdateConfig(config json.RawMessage) (*ConfigData, error) {
	resp, err := c.doRequestWithRefresh("POST", "/api/v1/sync/config", ConfigData{
		Config: config,
	})
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, &APIError{
			Message: string(body),
			Code:    resp.StatusCode,
		}
	}

	var configData ConfigData
	if err := json.NewDecoder(resp.Body).Decode(&configData); err != nil {
		return nil, err
	}

	return &configData, nil
}

func (c *Client) GetAccounts() (*AccountsData, error) {
	resp, err := c.doRequestWithRefresh("GET", "/api/v1/sync/accounts", nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, &APIError{
			Message: string(body),
			Code:    resp.StatusCode,
		}
	}

	var accountsData AccountsData
	if err := json.NewDecoder(resp.Body).Decode(&accountsData); err != nil {
		return nil, err
	}

	return &accountsData, nil
}

func (c *Client) UpdateAccounts(accounts json.RawMessage) (*AccountsData, error) {
	resp, err := c.doRequestWithRefresh("POST", "/api/v1/sync/accounts", AccountsData{
		Accounts: accounts,
	})
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, &APIError{
			Message: string(body),
			Code:    resp.StatusCode,
		}
	}

	var accountsData AccountsData
	if err := json.NewDecoder(resp.Body).Decode(&accountsData); err != nil {
		return nil, err
	}

	return &accountsData, nil
}

func (c *Client) ValidateLicense(licenseKey string) (*LicenseResponse, error) {
	resp, err := c.doRequestWithRefresh("GET", fmt.Sprintf("/api/v1/license/validate?key=%s", licenseKey), nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, &APIError{
			Message: string(body),
			Code:    resp.StatusCode,
		}
	}

	var licenseResp LicenseResponse
	if err := json.NewDecoder(resp.Body).Decode(&licenseResp); err != nil {
		return nil, err
	}

	return &licenseResp, nil
}

// doRequestWithRefresh performs a request and automatically refreshes token on 401
func (c *Client) doRequestWithRefresh(method, path string, body interface{}) (*http.Response, error) {
	resp, err := c.doRequest(method, path, body)
	if err != nil {
		return nil, err
	}

	// If we get 401 and have a refresh token, try to refresh
	if resp.StatusCode == http.StatusUnauthorized && c.RefreshToken != "" {
		resp.Body.Close() // Close the 401 response

		// Refresh token
		if err := c.refreshTokenIfNeeded(); err != nil {
			return nil, fmt.Errorf("token expired and refresh failed: %w", err)
		}

		// Retry the request with new token
		return c.doRequest(method, path, body)
	}

	return resp, nil
}

func (c *Client) GetLicenseFeatures() (map[string]interface{}, error) {
	resp, err := c.doRequestWithRefresh("GET", "/api/v1/license/features", nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, &APIError{
			Message: string(body),
			Code:    resp.StatusCode,
		}
	}

	var features map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&features); err != nil {
		return nil, err
	}

	return features, nil
}

// GetPlans returns available subscription plans
func (c *Client) GetPlans() ([]Plan, error) {
	resp, err := c.doRequestWithRefresh("GET", "/api/v1/subscriptions/plans", nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, &APIError{
			Message: string(body),
			Code:    resp.StatusCode,
		}
	}

	var plans []Plan
	if err := json.NewDecoder(resp.Body).Decode(&plans); err != nil {
		return nil, err
	}

	return plans, nil
}

// CreateCheckoutSession creates a Stripe Checkout session for subscription
func (c *Client) CreateCheckoutSession(planID string) (*CheckoutSessionResponse, error) {
	reqBody := map[string]string{"plan": planID}
	resp, err := c.doRequestWithRefresh("POST", "/api/v1/subscriptions/create-checkout", reqBody)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, &APIError{
			Message: string(body),
			Code:    resp.StatusCode,
		}
	}

	var session CheckoutSessionResponse
	if err := json.NewDecoder(resp.Body).Decode(&session); err != nil {
		return nil, err
	}

	return &session, nil
}

// CreatePortalSession creates a Stripe Customer Portal session for subscription management
func (c *Client) CreatePortalSession() (*PortalSessionResponse, error) {
	resp, err := c.doRequestWithRefresh("POST", "/api/v1/subscriptions/create-portal", nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, &APIError{
			Message: string(body),
			Code:    resp.StatusCode,
		}
	}

	var portal PortalSessionResponse
	if err := json.NewDecoder(resp.Body).Decode(&portal); err != nil {
		return nil, err
	}

	return &portal, nil
}

// GetCurrentSubscription returns user's current subscription
func (c *Client) GetCurrentSubscription() (*Subscription, error) {
	resp, err := c.doRequestWithRefresh("GET", "/api/v1/subscriptions/current", nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, &APIError{
			Message: string(body),
			Code:    resp.StatusCode,
		}
	}

	var subscription Subscription
	if err := json.NewDecoder(resp.Body).Decode(&subscription); err != nil {
		return nil, err
	}

	return &subscription, nil
}

// APISecretResponse represents the response from generating an API secret
type APISecretResponse struct {
	APISecret string `json:"api_secret"`
	Message   string `json:"message"`
}

// APISecretStatusResponse represents the status of API secret
type APISecretStatusResponse struct {
	HasSecret bool `json:"has_secret"`
}

// GenerateAPISecret generates a new API secret for HMAC signing
func (c *Client) GenerateAPISecret() (*APISecretResponse, error) {
	resp, err := c.doRequestWithRefresh("POST", "/api/v1/api-secret/generate", nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, &APIError{
			Message: string(body),
			Code:    resp.StatusCode,
		}
	}

	var secretResp APISecretResponse
	if err := json.NewDecoder(resp.Body).Decode(&secretResp); err != nil {
		return nil, err
	}

	return &secretResp, nil
}

// GetAPISecretStatus checks if user has an API secret configured
func (c *Client) GetAPISecretStatus() (*APISecretStatusResponse, error) {
	resp, err := c.doRequestWithRefresh("GET", "/api/v1/api-secret/status", nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, &APIError{
			Message: string(body),
			Code:    resp.StatusCode,
		}
	}

	var statusResp APISecretStatusResponse
	if err := json.NewDecoder(resp.Body).Decode(&statusResp); err != nil {
		return nil, err
	}

	return &statusResp, nil
}

// UsageStats represents usage statistics
type UsageStats struct {
	TotalRequests   int            `json:"total_requests"`
	UniqueEndpoints int            `json:"unique_endpoints"`
	HourlyRequests  map[string]int `json:"hourly_requests"`
}

// EndpointStats represents detailed endpoint statistics
type EndpointStats struct {
	Endpoint       string  `json:"endpoint"`
	Method         string  `json:"method"`
	RequestCount   int     `json:"request_count"`
	AvgRequestSize float64 `json:"avg_request_size"`
	ErrorCount     int     `json:"error_count"`
}

// DetailedUsageStats represents detailed usage statistics
type DetailedUsageStats struct {
	Since     time.Time      `json:"since"`
	Endpoints []EndpointStats `json:"endpoints"`
}

// GetUsageStats returns usage statistics for the authenticated user
func (c *Client) GetUsageStats(since time.Time) (*UsageStats, error) {
	// Format since as RFC3339
	sinceStr := since.Format(time.RFC3339)
	
	resp, err := c.doRequestWithRefresh("GET", fmt.Sprintf("/api/v1/usage/stats?since=%s", url.QueryEscape(sinceStr)), nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, &APIError{
			Message: string(body),
			Code:    resp.StatusCode,
		}
	}

	var stats UsageStats
	if err := json.NewDecoder(resp.Body).Decode(&stats); err != nil {
		return nil, err
	}

	return &stats, nil
}

// GetDetailedUsage returns detailed usage breakdown
func (c *Client) GetDetailedUsage(since time.Time) (*DetailedUsageStats, error) {
	// Format since as RFC3339
	sinceStr := since.Format(time.RFC3339)
	
	resp, err := c.doRequestWithRefresh("GET", fmt.Sprintf("/api/v1/usage/detailed?since=%s", url.QueryEscape(sinceStr)), nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, &APIError{
			Message: string(body),
			Code:    resp.StatusCode,
		}
	}

	var stats DetailedUsageStats
	if err := json.NewDecoder(resp.Body).Decode(&stats); err != nil {
		return nil, err
	}

	return &stats, nil
}

// NewsletterCategory represents a newsletter category with confidence
type NewsletterCategory struct {
	Category   string   `json:"category"`
	Confidence float64  `json:"confidence"`
	Tags       []string `json:"tags,omitempty"`
}

// EnrichNewsletter represents enriched newsletter data from API
type EnrichNewsletter struct {
	Sender       string             `json:"sender"`
	Category     NewsletterCategory `json:"category"`
	QualityScore int                `json:"quality_score"`
}

// EnrichNewslettersRequest represents the request for enriching newsletters
type EnrichNewslettersRequest struct {
	Newsletters []EnrichNewsletterInput `json:"newsletters"`
}

// EnrichNewsletterInput represents input for enrichment
type EnrichNewsletterInput struct {
	Sender         string `json:"sender"`
	EmailCount     int    `json:"email_count"`
	HasUnsubscribe bool   `json:"has_unsubscribe"`
}

// EnrichNewslettersResponse represents the response from enrichment API
type EnrichNewslettersResponse struct {
	Enriched []EnrichNewsletter `json:"enriched"`
}

// EnrichNewsletters enriches newsletters with categorization and quality scores
func (c *Client) EnrichNewsletters(newsletters []EnrichNewsletterInput) (*EnrichNewslettersResponse, error) {
	reqBody := EnrichNewslettersRequest{Newsletters: newsletters}
	resp, err := c.doRequestWithRefresh("POST", "/api/v1/premium/enrich-batch", reqBody)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, &APIError{
			Message: string(body),
			Code:    resp.StatusCode,
		}
	}

	var response EnrichNewslettersResponse
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return nil, err
	}

	return &response, nil
}

func (c *Client) GetUnsubscribed() (*UnsubscribedData, error) {
	resp, err := c.doRequestWithRefresh("GET", "/api/v1/sync/unsubscribed", nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, &APIError{
			Message: string(body),
			Code:    resp.StatusCode,
		}
	}

	var unsubscribedData UnsubscribedData
	if err := json.NewDecoder(resp.Body).Decode(&unsubscribedData); err != nil {
		return nil, err
	}

	return &unsubscribedData, nil
}

func (c *Client) UpdateUnsubscribed(unsubscribed json.RawMessage) (*UnsubscribedData, error) {
	resp, err := c.doRequestWithRefresh("POST", "/api/v1/sync/unsubscribed", UnsubscribedData{
		Unsubscribed: unsubscribed,
	})
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, &APIError{
			Message: string(body),
			Code:    resp.StatusCode,
		}
	}

	var unsubscribedData UnsubscribedData
	if err := json.NewDecoder(resp.Body).Decode(&unsubscribedData); err != nil {
		return nil, err
	}

	return &unsubscribedData, nil
}

// Refresh refreshes the access token using the refresh token
func (c *Client) Refresh() (*AuthResponse, error) {
	if c.RefreshToken == "" {
		return nil, fmt.Errorf("no refresh token available")
	}

	type RefreshRequest struct {
		RefreshToken string `json:"refresh_token"`
	}

	resp, err := c.doRequest("POST", "/api/v1/auth/refresh", RefreshRequest{
		RefreshToken: c.RefreshToken,
	})
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, &APIError{
			Message: string(body),
			Code:    resp.StatusCode,
		}
	}

	var authResp AuthResponse
	if err := json.NewDecoder(resp.Body).Decode(&authResp); err != nil {
		return nil, err
	}

	c.Token = authResp.Token
	if authResp.RefreshToken != "" {
		c.RefreshToken = authResp.RefreshToken
	}

	// Call callback to save new tokens
	if c.OnTokenRefresh != nil {
		if err := c.OnTokenRefresh(authResp.Token, authResp.RefreshToken); err != nil {
			return nil, fmt.Errorf("failed to save refreshed tokens: %w", err)
		}
	}

	return &authResp, nil
}

// refreshTokenIfNeeded attempts to refresh the token if we have a refresh token
func (c *Client) refreshTokenIfNeeded() error {
	if c.RefreshToken == "" {
		return fmt.Errorf("no refresh token available")
	}

	_, err := c.Refresh()
	return err
}

// DeleteAccount deletes all user data from the cloud (GDPR compliance)
func (c *Client) DeleteAccount() error {
	resp, err := c.doRequestWithRefresh("DELETE", "/api/v1/account", nil)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNoContent {
		body, _ := io.ReadAll(resp.Body)
		return &APIError{
			Message: string(body),
			Code:    resp.StatusCode,
		}
	}

	return nil
}
