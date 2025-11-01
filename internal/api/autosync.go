package api

import (
	"fmt"

	"github.com/loickal/newsletter-cli/internal/config"
)

// CheckAndSyncIfNeeded checks cloud versions and pulls if cloud is newer
// Returns true if sync was performed
func CheckAndSyncIfNeeded() (bool, error) {
	if !IsPremiumEnabled() {
		return false, nil // Silently skip if premium not enabled
	}

	client, err := GetAPIClient()
	if err != nil {
		return false, err
	}

	premiumConfig, err := GetPremiumConfig()
	if err != nil {
		return false, err
	}

	synced := false

	// Check accounts version
	cloudAccountsData, err := client.GetAccounts()
	if err == nil {
		if cloudAccountsData.Version > premiumConfig.LocalAccountsVersion {
			// Cloud has newer accounts, pull them
			cloudAccounts, err := SyncAccountsFromCloud()
			if err == nil {
				// Merge accounts
				cfg, err := config.Load()
				if err == nil {
					existingIDs := make(map[string]bool)
					for _, acc := range cfg.Accounts {
						existingIDs[acc.ID] = true
					}

					updated := false
					for _, cloudAcc := range cloudAccounts {
						if !existingIDs[cloudAcc.ID] {
							cfg.Accounts = append(cfg.Accounts, cloudAcc)
							updated = true
						}
					}

					if updated {
						if err := config.Save(*cfg); err == nil {
							// Update local version
							premiumConfig.LocalAccountsVersion = cloudAccountsData.Version
							synced = true
						}
					} else {
						// Even if no merge happened, update version to match cloud
						premiumConfig.LocalAccountsVersion = cloudAccountsData.Version
					}
				}
			}
		}
	}

	// Check unsubscribed version
	cloudUnsubscribedData, err := client.GetUnsubscribed()
	if err == nil {
		if cloudUnsubscribedData.Version > premiumConfig.LocalUnsubscribedVersion {
			// Cloud has newer unsubscribed data, pull it
			cloudUnsubscribed, err := SyncUnsubscribedFromCloud()
			if err == nil {
				localStore, _ := config.LoadUnsubscribed()
				if localStore == nil {
					localStore = &config.UnsubscribedStore{Newsletters: []config.UnsubscribedNewsletter{}}
				}

				localSenders := make(map[string]bool)
				for _, n := range localStore.Newsletters {
					localSenders[n.Sender] = true
				}

				updated := false
				for _, cloudNewsletter := range cloudUnsubscribed.Newsletters {
					if !localSenders[cloudNewsletter.Sender] {
						localStore.Newsletters = append(localStore.Newsletters, cloudNewsletter)
						updated = true
					}
				}

				if updated {
					if err := config.SaveUnsubscribed(localStore); err == nil {
						// Update local version
						premiumConfig.LocalUnsubscribedVersion = cloudUnsubscribedData.Version
						synced = true
					}
				} else {
					// Even if no merge happened, update version to match cloud
					premiumConfig.LocalUnsubscribedVersion = cloudUnsubscribedData.Version
				}
			}
		}
	}

	// Save updated versions
	if synced {
		if err := SavePremiumConfig(premiumConfig); err != nil {
			return false, fmt.Errorf("failed to save premium config: %w", err)
		}
	}

	return synced, nil
}

// PeriodicSync pushes local changes to cloud periodically
// Should be called periodically while app is running
func PeriodicSync() error {
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

	// Process sync queue first (retry failed operations)
	queue := GetSyncQueue()
	if queue.GetPendingCount() > 0 {
		if err := queue.ProcessQueue(); err != nil {
			// Don't fail completely if queue processing fails - continue with normal sync
			syncErr = err
		}
	}

	// Sync accounts if enabled
	if syncAccounts {
		if err := SyncAccountsToCloud(); err != nil {
			if syncErr == nil {
				syncErr = err
			}
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
