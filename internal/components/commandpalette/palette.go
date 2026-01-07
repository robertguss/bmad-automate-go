package commandpalette

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/robertguss/bmad-automate-go/internal/domain"
	"github.com/robertguss/bmad-automate-go/internal/theme"
)

// Command represents an action available in the command palette
type Command struct {
	Name        string
	Description string
	Shortcut    string
	Category    string
	Action      func() tea.Msg
}

// SelectCommandMsg is sent when a command is selected
type SelectCommandMsg struct {
	Command Command
}

// CloseMsg is sent when the palette is closed
type CloseMsg struct{}

// Model represents the command palette
type Model struct {
	width    int
	height   int
	input    string
	commands []Command
	filtered []Command
	cursor   int
	active   bool
	styles   theme.Styles
}

// New creates a new command palette
func New() Model {
	m := Model{
		styles: theme.NewStyles(),
	}
	m.commands = m.defaultCommands()
	m.filtered = m.commands
	return m
}

func (m Model) defaultCommands() []Command {
	return []Command{
		// Navigation
		{
			Name:        "Go to Dashboard",
			Description: "View dashboard with story overview",
			Shortcut:    "d",
			Category:    "Navigation",
			Action:      func() tea.Msg { return NavigateMsg{View: domain.ViewDashboard} },
		},
		{
			Name:        "Go to Stories",
			Description: "Browse and filter stories",
			Shortcut:    "s",
			Category:    "Navigation",
			Action:      func() tea.Msg { return NavigateMsg{View: domain.ViewStoryList} },
		},
		{
			Name:        "Go to Queue",
			Description: "Manage execution queue",
			Shortcut:    "q",
			Category:    "Navigation",
			Action:      func() tea.Msg { return NavigateMsg{View: domain.ViewQueue} },
		},
		{
			Name:        "Go to History",
			Description: "View execution history",
			Shortcut:    "h",
			Category:    "Navigation",
			Action:      func() tea.Msg { return NavigateMsg{View: domain.ViewHistory} },
		},
		{
			Name:        "Go to Statistics",
			Description: "View execution statistics",
			Shortcut:    "a",
			Category:    "Navigation",
			Action:      func() tea.Msg { return NavigateMsg{View: domain.ViewStats} },
		},
		{
			Name:        "Go to Settings",
			Description: "Configure application settings",
			Shortcut:    "o",
			Category:    "Navigation",
			Action:      func() tea.Msg { return NavigateMsg{View: domain.ViewSettings} },
		},
		// Theme
		{
			Name:        "Theme: Catppuccin",
			Description: "Switch to Catppuccin Mocha theme",
			Category:    "Theme",
			Action:      func() tea.Msg { return ThemeChangeMsg{Theme: "catppuccin"} },
		},
		{
			Name:        "Theme: Dracula",
			Description: "Switch to Dracula theme",
			Category:    "Theme",
			Action:      func() tea.Msg { return ThemeChangeMsg{Theme: "dracula"} },
		},
		{
			Name:        "Theme: Nord",
			Description: "Switch to Nord theme",
			Category:    "Theme",
			Action:      func() tea.Msg { return ThemeChangeMsg{Theme: "nord"} },
		},
		// Actions
		{
			Name:        "Start Queue",
			Description: "Begin processing queued stories",
			Category:    "Actions",
			Action:      func() tea.Msg { return ActionMsg{Action: "start_queue"} },
		},
		{
			Name:        "Pause Queue",
			Description: "Pause current queue execution",
			Category:    "Actions",
			Action:      func() tea.Msg { return ActionMsg{Action: "pause_queue"} },
		},
		{
			Name:        "Clear Queue",
			Description: "Remove all pending items from queue",
			Category:    "Actions",
			Action:      func() tea.Msg { return ActionMsg{Action: "clear_queue"} },
		},
		{
			Name:        "Refresh Stories",
			Description: "Reload stories from sprint-status.yaml",
			Category:    "Actions",
			Action:      func() tea.Msg { return ActionMsg{Action: "refresh"} },
		},
	}
}

// NavigateMsg requests navigation to a view
type NavigateMsg struct {
	View domain.View
}

// ThemeChangeMsg requests a theme change
type ThemeChangeMsg struct {
	Theme string
}

// ActionMsg requests an action to be performed
type ActionMsg struct {
	Action string
}

// Open opens the command palette
func (m *Model) Open() {
	m.active = true
	m.input = ""
	m.cursor = 0
	m.filtered = m.commands
}

// Close closes the command palette
func (m *Model) Close() {
	m.active = false
	m.input = ""
	m.cursor = 0
}

// IsActive returns whether the palette is open
func (m Model) IsActive() bool {
	return m.active
}

// SetSize sets the palette dimensions
func (m *Model) SetSize(width, height int) {
	m.width = width
	m.height = height
}

// Init initializes the palette
func (m Model) Init() tea.Cmd {
	return nil
}

// Update handles messages
func (m Model) Update(msg tea.Msg) (Model, tea.Cmd) {
	if !m.active {
		return m, nil
	}

	switch msg := msg.(type) {
	case tea.KeyMsg:
		return m.handleKeyMsg(msg)
	}
	return m, nil
}

func (m Model) handleKeyMsg(msg tea.KeyMsg) (Model, tea.Cmd) {
	switch msg.String() {
	case "esc", "ctrl+p":
		m.Close()
		return m, func() tea.Msg { return CloseMsg{} }

	case "enter":
		if len(m.filtered) > 0 && m.cursor < len(m.filtered) {
			cmd := m.filtered[m.cursor]
			m.Close()
			return m, func() tea.Msg { return SelectCommandMsg{Command: cmd} }
		}

	case "up", "ctrl+k":
		if m.cursor > 0 {
			m.cursor--
		}

	case "down", "ctrl+j":
		if m.cursor < len(m.filtered)-1 {
			m.cursor++
		}

	case "backspace":
		if len(m.input) > 0 {
			m.input = m.input[:len(m.input)-1]
			m.filterCommands()
		}

	default:
		// Handle character input
		if len(msg.String()) == 1 {
			m.input += msg.String()
			m.filterCommands()
		}
	}
	return m, nil
}

func (m *Model) filterCommands() {
	if m.input == "" {
		m.filtered = m.commands
		m.cursor = 0
		return
	}

	query := strings.ToLower(m.input)
	var filtered []Command

	for _, cmd := range m.commands {
		name := strings.ToLower(cmd.Name)
		desc := strings.ToLower(cmd.Description)
		cat := strings.ToLower(cmd.Category)

		// Simple fuzzy match - check if query letters appear in order
		if fuzzyMatch(name, query) || fuzzyMatch(desc, query) || strings.Contains(cat, query) {
			filtered = append(filtered, cmd)
		}
	}

	m.filtered = filtered
	if m.cursor >= len(m.filtered) {
		m.cursor = max(0, len(m.filtered)-1)
	}
}

// fuzzyMatch checks if query characters appear in target in order
func fuzzyMatch(target, query string) bool {
	targetIdx := 0
	for _, qChar := range query {
		found := false
		for targetIdx < len(target) {
			if rune(target[targetIdx]) == qChar {
				found = true
				targetIdx++
				break
			}
			targetIdx++
		}
		if !found {
			return false
		}
	}
	return true
}

// View renders the command palette
func (m Model) View() string {
	if !m.active {
		return ""
	}

	t := theme.Current

	// Calculate palette dimensions
	paletteWidth := min(60, m.width-4)
	maxItems := min(10, m.height-10)

	// Input field
	inputStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(t.Primary).
		Padding(0, 1).
		Width(paletteWidth - 2)

	prompt := lipgloss.NewStyle().Foreground(t.Primary).Render("> ")
	cursor := lipgloss.NewStyle().Foreground(t.Accent).Render("_")
	inputContent := prompt + m.input + cursor
	inputBox := inputStyle.Render(inputContent)

	// Results list
	var resultRows []string
	visibleStart := 0
	if m.cursor >= maxItems {
		visibleStart = m.cursor - maxItems + 1
	}

	for i := visibleStart; i < len(m.filtered) && i < visibleStart+maxItems; i++ {
		cmd := m.filtered[i]
		row := m.renderCommand(i, cmd, paletteWidth-4)
		resultRows = append(resultRows, row)
	}

	resultsList := lipgloss.JoinVertical(lipgloss.Left, resultRows...)

	resultsStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(t.Border).
		Width(paletteWidth - 2).
		MaxHeight(maxItems * 2)

	resultsBox := resultsStyle.Render(resultsList)

	// Combine input and results
	palette := lipgloss.JoinVertical(
		lipgloss.Left,
		inputBox,
		resultsBox,
	)

	// Center the palette
	paletteStyle := lipgloss.NewStyle().
		Background(t.Background).
		Padding(1).
		Border(lipgloss.DoubleBorder()).
		BorderForeground(t.Primary)

	return lipgloss.Place(
		m.width,
		m.height,
		lipgloss.Center,
		lipgloss.Center,
		paletteStyle.Render(palette),
	)
}

func (m Model) renderCommand(index int, cmd Command, width int) string {
	t := theme.Current

	isSelected := index == m.cursor

	// Name
	nameStyle := lipgloss.NewStyle().Bold(true)
	if isSelected {
		nameStyle = nameStyle.Foreground(t.Primary).Background(t.Selection)
	} else {
		nameStyle = nameStyle.Foreground(t.Foreground)
	}

	// Shortcut badge
	shortcutBadge := ""
	if cmd.Shortcut != "" {
		shortcutBadge = lipgloss.NewStyle().
			Foreground(t.Accent).
			Render(" [" + cmd.Shortcut + "]")
	}

	// Category badge
	categoryBadge := lipgloss.NewStyle().
		Foreground(t.Subtle).
		Render(" " + cmd.Category)

	// Description
	descStyle := lipgloss.NewStyle().Foreground(t.Subtle)
	if isSelected {
		descStyle = descStyle.Background(t.Selection)
	}

	name := nameStyle.Render(cmd.Name) + shortcutBadge + categoryBadge
	desc := descStyle.Render("  " + cmd.Description)

	// Row container
	rowStyle := lipgloss.NewStyle().Width(width).Padding(0, 1)
	if isSelected {
		rowStyle = rowStyle.Background(t.Selection)
	}

	return rowStyle.Render(name + "\n" + desc)
}

// Overlay renders the palette over content
func (m Model) Overlay(content string) string {
	if !m.active {
		return content
	}

	// Simple overlay: put palette view on top
	// In a real implementation, we'd blend this better
	paletteView := m.View()

	// Place palette centered over content
	t := theme.Current
	overlay := lipgloss.NewStyle().
		Background(t.Background).
		Width(m.width).
		Height(m.height).
		Render(paletteView)

	return overlay
}
