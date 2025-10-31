package ui

import (
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"sort"
	"strconv"

	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/loickal/newsletter-cli/internal/imap"
)

type model struct {
	list             list.Model
	stats            []imap.NewsletterStat
	msg              string
	msgStyle         lipgloss.Style
	totalEmails      int
	totalNewsletters int
}

func NewDashboard(stats []imap.NewsletterStat) model {
	// Sort by count (descending)
	sort.Slice(stats, func(i, j int) bool {
		return stats[i].Count > stats[j].Count
	})

	items := []list.Item{}
	totalEmails := 0
	for _, s := range stats {
		items = append(items, listItem{
			title: s.Sender,
			count: s.Count,
			link:  s.Unsubscribe,
		})
		totalEmails += s.Count
	}

	delegate := list.NewDefaultDelegate()
	delegate.Styles.SelectedTitle = delegate.Styles.SelectedTitle.
		Foreground(lipgloss.Color("229")).
		Bold(true)
	delegate.Styles.SelectedDesc = delegate.Styles.SelectedDesc.
		Foreground(lipgloss.Color("219"))

	l := list.New(items, delegate, 0, 0)
	l.Title = "📬  Newsletter Overview"
	l.SetShowStatusBar(false)
	l.SetFilteringEnabled(true)
	l.Styles.Title = lipgloss.NewStyle().
		Background(lipgloss.Color("63")).
		Foreground(lipgloss.Color("230")).
		Bold(true).
		Padding(0, 1)

	msgStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("220")).
		Padding(0, 1)

	return model{
		list:             l,
		stats:            stats,
		msgStyle:         msgStyle,
		totalEmails:      totalEmails,
		totalNewsletters: len(stats),
	}
}

type listItem struct {
	title string
	count int
	link  string
}

func (i listItem) Title() string {
	countStr := strconv.Itoa(i.count)
	color := getCountColor(i.count)
	countStyle := lipgloss.NewStyle().Foreground(color).Bold(true)
	return i.title + "  " + countStyle.Render(fmt.Sprintf("(%s)", countStr))
}

func (i listItem) Description() string {
	desc := fmt.Sprintf("%d email", i.count)
	if i.count != 1 {
		desc += "s"
	}
	if i.link != "" {
		// Truncate long URLs for display
		linkDisplay := i.link
		if len(linkDisplay) > 60 {
			linkDisplay = linkDisplay[:57] + "..."
		}
		return desc + "  •  🔗 " + linkDisplay
	}
	return desc + "  •  ⚠️  No unsubscribe link"
}

func (i listItem) FilterValue() string { return i.title }

func getCountColor(count int) lipgloss.Color {
	if count >= 20 {
		return lipgloss.Color("196") // Red for high counts
	} else if count >= 10 {
		return lipgloss.Color("208") // Orange
	} else if count >= 5 {
		return lipgloss.Color("220") // Yellow
	}
	return lipgloss.Color("46") // Green for low counts
}

func (m model) Init() tea.Cmd {
	return nil
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		h, v := docStyle.GetFrameSize()
		m.list.SetSize(msg.Width-h, msg.Height-v-7)
		return m, nil

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
					if err := openBrowser(i.link); err != nil {
						m.msg = "❌  Failed to open browser: " + err.Error() + " | Link: " + i.link
					} else {
						m.msg = "🔗  Opening: " + i.link
					}
				}
			}
			return m, nil
		case "/":
			m.list.ResetSelected()
			return m, nil
		case "esc":
			if m.list.FilterState() == list.Filtering {
				m.list.ResetFilter()
				return m, nil
			}
			m.msg = ""
			return m, nil
		}
	}

	var cmd tea.Cmd
	m.list, cmd = m.list.Update(msg)
	return m, cmd
}

func (m model) View() string {
	if len(m.stats) == 0 {
		return docStyle.Render(
			emptyStateStyle.Render(
				"📭\n\nNo newsletters found\n\nTry analyzing a different time period.",
			) + "\n\n" + helpStyle.Render("Press 'q' to quit"),
		)
	}

	// Header summary
	summary := headerStyle.Render(
		fmt.Sprintf("Total: %d newsletters • %d emails", m.totalNewsletters, m.totalEmails),
	)

	// List view
	listView := docStyle.Render(m.list.View())

	// Status message
	status := ""
	if m.msg != "" {
		status = "\n" + m.msgStyle.Render(m.msg)
	}

	// Help text
	help := helpStyle.Render(
		"[↑↓] Navigate  [u] Unsubscribe  [/] Search  [q] Quit",
	)

	return summary + "\n" + listView + status + "\n" + help
}

var (
	docStyle = lipgloss.NewStyle().Margin(1, 2)

	headerStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("240")).
			Margin(1, 0).
			Bold(true)

	helpStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("241")).
			MarginTop(1)

	emptyStateStyle = lipgloss.NewStyle().
			Align(lipgloss.Center).
			Padding(2, 4).
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("238"))
)

func openBrowser(url string) error {
	var cmd string
	var args []string

	switch runtime.GOOS {
	case "linux":
		cmd = "xdg-open"
		args = []string{url}
	case "windows":
		cmd = "rundll32"
		args = []string{"url.dll,FileProtocolHandler", url}
	case "darwin":
		cmd = "open"
		args = []string{url}
	default:
		return fmt.Errorf("unsupported platform: %s", runtime.GOOS)
	}

	browserCmd := exec.Command(cmd, args...)

	if err := browserCmd.Start(); err != nil {
		return fmt.Errorf("failed to start browser: %w", err)
	}

	// Detach the process so it doesn't block
	go func() {
		if browserCmd.Process != nil {
			browserCmd.Process.Release()
		}
	}()

	return nil
}

func Run(stats []imap.NewsletterStat) error {
	p := tea.NewProgram(NewDashboard(stats), tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error running TUI: %v\n", err)
		return err
	}
	return nil
}
