package theme

import (
	"os"

	"github.com/charmbracelet/lipgloss"
	"gopkg.in/yaml.v3"
)

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
	Border      lipgloss.Color
	Selection   lipgloss.Color
	ActiveTab   lipgloss.Color
	InactiveTab lipgloss.Color
	StatusBar   lipgloss.Color
	HeaderBg    lipgloss.Color
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

// Dracula theme
var Dracula = Theme{
	Name: "Dracula",

	// Base colors
	Background: lipgloss.Color("#282a36"),
	Foreground: lipgloss.Color("#f8f8f2"),
	Subtle:     lipgloss.Color("#6272a4"),
	Highlight:  lipgloss.Color("#f1fa8c"),

	// Status colors
	Success: lipgloss.Color("#50fa7b"),
	Warning: lipgloss.Color("#ffb86c"),
	Error:   lipgloss.Color("#ff5555"),
	Info:    lipgloss.Color("#8be9fd"),

	// Accent colors
	Primary:   lipgloss.Color("#bd93f9"),
	Secondary: lipgloss.Color("#ff79c6"),
	Accent:    lipgloss.Color("#8be9fd"),

	// UI element colors
	Border:      lipgloss.Color("#44475a"),
	Selection:   lipgloss.Color("#44475a"),
	ActiveTab:   lipgloss.Color("#bd93f9"),
	InactiveTab: lipgloss.Color("#6272a4"),
	StatusBar:   lipgloss.Color("#21222c"),
	HeaderBg:    lipgloss.Color("#21222c"),
}

// Nord theme
var Nord = Theme{
	Name: "Nord",

	// Base colors
	Background: lipgloss.Color("#2e3440"),
	Foreground: lipgloss.Color("#eceff4"),
	Subtle:     lipgloss.Color("#4c566a"),
	Highlight:  lipgloss.Color("#ebcb8b"),

	// Status colors
	Success: lipgloss.Color("#a3be8c"),
	Warning: lipgloss.Color("#ebcb8b"),
	Error:   lipgloss.Color("#bf616a"),
	Info:    lipgloss.Color("#81a1c1"),

	// Accent colors
	Primary:   lipgloss.Color("#88c0d0"),
	Secondary: lipgloss.Color("#b48ead"),
	Accent:    lipgloss.Color("#8fbcbb"),

	// UI element colors
	Border:      lipgloss.Color("#3b4252"),
	Selection:   lipgloss.Color("#434c5e"),
	ActiveTab:   lipgloss.Color("#88c0d0"),
	InactiveTab: lipgloss.Color("#4c566a"),
	StatusBar:   lipgloss.Color("#242933"),
	HeaderBg:    lipgloss.Color("#242933"),
}

// Current is the active theme
var Current = Catppuccin

// AvailableThemes returns a list of built-in theme names
func AvailableThemes() []string {
	return []string{"catppuccin", "dracula", "nord"}
}

// SetTheme sets the current theme by name
func SetTheme(name string) {
	switch name {
	case "dracula":
		Current = Dracula
	case "nord":
		Current = Nord
	case "catppuccin":
		fallthrough
	default:
		Current = Catppuccin
	}
}

// ThemeYAML represents a theme configuration in YAML format
type ThemeYAML struct {
	Name string `yaml:"name"`

	// Base colors
	Background string `yaml:"background"`
	Foreground string `yaml:"foreground"`
	Subtle     string `yaml:"subtle"`
	Highlight  string `yaml:"highlight"`

	// Status colors
	Success string `yaml:"success"`
	Warning string `yaml:"warning"`
	Error   string `yaml:"error"`
	Info    string `yaml:"info"`

	// Accent colors
	Primary   string `yaml:"primary"`
	Secondary string `yaml:"secondary"`
	Accent    string `yaml:"accent"`

	// UI element colors
	Border      string `yaml:"border"`
	Selection   string `yaml:"selection"`
	ActiveTab   string `yaml:"active_tab"`
	InactiveTab string `yaml:"inactive_tab"`
	StatusBar   string `yaml:"status_bar"`
	HeaderBg    string `yaml:"header_bg"`
}

// LoadThemeFromYAML loads a custom theme from a YAML file
func LoadThemeFromYAML(path string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}

	var themeYAML ThemeYAML
	if err := yaml.Unmarshal(data, &themeYAML); err != nil {
		return err
	}

	Current = Theme{
		Name:        themeYAML.Name,
		Background:  lipgloss.Color(themeYAML.Background),
		Foreground:  lipgloss.Color(themeYAML.Foreground),
		Subtle:      lipgloss.Color(themeYAML.Subtle),
		Highlight:   lipgloss.Color(themeYAML.Highlight),
		Success:     lipgloss.Color(themeYAML.Success),
		Warning:     lipgloss.Color(themeYAML.Warning),
		Error:       lipgloss.Color(themeYAML.Error),
		Info:        lipgloss.Color(themeYAML.Info),
		Primary:     lipgloss.Color(themeYAML.Primary),
		Secondary:   lipgloss.Color(themeYAML.Secondary),
		Accent:      lipgloss.Color(themeYAML.Accent),
		Border:      lipgloss.Color(themeYAML.Border),
		Selection:   lipgloss.Color(themeYAML.Selection),
		ActiveTab:   lipgloss.Color(themeYAML.ActiveTab),
		InactiveTab: lipgloss.Color(themeYAML.InactiveTab),
		StatusBar:   lipgloss.Color(themeYAML.StatusBar),
		HeaderBg:    lipgloss.Color(themeYAML.HeaderBg),
	}

	return nil
}

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
