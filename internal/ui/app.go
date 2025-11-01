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
	"github.com/loickal/newsletter-cli/internal/api"
	"github.com/loickal/newsletter-cli/internal/config"
	"github.com/loickal/newsletter-cli/internal/imap"
	"github.com/loickal/newsletter-cli/internal/unsubscribe"
	"github.com/loickal/newsletter-cli/internal/update"
)

type screen int

const (
	screenWelcome screen = iota
	screenLogin
	screenAnalyzeInput
	screenAnalyzing
	screenDashboard
	screenAccounts
	screenPremium
	screenQuitConfirm
	screenSyncSettings
	screenDeleteConfirm
	screenSubscription
)

type appModel struct {
	// Common
	screen screen
	width  int
	height int
	errMsg string

	// Welcome screen
	welcomeList     list.Model
	updateAvailable *updateInfo
	currentVersion  string

	// Login screen
	loginInputs         []textinput.Model
	loginFocused        int
	discoveringServer   bool
	serverStatusMsg     string
	lastDiscoveredEmail string // Track last email we discovered for to avoid re-checking same email

	// Analyze input screen
	analyzeInputs  []textinput.Model
	analyzeFocused int

	// Analyzing screen
	analyzingSpinner spinner.Model

	// Dashboard screen
	dashboardList         list.Model
	dashboardStats        []imap.NewsletterStat
	dashboardMsg          string
	dashboardSelected     map[string]bool // Track selected newsletters by sender
	dashboardUnsubscribed map[string]bool // Track which newsletters are already unsubscribed
	unsubscribing         bool
	unsubscribeResults    []unsubscribeResultMsg
	totalEmails           int
	totalNewsletters      int

	// Saved credentials (for skipping login)
	savedEmail    string
	savedPassword string
	savedServer   string

	// Account management screen
	accountsList     list.Model
	accounts         []config.Account
	accountsMsg      string
	accountToDelete  string // ID of account pending deletion
	deleteConfirming bool

	// Premium/premium screen
	premiumInputs   []textinput.Model
	premiumFocused  int
	premiumMsg      string
	premiumEnabled  bool
	premiumSyncing  bool
	premiumEmail    string
	premiumAPIURL   string
	premiumTier     string
	premiumFeatures []string

	// Quit confirmation
	quitConfirmSyncing bool

	// Sync status
	syncStatusMsg      string
	isSyncing          bool
	lastSyncStatusTime time.Time

	// Delete confirmation
	deleteConfirmDeleting bool

	// Subscription screen
	subscriptionList    list.Model
	subscriptionErr     string
	subscriptionMsg     string
	subscriptionLoading bool
	currentSubscription *api.Subscription
}

type updateInfo struct {
	version string
	url     string
	name    string
}

type appMenuItem struct {
	title       string
	description string
	action      screen
}

func (i appMenuItem) Title() string       { return i.title }
func (i appMenuItem) Description() string { return i.description }
func (i appMenuItem) FilterValue() string { return i.title }

func NewAppModel(savedEmail, savedPassword, savedServer string, currentVersion string) appModel {
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

	// Initialize premium inputs
	apiURLInput := textinput.New()
	apiURLInput.Placeholder = "https://api.newsletter-cli.apps.paas-01.pulseflow.cloud"
	apiURLInput.CharLimit = 200
	apiURLInput.Width = 50

	premiumEmailInput := textinput.New()
	premiumEmailInput.Placeholder = "your@email.com"
	premiumEmailInput.CharLimit = 100
	premiumEmailInput.Width = 50

	premiumPasswordInput := textinput.New()
	premiumPasswordInput.Placeholder = "Enter password"
	premiumPasswordInput.EchoMode = textinput.EchoPassword
	premiumPasswordInput.CharLimit = 100
	premiumPasswordInput.Width = 50

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

	// Always show Accounts option
	items = append(items, appMenuItem{
		title:       "👤 Accounts",
		description: "Manage email accounts",
		action:      screenAccounts,
	})

	// Add Premium option
	premiumDesc := "Enable cloud sync & premium features"
	if savedEmail != "" && savedPassword != "" && savedServer != "" {
		// Check if premium is enabled
		pc, _ := api.GetPremiumConfig()
		if pc != nil && pc.Enabled {
			premiumDesc = "☁️ Premium (Synced)"
		}
	}
	items = append(items, appMenuItem{
		title:       "☁️ Premium",
		description: premiumDesc,
		action:      screenPremium,
	})

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
	// Check if premium is enabled for title
	premiumConfig, _ := api.GetPremiumConfig()
	premiumBadge := ""
	if premiumConfig != nil && premiumConfig.Enabled {
		premiumBadge = " ☁️"
	}
	welcomeList.Title = "📬  Newsletter CLI" + premiumBadge
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

	// Initialize unsubscribed list
	unsubscribedList, _ := config.GetUnsubscribedList()

	// Check if premium is enabled
	pc, _ := api.GetPremiumConfig()
	premiumEnabled := pc != nil && pc.Enabled

	// Pre-fill premium inputs if configured
	if pc != nil {
		if pc.APIURL != "" {
			apiURLInput.SetValue(pc.APIURL)
		}
		if pc.Email != "" {
			premiumEmailInput.SetValue(pc.Email)
		}
	}

	return appModel{
		screen:                screenWelcome,
		welcomeList:           welcomeList,
		loginInputs:           []textinput.Model{emailInput, passwordInput, serverInput},
		loginFocused:          0,
		analyzeInputs:         []textinput.Model{daysInput},
		analyzeFocused:        0,
		analyzingSpinner:      sp,
		savedEmail:            savedEmail,
		savedPassword:         savedPassword,
		savedServer:           savedServer,
		currentVersion:        currentVersion,
		dashboardUnsubscribed: unsubscribedList,
		premiumInputs:         []textinput.Model{apiURLInput, premiumEmailInput, premiumPasswordInput},
		premiumFocused:        0,
		premiumEnabled:        premiumEnabled,
		premiumAPIURL: func() string {
			if premiumConfig != nil {
				return premiumConfig.APIURL
			}
			return ""
		}(),
		premiumEmail: func() string {
			if premiumConfig != nil {
				return premiumConfig.Email
			}
			return ""
		}(),
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

	// Start update check if on welcome screen and version is available
	if m.screen == screenWelcome && m.currentVersion != "" {
		cmds = append(cmds, m.checkForUpdate(m.currentVersion))
	}

	// Auto-sync on startup if premium enabled and setting is on
	if m.premiumEnabled {
		pc, _ := api.GetPremiumConfig()
		// Default to true if not set (for existing users)
		autoSyncOnStartup := true
		if pc != nil {
			// Check if all settings are unset (old config)
			if !pc.AutoSyncOnStartup && !pc.PeriodicSyncEnabled && pc.PeriodicSyncInterval == 0 && !pc.SyncAccounts && !pc.SyncUnsubscribed {
				// Old config - use defaults
				autoSyncOnStartup = true
			} else {
				autoSyncOnStartup = pc.AutoSyncOnStartup
			}
		}

		if autoSyncOnStartup {
			cmds = append(cmds, m.checkAndSyncOnStartup())
		}

		// Start periodic sync ticker if enabled
		periodicSyncEnabled := true
		periodicInterval := 5 * time.Minute
		if pc != nil {
			// Check if all settings are unset (old config)
			if !pc.AutoSyncOnStartup && !pc.PeriodicSyncEnabled && pc.PeriodicSyncInterval == 0 && !pc.SyncAccounts && !pc.SyncUnsubscribed {
				// Old config - use defaults
				periodicSyncEnabled = true
				periodicInterval = 5 * time.Minute
			} else {
				if pc.PeriodicSyncEnabled {
					periodicSyncEnabled = pc.PeriodicSyncEnabled
				}
				if pc.PeriodicSyncInterval > 0 {
					periodicInterval = time.Duration(pc.PeriodicSyncInterval) * time.Minute
				}
			}
		}

		if periodicSyncEnabled {
			cmds = append(cmds, tea.Tick(periodicInterval, func(t time.Time) tea.Msg {
				return periodicSyncTick{}
			}))
		}

		// Fetch subscription status on startup if premium enabled
		if pc != nil && pc.Enabled && pc.Token != "" {
			// Will be triggered when premium screen is viewed
		}
	}

	return tea.Batch(cmds...)
}

func (m appModel) checkForUpdate(currentVersion string) tea.Cmd {
	return func() tea.Msg {
		release, isNewer, err := update.CheckForUpdate(currentVersion)
		if err != nil || !isNewer {
			return updateCheckCompleteMsg{nil}
		}
		return updateCheckCompleteMsg{&updateInfo{
			version: release.TagName,
			url:     release.URL,
			name:    release.Name,
		}}
	}
}

func (m appModel) discoverServer(email string) tea.Cmd {
	return func() tea.Msg {
		server, err := imap.DiscoverIMAPServer(email)
		return serverDiscoveredMsg{server: server, err: err}
	}
}

type updateCheckCompleteMsg struct {
	update *updateInfo
}

type serverDiscoveredMsg struct {
	server string
	err    error
}

type unsubscribeResultMsg struct {
	results []unsubscribe.UnsubscribeResult
}

type periodicSyncTick struct{}

type autoSyncCompleteMsg struct {
	synced bool
	err    error
}

type quitSyncCompleteMsg struct {
	err error
}

type manualSyncCompleteMsg struct {
	err error
}

func (m appModel) checkAndSyncOnStartup() tea.Cmd {
	return func() tea.Msg {
		synced, err := api.CheckAndSyncIfNeeded()
		return autoSyncCompleteMsg{synced: synced, err: err}
	}
}

func (m appModel) periodicSync() tea.Cmd {
	return func() tea.Msg {
		err := api.PeriodicSync()
		if err != nil {
			// Silently log but don't show error to user
			return nil
		}
		return nil
	}
}

func (m appModel) syncBeforeQuit() tea.Cmd {
	return func() tea.Msg {
		err := api.PeriodicSync()
		return quitSyncCompleteMsg{err: err}
	}
}

func (m appModel) manualSync() tea.Cmd {
	return func() tea.Msg {
		err := api.PeriodicSync()
		return manualSyncCompleteMsg{err: err}
	}
}

func (m appModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	// Handle special messages first
	switch msg := msg.(type) {
	case periodicSyncTick:
		// Periodic sync tick - push local changes to cloud
		return m, m.periodicSync()
	case autoSyncCompleteMsg:
		// Auto-sync completed on startup - silently handle
		if msg.synced {
			m.lastSyncStatusTime = time.Now()
			m.syncStatusMsg = "✅ Synced"
			// Clear after 5 seconds
			go func() {
				time.Sleep(5 * time.Second)
				m.syncStatusMsg = ""
			}()
		}
		return m, nil
	case manualSyncCompleteMsg:
		// Manual sync completed
		m.isSyncing = false
		if msg.err != nil {
			m.syncStatusMsg = "❌ Sync failed: " + msg.err.Error()
		} else {
			m.lastSyncStatusTime = time.Now()
			m.syncStatusMsg = "✅ Synced"
			// Clear after 5 seconds
			go func() {
				time.Sleep(5 * time.Second)
				m.syncStatusMsg = ""
			}()
		}
		return m, nil
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
				title:       "👤 Accounts",
				description: "Manage email accounts",
				action:      screenAccounts,
			},
			appMenuItem{
				title:       "☁️ Premium",
				description: "Enable cloud sync & premium features",
				action:      screenPremium,
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

		// Load unsubscribed list
		unsubscribedList, _ := config.GetUnsubscribedList()
		m.dashboardUnsubscribed = unsubscribedList

		// Send analytics events (async, non-blocking)
		go func() {
			// Convert stats to analytics format
			analyticsStats := make([]api.NewsletterStatForAnalytics, 0, len(msg.stats))
			for _, s := range msg.stats {
				analyticsStats = append(analyticsStats, api.ConvertNewsletterStatsToAnalytics(
					s.Sender,
					s.Count,
					s.Unsubscribe,
				))
			}
			// Send analytics (silently fail if premium not enabled)
			_ = api.SendNewsletterAnalysisEvent(analyticsStats, m.savedEmail)
		}()

		// Create dashboard
		items := []list.Item{}
		totalEmails := 0

		// Check if premium is enabled AND user has active subscription (for categorization and quality scoring)
		premiumConfig, _ := api.GetPremiumConfig()
		hasPremiumConfig := premiumConfig != nil && premiumConfig.Enabled

		// Check if user has active subscription by checking license features
		isPremium := false
		if hasPremiumConfig {
			// Check subscription status by fetching features (which validates active subscription)
			features, err := api.GetLicenseFeatures()
			if err == nil {
				if tier, ok := features["tier"].(string); ok && tier != "" && tier != "free" {
					isPremium = true
				}
			}
		}

		// Prepare enrichment inputs for API call
		enrichInputs := make([]api.EnrichNewsletterInput, 0, len(msg.stats))
		for _, s := range msg.stats {
			enrichInputs = append(enrichInputs, api.EnrichNewsletterInput{
				Sender:         s.Sender,
				EmailCount:     s.Count,
				HasUnsubscribe: s.Unsubscribe != "",
			})
		}

		// Enrich newsletters using API (with caching)
		enrichedNewsletters := make(map[string]api.EnrichNewsletter)
		if isPremium && len(enrichInputs) > 0 {
			// Try to enrich via API (with caching)
			enriched, err := api.EnrichNewslettersWithCache(enrichInputs)
			if err == nil {
				for _, e := range enriched {
					enrichedNewsletters[e.Sender] = e
				}
			}
			// If API fails, silently fall back to showing without categories/scores
		}

		for _, s := range msg.stats {
			var category string
			var qualityScore int

			// Use enriched data if available
			if enriched, found := enrichedNewsletters[s.Sender]; found && isPremium {
				category = enriched.Category.Category
				qualityScore = enriched.QualityScore
			}

			items = append(items, dashboardListItem{
				title:        s.Sender,
				count:        s.Count,
				link:         s.Unsubscribe,
				selected:     m.dashboardSelected[s.Sender], // Preserve selection state
				unsubscribed: m.dashboardUnsubscribed[s.Sender],
				category:     category,
				qualityScore: qualityScore,
				isPremium:    isPremium,
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
		m.dashboardSelected = make(map[string]bool)
		// dashboardUnsubscribed already loaded above
		if m.dashboardUnsubscribed == nil {
			m.dashboardUnsubscribed = make(map[string]bool)
		}
		m.unsubscribing = false
		m.unsubscribeResults = nil
		m.totalEmails = totalEmails
		m.totalNewsletters = len(msg.stats)
		m.screen = screenDashboard
		m.errMsg = ""
		return m, nil

	case errorMsg:
		m.errMsg = string(msg)
		// Stay on current screen but show error
		// Ensure inputs remain focused and editable
		if m.screen == screenLogin {
			// Re-focus the current input field so user can continue editing
			for i := range m.loginInputs {
				if i == m.loginFocused {
					m.loginInputs[i].Focus()
				} else {
					m.loginInputs[i].Blur()
				}
			}
		}
		return m, nil

	case updateCheckCompleteMsg:
		m.updateAvailable = msg.update
		return m, nil

	case serverDiscoveredMsg:
		m.discoveringServer = false
		if msg.err != nil {
			m.serverStatusMsg = fmt.Sprintf("❌  Could not discover server: %v", msg.err)
			// Clear last discovered so we can retry
			m.lastDiscoveredEmail = ""
		} else {
			m.serverStatusMsg = fmt.Sprintf("✅ Discovered: %s", msg.server)
			// Auto-fill the server field
			m.loginInputs[2].SetValue(msg.server)
		}
		return m, nil
	}

	// Handle global shortcuts (before screen-specific handlers)
	if keyMsg, ok := msg.(tea.KeyMsg); ok && m.premiumEnabled {
		switch keyMsg.String() {
		case "ctrl+s":
			// Manual sync shortcut from any screen
			if !m.isSyncing {
				m.isSyncing = true
				m.syncStatusMsg = "☁️ Syncing..."
				return m, m.manualSync()
			}
			return m, nil
		}
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
	case screenAccounts:
		return m.updateAccounts(msg)
	case screenPremium:
		return m.updatePremium(msg)
	case screenQuitConfirm:
		return m.updateQuitConfirm(msg)
	case screenSyncSettings:
		return m.updateSyncSettings(msg)
	case screenDeleteConfirm:
		return m.updateDeleteConfirm(msg)
	case screenSubscription:
		return m.updateSubscription(msg)
	}

	return m, nil
}

func (m appModel) updateWelcome(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "q", "ctrl+c", "esc":
			// If premium enabled, ask if user wants to sync
			if m.premiumEnabled {
				m.screen = screenQuitConfirm
				return m, nil
			}
			return m, tea.Quit
		case "enter":
			i, ok := m.welcomeList.SelectedItem().(appMenuItem)
			if ok {
				if i.action == screenWelcome {
					// If premium enabled, ask if user wants to sync
					if m.premiumEnabled {
						m.screen = screenQuitConfirm
						return m, nil
					}
					return m, tea.Quit // Quit option
				}
				if i.action == screenAccounts {
					// Load accounts and initialize accounts screen
					accounts, err := config.GetAllAccounts()
					if err != nil {
						m.errMsg = "Failed to load accounts: " + err.Error()
						return m, nil
					}
					m.accounts = accounts
					m.screen = screenAccounts
					// Initialize accounts list
					return m.initAccountsList()
				}
				if i.action == screenPremium {
					m.screen = screenPremium
					m.premiumInputs[0].Focus()
					for i := 1; i < len(m.premiumInputs); i++ {
						m.premiumInputs[i].Blur()
					}
					m.premiumFocused = 0
					// Fetch license features and subscription status asynchronously if premium is enabled
					if m.premiumEnabled {
						return m, tea.Batch(m.fetchLicenseFeatures(), m.fetchSubscriptionStatus())
					}
					return m, nil
				}
				m.screen = i.action
				switch m.screen {
				case screenLogin:
					m.loginInputs[0].Focus()
					for i := 1; i < len(m.loginInputs); i++ {
						m.loginInputs[i].Blur()
					}
					// Try to discover server if email is already filled
					email := strings.TrimSpace(m.loginInputs[0].Value())
					if email != "" {
						m.discoveringServer = true
						m.serverStatusMsg = "🔍 Discovering IMAP server..."
						return m, m.discoverServer(email)
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
	case spinner.TickMsg:
		// Handle spinner updates during discovery
		if m.discoveringServer {
			var cmd tea.Cmd
			m.analyzingSpinner, cmd = m.analyzingSpinner.Update(msg)
			return m, cmd
		}
		return m, nil

	case tea.KeyMsg:
		switch msg.String() {
		case "esc", "ctrl+c":
			m.screen = screenWelcome
			m.discoveringServer = false
			m.serverStatusMsg = ""
			return m, nil
		case "ctrl+r":
			// Retry server discovery (Ctrl+R to avoid conflicts with typing 'r' in email)
			email := strings.TrimSpace(m.loginInputs[0].Value())
			if email != "" {
				m.lastDiscoveredEmail = "" // Clear so it will check again
				m.discoveringServer = true
				m.serverStatusMsg = "🔍 Discovering IMAP server..."
				return m, m.discoverServer(email)
			}
		case "tab", "shift+tab", "enter", "up", "down":
			// Handle tab/enter navigation
			if msg.String() == "enter" && m.loginFocused == len(m.loginInputs)-1 {
				// Submit login
				return m, m.submitLogin()
			}

			// Trigger discovery when leaving email field (tab/down) or pressing enter on email field
			if m.loginFocused == 0 && (msg.String() == "tab" || msg.String() == "down" || msg.String() == "enter") {
				email := strings.TrimSpace(m.loginInputs[0].Value())
				if email != "" && strings.Contains(email, "@") && strings.Count(email, "@") == 1 && !m.discoveringServer {
					parts := strings.Split(email, "@")
					if len(parts) == 2 {
						domain := strings.TrimSpace(parts[1])
						if domain != "" && strings.Contains(domain, ".") {
							// Try discovery when leaving email field or pressing enter
							m.discoveringServer = true
							m.serverStatusMsg = "🔍 Discovering IMAP server..."
							m.lastDiscoveredEmail = email // Track to avoid re-checking if they come back
							// If tab/down, switch to next field; if enter, stay on email field
							if msg.String() != "enter" {
								m.loginFocused++
								for i := range m.loginInputs {
									if i == m.loginFocused {
										m.loginInputs[i].Focus()
									} else {
										m.loginInputs[i].Blur()
									}
								}
							}
							return m, m.discoverServer(email)
						}
					}
				}
				// If discovery didn't trigger, continue with normal field navigation
				if msg.String() == "enter" {
					// Stay on email field if enter was pressed
					return m, nil
				}
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

	// Clear error message when user starts typing
	if m.errMsg != "" {
		// Check if this is a keypress that would modify input
		if keyMsg, ok := msg.(tea.KeyMsg); ok {
			switch keyMsg.String() {
			case "tab", "shift+tab", "enter", "esc", "ctrl+c", "ctrl+r":
				// Navigation keys - don't clear error
			default:
				// User is typing - clear the error
				m.errMsg = ""
			}
		}
	}

	// Update focused input
	var cmd tea.Cmd
	m.loginInputs[m.loginFocused], cmd = m.loginInputs[m.loginFocused].Update(msg)

	// Server discovery only happens when switching away from email field (handled above)
	// No automatic discovery while typing

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
	// Handle unsubscribe results
	if msg, ok := msg.(unsubscribeResultMsg); ok {
		m.unsubscribing = false
		m.unsubscribeResults = []unsubscribeResultMsg{msg}

		// Build result summary
		successCount := 0
		failCount := 0
		for _, result := range msg.results {
			if result.Success {
				successCount++
				// Remove from selected after successful unsubscribe
				delete(m.dashboardSelected, result.Sender)
				// Save to unsubscribed list
				m.dashboardUnsubscribed[result.Sender] = true
				config.AddUnsubscribed(result.Sender)
				// Send analytics event (async, non-blocking)
				go func(sender string) {
					_ = api.SendUnsubscribeEvent(sender, true, m.savedEmail)
				}(result.Sender)
				// Auto-sync to cloud if premium enabled
				go func() {
					_ = api.AutoSync() // Silently fail if premium not enabled
				}()
			} else {
				failCount++
				// Send analytics event for failed unsubscribe
				go func(sender string) {
					_ = api.SendUnsubscribeEvent(sender, false, m.savedEmail)
				}(result.Sender)
			}
		}

		if successCount > 0 {
			m.dashboardMsg = fmt.Sprintf("✅ Successfully unsubscribed from %d newsletter(s)", successCount)
		}
		if failCount > 0 {
			m.dashboardMsg += fmt.Sprintf(" | ❌ Failed: %d", failCount)
		}

		// Update list items to reflect unsubscribed status
		items := m.dashboardList.Items()
		for idx, item := range items {
			if item, ok := item.(dashboardListItem); ok {
				item.unsubscribed = m.dashboardUnsubscribed[item.title]
				items[idx] = item
			}
		}
		m.dashboardList.SetItems(items)

		return m, nil
	}

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "q", "ctrl+c":
			return m, tea.Quit
		case " ": // Spacebar for multiselect
			if m.unsubscribing {
				return m, nil // Don't allow selection while unsubscribing
			}
			i, ok := m.dashboardList.SelectedItem().(dashboardListItem)
			if ok {
				// Toggle selection
				if m.dashboardSelected[i.title] {
					delete(m.dashboardSelected, i.title)
				} else {
					m.dashboardSelected[i.title] = true
				}
				// Update the list item to reflect selection state
				items := m.dashboardList.Items()
				for idx, item := range items {
					if item, ok := item.(dashboardListItem); ok && item.title == i.title {
						item.selected = m.dashboardSelected[i.title]
						items[idx] = item
					}
				}
				m.dashboardList.SetItems(items)
			}
			return m, nil
		case "u":
			// Single unsubscribe (open browser)
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
		case "U": // Shift+U or uppercase U for mass unsubscribe
			selectedCount := len(m.dashboardSelected)
			if selectedCount == 0 {
				m.dashboardMsg = "⚠️  No newsletters selected. Use [Space] to select items."
				return m, nil
			}

			if m.unsubscribing {
				return m, nil
			}

			// Start mass unsubscribe
			m.unsubscribing = true
			m.dashboardMsg = fmt.Sprintf("🔄 Unsubscribing from %d newsletter(s)...", selectedCount)
			return m, m.batchUnsubscribe()
		case "/":
			m.dashboardList.ResetSelected()
			return m, nil
		case "esc":
			if m.dashboardList.FilterState() == list.Filtering {
				m.dashboardList.ResetFilter()
				return m, nil
			}
			// Clear selection on escape
			m.dashboardSelected = make(map[string]bool)
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

		// Check if this would be adding a second+ account (first account is free)
		cfg, _ := config.Load()
		if cfg != nil && len(cfg.Accounts) > 0 {
			// Check if account already exists (updating is allowed)
			accountExists := false
			for _, acc := range cfg.Accounts {
				if acc.Email == email {
					accountExists = true
					break
				}
			}

			// If adding a new account (not updating), check account limit
			if !accountExists {
				canAdd, reason := api.CanAddAccount(len(cfg.Accounts))
				if !canAdd {
					return errorMsg("⭐ " + reason + "\n\nNavigate to '☁️ Premium' to upgrade, or press [Esc] to go back.")
				}
			}
		}

		// Test connection
		if err := imap.ConnectIMAP(email, password, server); err != nil {
			return errorMsg("Connection failed: " + err.Error())
		}

		// Save account (use email as name if not provided)
		_, err := config.AddAccount(email, server, password, email)
		if err != nil {
			return errorMsg("Failed to save account: " + err.Error())
		}

		// Auto-sync to cloud if premium enabled
		go func() {
			_ = api.AutoSync() // Silently fail if premium not enabled
		}()

		return loginSuccessMsg{
			email:    email,
			password: password,
			server:   server,
		}
	}
}

func (m appModel) batchUnsubscribe() tea.Cmd {
	return func() tea.Msg {
		// Build unsubscribe requests from selected items
		var requests []struct {
			Sender string
			Link   string
		}

		for _, stat := range m.dashboardStats {
			if m.dashboardSelected[stat.Sender] {
				requests = append(requests, struct {
					Sender string
					Link   string
				}{
					Sender: stat.Sender,
					Link:   stat.Unsubscribe,
				})
			}
		}

		if len(requests) == 0 {
			return unsubscribeResultMsg{results: []unsubscribe.UnsubscribeResult{}}
		}

		// Pass credentials for mailto: links
		results := unsubscribe.BatchUnsubscribe(requests, m.savedEmail, m.savedPassword, m.savedServer)
		return unsubscribeResultMsg{results: results}
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
	case screenAccounts:
		view = m.viewAccounts()
	case screenPremium:
		view = m.viewPremium()
	case screenQuitConfirm:
		view = m.viewQuitConfirm()
	case screenSyncSettings:
		view = m.viewSyncSettings()
	case screenDeleteConfirm:
		view = m.viewDeleteConfirm()
	case screenSubscription:
		view = m.viewSubscription()
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

	// Update title with version if available
	if m.currentVersion != "" {
		// Add premium badge if enabled
		premiumConfig, _ := api.GetPremiumConfig()
		premiumBadge := ""
		if premiumConfig != nil && premiumConfig.Enabled {
			premiumBadge = " ☁️"
		}
		m.welcomeList.Title = fmt.Sprintf("📬  Newsletter CLI v%s%s", m.currentVersion, premiumBadge)
	}

	listView := docStyle.Render(m.welcomeList.View())

	// Show update notification if available
	updateNotice := ""
	if m.updateAvailable != nil {
		updateStyle := lipgloss.NewStyle().
			Foreground(lipgloss.Color("220")).
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("220")).
			Padding(0, 1).
			MarginTop(1)
		updateNotice = "\n" + updateStyle.Render(
			fmt.Sprintf("✨ Update available: %s\n   Visit: %s",
				m.updateAvailable.version, m.updateAvailable.url),
		)
	}

	// Show sync status if premium enabled
	syncStatusText := ""
	if m.premiumEnabled {
		if m.isSyncing {
			syncStatusText = "\n" + lipgloss.NewStyle().
				Foreground(lipgloss.Color("14")).
				Render("☁️ Syncing...")
		} else if m.syncStatusMsg != "" {
			syncStatusText = "\n" + lipgloss.NewStyle().
				Foreground(lipgloss.Color("10")).
				Render(m.syncStatusMsg)
		} else {
			pc, _ := api.GetPremiumConfig()
			if pc != nil && !pc.LastSyncTime.IsZero() {
				syncTime := formatTimeAgoSync(pc.LastSyncTime)
				syncStatusText = "\n" + lipgloss.NewStyle().
					Foreground(lipgloss.Color("241")).
					Render(fmt.Sprintf("☁️ Last sync: %s", syncTime))
			}
		}
	}

	helpText := "[↑↓] Navigate  [Enter] Select  [q/Esc] Quit"
	if m.premiumEnabled {
		helpText = "[↑↓] Navigate  [Enter] Select  [Ctrl+S] Sync  [q/Esc] Quit"
	}
	help := helpStyle.Render(helpText)

	return docStyle.Render(intro + "\n\n" + listView + updateNotice + syncStatusText + "\n" + help)
}

// formatTimeAgoSync formats time for sync status (shorter format)
func formatTimeAgoSync(t time.Time) string {
	now := time.Now()
	diff := now.Sub(t)

	if diff < time.Minute {
		return "just now"
	} else if diff < time.Hour {
		minutes := int(diff.Minutes())
		return fmt.Sprintf("%dm ago", minutes)
	} else if diff < 24*time.Hour {
		hours := int(diff.Hours())
		return fmt.Sprintf("%dh ago", hours)
	} else {
		days := int(diff.Hours() / 24)
		return fmt.Sprintf("%dd ago", days)
	}
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

	// Show server discovery status
	statusMsg := ""
	if m.discoveringServer || m.serverStatusMsg != "" {
		statusStyle := lipgloss.NewStyle().
			Foreground(lipgloss.Color("220")).
			MarginTop(1)
		if m.discoveringServer {
			statusMsg = "\n" + statusStyle.Render(m.analyzingSpinner.View()+" "+m.serverStatusMsg)
		} else if m.serverStatusMsg != "" {
			if strings.HasPrefix(m.serverStatusMsg, "✅") {
				statusStyle = statusStyle.Foreground(lipgloss.Color("82"))
			} else {
				statusStyle = statusStyle.Foreground(lipgloss.Color("196"))
			}
			statusMsg = "\n" + statusStyle.Render(m.serverStatusMsg)
		}
	}

	help := helpStyle.Render("[Tab] Next  [Shift+Tab] Previous  [Ctrl+R] Retry Discovery  [Enter] Submit  [Esc] Back")

	return docStyle.Render(content + statusMsg + "\n\n" + help)
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

	// Update list items to reflect selection and unsubscribed state
	items := m.dashboardList.Items()
	for idx, item := range items {
		if item, ok := item.(dashboardListItem); ok {
			item.selected = m.dashboardSelected[item.title]
			item.unsubscribed = m.dashboardUnsubscribed[item.title]
			items[idx] = item
		}
	}
	m.dashboardList.SetItems(items)

	selectedCount := len(m.dashboardSelected)
	summaryText := fmt.Sprintf("Total: %d newsletters • %d emails", m.totalNewsletters, m.totalEmails)
	if selectedCount > 0 {
		selectedStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("46")).Bold(true)
		summaryText += fmt.Sprintf(" • %s selected", selectedStyle.Render(fmt.Sprintf("%d", selectedCount)))
	}
	summary := headerStyle.Render(summaryText)

	listView := docStyle.Render(m.dashboardList.View())

	status := ""
	if m.dashboardMsg != "" {
		var msgStyle lipgloss.Style
		if m.unsubscribing {
			msgStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("63")).Bold(true).Padding(0, 1)
		} else {
			msgStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("220")).Padding(0, 1)
		}
		status = "\n" + msgStyle.Render(m.dashboardMsg)
	}

	helpText := "[↑↓] Navigate  [Space] Select  [u] Single  [U] Mass Unsubscribe  [/] Search  [Esc] Clear  [q] Quit"
	if m.unsubscribing {
		helpText = "[🔄 Unsubscribing... Please wait]"
	}
	help := helpStyle.Render(helpText)

	return summary + "\n" + listView + status + "\n" + help
}

type dashboardListItem struct {
	title        string
	count        int
	link         string
	selected     bool   // Track if this item is selected
	unsubscribed bool   // Track if this newsletter is already unsubscribed
	category     string // Newsletter category (premium only)
	qualityScore int    // Quality score 0-100 (premium only)
	isPremium    bool   // Whether premium features should be shown
}

func (i dashboardListItem) Title() string {
	countStr := strconv.Itoa(i.count)
	color := getCountColor(i.count)
	countStyle := lipgloss.NewStyle().Foreground(color).Bold(true)

	// Add prefix based on state
	prefix := ""
	if i.unsubscribed {
		prefix = "✓✓ " // Double checkmark for unsubscribed
	} else if i.selected {
		prefix = "✓ " // Single checkmark for selected
	}

	// Add quality score stars (⭐) for high scores (premium only)
	stars := ""
	if i.isPremium && i.qualityScore >= 80 {
		stars = " ⭐⭐⭐⭐⭐"
	} else if i.isPremium && i.qualityScore >= 70 {
		stars = " ⭐⭐⭐⭐"
	} else if i.isPremium && i.qualityScore >= 60 {
		stars = " ⭐⭐⭐"
	} else if i.isPremium && i.qualityScore >= 50 {
		stars = " ⭐⭐"
	} else if i.isPremium && i.qualityScore >= 40 {
		stars = " ⭐"
	}

	// Style unsubscribed items differently
	var titleStyle lipgloss.Style
	if i.unsubscribed {
		titleStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("240")).Strikethrough(true)
		return prefix + titleStyle.Render(i.title) + stars + "  " + countStyle.Render(fmt.Sprintf("(%s)", countStr))
	}

	return prefix + i.title + stars + "  " + countStyle.Render(fmt.Sprintf("(%s)", countStr))
}

func (i dashboardListItem) Description() string {
	desc := fmt.Sprintf("%d email", i.count)
	if i.count != 1 {
		desc += "s"
	}

	// Show unsubscribed status
	if i.unsubscribed {
		status := desc + "  •  ✅ Already unsubscribed"
		if i.isPremium && i.category != "" {
			status += "  •  📂 " + i.category
		}
		if i.isPremium && i.qualityScore > 0 {
			status += fmt.Sprintf("  •  Score: %d/100", i.qualityScore)
		}
		return status
	}

	// Build description with quality info (premium only)
	var parts []string
	parts = append(parts, desc)

	// Add category (premium only)
	if i.isPremium && i.category != "" {
		parts = append(parts, "📂 "+i.category)
	}

	// Add quality score (premium only)
	if i.isPremium && i.qualityScore > 0 {
		var scoreColor lipgloss.Color
		if i.qualityScore >= 80 {
			scoreColor = lipgloss.Color("10") // Green
		} else if i.qualityScore >= 60 {
			scoreColor = lipgloss.Color("11") // Yellow
		} else {
			scoreColor = lipgloss.Color("9") // Red
		}
		scoreStyle := lipgloss.NewStyle().Foreground(scoreColor).Bold(true)
		parts = append(parts, "⭐ "+scoreStyle.Render(fmt.Sprintf("%d/100", i.qualityScore)))
	}

	// Add unsubscribe link status
	if i.link != "" {
		linkDisplay := i.link
		if len(linkDisplay) > 40 {
			linkDisplay = linkDisplay[:37] + "..."
		}
		parts = append(parts, "🔗 "+linkDisplay)
	} else {
		parts = append(parts, "⚠️  No unsubscribe link")
	}

	return strings.Join(parts, "  •  ")
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

// Account list item
type accountListItem struct {
	account config.Account
}

func (i accountListItem) Title() string {
	prefix := ""
	cfg, _ := config.Load()
	if cfg != nil && cfg.SelectedID == i.account.ID {
		prefix = "✓ "
	}
	return prefix + i.account.Name
}

func (i accountListItem) Description() string {
	desc := i.account.Email + " @ " + i.account.Server
	cfg, _ := config.Load()
	if cfg != nil && cfg.SelectedID == i.account.ID {
		desc += " (active)"
	}
	return desc
}

func (i accountListItem) FilterValue() string {
	return i.account.Name + " " + i.account.Email
}

// initAccountsList initializes the accounts list
func (m appModel) initAccountsList() (tea.Model, tea.Cmd) {
	items := []list.Item{}
	for _, acc := range m.accounts {
		items = append(items, accountListItem{account: acc})
	}

	delegate := list.NewDefaultDelegate()
	delegate.Styles.SelectedTitle = delegate.Styles.SelectedTitle.
		Foreground(lipgloss.Color("229")).
		Bold(true)
	delegate.Styles.SelectedDesc = delegate.Styles.SelectedDesc.
		Foreground(lipgloss.Color("219"))

	l := list.New(items, delegate, 0, 0)
	l.Title = "👤  Manage Accounts"
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

	m.accountsList = l
	m.accountsMsg = ""
	m.deleteConfirming = false
	m.accountToDelete = ""

	return m, nil
}

// updateAccounts handles the accounts screen
func (m appModel) updateAccounts(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		h, v := docStyle.GetFrameSize()
		m.accountsList.SetSize(msg.Width-h, msg.Height-v-7)
		return m, nil

	case tea.KeyMsg:
		switch msg.String() {
		case "q", "ctrl+c":
			return m, tea.Quit
		case "esc":
			if m.deleteConfirming {
				m.deleteConfirming = false
				m.accountToDelete = ""
				return m, nil
			}
			if m.accountsList.FilterState() == list.Filtering {
				m.accountsList.ResetFilter()
				return m, nil
			}
			m.screen = screenWelcome
			return m, nil
		case "enter":
			if m.deleteConfirming {
				// Confirm deletion
				if err := config.DeleteAccount(m.accountToDelete); err != nil {
					m.accountsMsg = "❌ Failed to delete account: " + err.Error()
				} else {
					m.accountsMsg = "✅ Account deleted"
					// Reload accounts
					accounts, _ := config.GetAllAccounts()
					m.accounts = accounts
					// Reinitialize list
					return m.initAccountsList()
				}
				m.deleteConfirming = false
				m.accountToDelete = ""
				return m, nil
			}
			// Select account
			i, ok := m.accountsList.SelectedItem().(accountListItem)
			if ok {
				if err := config.SetSelectedAccount(i.account.ID); err != nil {
					m.accountsMsg = "❌ Failed to select account: " + err.Error()
				} else {
					m.accountsMsg = "✅ Selected account: " + i.account.Name

					// Update saved credentials to the selected account
					m.savedEmail = i.account.Email
					m.savedServer = i.account.Server
					decryptedPassword, err := config.Decrypt(i.account.Password)
					if err != nil {
						m.accountsMsg = "⚠️  Selected account but failed to decrypt password"
						m.savedPassword = ""
					} else {
						m.savedPassword = decryptedPassword
					}

					// Update welcome list to show Analyze option if credentials are available
					items := []list.Item{
						appMenuItem{
							title:       "🔐 Login",
							description: "Save your IMAP credentials",
							action:      screenLogin,
						},
					}
					if m.savedEmail != "" && m.savedPassword != "" && m.savedServer != "" {
						items = append(items, appMenuItem{
							title:       "📊 Analyze",
							description: "Analyze and manage newsletters",
							action:      screenAnalyzeInput,
						})
					}
					items = append(items, appMenuItem{
						title:       "👤 Accounts",
						description: "Manage email accounts",
						action:      screenAccounts,
					})
					items = append(items, appMenuItem{
						title:       "❌ Quit",
						description: "Exit the application",
						action:      screenWelcome,
					})
					m.welcomeList.SetItems(items)

					// Reload accounts list to update active indicator
					accounts, _ := config.GetAllAccounts()
					m.accounts = accounts
					updated, cmd := m.initAccountsList()
					m = updated.(appModel)
					return m, cmd
				}
			}
			return m, nil
		case "d":
			if m.deleteConfirming {
				return m, nil
			}
			// Delete account
			i, ok := m.accountsList.SelectedItem().(accountListItem)
			if ok {
				cfg, _ := config.Load()
				if cfg != nil && len(cfg.Accounts) <= 1 {
					m.accountsMsg = "⚠️  Cannot delete the last account"
					return m, nil
				}
				m.accountToDelete = i.account.ID
				m.deleteConfirming = true
				m.accountsMsg = fmt.Sprintf("⚠️  Delete %s? Press Enter to confirm, Esc to cancel", i.account.Name)
			}
			return m, nil
		case "a":
			// Add new account (go to login screen)
			// Check if this would be adding a second+ account (first account is free)
			cfg, _ := config.Load()
			if cfg != nil && len(cfg.Accounts) > 0 {
				// Check account limit based on subscription tier
				canAdd, reason := api.CanAddAccount(len(cfg.Accounts))
				if !canAdd {
					m.accountsMsg = "⭐ " + reason + "\nPress 'p' to go to Premium, or [Esc] to go back."
					return m, nil
				}
			}
			m.screen = screenLogin
			// Clear login inputs
			m.loginInputs[0].SetValue("")
			m.loginInputs[1].SetValue("")
			m.loginInputs[2].SetValue("")
			m.loginInputs[0].Focus()
			for i := 1; i < len(m.loginInputs); i++ {
				m.loginInputs[i].Blur()
			}
			return m, nil
		case "p":
			// Navigate to premium screen
			m.screen = screenPremium
			m.accountsMsg = "" // Clear any messages
			return m, nil
		case "/":
			m.accountsList.ResetSelected()
			return m, nil
		}
	}

	var cmd tea.Cmd
	m.accountsList, cmd = m.accountsList.Update(msg)
	return m, cmd
}

// viewAccounts renders the accounts screen
func (m appModel) viewAccounts() string {
	if len(m.accounts) == 0 {
		emptyMsg := "No accounts configured\n\nPress 'a' to add an account"
		if m.deleteConfirming {
			emptyMsg = "Cannot delete - no accounts available"
		}
		return docStyle.Render(
			emptyStateStyle.Render(emptyMsg) + "\n\n" +
				helpStyle.Render("Press 'a' to add account  [Esc] Back  [q] Quit"),
		)
	}

	listView := docStyle.Render(m.accountsList.View())

	status := ""
	if m.accountsMsg != "" {
		msgStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("220")).Padding(0, 1)
		status = "\n" + msgStyle.Render(m.accountsMsg)
	}

	helpText := "[↑↓] Navigate  [Enter] Select  [a] Add  [d] Delete  [p] Premium  [/] Search  [Esc] Back  [q] Quit"
	if m.deleteConfirming {
		helpText = "[Enter] Confirm Delete  [Esc] Cancel"
	}
	help := helpStyle.Render(helpText)

	return listView + status + "\n" + help
}

// RunAppSync runs the app synchronously (for use from commands)
// initialScreen can be "login", "analyze", or "" for welcome
func RunAppSync(savedEmail, savedPassword, savedServer string, days int, flagsProvided bool, initialScreen string, currentVersion string) error {
	m := NewAppModel(savedEmail, savedPassword, savedServer, currentVersion)

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
