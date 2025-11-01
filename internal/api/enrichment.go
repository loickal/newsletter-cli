package api

import (
	"fmt"
)

// EnrichNewslettersWithCache enriches newsletters using API with caching
func EnrichNewslettersWithCache(newsletters []EnrichNewsletterInput) ([]EnrichNewsletter, error) {
	cache := GetEnrichmentCache()
	client, err := GetAPIClient()
	if err != nil {
		return nil, fmt.Errorf("failed to get API client: %w", err)
	}

	// Separate newsletters into cached and uncached
	cached := make([]EnrichNewsletter, 0, len(newsletters))
	uncached := make([]EnrichNewsletterInput, 0)
	uncachedIndices := make([]int, 0) // Track original indices

	for i, newsletter := range newsletters {
		if cachedEntry, found := cache.Get(newsletter.Sender, newsletter.EmailCount); found {
			cached = append(cached, EnrichNewsletter{
				Sender:       cachedEntry.Sender,
				Category:     cachedEntry.Category,
				QualityScore: cachedEntry.QualityScore,
			})
		} else {
			uncached = append(uncached, newsletter)
			uncachedIndices = append(uncachedIndices, i)
		}
	}

	// If all were cached, return early
	if len(uncached) == 0 {
		return cached, nil
	}

	// Fetch uncached newsletters from API
	response, err := client.EnrichNewsletters(uncached)
	if err != nil {
		// If API fails, return cached entries only (graceful degradation)
		if len(cached) > 0 {
			return cached, nil
		}
		return nil, fmt.Errorf("failed to enrich newsletters: %w", err)
	}

	// Store newly fetched entries in cache
	// Create a map for quick lookup of email counts
	emailCountMap := make(map[string]int)
	for _, input := range uncached {
		emailCountMap[input.Sender] = input.EmailCount
	}

	for _, enriched := range response.Enriched {
		emailCount := emailCountMap[enriched.Sender]
		cache.Set(
			enriched.Sender,
			enriched.Category,
			enriched.QualityScore,
			emailCount,
		)
	}

	// Merge cached and newly fetched results, preserving original order
	result := make([]EnrichNewsletter, len(newsletters))

	// Create maps for quick lookup
	cachedMap := make(map[string]EnrichNewsletter)
	for _, c := range cached {
		cachedMap[c.Sender] = c
	}

	uncachedMap := make(map[string]EnrichNewsletter)
	for _, u := range response.Enriched {
		uncachedMap[u.Sender] = u
	}

	// Build result in original order
	for i, newsletter := range newsletters {
		if enriched, found := uncachedMap[newsletter.Sender]; found {
			result[i] = enriched
		} else if enriched, found := cachedMap[newsletter.Sender]; found {
			result[i] = enriched
		}
	}

	return result, nil
}

// EnrichNewslettersSimple enriches newsletters without caching (for testing or one-off use)
func EnrichNewslettersSimple(newsletters []EnrichNewsletterInput) ([]EnrichNewsletter, error) {
	client, err := GetAPIClient()
	if err != nil {
		return nil, fmt.Errorf("failed to get API client: %w", err)
	}

	response, err := client.EnrichNewsletters(newsletters)
	if err != nil {
		return nil, fmt.Errorf("failed to enrich newsletters: %w", err)
	}

	return response.Enriched, nil
}
