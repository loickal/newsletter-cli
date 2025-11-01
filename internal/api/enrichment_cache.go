package api

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// CachedEnrichment represents cached enrichment data
type CachedEnrichment struct {
	Sender       string             `json:"sender"`
	Category     NewsletterCategory `json:"category"`
	QualityScore int                `json:"quality_score"`
	EmailCount   int                `json:"email_count"` // Store count to invalidate if changed
	ExpiresAt    time.Time          `json:"expires_at"`
}

// EnrichmentCache manages cached enrichment data
type EnrichmentCache struct {
	cache     map[string]*CachedEnrichment
	mu        sync.RWMutex
	cacheFile string
	ttl       time.Duration // Time to live for cache entries (24 hours)
}

var (
	globalEnrichmentCache *EnrichmentCache
	cacheOnce             sync.Once
)

// GetEnrichmentCache returns the global enrichment cache instance
func GetEnrichmentCache() *EnrichmentCache {
	cacheOnce.Do(func() {
		var cacheDir string
		homeDir, _ := os.UserHomeDir()
		cacheDir = filepath.Join(homeDir, ".newsletter-cli", ".cache")

		os.MkdirAll(cacheDir, 0755)
		cacheFile := filepath.Join(cacheDir, "enrichment_cache.json")

		globalEnrichmentCache = &EnrichmentCache{
			cache:     make(map[string]*CachedEnrichment),
			cacheFile: cacheFile,
			ttl:       24 * time.Hour,
		}

		// Load existing cache
		globalEnrichmentCache.Load()
	})

	return globalEnrichmentCache
}

// Get retrieves cached enrichment data for a sender
func (ec *EnrichmentCache) Get(sender string, emailCount int) (*CachedEnrichment, bool) {
	ec.mu.RLock()
	defer ec.mu.RUnlock()

	cached, exists := ec.cache[sender]
	if !exists {
		return nil, false
	}

	// Check if expired
	if time.Now().After(cached.ExpiresAt) {
		return nil, false
	}

	// Check if email count changed (invalidate cache)
	if cached.EmailCount != emailCount {
		return nil, false
	}

	return cached, true
}

// Set stores enrichment data in cache
func (ec *EnrichmentCache) Set(sender string, category NewsletterCategory, qualityScore int, emailCount int) {
	ec.mu.Lock()
	defer ec.mu.Unlock()

	ec.cache[sender] = &CachedEnrichment{
		Sender:       sender,
		Category:     category,
		QualityScore: qualityScore,
		EmailCount:   emailCount,
		ExpiresAt:    time.Now().Add(ec.ttl),
	}

	// Persist to disk
	go ec.Save()
}

// Load loads cache from disk
func (ec *EnrichmentCache) Load() {
	ec.mu.Lock()
	defer ec.mu.Unlock()

	data, err := os.ReadFile(ec.cacheFile)
	if err != nil {
		return // Cache file doesn't exist yet
	}

	var cached []CachedEnrichment
	if err := json.Unmarshal(data, &cached); err != nil {
		return // Invalid cache file
	}

	now := time.Now()
	for _, entry := range cached {
		// Only load non-expired entries
		if now.Before(entry.ExpiresAt) {
			ec.cache[entry.Sender] = &entry
		}
	}
}

// Save saves cache to disk
func (ec *EnrichmentCache) Save() {
	ec.mu.RLock()
	defer ec.mu.RUnlock()

	now := time.Now()
	cached := make([]CachedEnrichment, 0, len(ec.cache))
	for _, entry := range ec.cache {
		// Only save non-expired entries
		if now.Before(entry.ExpiresAt) {
			cached = append(cached, *entry)
		}
	}

	data, err := json.Marshal(cached)
	if err != nil {
		return
	}

	os.WriteFile(ec.cacheFile, data, 0644)
}

// Clear removes all cached entries
func (ec *EnrichmentCache) Clear() {
	ec.mu.Lock()
	defer ec.mu.Unlock()

	ec.cache = make(map[string]*CachedEnrichment)
	os.Remove(ec.cacheFile)
}
