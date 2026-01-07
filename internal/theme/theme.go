package theme

import "github.com/charmbracelet/lipgloss"

// Theme defines the color palette and styles for the application
type Theme struct {
	Name string

	// Base colors
	Background lipgloss.Color
	Foreground lipgloss.Color
	Subtle     lipgloss.Color
	Highlight  lipgloss.Color

	// Status colors
	Success lipgloss.Color
	Warning lipgloss.Color
	Error   lipgloss.Color
	Info    lipgloss.Color

	// Accent colors
	Primary   lipgloss.Color
	Secondary lipgloss.Color
	Accent    lipgloss.Color

	// UI element colors
	Border       lipgloss.Color
	Selection    lipgloss.Color
	ActiveTab    lipgloss.Color
	InactiveTab  lipgloss.Color
	StatusBar    lipgloss.Color
	HeaderBg     lipgloss.Color
}

// Catppuccin Mocha theme (default)
var Catppuccin = Theme{
	Name: "Catppuccin Mocha",

	// Base colors
	Background: lipgloss.Color("#1e1e2e"),
	Foreground: lipgloss.Color("#cdd6f4"),
	Subtle:     lipgloss.Color("#6c7086"),
	Highlight:  lipgloss.Color("#f5e0dc"),

	// Status colors
	Success: lipgloss.Color("#a6e3a1"),
	Warning: lipgloss.Color("#f9e2af"),
	Error:   lipgloss.Color("#f38ba8"),
	Info:    lipgloss.Color("#89b4fa"),

	// Accent colors
	Primary:   lipgloss.Color("#cba6f7"),
	Secondary: lipgloss.Color("#f5c2e7"),
	Accent:    lipgloss.Color("#94e2d5"),

	// UI element colors
	Border:      lipgloss.Color("#313244"),
	Selection:   lipgloss.Color("#45475a"),
	ActiveTab:   lipgloss.Color("#cba6f7"),
	InactiveTab: lipgloss.Color("#6c7086"),
	StatusBar:   lipgloss.Color("#181825"),
	HeaderBg:    lipgloss.Color("#181825"),
}

// Current is the active theme
var Current = Catppuccin

// Styles contains pre-built lipgloss styles using the current theme
type Styles struct {
	// Base styles
	App       lipgloss.Style
	Header    lipgloss.Style
	StatusBar lipgloss.Style
	Content   lipgloss.Style

	// Text styles
	Title     lipgloss.Style
	Subtitle  lipgloss.Style
	Muted     lipgloss.Style
	Bold      lipgloss.Style
	Highlight lipgloss.Style

	// Status styles
	Success lipgloss.Style
	Warning lipgloss.Style
	Error   lipgloss.Style
	Info    lipgloss.Style

	// Interactive elements
	Selected   lipgloss.Style
	Unselected lipgloss.Style
	Active     lipgloss.Style
	Inactive   lipgloss.Style

	// Navigation
	NavItem       lipgloss.Style
	NavItemActive lipgloss.Style
	Shortcut      lipgloss.Style

	// Borders
	BorderedBox lipgloss.Style

	// Story status badges
	BadgeInProgress  lipgloss.Style
	BadgeReadyForDev lipgloss.Style
	BadgeBacklog     lipgloss.Style
	BadgeDone        lipgloss.Style
}

// NewStyles creates styles based on the current theme
func NewStyles() Styles {
	t := Current

	return Styles{
		// Base styles
		App: lipgloss.NewStyle().
			Background(t.Background).
			Foreground(t.Foreground),

		Header: lipgloss.NewStyle().
			Background(t.HeaderBg).
			Foreground(t.Foreground).
			Padding(0, 2).
			Bold(true),

		StatusBar: lipgloss.NewStyle().
			Background(t.StatusBar).
			Foreground(t.Subtle).
			Padding(0, 2),

		Content: lipgloss.NewStyle().
			Padding(1, 2),

		// Text styles
		Title: lipgloss.NewStyle().
			Foreground(t.Primary).
			Bold(true),

		Subtitle: lipgloss.NewStyle().
			Foreground(t.Secondary),

		Muted: lipgloss.NewStyle().
			Foreground(t.Subtle),

		Bold: lipgloss.NewStyle().
			Bold(true),

		Highlight: lipgloss.NewStyle().
			Foreground(t.Highlight).
			Bold(true),

		// Status styles
		Success: lipgloss.NewStyle().
			Foreground(t.Success),

		Warning: lipgloss.NewStyle().
			Foreground(t.Warning),

		Error: lipgloss.NewStyle().
			Foreground(t.Error),

		Info: lipgloss.NewStyle().
			Foreground(t.Info),

		// Interactive elements
		Selected: lipgloss.NewStyle().
			Background(t.Selection).
			Foreground(t.Foreground).
			Bold(true),

		Unselected: lipgloss.NewStyle().
			Foreground(t.Foreground),

		Active: lipgloss.NewStyle().
			Foreground(t.Primary).
			Bold(true),

		Inactive: lipgloss.NewStyle().
			Foreground(t.Subtle),

		// Navigation
		NavItem: lipgloss.NewStyle().
			Foreground(t.Subtle).
			Padding(0, 2),

		NavItemActive: lipgloss.NewStyle().
			Foreground(t.Primary).
			Background(t.Selection).
			Padding(0, 2).
			Bold(true),

		Shortcut: lipgloss.NewStyle().
			Foreground(t.Accent).
			Bold(true),

		// Borders
		BorderedBox: lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(t.Border).
			Padding(1, 2),

		// Story status badges
		BadgeInProgress: lipgloss.NewStyle().
			Foreground(t.Background).
			Background(t.Warning).
			Padding(0, 1).
			Bold(true),

		BadgeReadyForDev: lipgloss.NewStyle().
			Foreground(t.Background).
			Background(t.Success).
			Padding(0, 1).
			Bold(true),

		BadgeBacklog: lipgloss.NewStyle().
			Foreground(t.Background).
			Background(t.Subtle).
			Padding(0, 1),

		BadgeDone: lipgloss.NewStyle().
			Foreground(t.Background).
			Background(t.Info).
			Padding(0, 1),
	}
}
