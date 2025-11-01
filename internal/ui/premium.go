package ui

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/loickal/newsletter-cli/internal/api"
	"github.com/loickal/newsletter-cli/internal/config"
)

type premiumLoginMsg struct {
	success bool
	message string
}

func (m appModel) updatePremium(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "esc", "ctrl+c":
			m.screen = screenWelcome
			m.premiumMsg = ""
			return m, nil
		case "r":
			if m.premiumEnabled {
				// Refresh license features and subscription status
				return m, tea.Batch(m.fetchLicenseFeatures(), m.fetchSubscriptionStatus())
			}
		case "tab", "shift+tab", "enter", "up", "down":
			// Handle tab/enter navigation
			if msg.String() == "enter" && m.premiumFocused == len(m.premiumInputs)-1 {
				// Submit premium login/register
				return m, m.submitPremiumLogin()
			}

			// Navigate inputs
			s := msg.String()
			if s == "up" || s == "shift+tab" {
				m.premiumFocused--
			} else {
				m.premiumFocused++
			}

			if m.premiumFocused > len(m.premiumInputs)-1 {
				m.premiumFocused = 0
			} else if m.premiumFocused < 0 {
				m.premiumFocused = len(m.premiumInputs) - 1
			}

			// Update focus
			cmds := make([]tea.Cmd, len(m.premiumInputs))
			for i := 0; i <= len(m.premiumInputs)-1; i++ {
				if i == m.premiumFocused {
					cmds[i] = m.premiumInputs[i].Focus()
				} else {
					m.premiumInputs[i].Blur()
				}
			}
			return m, tea.Batch(cmds...)
		case "s":
			if m.premiumEnabled {
				// Verify active subscription before syncing
				if m.currentSubscription == nil || (m.currentSubscription.Status != "active" && m.currentSubscription.Status != "trialing") {
					m.premiumMsg = "❌ Active subscription required for cloud sync.\n   Press [u] to subscribe and enable sync features."
					return m, nil
				}
				// Sync to cloud
				m.premiumSyncing = true
				return m, m.syncToCloud()
			}
		case "p":
			if m.premiumEnabled {
				// Verify active subscription before pulling
				if m.currentSubscription == nil || (m.currentSubscription.Status != "active" && m.currentSubscription.Status != "trialing") {
					m.premiumMsg = "❌ Active subscription required for cloud sync.\n   Press [u] to subscribe and enable sync features."
					return m, nil
				}
				// Pull from cloud
				m.premiumSyncing = true
				return m, m.syncFromCloud()
			}
		case "o", "0":
			if m.premiumEnabled {
				m.screen = screenSyncSettings
				return m, nil
			}
		case "d":
			if m.premiumEnabled {
				m.screen = screenDeleteConfirm
				return m, nil
			}
		case "u":
			if m.premiumEnabled {
				// Subscribe / Upgrade - go to subscription screen
				m.subscriptionMsg = ""
				m.subscriptionErr = ""
				m.screen = screenSubscription
				return m, m.initSubscription()
			}
		case "m":
			if m.premiumEnabled {
				// Manage subscription - open Stripe Customer Portal
				return m, m.openSubscriptionPortal()
			}
		case "w":
			if m.premiumEnabled {
				// Verify active subscription before allowing dashboard access
				if m.currentSubscription == nil || (m.currentSubscription.Status != "active" && m.currentSubscription.Status != "trialing") {
					m.premiumMsg = "❌ Active subscription required to access analytics dashboard. Please subscribe first."
					return m, nil
				}

				// Open dashboard in browser
				dashboardURL := api.GetDashboardURL()
				if dashboardURL != "" {
					if err := openBrowser(dashboardURL); err != nil {
						m.premiumMsg = "❌ Failed to open dashboard: " + err.Error()
					} else {
						m.premiumMsg = "✅ Opening dashboard in browser..."
					}
				} else {
					m.premiumMsg = "❌ Dashboard URL not available. Please check your premium configuration."
				}
				return m, nil
			}
		case "v":
			if m.premiumEnabled {
				// View usage statistics
				return m, m.fetchUsageStats()
			}
		}
	case premiumLoginMsg:
		if msg.success {
			m.premiumMsg = "✅ " + msg.message
			m.premiumEnabled = true
			m.premiumEmail = strings.TrimSpace(m.premiumInputs[1].Value())
			m.premiumAPIURL = strings.TrimSpace(m.premiumInputs[0].Value())
			// Fetch license features asynchronously (non-blocking)
			return m, m.fetchLicenseFeatures()
		} else {
			m.premiumMsg = "❌ " + msg.message
		}
		m.premiumSyncing = false
		return m, nil
	case premiumSyncMsg:
		if msg.success {
			m.premiumMsg = "✅ " + msg.message
		} else {
			m.premiumMsg = msg.message // Message already includes emoji
			// If subscription is needed, offer to navigate to subscription screen
			if msg.needsSubscription {
				m.premiumMsg += "\n\n   💡 Press [u] to view subscription plans"
			}
		}
		m.premiumSyncing = false
		return m, nil
	case spinner.TickMsg:
		if m.premiumSyncing {
			var cmd tea.Cmd
			m.analyzingSpinner, cmd = m.analyzingSpinner.Update(msg)
			return m, cmd
		}
	case licenseFeaturesMsg:
		if msg.err == nil {
			m.premiumTier = msg.tier
			m.premiumFeatures = msg.features
		}
		// Also fetch subscription status
		return m, m.fetchSubscriptionStatus()
	case subscriptionStatusMsg:
		m.currentSubscription = msg.subscription
		if msg.err != nil {
			// Silently ignore errors - user might not have subscription yet
		}
		return m, nil
	case subscriptionPortalMsg:
		if msg.err != nil {
			m.premiumMsg = "❌ Failed to open subscription portal: " + msg.err.Error()
		} else if msg.url != "" {
			// Open browser
			if err := openBrowser(msg.url); err != nil {
				m.premiumMsg = "❌ Failed to open browser: " + err.Error()
			} else {
				m.premiumMsg = "✅ Opening subscription management in browser..."
			}
		}
		return m, nil
	case usageStatsMsg:
		if msg.err != nil {
			m.premiumMsg = "❌ Failed to fetch usage stats: " + msg.err.Error()
		} else if msg.stats != nil {
			// Display usage stats
			m.premiumMsg = fmt.Sprintf(
				"📊 API Usage Stats (Last 24 hours):\n"+
					"   Total Requests: %d\n"+
					"   Unique Endpoints: %d",
				msg.stats.TotalRequests,
				msg.stats.UniqueEndpoints,
			)
		}
		return m, nil
	}

	// Update inputs
	var cmds []tea.Cmd
	inputs := make([]textinput.Model, len(m.premiumInputs))
	for i, input := range m.premiumInputs {
		var cmd tea.Cmd
		inputs[i], cmd = input.Update(msg)
		cmds = append(cmds, cmd)
	}
	m.premiumInputs = inputs
	return m, tea.Batch(cmds...)
}

type premiumSyncMsg struct {
	success           bool
	message           string
	needsSubscription bool // Indicates user needs to subscribe
}

type licenseFeaturesMsg struct {
	tier     string
	features []string
	err      error
}

type subscriptionStatusMsg struct {
	subscription *api.Subscription
	err          error
}

type subscriptionPortalMsg struct {
	url string
	err error
}

type usageStatsMsg struct {
	stats *api.UsageStats
	err   error
}

func (m appModel) submitPremiumLogin() tea.Cmd {
	return func() tea.Msg {
		apiURL := strings.TrimSpace(m.premiumInputs[0].Value())
		email := strings.TrimSpace(m.premiumInputs[1].Value())
		password := strings.TrimSpace(m.premiumInputs[2].Value())

		if apiURL == "" || email == "" || password == "" {
			return premiumLoginMsg{
				success: false,
				message: "Please fill in all fields",
			}
		}

		client := api.NewClient(apiURL)

		// Try login first
		authResp, err := client.Login(email, password)
		if err != nil {
			// Try register if login fails
			authResp, err = client.Register(email, password)
			if err != nil {
				return premiumLoginMsg{
					success: false,
					message: "Failed to login or register: " + err.Error(),
				}
			}
			// Registration successful
		}

		// Save premium config
		premiumConfig := &api.PremiumConfig{
			APIURL:                 apiURL,
			Token:                  authResp.Token,
			RefreshToken:           authResp.RefreshToken,
			Email:                  email,
			Enabled:                true,
			AnalyticsEnabled:       true,  // Enable analytics by default for new premium users
			AnalyticsExplicitlySet: false, // Not explicitly set yet (default)
		}

		if err := api.SavePremiumConfig(premiumConfig); err != nil {
			return premiumLoginMsg{
				success: false,
				message: "Failed to save premium config: " + err.Error(),
			}
		}

		// Reset analytics collector to re-initialize with new premium config
		api.ResetAnalyticsCollector()

		return premiumLoginMsg{
			success: true,
			message: "Premium enabled! Token saved.",
		}
	}
}

func (m appModel) fetchLicenseFeatures() tea.Cmd {
	return func() tea.Msg {
		features, err := api.GetLicenseFeatures()
		if err != nil {
			// Return error but don't block - just use defaults
			return licenseFeaturesMsg{
				tier:     "starter",
				features: []string{},
				err:      err,
			}
		}

		tier := "starter"
		featureList := []string{}
		if t, ok := features["tier"].(string); ok && t != "" {
			tier = t
		}
		if fl, ok := features["features"].([]interface{}); ok {
			for _, f := range fl {
				if fStr, ok := f.(string); ok {
					featureList = append(featureList, fStr)
				}
			}
		}

		return licenseFeaturesMsg{
			tier:     tier,
			features: featureList,
			err:      nil,
		}
	}
}

func (m appModel) fetchSubscriptionStatus() tea.Cmd {
	return func() tea.Msg {
		client, err := api.GetAPIClient()
		if err != nil {
			return subscriptionStatusMsg{
				subscription: nil,
				err:          err,
			}
		}

		subscription, err := client.GetCurrentSubscription()
		if err != nil {
			// No subscription is okay - user might not have one yet
			return subscriptionStatusMsg{
				subscription: nil,
				err:          nil, // Don't treat as error
			}
		}

		return subscriptionStatusMsg{
			subscription: subscription,
			err:          nil,
		}
	}
}

func (m appModel) openSubscriptionPortal() tea.Cmd {
	return func() tea.Msg {
		client, err := api.GetAPIClient()
		if err != nil {
			return subscriptionPortalMsg{
				url: "",
				err: err,
			}
		}

		portal, err := client.CreatePortalSession()
		if err != nil {
			return subscriptionPortalMsg{
				url: "",
				err: err,
			}
		}

		// Return portal URL - browser will be opened in message handler
		return subscriptionPortalMsg{
			url: portal.URL,
			err: nil,
		}
	}
}

func (m appModel) fetchUsageStats() tea.Cmd {
	return func() tea.Msg {
		client, err := api.GetAPIClient()
		if err != nil {
			return usageStatsMsg{
				stats: nil,
				err:   err,
			}
		}

		// Get stats for last 24 hours
		since := time.Now().Add(-24 * time.Hour)
		stats, err := client.GetUsageStats(since)
		if err != nil {
			return usageStatsMsg{
				stats: nil,
				err:   err,
			}
		}

		return usageStatsMsg{
			stats: stats,
			err:   nil,
		}
	}
}

func (m appModel) syncToCloud() tea.Cmd {
	return func() tea.Msg {
		if !m.premiumEnabled {
			return premiumSyncMsg{
				success: false,
				message: "Premium not enabled",
			}
		}

		// Verify active subscription before syncing
		hasActive := api.HasActiveSubscription()
		if !hasActive {
			return premiumSyncMsg{
				success:           false,
				message:           "❌ Active subscription required for cloud sync.\n   Please subscribe to enable sync features.",
				needsSubscription: true,
			}
		}

		var messages []string
		var hasErrors bool

		// Sync accounts (continue even if it fails)
		err := api.SyncAccountsToCloud()
		if err != nil {
			hasErrors = true
			// Check if error is subscription-related
			if strings.Contains(err.Error(), "subscription") || strings.Contains(err.Error(), "403") || strings.Contains(err.Error(), "Forbidden") {
				return premiumSyncMsg{
					success:           false,
					message:           "❌ Active subscription required for cloud sync.\n   Please subscribe to enable sync features.",
					needsSubscription: true,
				}
			}
			// Check if error indicates it was queued for retry
			if strings.Contains(err.Error(), "queued for background retry") {
				messages = append(messages, "Accounts: queued for retry (will sync in background)")
			} else {
				messages = append(messages, "Accounts: "+err.Error())
			}
		} else {
			messages = append(messages, "Accounts: synced successfully")
		}

		// Sync unsubscribed newsletters (continue even if accounts failed)
		err = api.SyncUnsubscribedToCloud()
		if err != nil {
			hasErrors = true
			// Check if error is subscription-related
			if strings.Contains(err.Error(), "subscription") || strings.Contains(err.Error(), "403") || strings.Contains(err.Error(), "Forbidden") {
				return premiumSyncMsg{
					success:           false,
					message:           "❌ Active subscription required for cloud sync.\n   Please subscribe to enable sync features.",
					needsSubscription: true,
				}
			}
			// Check if error indicates it was queued for retry
			if strings.Contains(err.Error(), "queued for background retry") {
				messages = append(messages, "Unsubscribed: queued for retry (will sync in background)")
			} else {
				messages = append(messages, "Unsubscribed: "+err.Error())
			}
		} else {
			messages = append(messages, "Unsubscribed: synced successfully")
		}

		// Build response message
		message := strings.Join(messages, "\n")
		if !hasErrors {
			return premiumSyncMsg{
				success: true,
				message: "✅ All data synced successfully!",
			}
		}

		// Some operations failed but may have been queued
		return premiumSyncMsg{
			success: false,
			message: "⚠️ Sync completed with some issues:\n" + message,
		}
	}
}

func (m appModel) syncFromCloud() tea.Cmd {
	return func() tea.Msg {
		if !m.premiumEnabled {
			return premiumSyncMsg{
				success: false,
				message: "Premium not enabled",
			}
		}

		// Verify active subscription before pulling
		hasActive := api.HasActiveSubscription()
		if !hasActive {
			return premiumSyncMsg{
				success:           false,
				message:           "❌ Active subscription required for cloud sync.\n   Please subscribe to enable sync features.",
				needsSubscription: true,
			}
		}

		// Get accounts from cloud
		cloudAccounts, err := api.SyncAccountsFromCloud()
		if err != nil {
			// Check if error is subscription-related
			if strings.Contains(err.Error(), "subscription") || strings.Contains(err.Error(), "403") || strings.Contains(err.Error(), "Forbidden") {
				return premiumSyncMsg{
					success:           false,
					message:           "❌ Active subscription required for cloud sync.\n   Please subscribe to enable sync features.",
					needsSubscription: true,
				}
			}
			return premiumSyncMsg{
				success: false,
				message: "Failed to sync accounts from cloud: " + err.Error(),
			}
		}

		// Get unsubscribed from cloud
		cloudUnsubscribed, err := api.SyncUnsubscribedFromCloud()
		if err != nil {
			// Check if error is subscription-related
			if strings.Contains(err.Error(), "subscription") || strings.Contains(err.Error(), "403") || strings.Contains(err.Error(), "Forbidden") {
				return premiumSyncMsg{
					success:           false,
					message:           "❌ Active subscription required for cloud sync.\n   Please subscribe to enable sync features.",
					needsSubscription: true,
				}
			}
			return premiumSyncMsg{
				success: false,
				message: "Failed to sync unsubscribed from cloud: " + err.Error(),
			}
		}

		// Merge unsubscribed with local
		if cloudUnsubscribed != nil && len(cloudUnsubscribed.Newsletters) > 0 {
			localStore, _ := config.LoadUnsubscribed()
			if localStore == nil {
				localStore = &config.UnsubscribedStore{Newsletters: []config.UnsubscribedNewsletter{}}
			}

			// Create map of local senders
			localSenders := make(map[string]bool)
			for _, n := range localStore.Newsletters {
				localSenders[n.Sender] = true
			}

			// Add cloud newsletters that don't exist locally
			updated := false
			for _, cloudNewsletter := range cloudUnsubscribed.Newsletters {
				if !localSenders[cloudNewsletter.Sender] {
					localStore.Newsletters = append(localStore.Newsletters, cloudNewsletter)
					updated = true
				}
			}

			if updated {
				config.SaveUnsubscribed(localStore)
			}
		}

		// Merge with local accounts
		cfg, err := config.Load()
		if err != nil {
			return premiumSyncMsg{
				success: false,
				message: "Failed to load local config: " + err.Error(),
			}
		}

		// Create map of existing accounts
		existingIDs := make(map[string]bool)
		for _, acc := range cfg.Accounts {
			existingIDs[acc.ID] = true
		}

		// Add new accounts from cloud
		added := 0
		for _, cloudAcc := range cloudAccounts {
			if !existingIDs[cloudAcc.ID] {
				cfg.Accounts = append(cfg.Accounts, cloudAcc)
				added++
			}
		}

		if added > 0 {
			if err := config.Save(*cfg); err != nil {
				return premiumSyncMsg{
					success: false,
					message: "Failed to save merged accounts: " + err.Error(),
				}
			}

			return premiumSyncMsg{
				success: true,
				message: fmt.Sprintf("Pulled %d account(s) from cloud!", added),
			}
		}

		return premiumSyncMsg{
			success: true,
			message: "Already in sync - no new accounts from cloud",
		}
	}
}

func (m appModel) viewPremium() string {
	if m.premiumSyncing {
		return docStyle.Render(
			lipgloss.JoinVertical(lipgloss.Left,
				titleStyle.Render("☁️ Premium"),
				"\n",
				m.analyzingSpinner.View()+" Syncing...",
			),
		)
	}

	var content strings.Builder
	content.WriteString(titleStyle.Render("☁️ Premium"))

	if m.premiumEnabled {
		content.WriteString("\n\n")
		content.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color("10")).Render("✅ Premium enabled"))
		content.WriteString(fmt.Sprintf("\nEmail: %s", m.premiumEmail))
		content.WriteString(fmt.Sprintf("\nAPI: %s", m.premiumAPIURL))

		// Show tier (from cached value or default)
		// Don't make blocking API calls in view - fetch asynchronously if needed
		tierDisplay := "Starter"
		if m.premiumTier != "" {
			tierDisplay = strings.ToUpper(m.premiumTier[:1]) + strings.ToLower(m.premiumTier[1:])
		}
		content.WriteString(fmt.Sprintf("\nTier: %s", tierDisplay))

		// Get premium config for sync stats and dashboard link
		premiumConfig, _ := api.GetPremiumConfig()
		if premiumConfig != nil {
			// Analytics status
			content.WriteString("\n\n")
			content.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color("14")).Bold(true).Render("📊 Analytics"))

			// Determine analytics status
			// For premium users, analytics defaults to enabled unless explicitly disabled
			// GetPremiumConfig handles setting the default to true for premium users
			analyticsEnabled := premiumConfig.AnalyticsEnabled

			// Check if user has active subscription
			hasActiveSubscription := m.currentSubscription != nil &&
				(m.currentSubscription.Status == "active" || m.currentSubscription.Status == "trialing")

			if hasActiveSubscription {
				if analyticsEnabled {
					content.WriteString("\n  ✅ Analytics: Enabled")
					content.WriteString("\n    Anonymous stats are being collected")
					content.WriteString("\n    (Toggle in Sync Settings)")
				} else {
					content.WriteString("\n  ❌ Analytics: Disabled")
					content.WriteString("\n    No data is being collected")
				}
			} else {
				content.WriteString("\n  ⚠️  Analytics: Requires Active Subscription")
				content.WriteString("\n    Subscribe to enable analytics features")
			}

			// Sync status
			content.WriteString("\n\n")
			content.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color("14")).Bold(true).Render("🔄 Sync Status"))

			// Last sync time
			if !premiumConfig.LastSyncTime.IsZero() {
				syncTime := formatTimeAgo(premiumConfig.LastSyncTime)
				content.WriteString(fmt.Sprintf("\nLast Sync: %s", syncTime))
			} else {
				content.WriteString("\nLast Sync: Never")
			}

			// Accounts sync status
			if !premiumConfig.LastAccountsSync.IsZero() {
				accountsTime := formatTimeAgo(premiumConfig.LastAccountsSync)
				content.WriteString(fmt.Sprintf("\n  • Accounts: %d synced (%s)", premiumConfig.AccountsSynced, accountsTime))
			} else if premiumConfig.SyncAccounts {
				content.WriteString("\n  • Accounts: Pending sync")
			}

			// Unsubscribed sync status
			if !premiumConfig.LastUnsubSync.IsZero() {
				unsubTime := formatTimeAgo(premiumConfig.LastUnsubSync)
				content.WriteString(fmt.Sprintf("\n  • Unsubscribed: %d items (%s)", premiumConfig.UnsubscribedCount, unsubTime))
			} else if premiumConfig.SyncUnsubscribed {
				content.WriteString("\n  • Unsubscribed: Pending sync")
			}

			// Show pending sync queue
			queue := api.GetSyncQueue()
			pendingCount := queue.GetPendingCount()
			if pendingCount > 0 {
				content.WriteString(fmt.Sprintf("\n  ⚠️  Pending: %d operation(s) queued for retry", pendingCount))
			}
		}

		// Show available features (from cached value)
		if len(m.premiumFeatures) > 0 {
			content.WriteString("\n\n")
			content.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color("14")).Bold(true).Render("✨ Features"))
			for _, feature := range m.premiumFeatures {
				content.WriteString(fmt.Sprintf("\n  ✓ %s", feature))
			}
		}

		// Subscription status
		if m.currentSubscription != nil && m.currentSubscription.Status != "" {
			content.WriteString("\n\n")
			content.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color("14")).Bold(true).Render("💳 Subscription"))

			// Check if subscription is effectively canceled (either status is canceled OR canceled_at is set)
			isCanceled := m.currentSubscription.Status == "canceled" || 
				(m.currentSubscription.CanceledAt != nil && m.currentSubscription.CurrentPeriodEnd != nil && 
				 m.currentSubscription.CurrentPeriodEnd.Before(time.Now()))

			statusColor := "10" // green
			statusText := strings.ToUpper(m.currentSubscription.Status)
			if isCanceled || m.currentSubscription.Status == "canceled" {
				statusColor = "196" // red
				statusText = "CANCELED"
			} else if m.currentSubscription.CanceledAt != nil {
				// Scheduled to cancel at period end
				statusColor = "220" // yellow
				statusText = strings.ToUpper(m.currentSubscription.Status) + " (Will Cancel)"
			} else if m.currentSubscription.Status != "active" && m.currentSubscription.Status != "trialing" {
				statusColor = "220" // yellow
			}

			content.WriteString(fmt.Sprintf("\n  Status: %s", lipgloss.NewStyle().Foreground(lipgloss.Color(statusColor)).Render(statusText)))
			content.WriteString(fmt.Sprintf("\n  Tier: %s", strings.Title(m.currentSubscription.Tier)))

			if isCanceled || m.currentSubscription.Status == "canceled" {
				// Show cancellation date (when subscription was canceled)
				if m.currentSubscription.CanceledAt != nil {
					cancelDate := m.currentSubscription.CanceledAt.Format("January 2, 2006")
					content.WriteString(fmt.Sprintf("\n  ❌ Canceled on: %s", lipgloss.NewStyle().Foreground(lipgloss.Color("196")).Render(cancelDate)))
				}
				// Show when access ends (current_period_end)
				if m.currentSubscription.CurrentPeriodEnd != nil {
					if m.currentSubscription.CurrentPeriodEnd.After(time.Now()) {
						accessEndDate := m.currentSubscription.CurrentPeriodEnd.Format("January 2, 2006")
						timeUntil := formatTimeAgo(*m.currentSubscription.CurrentPeriodEnd)
						content.WriteString(fmt.Sprintf("\n     Access ends: %s (%s)", accessEndDate, timeUntil))
					} else {
						content.WriteString("\n     ❌ Access has ended")
					}
				}
			} else {
				// For active/subscription subscriptions, show renewal/period end info
				if m.currentSubscription.CurrentPeriodEnd != nil {
					if m.currentSubscription.Status == "active" || m.currentSubscription.Status == "trialing" {
						renewalDate := m.currentSubscription.CurrentPeriodEnd.Format("January 2, 2006")
						timeUntil := formatTimeAgo(*m.currentSubscription.CurrentPeriodEnd)
						content.WriteString(fmt.Sprintf("\n  Renews: %s (%s)", renewalDate, timeUntil))
						
						// If subscription was canceled but still active (cancel_at_period_end), show cancellation warning
						if m.currentSubscription.CanceledAt != nil {
							content.WriteString(fmt.Sprintf("\n  ⚠️  Will cancel at period end (canceled on %s)", 
								m.currentSubscription.CanceledAt.Format("January 2, 2006")))
						}
					} else {
						// For other statuses (past_due, etc), show when period ends
						periodEndDate := m.currentSubscription.CurrentPeriodEnd.Format("January 2, 2006")
						timeUntil := formatTimeAgo(*m.currentSubscription.CurrentPeriodEnd)
						content.WriteString(fmt.Sprintf("\n  Period ends: %s (%s)", periodEndDate, timeUntil))
					}
				}
			}
		}

		content.WriteString("\n\n")
		content.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color("14")).Bold(true).Render("Actions"))
		content.WriteString("\n")
		content.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color("14")).Render("[s] Sync to Cloud"))
		content.WriteString("\n")
		content.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color("14")).Render("[p] Pull from Cloud"))
		content.WriteString("\n")
		content.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color("14")).Render("[o] Sync Settings"))

		// Subscription actions
		if m.currentSubscription != nil && m.currentSubscription.Status == "active" {
			content.WriteString("\n")
			content.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color("14")).Render("[m] Manage Subscription"))
		} else {
			content.WriteString("\n")
			content.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color("14")).Render("[u] Subscribe / Upgrade"))
		}

		content.WriteString("\n")
		content.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color("196")).Render("[d] Delete All Data (GDPR)"))

		// Add dashboard button if analytics is enabled AND user has active subscription
		if premiumConfig != nil && premiumConfig.Enabled && premiumConfig.AnalyticsEnabled {
			// Check if user has active subscription
			hasActiveSubscription := m.currentSubscription != nil &&
				(m.currentSubscription.Status == "active" || m.currentSubscription.Status == "trialing")

			if hasActiveSubscription {
				dashboardURL := api.GetDashboardURL()
				if dashboardURL != "" {
					content.WriteString("\n")
					content.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color("14")).Render("[w] Open Dashboard"))
					content.WriteString("\n")
					content.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color("8")).Render("💡 Opens analytics dashboard in your browser"))
				}
			} else {
				content.WriteString("\n")
				content.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color("8")).Render("💡 Subscribe to access analytics dashboard"))
			}
		} else if premiumConfig != nil && premiumConfig.Enabled && premiumConfig.Token != "" {
			content.WriteString("\n")
			content.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color("8")).Render("💡 Enable analytics to view dashboard"))
		}

		// Add usage stats action (available for all premium users)
		if premiumConfig != nil && premiumConfig.Enabled {
			content.WriteString("\n")
			content.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color("14")).Render("[v] View API Usage Stats"))
			content.WriteString("\n")
			content.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color("8")).Render("💡 View your API request statistics"))
		}
	} else {
		content.WriteString("\n\n")
		content.WriteString("API URL:")
		content.WriteString("\n")
		content.WriteString(m.premiumInputs[0].View())
		content.WriteString("\n\n")
		content.WriteString("Email:")
		content.WriteString("\n")
		content.WriteString(m.premiumInputs[1].View())
		content.WriteString("\n\n")
		content.WriteString("Password:")
		content.WriteString("\n")
		content.WriteString(m.premiumInputs[2].View())
	}

	if m.premiumMsg != "" {
		content.WriteString("\n\n")
		content.WriteString(m.premiumMsg)
	}

	helpText := "[Tab] Next  [Enter] Login/Register  [Esc] Back"
	if m.premiumEnabled {
		helpText = "[s] Sync  [p] Pull  [o] Settings  [Esc] Back"
	}
	help := helpStyle.Render(helpText)
	content.WriteString("\n\n")
	content.WriteString(help)

	return docStyle.Render(content.String())
}

// formatTimeAgo formats a time as "X ago" or relative time
func formatTimeAgo(t time.Time) string {
	now := time.Now()
	diff := now.Sub(t)

	if diff < time.Minute {
		return "just now"
	} else if diff < time.Hour {
		minutes := int(diff.Minutes())
		return fmt.Sprintf("%d minute%s ago", minutes, pluralize(minutes))
	} else if diff < 24*time.Hour {
		hours := int(diff.Hours())
		return fmt.Sprintf("%d hour%s ago", hours, pluralize(hours))
	} else if diff < 7*24*time.Hour {
		days := int(diff.Hours() / 24)
		return fmt.Sprintf("%d day%s ago", days, pluralize(days))
	} else {
		return t.Format("Jan 2, 2006 at 15:04")
	}
}

func pluralize(n int) string {
	if n == 1 {
		return ""
	}
	return "s"
}
