package settings

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/robertguss/bmad-automate-go/internal/config"
	"github.com/robertguss/bmad-automate-go/internal/messages"
	"github.com/robertguss/bmad-automate-go/internal/theme"
)

// SettingType represents the type of a setting
type SettingType int

const (
	SettingTypeSelect SettingType = iota
	SettingTypeToggle
	SettingTypeNumber
	SettingTypeText
)

// Setting represents a configurable option
type Setting struct {
	Name        string
	Description string
	Type        SettingType
	Options     []string      // For select type
	Value       interface{}   // Current value
	Min, Max    int           // For number type
	OnChange    func(interface{}) tea.Cmd
}

// Model represents the settings view
type Model struct {
	width    int
	height   int
	config   *config.Config
	settings []Setting
	cursor   int
	editing  bool
	styles   theme.Styles
}

// ThemeChangedMsg is sent when the theme is changed
type ThemeChangedMsg struct {
	Theme string
}

// SettingChangedMsg is sent when any setting changes
type SettingChangedMsg struct {
	Name  string
	Value interface{}
}

// New creates a new settings view
func New(cfg *config.Config) Model {
	m := Model{
		config: cfg,
		styles: theme.NewStyles(),
	}
	m.buildSettings()
	return m
}

func (m *Model) buildSettings() {
	m.settings = []Setting{
		{
			Name:        "Theme",
			Description: "Color theme for the application",
			Type:        SettingTypeSelect,
			Options:     theme.AvailableThemes(),
			Value:       m.config.Theme,
		},
		{
			Name:        "Timeout",
			Description: "Maximum time (seconds) for each step execution",
			Type:        SettingTypeNumber,
			Value:       m.config.Timeout,
			Min:         60,
			Max:         3600,
		},
		{
			Name:        "Retries",
			Description: "Number of retry attempts for failed steps",
			Type:        SettingTypeNumber,
			Value:       m.config.Retries,
			Min:         0,
			Max:         5,
		},
		{
			Name:        "Notifications",
			Description: "Enable desktop notifications when tasks complete",
			Type:        SettingTypeToggle,
			Value:       m.config.NotificationsEnabled,
		},
		{
			Name:        "Sound",
			Description: "Enable sound feedback for events",
			Type:        SettingTypeToggle,
			Value:       m.config.SoundEnabled,
		},
	}
}

// Init initializes the settings view
func (m Model) Init() tea.Cmd {
	return nil
}

// Update handles messages for the settings view
func (m Model) Update(msg tea.Msg) (Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		return m.handleKeyMsg(msg)
	case messages.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
	}
	return m, nil
}

func (m Model) handleKeyMsg(msg tea.KeyMsg) (Model, tea.Cmd) {
	switch msg.String() {
	case "up", "k":
		if m.cursor > 0 {
			m.cursor--
		}
	case "down", "j":
		if m.cursor < len(m.settings)-1 {
			m.cursor++
		}
	case "left", "h":
		return m.adjustValue(-1)
	case "right", "l":
		return m.adjustValue(1)
	case "enter", " ":
		return m.toggleOrCycle()
	}
	return m, nil
}

func (m Model) adjustValue(delta int) (Model, tea.Cmd) {
	setting := &m.settings[m.cursor]
	var cmd tea.Cmd

	switch setting.Type {
	case SettingTypeSelect:
		options := setting.Options
		current := 0
		for i, opt := range options {
			if opt == setting.Value.(string) {
				current = i
				break
			}
		}
		newIdx := (current + delta + len(options)) % len(options)
		setting.Value = options[newIdx]
		cmd = m.applySettingChange(setting)
	case SettingTypeNumber:
		val := setting.Value.(int)
		newVal := val + (delta * 10) // Adjust by 10 for numbers
		if newVal < setting.Min {
			newVal = setting.Min
		}
		if newVal > setting.Max {
			newVal = setting.Max
		}
		setting.Value = newVal
		cmd = m.applySettingChange(setting)
	case SettingTypeToggle:
		setting.Value = !setting.Value.(bool)
		cmd = m.applySettingChange(setting)
	}

	return m, cmd
}

func (m Model) toggleOrCycle() (Model, tea.Cmd) {
	setting := &m.settings[m.cursor]
	var cmd tea.Cmd

	switch setting.Type {
	case SettingTypeToggle:
		setting.Value = !setting.Value.(bool)
		cmd = m.applySettingChange(setting)
	case SettingTypeSelect:
		// Cycle to next option
		options := setting.Options
		current := 0
		for i, opt := range options {
			if opt == setting.Value.(string) {
				current = i
				break
			}
		}
		newIdx := (current + 1) % len(options)
		setting.Value = options[newIdx]
		cmd = m.applySettingChange(setting)
	}

	return m, cmd
}

func (m *Model) applySettingChange(setting *Setting) tea.Cmd {
	switch setting.Name {
	case "Theme":
		themeName := setting.Value.(string)
		theme.SetTheme(themeName)
		m.config.Theme = themeName
		m.styles = theme.NewStyles() // Rebuild styles with new theme
		return func() tea.Msg {
			return ThemeChangedMsg{Theme: themeName}
		}
	case "Timeout":
		m.config.Timeout = setting.Value.(int)
	case "Retries":
		m.config.Retries = setting.Value.(int)
	case "Notifications":
		m.config.NotificationsEnabled = setting.Value.(bool)
	case "Sound":
		m.config.SoundEnabled = setting.Value.(bool)
	}

	return func() tea.Msg {
		return SettingChangedMsg{
			Name:  setting.Name,
			Value: setting.Value,
		}
	}
}

// SetSize sets the view dimensions
func (m *Model) SetSize(width, height int) {
	m.width = width
	m.height = height
}

// SetConfig updates the config reference
func (m *Model) SetConfig(cfg *config.Config) {
	m.config = cfg
	m.buildSettings()
}

// RefreshStyles rebuilds styles after theme change
func (m *Model) RefreshStyles() {
	m.styles = theme.NewStyles()
}

// View renders the settings view
func (m Model) View() string {
	if m.width == 0 {
		return "Loading..."
	}

	t := theme.Current

	// Title
	title := m.styles.Title.Render("Settings")

	// Build settings list
	var settingsRows []string

	for i, setting := range m.settings {
		row := m.renderSetting(i, setting)
		settingsRows = append(settingsRows, row)
	}

	settingsList := lipgloss.JoinVertical(lipgloss.Left, settingsRows...)

	// Settings box
	settingsBox := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(t.Border).
		Padding(1, 2).
		Width(m.width - 4).
		Render(settingsList)

	// Help text
	help := m.styles.Muted.Render("Arrow keys: Navigate/Adjust  Enter/Space: Toggle  Esc: Back")

	// Combine all elements
	content := lipgloss.JoinVertical(
		lipgloss.Left,
		title,
		"",
		settingsBox,
		"",
		help,
	)

	return lipgloss.NewStyle().
		Padding(1, 2).
		Render(content)
}

func (m Model) renderSetting(index int, setting Setting) string {
	t := theme.Current

	// Cursor indicator
	cursor := "  "
	if index == m.cursor {
		cursor = m.styles.Shortcut.Render("> ")
	}

	// Name and description
	nameStyle := lipgloss.NewStyle().Foreground(t.Foreground).Bold(true).Width(16)
	if index == m.cursor {
		nameStyle = nameStyle.Foreground(t.Primary)
	}
	name := nameStyle.Render(setting.Name)

	// Value display
	var valueDisplay string
	switch setting.Type {
	case SettingTypeSelect:
		val := setting.Value.(string)
		options := setting.Options
		var parts []string
		for _, opt := range options {
			if opt == val {
				parts = append(parts, lipgloss.NewStyle().
					Background(t.Selection).
					Foreground(t.Primary).
					Padding(0, 1).
					Bold(true).
					Render(opt))
			} else {
				parts = append(parts, lipgloss.NewStyle().
					Foreground(t.Subtle).
					Padding(0, 1).
					Render(opt))
			}
		}
		valueDisplay = strings.Join(parts, " ")
	case SettingTypeToggle:
		if setting.Value.(bool) {
			valueDisplay = lipgloss.NewStyle().
				Background(t.Success).
				Foreground(t.Background).
				Padding(0, 1).
				Bold(true).
				Render("ON")
		} else {
			valueDisplay = lipgloss.NewStyle().
				Background(t.Subtle).
				Foreground(t.Background).
				Padding(0, 1).
				Render("OFF")
		}
	case SettingTypeNumber:
		val := setting.Value.(int)
		// Show as slider-like display
		valueDisplay = fmt.Sprintf("%s %d %s",
			m.styles.Muted.Render("<"),
			val,
			m.styles.Muted.Render(">"))
	}

	// Description
	desc := m.styles.Muted.Render(setting.Description)

	// First line: cursor + name + value
	firstLine := fmt.Sprintf("%s%s  %s", cursor, name, valueDisplay)

	// Second line: description (indented)
	secondLine := fmt.Sprintf("                    %s", desc)

	return firstLine + "\n" + secondLine + "\n"
}
