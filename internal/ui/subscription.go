package ui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/loickal/newsletter-cli/internal/api"
)

var errorStyle = lipgloss.NewStyle().
	Foreground(lipgloss.Color("196")).
	Padding(0, 1)

type planItem struct {
	id       string
	name     string
	amount   int64
	interval string
}

func (i planItem) FilterValue() string { return i.name }
func (i planItem) Title() string {
	price := fmt.Sprintf("$%.2f", float64(i.amount)/100)
	return fmt.Sprintf("%s - %s/%s", i.name, price, i.interval)
}
func (i planItem) Description() string {
	features := getPlanFeatures(i.id)
	return strings.Join(features, " • ")
}

func getPlanFeatures(planID string) []string {
	features := map[string][]string{
		"starter": {
			"Cloud Sync",
			"Basic Analytics",
			"Web Dashboard",
		},
		"pro": {
			"Everything in Starter",
			"Smart Scheduling",
			"Advanced Analytics",
			"Integrations",
		},
		"enterprise": {
			"Everything in Pro",
			"Team Workspaces",
			"Compliance Reporting",
			"Priority Support",
		},
	}
	if f, ok := features[planID]; ok {
		return f
	}
	return []string{}
}

// Subscription UI functions are methods on appModel, not a separate model

func (m *appModel) initSubscriptionList() {
	items := []list.Item{
		planItem{id: "starter", name: "Starter", amount: 500, interval: "month"},
		planItem{id: "pro", name: "Pro", amount: 1200, interval: "month"},
		planItem{id: "enterprise", name: "Enterprise", amount: 5000, interval: "month"},
	}

	delegate := list.NewDefaultDelegate()
	delegate.Styles.SelectedTitle = delegate.Styles.SelectedTitle.
		Foreground(lipgloss.Color("229")).
		Bold(true)
	delegate.Styles.SelectedDesc = delegate.Styles.SelectedDesc.
		Foreground(lipgloss.Color("219"))

	// Use actual dimensions if available, otherwise default
	width := 50
	height := 14
	if m.width > 0 && m.height > 0 {
		width = m.width - 4
		height = m.height - 10 // Leave room for header and help text
	}

	l := list.New(items, delegate, width, height)
	l.Title = "Select Subscription Plan"
	l.SetShowStatusBar(false)
	l.SetFilteringEnabled(false)
	m.subscriptionList = l
}

func (m *appModel) initSubscription() tea.Cmd {
	return func() tea.Msg {
		client, err := api.GetAPIClient()
		if err != nil {
			return subscriptionPlansMsg{
				err: err.Error(),
			}
		}

		plans, err := client.GetPlans()
		if err != nil {
			return subscriptionPlansMsg{
				err: err.Error(),
			}
		}

		return subscriptionPlansMsg{
			plans: plans,
		}
	}
}

type subscriptionPlansMsg struct {
	plans []api.Plan
	err   string
}

type subscriptionCheckoutMsg struct {
	checkoutURL string
	err         string
}

func (m appModel) updateSubscription(msg tea.Msg) (tea.Model, tea.Cmd) {
	if m.subscriptionList.Items() == nil || len(m.subscriptionList.Items()) == 0 {
		// Initialize list if not already done
		m.initSubscriptionList()
	}

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		// Update list dimensions
		width := msg.Width - 4
		height := msg.Height - 10
		m.subscriptionList.SetWidth(width)
		m.subscriptionList.SetHeight(height)
		return m, nil
	case tea.KeyMsg:
		switch msg.String() {
		case "esc", "q":
			m.screen = screenPremium
			return m, nil
		case "enter":
			selected := m.subscriptionList.SelectedItem()
			if selected == nil {
				return m, nil
			}
			plan, ok := selected.(planItem)
			if !ok {
				return m, nil
			}
			// Create checkout session
			m.subscriptionLoading = true
			return m, m.createCheckoutSession(plan.id)
		}
	case subscriptionPlansMsg:
		m.subscriptionLoading = false
		if msg.err != "" {
			m.subscriptionErr = msg.err
			return m, nil
		}
		// Update list items with actual plans from API
		items := make([]list.Item, len(msg.plans))
		for i, plan := range msg.plans {
			items[i] = planItem{
				id:       plan.ID,
				name:     plan.Name,
				amount:   plan.Amount,
				interval: plan.Interval,
			}
		}
		m.subscriptionList.SetItems(items)
		// Ensure list has proper dimensions
		if m.width > 0 && m.height > 0 {
			m.subscriptionList.SetWidth(m.width - 4)
			m.subscriptionList.SetHeight(m.height - 10)
		}
		return m, nil
	case subscriptionCheckoutMsg:
		m.subscriptionLoading = false
		if msg.err != "" {
			m.subscriptionErr = msg.err
			return m, nil
		}
		// Open browser with checkout URL
		if err := openBrowser(msg.checkoutURL); err != nil {
			m.subscriptionErr = "Failed to open browser: " + err.Error()
			return m, nil
		}
		m.subscriptionErr = ""
		m.subscriptionMsg = fmt.Sprintf("✅ Opening checkout page in browser...\n   Complete payment to activate subscription.")
		return m, nil
	}

	var cmd tea.Cmd
	m.subscriptionList, cmd = m.subscriptionList.Update(msg)
	return m, cmd
}

func (m appModel) createCheckoutSession(planID string) tea.Cmd {
	return func() tea.Msg {
		client, err := api.GetAPIClient()
		if err != nil {
			return subscriptionCheckoutMsg{
				err: err.Error(),
			}
		}

		session, err := client.CreateCheckoutSession(planID)
		if err != nil {
			return subscriptionCheckoutMsg{
				err: err.Error(),
			}
		}

		return subscriptionCheckoutMsg{
			checkoutURL: session.CheckoutURL,
		}
	}
}

func (m appModel) viewSubscription() string {
	if m.subscriptionLoading {
		return docStyle.Render(
			lipgloss.JoinVertical(lipgloss.Left,
				titleStyle.Render("💳 Subscribe"),
				"\n",
				m.analyzingSpinner.View()+" Loading plans...",
			),
		)
	}

	var content strings.Builder
	content.WriteString(titleStyle.Render("💳 Subscribe"))

	if m.subscriptionErr != "" {
		content.WriteString("\n\n")
		content.WriteString(errorStyle.Render("❌ " + m.subscriptionErr))
	}

	if m.subscriptionMsg != "" {
		content.WriteString("\n\n")
		content.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color("10")).Render(m.subscriptionMsg))
	}

	content.WriteString("\n\n")
	if len(m.subscriptionList.Items()) > 0 {
		content.WriteString(m.subscriptionList.View())
	} else {
		content.WriteString("Loading plans...")
	}
	content.WriteString("\n\n")
	content.WriteString(helpStyle.Render("[Enter] Select plan  [Esc] Back  [q] Quit"))

	return docStyle.Render(content.String())
}
