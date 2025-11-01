package api

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/loickal/newsletter-cli/internal/config"
)

// PendingSync represents a sync operation that failed and is queued for retry
type PendingSync struct {
	Type      string          `json:"type"`      // "accounts" or "unsubscribed"
	Data      json.RawMessage `json:"data"`      // The data to sync
	QueuedAt  time.Time       `json:"queued_at"` // When it was queued
	Retries   int             `json:"retries"`   // Number of retry attempts
	LastError string          `json:"last_error,omitempty"`
}

// SyncQueue manages pending sync operations
type SyncQueue struct {
	mu      sync.Mutex
	pending []PendingSync
}

var globalSyncQueue *SyncQueue
var syncQueueOnce sync.Once

// GetSyncQueue returns the global sync queue instance
func GetSyncQueue() *SyncQueue {
	syncQueueOnce.Do(func() {
		globalSyncQueue = &SyncQueue{
			pending: []PendingSync{},
		}
		// Load pending syncs from disk
		globalSyncQueue.load()
	})
	return globalSyncQueue
}

// QueueSync adds a sync operation to the queue
func (sq *SyncQueue) QueueSync(syncType string, data interface{}) error {
	sq.mu.Lock()
	defer sq.mu.Unlock()

	dataJSON, err := json.Marshal(data)
	if err != nil {
		return err
	}

	pending := PendingSync{
		Type:     syncType,
		Data:     dataJSON,
		QueuedAt: time.Now(),
		Retries:  0,
	}

	sq.pending = append(sq.pending, pending)
	return sq.save()
}

// ProcessQueue processes pending sync operations with retry logic
func (sq *SyncQueue) ProcessQueue() error {
	if !IsPremiumEnabled() {
		return nil
	}

	sq.mu.Lock()
	defer sq.mu.Unlock()

	if len(sq.pending) == 0 {
		return nil
	}

	var remaining []PendingSync
	var lastErr error

	for _, pending := range sq.pending {
		var err error

		switch pending.Type {
		case "accounts":
			var accounts []config.Account
			if err := json.Unmarshal(pending.Data, &accounts); err == nil {
				// Try to sync accounts
				err = syncAccountsWithRetry(accounts, pending.Retries)
			}
		case "unsubscribed":
			var unsubscribed *config.UnsubscribedStore
			if err := json.Unmarshal(pending.Data, &unsubscribed); err == nil {
				// Try to sync unsubscribed
				err = syncUnsubscribedWithRetry(unsubscribed, pending.Retries)
			}
		}

		if err != nil {
			// Check if error is subscription-related - don't retry those
			errStr := err.Error()
			if isSubscriptionError(errStr) {
				// Skip subscription errors - don't retry or keep in queue
				lastErr = err
				continue
			}

			pending.Retries++
			pending.LastError = errStr

			// Exponential backoff: max 3 retries
			if pending.Retries < 3 {
				remaining = append(remaining, pending)
			} else {
				// Max retries reached - keep in queue but mark as failed
				remaining = append(remaining, pending)
			}
			lastErr = err
		}
		// Success - don't add back to queue
	}

	sq.pending = remaining
	sq.save()

	return lastErr
}

// syncAccountsWithRetry syncs accounts with exponential backoff
func syncAccountsWithRetry(accounts []config.Account, retries int) error {
	// Calculate delay: 1s, 2s, 4s
	delay := time.Duration(1<<uint(retries)) * time.Second
	if delay > 5*time.Second {
		delay = 5 * time.Second // Cap at 5 seconds
	}

	time.Sleep(delay)

	client, err := GetAPIClient()
	if err != nil {
		return err
	}

	accountsJSON, err := json.Marshal(accounts)
	if err != nil {
		return err
	}

	_, err = client.UpdateAccounts(accountsJSON)
	return err
}

// syncUnsubscribedWithRetry syncs unsubscribed with exponential backoff
func syncUnsubscribedWithRetry(unsubscribed *config.UnsubscribedStore, retries int) error {
	// Calculate delay: 1s, 2s, 4s
	delay := time.Duration(1<<uint(retries)) * time.Second
	if delay > 5*time.Second {
		delay = 5 * time.Second // Cap at 5 seconds
	}

	time.Sleep(delay)

	client, err := GetAPIClient()
	if err != nil {
		return err
	}

	unsubscribedJSON, err := json.Marshal(unsubscribed.Newsletters)
	if err != nil {
		return err
	}

	_, err = client.UpdateUnsubscribed(unsubscribedJSON)
	return err
}

// GetPendingCount returns the number of pending sync operations
func (sq *SyncQueue) GetPendingCount() int {
	sq.mu.Lock()
	defer sq.mu.Unlock()
	return len(sq.pending)
}

// Clear removes all pending syncs
func (sq *SyncQueue) Clear() error {
	sq.mu.Lock()
	defer sq.mu.Unlock()
	sq.pending = []PendingSync{}
	return sq.save()
}

// save persists the queue to disk
func (sq *SyncQueue) save() error {
	configDir, err := config.ConfigDir()
	if err != nil {
		return err
	}

	queuePath := filepath.Join(configDir, "sync_queue.json")
	data, err := json.MarshalIndent(sq.pending, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(queuePath, data, 0600)
}

// load loads the queue from disk
func (sq *SyncQueue) load() {
	configDir, err := config.ConfigDir()
	if err != nil {
		return
	}

	queuePath := filepath.Join(configDir, "sync_queue.json")
	data, err := os.ReadFile(queuePath)
	if err != nil {
		return // No queue file exists yet
	}

	json.Unmarshal(data, &sq.pending)
}
