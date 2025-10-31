package ui

import (
	"fmt"
	"os"

	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/loickal/newsletter-cli/internal/imap"
)

type model struct {
	list  list.Model
	stats []imap.NewsletterStat
}

func NewDashboard(stats []imap.NewsletterStat) model {
	items := []list.Item{}
	for _, s := range stats {
		items = append(items, listItem{
			title: fmt.Sprintf("%s", s.Sender),
			desc:  fmt.Sprintf("%d emails", s.Count),
		})
	}
	l := list.New(items, list.NewDefaultDelegate(), 60, 20)
	l.Title = "📬  Newsletter Overview"
	l.SetShowHelp(false)

	return model{list: l, stats: stats}
}

type listItem struct {
	title, desc string
}

func (i listItem) Title() string       { return i.title }
func (i listItem) Description() string { return i.desc }
func (i listItem) FilterValue() string { return i.title }

func (m model) Init() tea.Cmd { return nil }

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "q", "ctrl+c":
			return m, tea.Quit
		case "u":
			i, ok := m.list.SelectedItem().(listItem)
			if ok {
				fmt.Printf("\n🔗 Would unsubscribe from %s\n", i.title)
			}
			return m, nil
		}
	}
	var cmd tea.Cmd
	m.list, cmd = m.list.Update(msg)
	return m, cmd
}

func (m model) View() string {
	style := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("63")).
		Padding(1, 2).
		Margin(1)
	return style.Render(m.list.View()) + "\n[↑↓] Navigate  [u] Unsubscribe  [q] Quit\n"
}

func Run(stats []imap.NewsletterStat) error {
	p := tea.NewProgram(NewDashboard(stats))
	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error running TUI: %v\n", err)
		return err
	}
	return nil
}
