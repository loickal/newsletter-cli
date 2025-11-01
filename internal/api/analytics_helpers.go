package api

import (
	"time"

	"github.com/loickal/newsletter-cli/internal/config"
)

const (
	// Analytics event types
	EventTypeNewsletterAnalyzed = "newsletter_analyzed"
	EventTypeUnsubscribed       = "unsubscribed"
	EventTypeAnalysisCompleted  = "analysis_completed"
)

// analyticsSalt is a constant salt for hashing (could be made configurable)
// In production, you might want to use a user-specific salt stored in premium config
const analyticsSalt = "newsletter-cli-analytics-2025"

// SendNewsletterAnalysisEvent sends analytics after analyzing newsletters
// stats: slice of newsletter statistics
// accountEmail: email address of the account being analyzed (will be hashed)
// Returns error only for logging purposes - failures are silent to not interrupt user flow
func SendNewsletterAnalysisEvent(stats []NewsletterStatForAnalytics, accountEmail string) error {
	// Check premium and analytics status first
	cfg, err := GetPremiumConfig()
	if err != nil || !cfg.Enabled || !cfg.AnalyticsEnabled {
		return nil // Analytics not enabled - not an error
	}
	
	// Verify active subscription - analytics requires active subscription
	if !HasActiveSubscription() {
		return nil // No active subscription - silently skip analytics
	}

	collector, err := GetAnalyticsCollector()
	if err != nil {
		// Analytics is optional - don't fail if collector can't be created
		return nil
	}

	if collector == nil {
		return nil
	}

	// Hash account identifier
	accountID := HashAccountID(accountEmail, analyticsSalt)

	// Enrich newsletters using API (for categorization and quality scoring)
	enrichInputs := make([]EnrichNewsletterInput, 0, len(stats))
	for _, stat := range stats {
		enrichInputs = append(enrichInputs, EnrichNewsletterInput{
			Sender:         stat.Sender,
			EmailCount:     stat.Count,
			HasUnsubscribe: stat.HasUnsubscribeLink,
		})
	}

	// Try to enrich via API (with caching), but don't fail if it doesn't work
	enrichedMap := make(map[string]EnrichNewsletter)
	if len(enrichInputs) > 0 {
		enriched, err := EnrichNewslettersWithCache(enrichInputs)
		if err == nil {
			for _, e := range enriched {
				enrichedMap[e.Sender] = e
			}
		}
		// If enrichment fails, continue without categories/scores in analytics
	}

	// Send individual newsletter events with categorization and quality scoring
	for _, stat := range stats {
		var category string
		var categoryConfidence float64
		var qualityScore int

		// Use enriched data if available
		if enriched, found := enrichedMap[stat.Sender]; found {
			category = enriched.Category.Category
			categoryConfidence = enriched.Category.Confidence
			qualityScore = enriched.QualityScore
		}

		event := AnalyticsEvent{
			EventType:    EventTypeNewsletterAnalyzed,
			Timestamp:    time.Now(),
			SenderDomain: HashSenderDomain(stat.Sender, analyticsSalt),
			EmailCount:   stat.Count,
			AccountID:    accountID,
			Metadata: map[string]interface{}{
				"has_unsubscribe_link": stat.HasUnsubscribeLink,
				"category":              category,
				"category_confidence":    categoryConfidence,
				"quality_score":          qualityScore,
			},
		}
		collector.Collect(event)
	}

	// Send summary event
	summaryEvent := AnalyticsEvent{
		EventType:  EventTypeAnalysisCompleted,
		Timestamp:  time.Now(),
		AccountID:  accountID,
		EmailCount: len(stats),
		Metadata: map[string]interface{}{
			"total_newsletters": len(stats),
			"total_emails":      calculateTotalEmails(stats),
		},
	}
	collector.Collect(summaryEvent)

	// Trigger immediate flush for analysis events
	go func() {
		_ = collector.Flush()
	}()

	return nil
}

// SendUnsubscribeEvent sends analytics when a newsletter is unsubscribed
// Returns error only for logging purposes - failures are silent to not interrupt user flow
func SendUnsubscribeEvent(sender string, success bool, accountEmail string) error {
	// Check premium and analytics status first
	cfg, err := GetPremiumConfig()
	if err != nil || !cfg.Enabled || !cfg.AnalyticsEnabled {
		return nil // Analytics not enabled - not an error
	}
	
	// Verify active subscription - analytics requires active subscription
	if !HasActiveSubscription() {
		return nil // No active subscription - silently skip analytics
	}

	collector, err := GetAnalyticsCollector()
	if err != nil {
		// Analytics is optional - don't fail if collector can't be created
		return nil
	}

	if collector == nil {
		return nil
	}

	accountID := HashAccountID(accountEmail, analyticsSalt)

	event := AnalyticsEvent{
		EventType:    EventTypeUnsubscribed,
		Timestamp:    time.Now(),
		SenderDomain: HashSenderDomain(sender, analyticsSalt),
		AccountID:    accountID,
		Metadata: map[string]interface{}{
			"success": success,
		},
	}
	collector.Collect(event)

	// Trigger flush
	go func() {
		_ = collector.Flush()
	}()

	return nil
}

// NewsletterStatForAnalytics is a simplified version for analytics
type NewsletterStatForAnalytics struct {
	Sender             string
	Count              int
	HasUnsubscribeLink bool
}

// calculateTotalEmails sums up email counts from stats
func calculateTotalEmails(stats []NewsletterStatForAnalytics) int {
	total := 0
	for _, stat := range stats {
		total += stat.Count
	}
	return total
}

// ConvertNewsletterStatsToAnalytics converts newsletter stats to analytics format
// This helper is called from the UI layer to convert imap.NewsletterStat[] to analytics format
func ConvertNewsletterStatsToAnalytics(sender string, count int, unsubscribeLink string) NewsletterStatForAnalytics {
	return NewsletterStatForAnalytics{
		Sender:             sender,
		Count:              count,
		HasUnsubscribeLink: unsubscribeLink != "",
	}
}

// GetAllAccounts returns all accounts for analytics hashing
func GetAllAccountsForAnalytics() ([]config.Account, error) {
	return config.GetAllAccounts()
}
