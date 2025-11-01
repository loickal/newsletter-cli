package ui

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/loickal/newsletter-cli/internal/api"
)

func (m appModel) updateSyncSettings(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "esc", "q":
			m.screen = screenPremium
			return m, nil
		case "1":
			// Toggle auto-sync on startup
			pc, _ := api.GetPremiumConfig()
			if pc != nil {
				pc.AutoSyncOnStartup = !pc.AutoSyncOnStartup
				api.SavePremiumConfig(pc)
			}
			return m, nil
		case "2":
			// Toggle periodic sync
			pc, _ := api.GetPremiumConfig()
			if pc != nil {
				pc.PeriodicSyncEnabled = !pc.PeriodicSyncEnabled
				api.SavePremiumConfig(pc)
			}
			return m, nil
		case "3":
			// Toggle sync accounts
			pc, _ := api.GetPremiumConfig()
			if pc != nil {
				pc.SyncAccounts = !pc.SyncAccounts
				api.SavePremiumConfig(pc)
			}
			return m, nil
		case "4":
			// Toggle sync unsubscribed
			pc, _ := api.GetPremiumConfig()
			if pc != nil {
				pc.SyncUnsubscribed = !pc.SyncUnsubscribed
				api.SavePremiumConfig(pc)
			}
			return m, nil
		case "5":
			// Toggle analytics
			pc, _ := api.GetPremiumConfig()
			if pc != nil {
				pc.AnalyticsEnabled = !pc.AnalyticsEnabled
				pc.AnalyticsExplicitlySet = true // Mark as explicitly set by user
				api.SavePremiumConfig(pc)
				// Reset analytics collector to apply changes
				api.ResetAnalyticsCollector()
			}
			return m, nil
		case "+":
			// Increase periodic sync interval
			pc, _ := api.GetPremiumConfig()
			if pc != nil {
				if pc.PeriodicSyncInterval < 60 {
					pc.PeriodicSyncInterval += 1
					if pc.PeriodicSyncInterval == 0 {
						pc.PeriodicSyncInterval = 5 // Default to 5 if was 0
					}
					api.SavePremiumConfig(pc)
				}
			}
			return m, nil
		case "-":
			// Decrease periodic sync interval
			pc, _ := api.GetPremiumConfig()
			if pc != nil {
				if pc.PeriodicSyncInterval > 1 {
					pc.PeriodicSyncInterval -= 1
					api.SavePremiumConfig(pc)
				}
			}
			return m, nil
		}
	}

	return m, nil
}

func (m appModel) viewSyncSettings() string {
	var content strings.Builder

	titleStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("229")).
		Background(lipgloss.Color("63")).
		Bold(true).
		Padding(0, 1).
		MarginBottom(1)

	content.WriteString(titleStyle.Render("⚙️  Sync Settings"))

	pc, err := api.GetPremiumConfig()
	if err != nil || pc == nil {
		content.WriteString("\n\n❌ Failed to load settings")
		return docStyle.Render(content.String())
	}

	// Default values if not set
	autoSyncOnStartup := true
	periodicSyncEnabled := true
	periodicInterval := 5
	syncAccounts := true
	syncUnsubscribed := true
	analyticsEnabled := true

	if pc.AutoSyncOnStartup || (pc.AutoSyncOnStartup == false && pc.PeriodicSyncEnabled == false && pc.PeriodicSyncInterval == 0) {
		autoSyncOnStartup = pc.AutoSyncOnStartup || (pc.AutoSyncOnStartup == false && pc.PeriodicSyncEnabled == false && pc.PeriodicSyncInterval == 0 && !pc.SyncAccounts && !pc.SyncUnsubscribed)
		// If all settings are default/unset, assume defaults
		if !pc.AutoSyncOnStartup && !pc.PeriodicSyncEnabled && pc.PeriodicSyncInterval == 0 && !pc.SyncAccounts && !pc.SyncUnsubscribed {
			autoSyncOnStartup = true
			periodicSyncEnabled = true
			periodicInterval = 5
			syncAccounts = true
			syncUnsubscribed = true
			analyticsEnabled = true
		} else {
			autoSyncOnStartup = pc.AutoSyncOnStartup
			periodicSyncEnabled = pc.PeriodicSyncEnabled
			if pc.PeriodicSyncInterval > 0 {
				periodicInterval = pc.PeriodicSyncInterval
			}
			syncAccounts = pc.SyncAccounts
			syncUnsubscribed = pc.SyncUnsubscribed
			// Use actual config value (GetPremiumConfig already handles defaulting)
			analyticsEnabled = pc.AnalyticsEnabled
		}
	} else {
		autoSyncOnStartup = pc.AutoSyncOnStartup
		if pc.PeriodicSyncInterval > 0 {
			periodicInterval = pc.PeriodicSyncInterval
		}
		if pc.PeriodicSyncEnabled {
			periodicSyncEnabled = pc.PeriodicSyncEnabled
		}
		if pc.SyncAccounts {
			syncAccounts = pc.SyncAccounts
		}
		if pc.SyncUnsubscribed {
			syncUnsubscribed = pc.SyncUnsubscribed
		}
		// Use actual config value (GetPremiumConfig already handles defaulting)
		analyticsEnabled = pc.AnalyticsEnabled
	}

	content.WriteString("\n\n")

	// Auto-sync on startup
	toggleSymbol := "❌"
	if autoSyncOnStartup {
		toggleSymbol = "✅"
	}
	content.WriteString(fmt.Sprintf("[1] Auto-sync on startup: %s", toggleSymbol))

	// Periodic sync
	toggleSymbol = "❌"
	if periodicSyncEnabled {
		toggleSymbol = "✅"
	}
	content.WriteString(fmt.Sprintf("\n[2] Periodic sync: %s", toggleSymbol))
	if periodicSyncEnabled {
		content.WriteString(fmt.Sprintf(" (Every %d minutes)", periodicInterval))
		content.WriteString("\n    [+/-] Adjust interval")
	}

	// What to sync
	content.WriteString("\n\n")
	content.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color("14")).Bold(true).Render("What to sync:"))

	toggleSymbol = "❌"
	if syncAccounts {
		toggleSymbol = "✅"
	}
	content.WriteString(fmt.Sprintf("\n[3] Accounts: %s", toggleSymbol))

	toggleSymbol = "❌"
	if syncUnsubscribed {
		toggleSymbol = "✅"
	}
	content.WriteString(fmt.Sprintf("\n[4] Unsubscribed newsletters: %s", toggleSymbol))

	// Analytics setting
	content.WriteString("\n\n")
	content.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color("14")).Bold(true).Render("Analytics:"))

	toggleSymbol = "❌"
	if analyticsEnabled {
		toggleSymbol = "✅"
	}
	content.WriteString(fmt.Sprintf("\n[5] Analytics collection: %s", toggleSymbol))

	help := helpStyle.Render("[1-5] Toggle  [+/-] Adjust interval  [Esc] Back")
	content.WriteString("\n\n")
	content.WriteString(help)

	return docStyle.Render(content.String())
}
