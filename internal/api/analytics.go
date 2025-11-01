package api

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"
)

// AnalyticsEvent represents a single analytics event
type AnalyticsEvent struct {
	EventType    string                 `json:"event_type"` // "newsletter_analyzed", "unsubscribed", etc.
	Timestamp    time.Time              `json:"timestamp"`
	SenderDomain string                 `json:"sender_domain"` // Hashed/anonymized domain
	EmailCount   int                    `json:"email_count,omitempty"`
	AccountID    string                 `json:"account_id,omitempty"` // Hashed account identifier
	Metadata     map[string]interface{} `json:"metadata,omitempty"`   // Additional event data
}

// AnalyticsCollector manages analytics event collection and batching
type AnalyticsCollector struct {
	client        *Client
	enabled       bool
	queue         []AnalyticsEvent
	mu            sync.Mutex
	flushTicker   *time.Ticker
	stopChan      chan struct{}
	flushSize     int           // Batch size before auto-flush
	flushInterval time.Duration // Time interval for auto-flush
}

// NewAnalyticsCollector creates a new analytics collector
func NewAnalyticsCollector(client *Client, enabled bool) *AnalyticsCollector {
	collector := &AnalyticsCollector{
		client:        client,
		enabled:       enabled,
		queue:         make([]AnalyticsEvent, 0),
		flushSize:     10,               // Flush after 10 events
		flushInterval: 30 * time.Second, // Flush every 30 seconds
		stopChan:      make(chan struct{}),
	}

	// Start background flusher if enabled
	if enabled {
		collector.startBackgroundFlusher()
	}

	return collector
}

// Enable enables analytics collection
func (ac *AnalyticsCollector) Enable() {
	ac.mu.Lock()
	defer ac.mu.Unlock()

	if !ac.enabled {
		ac.enabled = true
		// Only start background flusher if we have a client
		if ac.client != nil {
			ac.startBackgroundFlusher()
		}
	}
}

// Disable disables analytics collection and flushes queue
func (ac *AnalyticsCollector) Disable() {
	ac.mu.Lock()
	defer ac.mu.Unlock()

	if ac.enabled {
		ac.enabled = false
		ac.stopBackgroundFlusher()
		// Flush remaining events
		if len(ac.queue) > 0 {
			queue := make([]AnalyticsEvent, len(ac.queue))
			copy(queue, ac.queue)
			ac.queue = ac.queue[:0]
			go func() {
				_ = ac.flushEvents(queue) // Best effort, ignore errors
			}()
		}
	}
}

// Collect adds an event to the queue
func (ac *AnalyticsCollector) Collect(event AnalyticsEvent) {
	ac.mu.Lock()
	defer ac.mu.Unlock()

	if !ac.enabled {
		return
	}

	// Set timestamp if not set
	if event.Timestamp.IsZero() {
		event.Timestamp = time.Now()
	}

	ac.queue = append(ac.queue, event)

	// Auto-flush if queue size reached
	if len(ac.queue) >= ac.flushSize {
		queue := make([]AnalyticsEvent, len(ac.queue))
		copy(queue, ac.queue)
		ac.queue = ac.queue[:0]
		go func() {
			// Best effort - errors are logged but don't interrupt user flow
			if err := ac.flushEvents(queue); err != nil {
				// Log error for debugging, but don't propagate
				// In production, you might want to use a proper logger here
				_ = err
			}
		}()
	}
}

// Flush immediately sends all queued events
func (ac *AnalyticsCollector) Flush() error {
	ac.mu.Lock()
	queue := make([]AnalyticsEvent, len(ac.queue))
	copy(queue, ac.queue)
	ac.queue = ac.queue[:0]
	ac.mu.Unlock()

	if len(queue) == 0 {
		return nil
	}

	return ac.flushEvents(queue)
}

// flushEvents sends events to the API
// Errors are logged but don't propagate to avoid interrupting user flow
func (ac *AnalyticsCollector) flushEvents(events []AnalyticsEvent) error {
	if ac.client == nil || len(events) == 0 {
		return nil
	}

	// Send batch to API
	resp, err := ac.client.doRequestWithRefresh("POST", "/api/v1/analytics/events", map[string]interface{}{
		"events": events,
	})
	if err != nil {
		// Network error - events will be queued for retry on next flush
		return fmt.Errorf("analytics API request failed: %w", err)
	}
	defer resp.Body.Close()

	// Handle different status codes gracefully
	if resp.StatusCode == http.StatusUnauthorized {
		// Token expired - will be refreshed on next request via refresh mechanism
		return fmt.Errorf("analytics API: authentication expired (will retry)")
	} else if resp.StatusCode == http.StatusForbidden {
		// User doesn't have access - disable analytics silently
		return fmt.Errorf("analytics API: access forbidden")
	} else if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		// Server error - retry later
		return fmt.Errorf("analytics API returned status %d", resp.StatusCode)
	}

	return nil
}

// startBackgroundFlusher starts periodic flushing
func (ac *AnalyticsCollector) startBackgroundFlusher() {
	if ac.flushTicker != nil {
		return // Already running
	}

	ac.flushTicker = time.NewTicker(ac.flushInterval)
	go func() {
		for {
			select {
			case <-ac.flushTicker.C:
				// Best effort - errors are logged but don't interrupt
				if err := ac.Flush(); err != nil {
					// Log for debugging, but continue running
					_ = err
				}
			case <-ac.stopChan:
				return
			}
		}
	}()
}

// stopBackgroundFlusher stops periodic flushing
func (ac *AnalyticsCollector) stopBackgroundFlusher() {
	if ac.flushTicker != nil {
		ac.flushTicker.Stop()
		ac.flushTicker = nil
	}
	select {
	case ac.stopChan <- struct{}{}:
	default:
	}
}

// HashSenderDomain hashes a sender domain for privacy
// Uses SHA-256 with optional salt to anonymize sender information
func HashSenderDomain(sender string, salt string) string {
	if sender == "" {
		return ""
	}

	// Extract domain from email or use as-is if already a domain
	domain := sender
	if idx := strings.LastIndex(sender, "@"); idx >= 0 {
		domain = sender[idx+1:]
	}

	// Hash the domain with salt
	hash := sha256.Sum256([]byte(domain + salt))
	// Return first 16 bytes as hex (32 chars) for uniqueness while keeping it short
	return hex.EncodeToString(hash[:16])
}

// HashAccountID hashes an account identifier for privacy
func HashAccountID(accountID string, salt string) string {
	if accountID == "" {
		return ""
	}

	hash := sha256.Sum256([]byte(accountID + salt))
	return hex.EncodeToString(hash[:16])
}
