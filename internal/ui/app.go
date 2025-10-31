package ui

import (
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/loickal/newsletter-cli/internal/config"
	"github.com/loickal/newsletter-cli/internal/imap"
)

type screen int

const (
	screenWelcome screen = iota
	screenLogin
	screenAnalyzeInput
	screenAnalyzing
	screenDashboard
)

type appModel struct {
	// Common
	screen screen
	width  int
	height int
	errMsg string

	// Welcome screen
	welcomeList list.Model

	// Login screen
	loginInputs  []textinput.Model
	loginFocused int

	// Analyze input screen
	analyzeInputs  []textinput.Model
	analyzeFocused int

	// Analyzing screen
	analyzingSpinner spinner.Model

	// Dashboard screen
	dashboardList    list.Model
	dashboardStats   []imap.NewsletterStat
	dashboardMsg     string
	totalEmails      int
	totalNewsletters int

	// Saved credentials (for skipping login)
	savedEmail    string
	savedPassword string
	savedServer   string
}

type appMenuItem struct {
	title       string
	description string
	action      screen
}

func (i appMenuItem) Title() string       { return i.title }
func (i appMenuItem) Description() string { return i.description }
func (i appMenuItem) FilterValue() string { return i.title }

func NewAppModel(savedEmail, savedPassword, savedServer string) appModel {
	// Initialize login inputs
	emailInput := textinput.New()
	emailInput.Placeholder = "you@example.com"
	emailInput.Focus()
	emailInput.CharLimit = 100
	emailInput.Width = 50

	passwordInput := textinput.New()
	passwordInput.Placeholder = "Enter password"
	passwordInput.EchoMode = textinput.EchoPassword
	passwordInput.CharLimit = 100
	passwordInput.Width = 50

	serverInput := textinput.New()
	serverInput.Placeholder = "imap.gmail.com:993"
	serverInput.CharLimit = 100
	serverInput.Width = 50

	// Initialize analyze inputs
	daysInput := textinput.New()
	daysInput.Placeholder = "30"
	daysInput.Focus()
	daysInput.CharLimit = 3
	daysInput.Width = 10

	// Initialize spinner
	sp := spinner.New()
	sp.Spinner = spinner.Dot
	sp.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("63"))

	// Initialize welcome list
	items := []list.Item{
		appMenuItem{
			title:       "🔐 Login",
			description: "Save your IMAP credentials",
			action:      screenLogin,
		},
	}

	// Only show Analyze option if user is logged in
	if savedEmail != "" && savedPassword != "" && savedServer != "" {
		items = append(items, appMenuItem{
			title:       "📊 Analyze",
			description: "Analyze and manage newsletters",
			action:      screenAnalyzeInput,
		})
	}

	// Add Quit option at the end
	items = append(items, appMenuItem{
		title:       "❌ Quit",
		description: "Exit the application",
		action:      screenWelcome, // Will quit anyway
	})

	delegate := list.NewDefaultDelegate()
	delegate.Styles.SelectedTitle = delegate.Styles.SelectedTitle.
		Foreground(lipgloss.Color("229")).
		Bold(true)
	delegate.Styles.SelectedDesc = delegate.Styles.SelectedDesc.
		Foreground(lipgloss.Color("219"))

	welcomeList := list.New(items, delegate, 0, 0)
	welcomeList.Title = "📬  Newsletter CLI"
	welcomeList.SetShowStatusBar(false)
	welcomeList.SetFilteringEnabled(false)
	welcomeList.Styles.Title = lipgloss.NewStyle().
		Background(lipgloss.Color("63")).
		Foreground(lipgloss.Color("230")).
		Bold(true).
		Padding(0, 1)

	// Pre-fill inputs if credentials exist
	if savedEmail != "" {
		emailInput.SetValue(savedEmail)
	}
	if savedServer != "" {
		serverInput.SetValue(savedServer)
	}

	return appModel{
		screen:           screenWelcome,
		welcomeList:      welcomeList,
		loginInputs:      []textinput.Model{emailInput, passwordInput, serverInput},
		loginFocused:     0,
		analyzeInputs:    []textinput.Model{daysInput},
		analyzeFocused:   0,
		analyzingSpinner: sp,
		savedEmail:       savedEmail,
		savedPassword:    savedPassword,
		savedServer:      savedServer,
	}
}

func (m appModel) Init() tea.Cmd {
	cmds := []tea.Cmd{
		m.analyzingSpinner.Tick,
		textinput.Blink,
	}

	// If we're in analyzing screen with saved credentials, start analysis immediately
	if m.screen == screenAnalyzing && m.savedEmail != "" && m.savedPassword != "" && m.savedServer != "" {
		cmds = append(cmds, m.startAnalysis())
	}

	return tea.Batch(cmds...)
}

func (m appModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	// Handle special messages first
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		h, v := docStyle.GetFrameSize()
		m.welcomeList.SetSize(msg.Width-h, msg.Height-v-6)
		if m.dashboardList.Width() > 0 {
			m.dashboardList.SetSize(msg.Width-h, msg.Height-v-7)
		}
		return m, nil

	case loginSuccessMsg:
		m.savedEmail = msg.email
		m.savedPassword = msg.password
		m.savedServer = msg.server
		m.screen = screenWelcome
		m.errMsg = ""

		// Update welcome list to include Analyze option now that user is logged in
		items := []list.Item{
			appMenuItem{
				title:       "🔐 Login",
				description: "Save your IMAP credentials",
				action:      screenLogin,
			},
			appMenuItem{
				title:       "📊 Analyze",
				description: "Analyze and manage newsletters",
				action:      screenAnalyzeInput,
			},
			appMenuItem{
				title:       "❌ Quit",
				description: "Exit the application",
				action:      screenWelcome, // Will quit anyway
			},
		}

		// Create new list with updated items
		delegate := list.NewDefaultDelegate()
		delegate.Styles.SelectedTitle = delegate.Styles.SelectedTitle.
			Foreground(lipgloss.Color("229")).
			Bold(true)
		delegate.Styles.SelectedDesc = delegate.Styles.SelectedDesc.
			Foreground(lipgloss.Color("219"))

		m.welcomeList.SetItems(items)

		return m, nil

	case analysisCompleteMsg:
		// Sort stats
		sort.Slice(msg.stats, func(i, j int) bool {
			return msg.stats[i].Count > msg.stats[j].Count
		})

		// Create dashboard
		items := []list.Item{}
		totalEmails := 0
		for _, s := range msg.stats {
			items = append(items, dashboardListItem{
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

		h, v := docStyle.GetFrameSize()
		if m.width > 0 && m.height > 0 {
			l.SetSize(m.width-h, m.height-v-7)
		}

		m.dashboardList = l
		m.dashboardStats = msg.stats
		m.totalEmails = totalEmails
		m.totalNewsletters = len(msg.stats)
		m.screen = screenDashboard
		m.errMsg = ""
		return m, nil

	case errorMsg:
		m.errMsg = string(msg)
		// Stay on current screen but show error
		return m, nil
	}

	// Handle screen-specific updates
	switch m.screen {
	case screenWelcome:
		return m.updateWelcome(msg)
	case screenLogin:
		return m.updateLogin(msg)
	case screenAnalyzeInput:
		return m.updateAnalyzeInput(msg)
	case screenAnalyzing:
		return m.updateAnalyzing(msg)
	case screenDashboard:
		return m.updateDashboard(msg)
	}

	return m, nil
}

func (m appModel) updateWelcome(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "q", "ctrl+c", "esc":
			return m, tea.Quit
		case "enter":
			i, ok := m.welcomeList.SelectedItem().(appMenuItem)
			if ok {
				if i.action == screenWelcome {
					return m, tea.Quit // Quit option
				}
				m.screen = i.action
				switch m.screen {
				case screenLogin:
					m.loginInputs[0].Focus()
					for i := 1; i < len(m.loginInputs); i++ {
						m.loginInputs[i].Blur()
					}
				case screenAnalyzeInput:
					// Always show the input screen to let user specify days
					m.analyzeInputs[0].Focus()
				}
				return m, nil
			}
		}
	}

	var cmd tea.Cmd
	m.welcomeList, cmd = m.welcomeList.Update(msg)
	return m, cmd
}

func (m appModel) updateLogin(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "esc", "ctrl+c":
			m.screen = screenWelcome
			return m, nil
		case "tab", "shift+tab", "enter", "up", "down":
			// Handle tab/enter navigation
			if msg.String() == "enter" && m.loginFocused == len(m.loginInputs)-1 {
				// Submit login
				return m, m.submitLogin()
			}

			// Cycle through inputs
			if msg.String() == "tab" || msg.String() == "enter" || msg.String() == "down" {
				m.loginFocused++
				if m.loginFocused >= len(m.loginInputs) {
					m.loginFocused = 0
				}
			} else {
				m.loginFocused--
				if m.loginFocused < 0 {
					m.loginFocused = len(m.loginInputs) - 1
				}
			}

			for i := range m.loginInputs {
				if i == m.loginFocused {
					m.loginInputs[i].Focus()
				} else {
					m.loginInputs[i].Blur()
				}
			}
			return m, nil
		}
	}

	// Update focused input
	var cmd tea.Cmd
	m.loginInputs[m.loginFocused], cmd = m.loginInputs[m.loginFocused].Update(msg)
	return m, cmd
}

func (m appModel) updateAnalyzeInput(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "esc", "ctrl+c":
			m.screen = screenWelcome
			return m, nil
		case "enter":
			// Start analysis
			m.screen = screenAnalyzing
			return m, m.startAnalysis()
		}
	}

	var cmd tea.Cmd
	m.analyzeInputs[m.analyzeFocused], cmd = m.analyzeInputs[m.analyzeFocused].Update(msg)
	return m, cmd
}

func (m appModel) updateAnalyzing(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		if msg.String() == "ctrl+c" {
			return m, tea.Quit
		}
	case spinner.TickMsg:
		var cmd tea.Cmd
		m.analyzingSpinner, cmd = m.analyzingSpinner.Update(msg)
		return m, cmd
	}
	return m, nil
}

func (m appModel) updateDashboard(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "q", "ctrl+c":
			return m, tea.Quit
		case "u":
			i, ok := m.dashboardList.SelectedItem().(dashboardListItem)
			if ok {
				if i.link == "" {
					m.dashboardMsg = "❌  No unsubscribe link found for " + i.title
				} else {
					if err := openBrowser(i.link); err != nil {
						m.dashboardMsg = "❌  Failed to open browser: " + err.Error() + " | Link: " + i.link
					} else {
						m.dashboardMsg = "🔗  Opening: " + i.link
					}
				}
			}
			return m, nil
		case "/":
			m.dashboardList.ResetSelected()
			return m, nil
		case "esc":
			if m.dashboardList.FilterState() == list.Filtering {
				m.dashboardList.ResetFilter()
				return m, nil
			}
			m.dashboardMsg = ""
			return m, nil
		}
	}

	var cmd tea.Cmd
	m.dashboardList, cmd = m.dashboardList.Update(msg)
	return m, cmd
}

func (m appModel) submitLogin() tea.Cmd {
	return func() tea.Msg {
		email := strings.TrimSpace(m.loginInputs[0].Value())
		password := strings.TrimSpace(m.loginInputs[1].Value())
		server := strings.TrimSpace(m.loginInputs[2].Value())

		if email == "" || password == "" || server == "" {
			return errorMsg("All fields are required")
		}

		// Test connection
		if err := imap.ConnectIMAP(email, password, server); err != nil {
			return errorMsg("Connection failed: " + err.Error())
		}

		// Save config
		cfg := config.Config{
			Email:    email,
			Server:   server,
			Password: config.Encrypt(password),
		}
		if err := config.Save(cfg); err != nil {
			return errorMsg("Failed to save config: " + err.Error())
		}

		return loginSuccessMsg{
			email:    email,
			password: password,
			server:   server,
		}
	}
}

func (m appModel) startAnalysis() tea.Cmd {
	return func() tea.Msg {
		// Get days
		daysStr := strings.TrimSpace(m.analyzeInputs[0].Value())
		if daysStr == "" {
			daysStr = "30"
		}
		daysInt, err := strconv.Atoi(daysStr)
		if err != nil || daysInt <= 0 {
			return errorMsg("Invalid number of days")
		}

		// Use saved credentials or input
		email := m.savedEmail
		password := m.savedPassword
		server := m.savedServer

		if email == "" || password == "" || server == "" {
			return errorMsg("Please login first")
		}

		days := time.Duration(daysInt) * 24 * time.Hour
		since := time.Now().Add(-days)

		stats, err := imap.FetchNewsletterStats(server, email, password, since)
		if err != nil {
			return errorMsg("Failed to fetch newsletters: " + err.Error())
		}

		return analysisCompleteMsg{stats: stats}
	}
}

type loginSuccessMsg struct {
	email    string
	password string
	server   string
}

type analysisCompleteMsg struct {
	stats []imap.NewsletterStat
}

type errorMsg string

func (m appModel) View() string {
	if m.width == 0 || m.height == 0 {
		return "Initializing..."
	}

	// Handle special messages in view (for async updates)
	var view string

	switch m.screen {
	case screenWelcome:
		view = m.viewWelcome()
	case screenLogin:
		view = m.viewLogin()
	case screenAnalyzeInput:
		view = m.viewAnalyzeInput()
	case screenAnalyzing:
		view = m.viewAnalyzing()
	case screenDashboard:
		view = m.viewDashboard()
	}

	// Add error message if present
	if m.errMsg != "" {
		errorStyle := lipgloss.NewStyle().
			Foreground(lipgloss.Color("196")).
			Padding(0, 1).
			MarginTop(1)
		view += "\n" + errorStyle.Render("❌ "+m.errMsg)
	}

	return view
}

func (m appModel) viewWelcome() string {
	intro := introStyle.Render(
		"A beautiful TUI-based CLI to analyze, list and unsubscribe\nfrom newsletters using your IMAP inbox.",
	)

	listView := docStyle.Render(m.welcomeList.View())
	help := helpStyle.Render("[↑↓] Navigate  [Enter] Select  [q/Esc] Quit")

	return docStyle.Render(intro + "\n\n" + listView + "\n" + help)
}

func (m appModel) viewLogin() string {
	title := titleStyle.Render("🔐  Login")

	var inputs []string
	labels := []string{"📧 Email:", "🔒 Password:", "🌐 IMAP Server:"}

	for i, input := range m.loginInputs {
		labelStyle := lipgloss.NewStyle().Width(20).Foreground(lipgloss.Color("240"))
		inputStyle := lipgloss.NewStyle()
		if i == m.loginFocused {
			inputStyle = inputStyle.Border(lipgloss.RoundedBorder()).
				BorderForeground(lipgloss.Color("63")).
				Padding(0, 1)
		} else {
			inputStyle = inputStyle.Border(lipgloss.RoundedBorder()).
				BorderForeground(lipgloss.Color("238")).
				Padding(0, 1)
		}

		inputs = append(inputs,
			labelStyle.Render(labels[i])+" "+
				inputStyle.Render(input.View()),
		)
	}

	content := title + "\n\n" + strings.Join(inputs, "\n\n")
	help := helpStyle.Render("[Tab] Next  [Shift+Tab] Previous  [Enter] Submit  [Esc] Back")

	return docStyle.Render(content + "\n\n" + help)
}

func (m appModel) viewAnalyzeInput() string {
	title := titleStyle.Render("📊  Analyze Newsletters")

	daysLabel := lipgloss.NewStyle().Width(20).Foreground(lipgloss.Color("240")).Render("📅 Days:")
	daysInput := m.analyzeInputs[0]
	inputStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("63")).
		Padding(0, 1)

	content := title + "\n\n" + daysLabel + " " + inputStyle.Render(daysInput.View())

	accountInfo := ""
	if m.savedEmail != "" {
		accountStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("240")).MarginTop(1)
		accountInfo = "\n\n" + accountStyle.Render(fmt.Sprintf("🔐 Using saved account: %s @ %s", m.savedEmail, m.savedServer))
	}

	help := helpStyle.Render("[Enter] Analyze  [Esc] Back")

	return docStyle.Render(content + accountInfo + "\n\n" + help)
}

func (m appModel) viewAnalyzing() string {
	spinnerView := m.analyzingSpinner.View()
	msg := lipgloss.NewStyle().Foreground(lipgloss.Color("240")).Render("Fetching newsletters...")

	return docStyle.Render(
		titleStyle.Render("🔍  Analyzing") + "\n\n" +
			spinnerView + " " + msg + "\n\n" +
			helpStyle.Render("Please wait..."),
	)
}

func (m appModel) viewDashboard() string {
	if len(m.dashboardStats) == 0 {
		return docStyle.Render(
			emptyStateStyle.Render(
				"📭\n\nNo newsletters found\n\nTry analyzing a different time period.",
			) + "\n\n" + helpStyle.Render("Press 'q' to quit"),
		)
	}

	summary := headerStyle.Render(
		fmt.Sprintf("Total: %d newsletters • %d emails", m.totalNewsletters, m.totalEmails),
	)

	listView := docStyle.Render(m.dashboardList.View())

	status := ""
	if m.dashboardMsg != "" {
		msgStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("220")).Padding(0, 1)
		status = "\n" + msgStyle.Render(m.dashboardMsg)
	}

	help := helpStyle.Render("[↑↓] Navigate  [u] Unsubscribe  [/] Search  [q] Quit")

	return summary + "\n" + listView + status + "\n" + help
}

type dashboardListItem struct {
	title string
	count int
	link  string
}

func (i dashboardListItem) Title() string {
	countStr := strconv.Itoa(i.count)
	color := getCountColor(i.count)
	countStyle := lipgloss.NewStyle().Foreground(color).Bold(true)
	return i.title + "  " + countStyle.Render(fmt.Sprintf("(%s)", countStr))
}

func (i dashboardListItem) Description() string {
	desc := fmt.Sprintf("%d email", i.count)
	if i.count != 1 {
		desc += "s"
	}
	if i.link != "" {
		linkDisplay := i.link
		if len(linkDisplay) > 60 {
			linkDisplay = linkDisplay[:57] + "..."
		}
		return desc + "  •  🔗 " + linkDisplay
	}
	return desc + "  •  ⚠️  No unsubscribe link"
}

func (i dashboardListItem) FilterValue() string { return i.title }

var (
	titleStyle = lipgloss.NewStyle().
			Background(lipgloss.Color("63")).
			Foreground(lipgloss.Color("230")).
			Bold(true).
			Padding(0, 1)

	introStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("240")).
			Align(lipgloss.Center).
			Padding(0, 2)
)

// RunAppSync runs the app synchronously (for use from commands)
// initialScreen can be "login", "analyze", or "" for welcome
func RunAppSync(savedEmail, savedPassword, savedServer string, days int, flagsProvided bool, initialScreen string) error {
	m := NewAppModel(savedEmail, savedPassword, savedServer)

	// Determine initial screen
	if initialScreen == "login" {
		m.screen = screenLogin
		m.loginInputs[0].Focus()
		for i := 1; i < len(m.loginInputs); i++ {
			m.loginInputs[i].Blur()
		}
	} else if initialScreen == "analyze" || (flagsProvided && savedEmail != "" && savedPassword != "" && savedServer != "") {
		// Go directly to analyze input or analysis
		if savedEmail != "" && savedPassword != "" && savedServer != "" {
			m.screen = screenAnalyzing
			if days == 0 {
				days = 30
			}
			m.analyzeInputs[0].SetValue(strconv.Itoa(days))
			m.savedEmail = savedEmail
			m.savedPassword = savedPassword
			m.savedServer = savedServer
		} else {
			m.screen = screenAnalyzeInput
			m.analyzeInputs[0].Focus()
		}
	} else if flagsProvided {
		// Flags provided but no saved credentials - show login
		m.screen = screenLogin
		m.loginInputs[0].Focus()
		for i := 1; i < len(m.loginInputs); i++ {
			m.loginInputs[i].Blur()
		}
	}
	// Otherwise show welcome screen (default)

	p := tea.NewProgram(m, tea.WithAltScreen())
	_, err := p.Run()
	if err != nil {
		return fmt.Errorf("error running app: %w", err)
	}

	return nil
}
