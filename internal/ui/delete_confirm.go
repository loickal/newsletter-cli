package ui

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/loickal/newsletter-cli/internal/api"
)

type deleteCompleteMsg struct {
	err error
}

func (m appModel) updateDeleteConfirm(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case deleteCompleteMsg:
		m.deleteConfirmDeleting = false
		if msg.err != nil {
			m.errMsg = "Failed to delete data: " + msg.err.Error()
			return m, nil
		}
		// Success - clear premium config locally
		pc, _ := api.GetPremiumConfig()
		if pc != nil {
			pc.Enabled = false
			pc.Token = ""
			pc.RefreshToken = ""
			api.SavePremiumConfig(pc)
		}
		// Return to welcome screen
		m.premiumEnabled = false
		m.screen = screenWelcome
		m.errMsg = "" // Clear any errors
		return m, nil

	case tea.KeyMsg:
		switch msg.String() {
		case "y", "Y":
			// User confirmed deletion
			if !m.deleteConfirmDeleting {
				m.deleteConfirmDeleting = true
				return m, m.deleteAccountFromCloud()
			}
			return m, nil
		case "n", "N", "esc", "q":
			// User cancelled - go back to premium screen
			m.screen = screenPremium
			return m, nil
		}
	}

	return m, nil
}

func (m appModel) deleteAccountFromCloud() tea.Cmd {
	return func() tea.Msg {
		err := api.DeleteAccountFromCloud()
		return deleteCompleteMsg{err: err}
	}
}

func (m appModel) viewDeleteConfirm() string {
	var content strings.Builder

	titleStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("229")).
		Background(lipgloss.Color("196")).
		Bold(true).
		Padding(0, 1).
		MarginBottom(1)

	content.WriteString(titleStyle.Render("⚠️  Delete All Data (GDPR)"))

	content.WriteString("\n\n")
	warningStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("196")).
		Bold(true)
	content.WriteString(warningStyle.Render("⚠️  WARNING: This action cannot be undone!"))
	content.WriteString("\n\n")
	content.WriteString("This will permanently delete ALL your data from the cloud:")
	content.WriteString("\n  • All synced accounts")
	content.WriteString("\n  • All unsubscribed newsletter history")
	content.WriteString("\n  • All configuration data")
	content.WriteString("\n  • Your premium account")
	content.WriteString("\n\n")
	content.WriteString("Your local data will NOT be deleted.")
	content.WriteString("\n")
	content.WriteString("This action is required for GDPR compliance.")
	content.WriteString("\n")

	if m.deleteConfirmDeleting {
		content.WriteString("\n")
		syncStyle := lipgloss.NewStyle().
			Foreground(lipgloss.Color("196")).
			Bold(true)
		content.WriteString(syncStyle.Render("🗑️  Deleting all data from cloud..."))
		content.WriteString("\n")
		content.WriteString("Please wait...")
	} else {
		content.WriteString("\n")
		helpStyle := lipgloss.NewStyle().
			Foreground(lipgloss.Color("241")).
			MarginTop(1)
		content.WriteString(helpStyle.Render("[y] Confirm deletion  [n/Esc] Cancel"))
	}

	return docStyle.Render(content.String())
}

