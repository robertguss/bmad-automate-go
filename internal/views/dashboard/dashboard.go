package dashboard

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/robertguss/bmad-automate-go/internal/domain"
	"github.com/robertguss/bmad-automate-go/internal/messages"
	"github.com/robertguss/bmad-automate-go/internal/theme"
)

// Model represents the dashboard view
type Model struct {
	width   int
	height  int
	stories []domain.Story
	styles  theme.Styles
}

// New creates a new dashboard model
func New() Model {
	return Model{
		styles: theme.NewStyles(),
	}
}

// Init initializes the dashboard
func (m Model) Init() tea.Cmd {
	return nil
}

// Update handles messages
func (m Model) Update(msg tea.Msg) (Model, tea.Cmd) {
	switch msg := msg.(type) {
	case messages.StoriesLoadedMsg:
		if msg.Error == nil {
			m.stories = msg.Stories
		}
	case messages.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
	}
	return m, nil
}

// SetSize sets the view dimensions
func (m *Model) SetSize(width, height int) {
	m.width = width
	m.height = height
}

// SetStories sets the story data
func (m *Model) SetStories(stories []domain.Story) {
	m.stories = stories
}

// View renders the dashboard
func (m Model) View() string {
	t := theme.Current

	// Count stories by status
	counts := make(map[domain.StoryStatus]int)
	for _, s := range m.stories {
		counts[s.Status]++
	}

	// Build the overview section
	overviewTitle := lipgloss.NewStyle().
		Foreground(t.Primary).
		Bold(true).
		MarginBottom(1).
		Render("Stories Overview")

	// Status rows
	statusRows := []struct {
		label string
		count int
		style lipgloss.Style
	}{
		{"In Progress", counts[domain.StatusInProgress], m.styles.BadgeInProgress},
		{"Ready for Dev", counts[domain.StatusReadyForDev], m.styles.BadgeReadyForDev},
		{"Backlog", counts[domain.StatusBacklog], m.styles.BadgeBacklog},
		{"Done", counts[domain.StatusDone], m.styles.BadgeDone},
	}

	var rows []string
	for _, r := range statusRows {
		badge := r.style.Render(fmt.Sprintf(" %d ", r.count))
		label := lipgloss.NewStyle().Foreground(t.Foreground).Width(15).Render(r.label)
		rows = append(rows, fmt.Sprintf("  %s  %s", label, badge))
	}

	total := len(m.stories)
	totalRow := fmt.Sprintf("  %s  %s",
		lipgloss.NewStyle().Foreground(t.Foreground).Bold(true).Width(15).Render("Total"),
		lipgloss.NewStyle().Foreground(t.Highlight).Bold(true).Render(fmt.Sprintf(" %d ", total)),
	)
	rows = append(rows, "")
	rows = append(rows, totalRow)

	overviewBox := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(t.Border).
		Padding(1, 2).
		Width(40).
		Render(lipgloss.JoinVertical(lipgloss.Left, append([]string{overviewTitle}, rows...)...))

	// Quick actions section
	actionsTitle := lipgloss.NewStyle().
		Foreground(t.Primary).
		Bold(true).
		MarginBottom(1).
		Render("Quick Actions")

	actions := []struct {
		key  string
		desc string
	}{
		{"s", "View story list"},
		{"q", "View queue"},
		{"Enter", "Start processing"},
		{"h", "View history"},
		{"o", "Open settings"},
	}

	var actionRows []string
	for _, a := range actions {
		key := lipgloss.NewStyle().
			Foreground(t.Accent).
			Bold(true).
			Width(8).
			Render("[" + a.key + "]")
		desc := lipgloss.NewStyle().
			Foreground(t.Foreground).
			Render(a.desc)
		actionRows = append(actionRows, "  "+key+" "+desc)
	}

	actionsBox := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(t.Border).
		Padding(1, 2).
		Width(35).
		Render(lipgloss.JoinVertical(lipgloss.Left, append([]string{actionsTitle}, actionRows...)...))

	// Recent activity placeholder
	recentTitle := lipgloss.NewStyle().
		Foreground(t.Primary).
		Bold(true).
		MarginBottom(1).
		Render("Recent Activity")

	recentContent := lipgloss.NewStyle().
		Foreground(t.Subtle).
		Italic(true).
		Render("No recent activity")

	recentBox := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(t.Border).
		Padding(1, 2).
		Width(35).
		Render(lipgloss.JoinVertical(lipgloss.Left, recentTitle, recentContent))

	// Layout
	leftColumn := overviewBox
	rightColumn := lipgloss.JoinVertical(lipgloss.Left, actionsBox, "", recentBox)

	content := lipgloss.JoinHorizontal(lipgloss.Top, leftColumn, "  ", rightColumn)

	// Welcome message
	welcome := lipgloss.NewStyle().
		Foreground(t.Foreground).
		MarginBottom(2).
		Render("Welcome to BMAD Automate - your AI-powered development workflow assistant.")

	// Wrap in container
	container := lipgloss.NewStyle().
		Padding(1, 2).
		Render(lipgloss.JoinVertical(lipgloss.Left, welcome, content))

	// Add bottom padding to fill space
	lines := strings.Count(container, "\n") + 1
	if m.height > lines {
		container += strings.Repeat("\n", m.height-lines-1)
	}

	return container
}
