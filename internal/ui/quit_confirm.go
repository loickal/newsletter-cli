package ui

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

func (m appModel) updateQuitConfirm(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case quitSyncCompleteMsg:
		m.quitConfirmSyncing = false
		if msg.err != nil {
			// Show error but still allow quit
			return m, tea.Quit
		}
		// Sync successful, quit
		return m, tea.Quit

	case tea.KeyMsg:
		switch msg.String() {
		case "y", "Y":
			// User wants to sync before quitting
			if m.premiumEnabled && !m.quitConfirmSyncing {
				m.quitConfirmSyncing = true
				return m, m.syncBeforeQuit()
			}
			return m, tea.Quit
		case "n", "N", "q", "ctrl+c":
			// User wants to quit without syncing
			return m, tea.Quit
		case "esc":
			// Cancel and go back
			m.screen = screenWelcome
			return m, nil
		}
	}

	return m, nil
}

func (m appModel) viewQuitConfirm() string {
	var content strings.Builder

	titleStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("229")).
		Background(lipgloss.Color("63")).
		Bold(true).
		Padding(0, 1).
		MarginBottom(1)

	content.WriteString(titleStyle.Render("⚠️  Quit Confirmation"))

	if m.premiumEnabled {
		content.WriteString("\n\n")
		content.WriteString("You have premium enabled with cloud sync.")
		content.WriteString("\n")
		content.WriteString("Would you like to sync your data before quitting?")

		if m.quitConfirmSyncing {
			content.WriteString("\n\n")
			syncStyle := lipgloss.NewStyle().
				Foreground(lipgloss.Color("14")).
				Bold(true)
			content.WriteString(syncStyle.Render("☁️  Syncing to cloud..."))
			content.WriteString("\n")
			content.WriteString("Please wait...")
		} else {
			content.WriteString("\n\n")
			helpStyle := lipgloss.NewStyle().
				Foreground(lipgloss.Color("241")).
				MarginTop(1)
			content.WriteString(helpStyle.Render("[y] Yes, sync & quit  [n] Quit without sync  [Esc] Cancel"))
		}
	} else {
		content.WriteString("\n\n")
		content.WriteString("Are you sure you want to quit?")
		content.WriteString("\n\n")
		helpStyle := lipgloss.NewStyle().
			Foreground(lipgloss.Color("241")).
			MarginTop(1)
		content.WriteString(helpStyle.Render("[y] Yes, quit  [n/Esc] Cancel"))
	}

	return docStyle.Render(content.String())
}
