package app

import (
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/robertguss/bmad-automate-go/internal/components/header"
	"github.com/robertguss/bmad-automate-go/internal/components/statusbar"
	"github.com/robertguss/bmad-automate-go/internal/config"
	"github.com/robertguss/bmad-automate-go/internal/domain"
	"github.com/robertguss/bmad-automate-go/internal/messages"
	"github.com/robertguss/bmad-automate-go/internal/parser"
	"github.com/robertguss/bmad-automate-go/internal/theme"
	"github.com/robertguss/bmad-automate-go/internal/views/dashboard"
	"github.com/robertguss/bmad-automate-go/internal/views/storylist"
)

// Model is the main application model
type Model struct {
	// Dimensions
	width  int
	height int
	ready  bool

	// Navigation
	activeView domain.View
	prevView   domain.View

	// Configuration
	config *config.Config

	// Data
	stories []domain.Story
	err     error

	// Components
	header    header.Model
	statusbar statusbar.Model

	// Views
	dashboard dashboard.Model
	storylist storylist.Model

	// Styles
	styles theme.Styles
}

// New creates a new application model
func New(cfg *config.Config) Model {
	return Model{
		activeView: domain.ViewDashboard,
		config:     cfg,
		header:     header.New(),
		statusbar:  statusbar.New(),
		dashboard:  dashboard.New(),
		storylist:  storylist.New(),
		styles:     theme.NewStyles(),
	}
}

// Init initializes the application
func (m Model) Init() tea.Cmd {
	return m.loadStories
}

// loadStories loads stories from sprint-status.yaml
func (m Model) loadStories() tea.Msg {
	stories, err := parser.ParseSprintStatus(m.config)
	return messages.StoriesLoadedMsg{Stories: stories, Error: err}
}

// Update handles all messages
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		// Global key handling
		switch msg.String() {
		case "ctrl+c", "ctrl+q":
			return m, tea.Quit

		case "?":
			// TODO: Show help modal
			return m, nil

		// View navigation
		case "d":
			m.prevView = m.activeView
			m.activeView = domain.ViewDashboard
			m.header.SetActiveView(m.activeView)
			return m, nil

		case "s":
			m.prevView = m.activeView
			m.activeView = domain.ViewStoryList
			m.header.SetActiveView(m.activeView)
			return m, nil

		case "q":
			m.prevView = m.activeView
			m.activeView = domain.ViewQueue
			m.header.SetActiveView(m.activeView)
			return m, nil

		case "h":
			m.prevView = m.activeView
			m.activeView = domain.ViewHistory
			m.header.SetActiveView(m.activeView)
			return m, nil

		case "a":
			// Only navigate to stats if not in storylist (where 'a' means select all)
			if m.activeView != domain.ViewStoryList {
				m.prevView = m.activeView
				m.activeView = domain.ViewStats
				m.header.SetActiveView(m.activeView)
				return m, nil
			}

		case "o":
			m.prevView = m.activeView
			m.activeView = domain.ViewSettings
			m.header.SetActiveView(m.activeView)
			return m, nil

		case "esc":
			// Go back to previous view or dashboard
			if m.activeView != domain.ViewDashboard {
				m.activeView = m.prevView
				if m.activeView == m.activeView { // same view
					m.activeView = domain.ViewDashboard
				}
				m.header.SetActiveView(m.activeView)
			}
			return m, nil
		}

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.ready = true

		// Update component sizes
		m.header.SetWidth(msg.Width)
		m.statusbar.SetWidth(msg.Width)

		// Calculate content height (total - header - statusbar)
		contentHeight := msg.Height - 4 // header(2) + statusbar(2)

		m.dashboard.SetSize(msg.Width, contentHeight)
		m.storylist.SetSize(msg.Width, contentHeight)

		// Propagate to views
		sizeMsg := messages.WindowSizeMsg{Width: msg.Width, Height: contentHeight}
		m.dashboard, _ = m.dashboard.Update(sizeMsg)
		m.storylist, _ = m.storylist.Update(sizeMsg)

	case messages.StoriesLoadedMsg:
		if msg.Error != nil {
			m.err = msg.Error
			m.statusbar.SetMessage(fmt.Sprintf("Error: %v", msg.Error))
		} else {
			m.stories = msg.Stories
			m.statusbar.SetStoryCounts(len(m.stories), 0)

			// Update views with stories
			m.dashboard.SetStories(m.stories)
			m.storylist.SetStories(m.stories)
		}
	}

	// Route to active view
	switch m.activeView {
	case domain.ViewDashboard:
		var cmd tea.Cmd
		m.dashboard, cmd = m.dashboard.Update(msg)
		cmds = append(cmds, cmd)

	case domain.ViewStoryList:
		var cmd tea.Cmd
		m.storylist, cmd = m.storylist.Update(msg)
		cmds = append(cmds, cmd)

	// TODO: Add other views
	}

	return m, tea.Batch(cmds...)
}

// View renders the application
func (m Model) View() string {
	if !m.ready {
		return "\n  Initializing BMAD Automate..."
	}

	// Header
	m.header.SetActiveView(m.activeView)
	headerView := m.header.View()

	// Content based on active view
	var content string
	switch m.activeView {
	case domain.ViewDashboard:
		content = m.dashboard.View()
	case domain.ViewStoryList:
		content = m.storylist.View()
	case domain.ViewQueue:
		content = m.renderPlaceholder("Queue Manager", "Coming in Phase 3")
	case domain.ViewExecution:
		content = m.renderPlaceholder("Execution", "Coming in Phase 2")
	case domain.ViewTimeline:
		content = m.renderPlaceholder("Timeline", "Coming in Phase 3")
	case domain.ViewDiff:
		content = m.renderPlaceholder("Diff Preview", "Coming in Phase 4")
	case domain.ViewHistory:
		content = m.renderPlaceholder("History", "Coming in Phase 4")
	case domain.ViewStats:
		content = m.renderPlaceholder("Statistics", "Coming in Phase 4")
	case domain.ViewSettings:
		content = m.renderPlaceholder("Settings", "Coming in Phase 5")
	default:
		content = m.renderPlaceholder("Unknown View", "")
	}

	// Status bar
	statusView := m.statusbar.View()

	// Combine all sections
	return lipgloss.JoinVertical(lipgloss.Left,
		headerView,
		content,
		statusView,
	)
}

func (m Model) renderPlaceholder(title, subtitle string) string {
	t := theme.Current

	titleStyle := lipgloss.NewStyle().
		Foreground(t.Primary).
		Bold(true).
		MarginBottom(1)

	subtitleStyle := lipgloss.NewStyle().
		Foreground(t.Subtle).
		Italic(true)

	content := titleStyle.Render(title)
	if subtitle != "" {
		content = lipgloss.JoinVertical(lipgloss.Left,
			content,
			subtitleStyle.Render(subtitle),
		)
	}

	box := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(t.Border).
		Padding(2, 4).
		Render(content)

	// Center in available space
	return lipgloss.NewStyle().
		Width(m.width).
		Height(m.height - 4).
		Align(lipgloss.Center, lipgloss.Center).
		Render(box)
}
