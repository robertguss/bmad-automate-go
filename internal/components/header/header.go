package header

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/robertguss/bmad-automate-go/internal/domain"
	"github.com/robertguss/bmad-automate-go/internal/theme"
)

// Model represents the header component
type Model struct {
	width      int
	activeView domain.View
	styles     theme.Styles
}

// New creates a new header model
func New() Model {
	return Model{
		styles: theme.NewStyles(),
	}
}

// SetWidth sets the header width
func (m *Model) SetWidth(width int) {
	m.width = width
}

// SetActiveView sets the currently active view
func (m *Model) SetActiveView(view domain.View) {
	m.activeView = view
}

// View renders the header
func (m Model) View() string {
	t := theme.Current

	// Title
	title := lipgloss.NewStyle().
		Foreground(t.Primary).
		Bold(true).
		Render("BMAD Automate")

	// Navigation items
	navViews := []domain.View{
		domain.ViewDashboard,
		domain.ViewStoryList,
		domain.ViewQueue,
		domain.ViewHistory,
		domain.ViewStats,
		domain.ViewSettings,
	}

	var navItems []string
	for _, v := range navViews {
		shortcut := lipgloss.NewStyle().
			Foreground(t.Accent).
			Bold(true).
			Render("[" + v.Shortcut() + "]")

		name := v.String()

		var item string
		if v == m.activeView {
			item = lipgloss.NewStyle().
				Foreground(t.Primary).
				Bold(true).
				Render(shortcut + " " + name)
		} else {
			item = lipgloss.NewStyle().
				Foreground(t.Subtle).
				Render(shortcut + " " + name)
		}
		navItems = append(navItems, item)
	}

	nav := strings.Join(navItems, "  ")

	// Command palette hint
	paletteHint := lipgloss.NewStyle().
		Foreground(t.Subtle).
		Render("[Ctrl+P] Command Palette")

	// Calculate spacing
	titleWidth := lipgloss.Width(title)
	navWidth := lipgloss.Width(nav)
	hintWidth := lipgloss.Width(paletteHint)
	totalContent := titleWidth + navWidth + hintWidth + 8 // padding

	spacing := ""
	if m.width > totalContent {
		gap1 := (m.width - totalContent) / 2
		gap2 := m.width - totalContent - gap1
		spacing = strings.Repeat(" ", gap1)
		paletteHint = strings.Repeat(" ", gap2) + paletteHint
	}

	content := title + spacing + nav + paletteHint

	// Header container
	header := lipgloss.NewStyle().
		Background(t.HeaderBg).
		Foreground(t.Foreground).
		Width(m.width).
		Padding(0, 2).
		Render(content)

	// Bottom border
	border := lipgloss.NewStyle().
		Foreground(t.Border).
		Width(m.width).
		Render(strings.Repeat("─", m.width))

	return lipgloss.JoinVertical(lipgloss.Left, header, border)
}
