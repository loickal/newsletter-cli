package ui

import (
	"fmt"
	"os"
	"os/exec"
	"runtime"

	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/loickal/newsletter-cli/internal/imap"
)

type model struct {
	list  list.Model
	stats []imap.NewsletterStat
	msg   string
}

func NewDashboard(stats []imap.NewsletterStat) model {
	items := []list.Item{}
	for _, s := range stats {
		items = append(items, listItem{
			title: s.Sender,
			desc:  fmt.Sprintf("%d emails", s.Count),
			link:  s.Unsubscribe,
		})
	}
	l := list.New(items, list.NewDefaultDelegate(), 60, 20)
	l.Title = "📬  Newsletter Overview"
	l.SetShowHelp(false)
	return model{list: l, stats: stats}
}

type listItem struct {
	title, desc, link string
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
				if i.link == "" {
					m.msg = "❌  No unsubscribe link found for " + i.title
				} else {
					m.msg = "🔗  Opening: " + i.link
					openBrowser(i.link)
				}
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

	msg := ""
	if m.msg != "" {
		msg = "\n" + m.msg
	}
	return style.Render(m.list.View()) + msg + "\n[↑↓] Navigate  [u] Unsubscribe  [q] Quit\n"
}

func openBrowser(url string) {
	var cmd string
	var args []string

	switch runtime.GOOS {
	case "linux":
		cmd = "xdg-open"
	case "windows":
		cmd = "rundll32"
		args = []string{"url.dll,FileProtocolHandler", url}
	case "darwin":
		cmd = "open"
	default:
		fmt.Println("Please open manually:", url)
		return
	}

	if cmd == "xdg-open" {
		args = []string{url}
	}
	exec.Command(cmd, args...).Start()
}

func Run(stats []imap.NewsletterStat) error {
	p := tea.NewProgram(NewDashboard(stats))
	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error running TUI: %v\n", err)
		return err
	}
	return nil
}
