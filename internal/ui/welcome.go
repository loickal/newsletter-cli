package ui

import (
	"fmt"

	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type welcomeModel struct {
	list     list.Model
	choice   string
	quitting bool
}

type menuItem struct {
	title       string
	description string
	action      string
}

func (i menuItem) Title() string       { return i.title }
func (i menuItem) Description() string { return i.description }
func (i menuItem) FilterValue() string { return i.title }

func NewWelcomeScreen() welcomeModel {
	items := []list.Item{
		menuItem{
			title:       "🔐 Login",
			description: "Save your IMAP credentials",
			action:      "login",
		},
		menuItem{
			title:       "📊 Analyze",
			description: "Analyze and manage newsletters",
			action:      "analyze",
		},
		menuItem{
			title:       "❌ Quit",
			description: "Exit the application",
			action:      "quit",
		},
	}

	delegate := list.NewDefaultDelegate()
	delegate.Styles.SelectedTitle = delegate.Styles.SelectedTitle.
		Foreground(lipgloss.Color("229")).
		Bold(true)
	delegate.Styles.SelectedDesc = delegate.Styles.SelectedDesc.
		Foreground(lipgloss.Color("219"))

	l := list.New(items, delegate, 0, 0)
	l.Title = "📬  Newsletter CLI"
	l.SetShowStatusBar(false)
	l.SetFilteringEnabled(false)
	l.Styles.Title = lipgloss.NewStyle().
		Background(lipgloss.Color("63")).
		Foreground(lipgloss.Color("230")).
		Bold(true).
		Padding(0, 1)

	return welcomeModel{
		list: l,
	}
}

func (m welcomeModel) Init() tea.Cmd {
	return nil
}

func (m welcomeModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		h, v := welcomeDocStyle.GetFrameSize()
		m.list.SetSize(msg.Width-h, msg.Height-v-6)
		return m, nil

	case tea.KeyMsg:
		switch msg.String() {
		case "q", "ctrl+c", "esc":
			m.choice = "quit"
			return m, tea.Quit
		case "enter":
			i, ok := m.list.SelectedItem().(menuItem)
			if ok {
				m.choice = i.action
				if i.action == "quit" {
					return m, tea.Quit
				}
				// Return a custom message to indicate the choice
				return m, tea.Quit
			}
		}
	}

	var cmd tea.Cmd
	m.list, cmd = m.list.Update(msg)
	return m, cmd
}

func (m welcomeModel) View() string {
	if m.quitting {
		return ""
	}

	intro := welcomeIntroStyle.Render(
		"A beautiful TUI-based CLI to analyze, list and unsubscribe\nfrom newsletters using your IMAP inbox.",
	)

	listView := welcomeDocStyle.Render(m.list.View())

	help := welcomeHelpStyle.Render(
		"[↑↓] Navigate  [Enter] Select  [q/Esc] Quit",
	)

	return welcomeDocStyle.Render(intro + "\n\n" + listView + "\n" + help)
}

var (
	welcomeDocStyle = lipgloss.NewStyle().Margin(1, 2)

	welcomeIntroStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("240")).
				Align(lipgloss.Center).
				Padding(0, 2)

	welcomeHelpStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("241")).
				MarginTop(1)
)

func RunWelcome() (string, error) {
	m := NewWelcomeScreen()
	p := tea.NewProgram(m, tea.WithAltScreen())
	finalModel, err := p.Run()
	if err != nil {
		return "", fmt.Errorf("error running welcome screen: %w", err)
	}

	if wm, ok := finalModel.(welcomeModel); ok {
		return wm.choice, nil
	}

	return "", nil
}
