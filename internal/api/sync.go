package api

import (
	"fmt"

	"github.com/loickal/newsletter-cli/internal/config"
)

// SyncAllAccounts syncs accounts bidirectionally with conflict resolution
// Returns sync result with conflicts and changes
func SyncAllAccounts() (*SyncResult, error) {
	if !IsPremiumEnabled() {
		return nil, fmt.Errorf("premium features not enabled")
	}

	result := &SyncResult{
		Success:   true,
		Conflicts: []SyncConflict{},
		Errors:    []string{},
	}

	// Get local accounts
	localAccounts, err := config.GetAllAccounts()
	if err != nil {
		return nil, err
	}

	// Get cloud accounts
	cloudAccounts, err := SyncAccountsFromCloud()
	if err != nil {
		// If cloud sync fails, try to push local (one-way sync)
		if pushErr := SyncAccountsToCloud(); pushErr != nil {
			// Queue for retry
			queue := GetSyncQueue()
			queue.QueueSync("accounts", localAccounts)
			return nil, fmt.Errorf("sync failed: %v (queued for retry)", pushErr)
		}
		// Push succeeded but pull failed - return partial success
		return result, nil
	}

	// Load base (last synced version) - for now use empty as base
	// In a real implementation, you'd store the last synced version
	baseAccounts := []config.Account{}

	// Perform three-way merge
	mergedAccounts, conflicts := ThreeWayMergeAccounts(localAccounts, cloudAccounts, baseAccounts)
	result.Conflicts = conflicts

	// Count changes
	localMap := make(map[string]bool)
	cloudMap := make(map[string]bool)
	for _, acc := range localAccounts {
		localMap[acc.ID] = true
	}
	for _, acc := range cloudAccounts {
		cloudMap[acc.ID] = true
	}

	for _, acc := range mergedAccounts {
		if !localMap[acc.ID] {
			result.AccountsAdded++
		} else if cloudMap[acc.ID] {
			// Check if account was updated
			for _, localAcc := range localAccounts {
				if localAcc.ID == acc.ID {
					if localAcc.Name != acc.Name || localAcc.Server != acc.Server || localAcc.Email != acc.Email {
						result.AccountsUpdated++
					}
					break
				}
			}
		}
	}

	// Save merged accounts if different from local
	if len(mergedAccounts) != len(localAccounts) || hasAccountChanges(localAccounts, mergedAccounts) {
		cfg, err := config.Load()
		if err != nil {
			result.Success = false
			result.Errors = append(result.Errors, fmt.Sprintf("Failed to load config: %v", err))
			return result, err
		}

		cfg.Accounts = mergedAccounts
		if err := config.Save(*cfg); err != nil {
			result.Success = false
			result.Errors = append(result.Errors, fmt.Sprintf("Failed to save config: %v", err))
			return result, err
		}
	}

	// Push merged result to cloud (if there are conflicts, we prefer local)
	if len(conflicts) == 0 {
		if err := SyncAccountsToCloud(); err != nil {
			// Queue for retry
			queue := GetSyncQueue()
			queue.QueueSync("accounts", mergedAccounts)
			result.Errors = append(result.Errors, fmt.Sprintf("Failed to push to cloud: %v (queued for retry)", err))
		}
	}

	return result, nil
}

// hasAccountChanges checks if accounts have changed
func hasAccountChanges(old, new []config.Account) bool {
	if len(old) != len(new) {
		return true
	}
	
	oldMap := make(map[string]config.Account)
	for _, acc := range old {
		oldMap[acc.ID] = acc
	}
	
	for _, newAcc := range new {
		oldAcc, exists := oldMap[newAcc.ID]
		if !exists {
			return true
		}
		if oldAcc.Name != newAcc.Name || oldAcc.Server != newAcc.Server || oldAcc.Email != newAcc.Email {
			return true
		}
	}
	return false
}

// AutoSync syncs accounts and unsubscribed newsletters automatically when local config changes
func AutoSync() error {
	if !IsPremiumEnabled() {
		return nil // Silently skip if premium not enabled
	}

	pc, err := GetPremiumConfig()
	if err != nil {
		return err
	}

	var syncErr error

	// Determine if we should sync accounts (default to true for old configs)
	syncAccounts := pc.SyncAccounts
	if !pc.AutoSyncOnStartup && !pc.PeriodicSyncEnabled && pc.PeriodicSyncInterval == 0 && !pc.SyncAccounts && !pc.SyncUnsubscribed {
		syncAccounts = true // Old config - default to true
	}

	// Determine if we should sync unsubscribed (default to true for old configs)
	syncUnsubscribed := pc.SyncUnsubscribed
	if !pc.AutoSyncOnStartup && !pc.PeriodicSyncEnabled && pc.PeriodicSyncInterval == 0 && !pc.SyncAccounts && !pc.SyncUnsubscribed {
		syncUnsubscribed = true // Old config - default to true
	}

	// Sync accounts if enabled
	if syncAccounts {
		if err := SyncAccountsToCloud(); err != nil {
			syncErr = err
		}
	}

	// Sync unsubscribed if enabled
	if syncUnsubscribed {
		if err := SyncUnsubscribedToCloud(); err != nil {
			if syncErr == nil {
				syncErr = err
			}
		}
	}

	return syncErr
}
