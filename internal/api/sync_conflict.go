package api

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"time"

	"github.com/loickal/newsletter-cli/internal/config"
)

// SyncMetadata tracks sync metadata for conflict detection
type SyncMetadata struct {
	Version      int64     `json:"version"`
	LastModified time.Time `json:"last_modified"`
	ModifiedBy   string    `json:"modified_by"` // "local" or "cloud"
	Checksum     string    `json:"checksum"`     // SHA256 hash of content
}

// SyncConflict represents a detected conflict that needs resolution
type SyncConflict struct {
	Type      string      `json:"type"`       // "account" or "unsubscribed"
	ID        string      `json:"id"`         // Account ID or sender
	Field     string      `json:"field"`      // Field name (e.g., "name", "server")
	Local     interface{} `json:"local"`      // Local value
	Cloud     interface{} `json:"cloud"`      // Cloud value
	Resolved  bool        `json:"resolved"`   // Whether conflict has been resolved
	LocalTime time.Time   `json:"local_time"` // When local was modified
	CloudTime time.Time   `json:"cloud_time"` // When cloud was modified
}

// SyncResult contains sync operation results and conflicts
type SyncResult struct {
	Success          bool           `json:"success"`
	AccountsAdded    int            `json:"accounts_added"`
	AccountsUpdated  int            `json:"accounts_updated"`
	UnsubscribedAdded int           `json:"unsubscribed_added"`
	Conflicts        []SyncConflict `json:"conflicts"`
	Errors           []string       `json:"errors"`
}

// CalculateChecksum calculates SHA256 checksum of accounts or unsubscribed list
func CalculateChecksum(accounts []config.Account) string {
	// Create deterministic string representation
	data := ""
	for _, acc := range accounts {
		data += fmt.Sprintf("%s|%s|%s|%s|", acc.ID, acc.Name, acc.Email, acc.Server)
	}
	hash := sha256.Sum256([]byte(data))
	return hex.EncodeToString(hash[:])
}

// CalculateUnsubscribedChecksum calculates checksum for unsubscribed list
func CalculateUnsubscribedChecksum(newsletters []config.UnsubscribedNewsletter) string {
	data := ""
	for _, n := range newsletters {
		data += fmt.Sprintf("%s|%s|", n.Sender, n.UnsubscribedAt.Format(time.RFC3339))
	}
	hash := sha256.Sum256([]byte(data))
	return hex.EncodeToString(hash[:])
}

// DetectAccountConflicts detects conflicts between local and cloud accounts
func DetectAccountConflicts(localAccounts []config.Account, cloudAccounts []config.Account) []SyncConflict {
	conflicts := []SyncConflict{}
	
	// Create maps for efficient lookup
	localMap := make(map[string]config.Account)
	cloudMap := make(map[string]config.Account)
	
	for _, acc := range localAccounts {
		localMap[acc.ID] = acc
	}
	for _, acc := range cloudAccounts {
		cloudMap[acc.ID] = acc
	}
	
	// Check for conflicts in accounts that exist in both
	for id, localAcc := range localMap {
		if cloudAcc, exists := cloudMap[id]; exists {
			// Both exist - check for differences
			if localAcc.Name != cloudAcc.Name {
				conflicts = append(conflicts, SyncConflict{
					Type:      "account",
					ID:        id,
					Field:     "name",
					Local:     localAcc.Name,
					Cloud:     cloudAcc.Name,
					Resolved:  false,
					LocalTime: localAcc.CreatedAt,
					CloudTime: cloudAcc.CreatedAt,
				})
			}
			if localAcc.Server != cloudAcc.Server {
				conflicts = append(conflicts, SyncConflict{
					Type:      "account",
					ID:        id,
					Field:     "server",
					Local:     localAcc.Server,
					Cloud:     cloudAcc.Server,
					Resolved:  false,
					LocalTime: localAcc.CreatedAt,
					CloudTime: cloudAcc.CreatedAt,
				})
			}
			if localAcc.Email != cloudAcc.Email {
				conflicts = append(conflicts, SyncConflict{
					Type:      "account",
					ID:        id,
					Field:     "email",
					Local:     localAcc.Email,
					Cloud:     cloudAcc.Email,
					Resolved:  false,
					LocalTime: localAcc.CreatedAt,
					CloudTime: cloudAcc.CreatedAt,
				})
			}
		}
	}
	
	return conflicts
}

// ThreeWayMergeAccounts performs three-way merge of accounts
// Strategy:
// 1. Compare timestamps (LastModified or CreatedAt)
// 2. If timestamps equal, prefer local (user is editing now)
// 3. Track conflicts explicitly
func ThreeWayMergeAccounts(localAccounts []config.Account, cloudAccounts []config.Account, baseAccounts []config.Account) ([]config.Account, []SyncConflict) {
	result := []config.Account{}
	conflicts := []SyncConflict{}
	
	// Create maps for efficient lookup
	localMap := make(map[string]config.Account)
	cloudMap := make(map[string]config.Account)
	baseMap := make(map[string]config.Account)
	
	for _, acc := range localAccounts {
		localMap[acc.ID] = acc
	}
	for _, acc := range cloudAccounts {
		cloudMap[acc.ID] = acc
	}
	for _, acc := range baseAccounts {
		baseMap[acc.ID] = acc
	}
	
	// Collect all account IDs
	allIDs := make(map[string]bool)
	for id := range localMap {
		allIDs[id] = true
	}
	for id := range cloudMap {
		allIDs[id] = true
	}
	
	// Process each account
	for id := range allIDs {
		localAcc, localExists := localMap[id]
		cloudAcc, cloudExists := cloudMap[id]
		_, baseExists := baseMap[id]
		
		if !localExists && cloudExists {
			// New account from cloud - add it
			result = append(result, cloudAcc)
		} else if localExists && !cloudExists {
			// New account locally - keep it
			result = append(result, localAcc)
		} else if localExists && cloudExists {
			// Account exists in both - need to merge
			if !baseExists {
				// No base - compare timestamps
				if localAcc.CreatedAt.After(cloudAcc.CreatedAt) {
					result = append(result, localAcc)
				} else if cloudAcc.CreatedAt.After(localAcc.CreatedAt) {
					result = append(result, cloudAcc)
				} else {
					// Same timestamp - prefer local (user editing now)
					result = append(result, localAcc)
				}
			} else {
				// Three-way merge: compare local vs base and cloud vs base
				baseAcc := baseMap[id]
				localChanged := (localAcc.Name != baseAcc.Name || localAcc.Server != baseAcc.Server || localAcc.Email != baseAcc.Email)
				cloudChanged := (cloudAcc.Name != baseAcc.Name || cloudAcc.Server != baseAcc.Server || cloudAcc.Email != baseAcc.Email)
				
				if localChanged && cloudChanged {
					// Both changed - conflict!
					if localAcc.Name != cloudAcc.Name {
						conflicts = append(conflicts, SyncConflict{
							Type:      "account",
							ID:        id,
							Field:     "name",
							Local:     localAcc.Name,
							Cloud:     cloudAcc.Name,
							Resolved:  false,
							LocalTime: localAcc.CreatedAt,
							CloudTime: cloudAcc.CreatedAt,
						})
					}
					if localAcc.Server != cloudAcc.Server {
						conflicts = append(conflicts, SyncConflict{
							Type:      "account",
							ID:        id,
							Field:     "server",
							Local:     localAcc.Server,
							Cloud:     cloudAcc.Server,
							Resolved:  false,
							LocalTime: localAcc.CreatedAt,
							CloudTime: cloudAcc.CreatedAt,
						})
					}
					// For conflicts, prefer local (user is editing)
					result = append(result, localAcc)
				} else if localChanged {
					// Only local changed - use local
					result = append(result, localAcc)
				} else if cloudChanged {
					// Only cloud changed - use cloud
					result = append(result, cloudAcc)
				} else {
					// Neither changed - use either (prefer local)
					result = append(result, localAcc)
				}
			}
		}
	}
	
	return result, conflicts
}

// ThreeWayMergeUnsubscribed performs three-way merge of unsubscribed lists
func ThreeWayMergeUnsubscribed(localList []config.UnsubscribedNewsletter, cloudList []config.UnsubscribedNewsletter, baseList []config.UnsubscribedNewsletter) ([]config.UnsubscribedNewsletter, []SyncConflict) {
	result := []config.UnsubscribedNewsletter{}
	conflicts := []SyncConflict{}
	
	// Create maps
	localMap := make(map[string]config.UnsubscribedNewsletter)
	cloudMap := make(map[string]config.UnsubscribedNewsletter)
	baseMap := make(map[string]bool)
	
	for _, n := range localList {
		localMap[n.Sender] = n
	}
	for _, n := range cloudList {
		cloudMap[n.Sender] = n
	}
	for _, n := range baseList {
		baseMap[n.Sender] = true
	}
	
	// Collect all senders
	allSenders := make(map[string]bool)
	for sender := range localMap {
		allSenders[sender] = true
	}
	for sender := range cloudMap {
		allSenders[sender] = true
	}
	
	// For unsubscribed list, conflicts are rare (usually just additions)
	// Strategy: Union of both lists, prefer local timestamp if both exist
	for sender := range allSenders {
		localItem, localExists := localMap[sender]
		cloudItem, cloudExists := cloudMap[sender]
		
		if !localExists && cloudExists {
			// New from cloud
			result = append(result, cloudItem)
		} else if localExists && !cloudExists {
			// New locally
			result = append(result, localItem)
		} else if localExists && cloudExists {
			// Both exist - prefer earlier timestamp (earlier unsubscribe)
			if localItem.UnsubscribedAt.Before(cloudItem.UnsubscribedAt) {
				result = append(result, localItem)
			} else {
				result = append(result, cloudItem)
			}
			
			// Check for timestamp conflicts (significant difference)
			diff := localItem.UnsubscribedAt.Sub(cloudItem.UnsubscribedAt)
			if diff > 24*time.Hour || diff < -24*time.Hour {
				// Significant difference - might be conflict
				conflicts = append(conflicts, SyncConflict{
					Type:      "unsubscribed",
					ID:        sender,
					Field:     "unsubscribed_at",
					Local:     localItem.UnsubscribedAt,
					Cloud:     cloudItem.UnsubscribedAt,
					Resolved:  false,
					LocalTime: localItem.UnsubscribedAt,
					CloudTime: cloudItem.UnsubscribedAt,
				})
			}
		}
	}
	
	return result, conflicts
}

