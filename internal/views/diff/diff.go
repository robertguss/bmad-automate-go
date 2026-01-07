package diff

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/robertguss/bmad-automate-go/internal/messages"
	"github.com/robertguss/bmad-automate-go/internal/theme"
)

// Model represents the diff preview view state
type Model struct {
	width    int
	height   int
	styles   theme.Styles
	storyKey string
	content  string
	lines    []diffLine
	scroll   int
	loading  bool
	errorMsg string
}

// diffLine represents a parsed diff line
type diffLine struct {
	content  string
	lineType lineType
}

// lineType represents the type of diff line
type lineType int

const (
	lineNormal lineType = iota
	lineAdded
	lineRemoved
	lineHeader
	lineHunk
	lineContext
)

// New creates a new diff view model
func New() Model {
	return Model{
		styles: theme.NewStyles(),
		lines:  make([]diffLine, 0),
	}
}

// Init initializes the model
func (m Model) Init() tea.Cmd {
	return nil
}

// Update handles messages
func (m Model) Update(msg tea.Msg) (Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		return m.handleKeyMsg(msg)

	case messages.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height

	case messages.DiffLoadedMsg:
		m.loading = false
		if msg.Error != nil {
			m.errorMsg = msg.Error.Error()
			return m, nil
		}
		m.storyKey = msg.StoryKey
		m.content = msg.Content
		m.lines = parseDiff(msg.Content)
		m.errorMsg = ""
		m.scroll = 0
	}

	return m, nil
}

func (m Model) handleKeyMsg(msg tea.KeyMsg) (Model, tea.Cmd) {
	switch msg.String() {
	case "up":
		if m.scroll > 0 {
			m.scroll--
		}

	case "down":
		maxScroll := m.maxScroll()
		if m.scroll < maxScroll {
			m.scroll++
		}

	case "home":
		m.scroll = 0

	case "end":
		m.scroll = m.maxScroll()

	case "pgup":
		contentHeight := m.contentHeight()
		m.scroll -= contentHeight
		if m.scroll < 0 {
			m.scroll = 0
		}

	case "pgdown":
		contentHeight := m.contentHeight()
		m.scroll += contentHeight
		maxScroll := m.maxScroll()
		if m.scroll > maxScroll {
			m.scroll = maxScroll
		}
	}

	return m, nil
}

// View renders the diff view
func (m Model) View() string {
	if m.loading {
		return m.renderLoading()
	}

	if m.errorMsg != "" {
		return m.renderError()
	}

	if len(m.lines) == 0 {
		return m.renderNoDiff()
	}

	var sections []string

	// Header
	sections = append(sections, m.renderHeader())

	// Diff content
	sections = append(sections, m.renderDiffContent())

	// Footer with scroll info
	sections = append(sections, m.renderFooter())

	return lipgloss.JoinVertical(lipgloss.Left, sections...)
}

func (m Model) renderLoading() string {
	t := theme.Current
	return lipgloss.NewStyle().
		Foreground(t.Subtle).
		Padding(2, 0).
		Render("Loading diff...")
}

func (m Model) renderError() string {
	t := theme.Current
	return lipgloss.NewStyle().
		Foreground(t.Error).
		Padding(2, 0).
		Render(fmt.Sprintf("Error: %s", m.errorMsg))
}

func (m Model) renderNoDiff() string {
	t := theme.Current
	return lipgloss.NewStyle().
		Foreground(t.Subtle).
		Padding(2, 0).
		Render("No diff available. Select a story and run 'git diff' to preview changes.")
}

func (m Model) renderHeader() string {
	t := theme.Current

	title := lipgloss.NewStyle().
		Foreground(t.Primary).
		Bold(true).
		Render("Diff Preview")

	var subtitle string
	if m.storyKey != "" {
		subtitle = lipgloss.NewStyle().
			Foreground(t.Secondary).
			Render(fmt.Sprintf(" - %s", m.storyKey))
	}

	stats := m.getDiffStats()
	statsText := lipgloss.NewStyle().
		Foreground(t.Subtle).
		Render(fmt.Sprintf(" (%d lines, +%d/-%d)", len(m.lines), stats.added, stats.removed))

	return lipgloss.JoinHorizontal(lipgloss.Left, title, subtitle, statsText)
}

func (m Model) renderDiffContent() string {
	t := theme.Current
	contentHeight := m.contentHeight()

	// Calculate visible range
	start := m.scroll
	end := start + contentHeight
	if end > len(m.lines) {
		end = len(m.lines)
	}

	var renderedLines []string
	for i := start; i < end; i++ {
		line := m.lines[i]
		rendered := m.renderDiffLine(line, i+1) // 1-based line numbers
		renderedLines = append(renderedLines, rendered)
	}

	// If not enough lines to fill the view, pad with empty lines
	for len(renderedLines) < contentHeight {
		renderedLines = append(renderedLines, "")
	}

	box := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(t.Border).
		Width(m.width - 4).
		Render(strings.Join(renderedLines, "\n"))

	return box
}

func (m Model) renderDiffLine(line diffLine, lineNum int) string {
	t := theme.Current

	// Line number
	lineNumStyle := lipgloss.NewStyle().
		Foreground(t.Subtle).
		Width(5).
		Align(lipgloss.Right)
	lineNumStr := lineNumStyle.Render(fmt.Sprintf("%d", lineNum))

	// Content styling based on line type
	var contentStyle lipgloss.Style
	var prefix string

	switch line.lineType {
	case lineAdded:
		contentStyle = lipgloss.NewStyle().
			Foreground(t.Success).
			Background(lipgloss.Color("#1a3d1a"))
		prefix = "+"

	case lineRemoved:
		contentStyle = lipgloss.NewStyle().
			Foreground(t.Error).
			Background(lipgloss.Color("#3d1a1a"))
		prefix = "-"

	case lineHeader:
		contentStyle = lipgloss.NewStyle().
			Foreground(t.Primary).
			Bold(true)
		prefix = ""

	case lineHunk:
		contentStyle = lipgloss.NewStyle().
			Foreground(t.Info).
			Bold(true)
		prefix = ""

	case lineContext:
		contentStyle = lipgloss.NewStyle().
			Foreground(t.Subtle)
		prefix = " "

	default:
		contentStyle = lipgloss.NewStyle().
			Foreground(t.Foreground)
		prefix = " "
	}

	// Truncate content if too wide
	maxWidth := m.width - 12 // Account for line number and padding
	content := line.content
	if len(content) > maxWidth && maxWidth > 3 {
		content = content[:maxWidth-3] + "..."
	}

	// Add prefix for added/removed/context lines
	if prefix != "" && !strings.HasPrefix(content, prefix) {
		content = prefix + content
	}

	contentStr := contentStyle.Render(content)

	return lipgloss.JoinHorizontal(lipgloss.Left, lineNumStr, " ", contentStr)
}

func (m Model) renderFooter() string {
	t := theme.Current

	// Scroll indicator
	var scrollInfo string
	if len(m.lines) > m.contentHeight() {
		scrollInfo = fmt.Sprintf(" [%d-%d of %d lines]",
			m.scroll+1,
			min(m.scroll+m.contentHeight(), len(m.lines)),
			len(m.lines),
		)
	}

	help := lipgloss.NewStyle().
		Foreground(t.Subtle).
		Padding(1, 0, 0, 0).
		Render(fmt.Sprintf("Up/Down/PgUp/PgDown: Scroll%s", scrollInfo))

	return help
}

// SetSize updates the view dimensions
func (m *Model) SetSize(width, height int) {
	m.width = width
	m.height = height
}

// SetDiff sets the diff content
func (m *Model) SetDiff(storyKey, content string) {
	m.storyKey = storyKey
	m.content = content
	m.lines = parseDiff(content)
	m.loading = false
	m.scroll = 0
}

// SetLoading sets the loading state
func (m *Model) SetLoading(loading bool) {
	m.loading = loading
}

// Clear clears the diff content
func (m *Model) Clear() {
	m.storyKey = ""
	m.content = ""
	m.lines = nil
	m.scroll = 0
}

// contentHeight returns the available height for diff content
func (m Model) contentHeight() int {
	// Reserve space for header and footer
	height := m.height - 6
	if height < 1 {
		height = 1
	}
	return height
}

// maxScroll returns the maximum scroll position
func (m Model) maxScroll() int {
	contentHeight := m.contentHeight()
	if len(m.lines) <= contentHeight {
		return 0
	}
	return len(m.lines) - contentHeight
}

// diffStats holds diff statistics
type diffStats struct {
	added   int
	removed int
}

// getDiffStats returns statistics about the diff
func (m Model) getDiffStats() diffStats {
	var stats diffStats
	for _, line := range m.lines {
		switch line.lineType {
		case lineAdded:
			stats.added++
		case lineRemoved:
			stats.removed++
		}
	}
	return stats
}

// parseDiff parses raw diff content into typed lines
func parseDiff(content string) []diffLine {
	if content == "" {
		return nil
	}

	rawLines := strings.Split(content, "\n")
	lines := make([]diffLine, 0, len(rawLines))

	for _, raw := range rawLines {
		line := diffLine{content: raw}

		switch {
		case strings.HasPrefix(raw, "diff --git"):
			line.lineType = lineHeader
		case strings.HasPrefix(raw, "index "):
			line.lineType = lineHeader
		case strings.HasPrefix(raw, "---"):
			line.lineType = lineHeader
		case strings.HasPrefix(raw, "+++"):
			line.lineType = lineHeader
		case strings.HasPrefix(raw, "@@"):
			line.lineType = lineHunk
		case strings.HasPrefix(raw, "+"):
			line.lineType = lineAdded
		case strings.HasPrefix(raw, "-"):
			line.lineType = lineRemoved
		case strings.HasPrefix(raw, " "):
			line.lineType = lineContext
		default:
			line.lineType = lineNormal
		}

		lines = append(lines, line)
	}

	return lines
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
