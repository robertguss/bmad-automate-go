package storylist

import (
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/robertguss/bmad-automate-go/internal/domain"
	"github.com/robertguss/bmad-automate-go/internal/messages"
	"github.com/robertguss/bmad-automate-go/internal/parser"
	"github.com/robertguss/bmad-automate-go/internal/theme"
)

// Model represents the story list view
type Model struct {
	width        int
	height       int
	stories      []domain.Story
	filtered     []domain.Story
	cursor       int
	selected     map[string]bool
	filterEpic   int
	filterStatus domain.StoryStatus
	epics        []int
	styles       theme.Styles
}

// New creates a new story list model
func New() Model {
	return Model{
		selected: make(map[string]bool),
		styles:   theme.NewStyles(),
	}
}

// Init initializes the story list
func (m Model) Init() tea.Cmd {
	return nil
}

// Update handles messages
func (m Model) Update(msg tea.Msg) (Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "up":
			if m.cursor > 0 {
				m.cursor--
			}
		case "down":
			if m.cursor < len(m.filtered)-1 {
				m.cursor++
			}
		case " ": // Space to toggle selection
			if len(m.filtered) > 0 {
				key := m.filtered[m.cursor].Key
				m.selected[key] = !m.selected[key]
				if !m.selected[key] {
					delete(m.selected, key)
				}
			}
		case "a": // Select all visible
			for _, s := range m.filtered {
				m.selected[s.Key] = true
			}
		case "n": // Deselect all
			m.selected = make(map[string]bool)
		case "e": // Cycle epic filter
			m.cycleEpicFilter()
		case "f": // Cycle status filter
			m.cycleStatusFilter()
		}

	case messages.StoriesLoadedMsg:
		if msg.Error == nil {
			m.stories = msg.Stories
			m.epics = parser.GetUniqueEpics(m.stories)
			m.applyFilters()
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
	m.epics = parser.GetUniqueEpics(stories)
	m.applyFilters()
}

// GetSelected returns the selected stories
func (m Model) GetSelected() []domain.Story {
	var selected []domain.Story
	for _, s := range m.stories {
		if m.selected[s.Key] {
			selected = append(selected, s)
		}
	}
	return selected
}

// GetCurrent returns the currently highlighted story
func (m Model) GetCurrent() *domain.Story {
	if len(m.filtered) > 0 && m.cursor < len(m.filtered) {
		return &m.filtered[m.cursor]
	}
	return nil
}

func (m *Model) cycleEpicFilter() {
	if len(m.epics) == 0 {
		return
	}

	if m.filterEpic == 0 {
		m.filterEpic = m.epics[0]
	} else {
		// Find current index and move to next
		for i, e := range m.epics {
			if e == m.filterEpic {
				if i+1 < len(m.epics) {
					m.filterEpic = m.epics[i+1]
				} else {
					m.filterEpic = 0 // Back to all
				}
				break
			}
		}
	}
	m.applyFilters()
}

func (m *Model) cycleStatusFilter() {
	statuses := []domain.StoryStatus{
		"", // All
		domain.StatusInProgress,
		domain.StatusReadyForDev,
		domain.StatusBacklog,
		domain.StatusDone,
	}

	for i, s := range statuses {
		if s == m.filterStatus {
			if i+1 < len(statuses) {
				m.filterStatus = statuses[i+1]
			} else {
				m.filterStatus = ""
			}
			break
		}
	}
	m.applyFilters()
}

func (m *Model) applyFilters() {
	m.filtered = m.stories

	// Apply epic filter
	if m.filterEpic > 0 {
		m.filtered = parser.FilterStoriesByEpic(m.filtered, m.filterEpic)
	}

	// Apply status filter
	if m.filterStatus != "" {
		m.filtered = parser.FilterStoriesByStatus(m.filtered, m.filterStatus)
	}

	// Reset cursor if out of bounds
	if m.cursor >= len(m.filtered) {
		m.cursor = max(0, len(m.filtered)-1)
	}
}

// View renders the story list
func (m Model) View() string {
	t := theme.Current

	// Header with filters
	filterInfo := "All Stories"
	if m.filterEpic > 0 {
		filterInfo = fmt.Sprintf("Epic %d", m.filterEpic)
	}
	if m.filterStatus != "" {
		filterInfo += fmt.Sprintf(" | %s", m.filterStatus)
	}

	header := lipgloss.NewStyle().
		Foreground(t.Primary).
		Bold(true).
		Render(fmt.Sprintf("Stories (%d)", len(m.filtered)))

	filterText := lipgloss.NewStyle().
		Foreground(t.Subtle).
		Render("  " + filterInfo)

	selectedCount := len(m.selected)
	selectedText := ""
	if selectedCount > 0 {
		selectedText = lipgloss.NewStyle().
			Foreground(t.Success).
			Bold(true).
			Render(fmt.Sprintf("  [%d selected]", selectedCount))
	}

	titleLine := header + filterText + selectedText

	// Help line
	help := lipgloss.NewStyle().
		Foreground(t.Subtle).
		Render("[Up/Down] Navigate  [Space] Select  [a] All  [n] None  [e] Epic  [f] Status  [Enter] Queue")

	// Story list
	var rows []string
	visibleHeight := m.height - 6 // Account for header, help, padding
	startIdx := 0
	if m.cursor >= visibleHeight {
		startIdx = m.cursor - visibleHeight + 1
	}

	for i := startIdx; i < len(m.filtered) && i < startIdx+visibleHeight; i++ {
		story := m.filtered[i]
		rows = append(rows, m.renderStoryRow(story, i == m.cursor))
	}

	if len(rows) == 0 {
		rows = append(rows, lipgloss.NewStyle().
			Foreground(t.Subtle).
			Italic(true).
			Render("  No stories match the current filters"))
	}

	content := lipgloss.JoinVertical(lipgloss.Left, rows...)

	// Combine everything
	view := lipgloss.JoinVertical(lipgloss.Left,
		titleLine,
		"",
		content,
		"",
		help,
	)

	return lipgloss.NewStyle().Padding(1, 2).Render(view)
}

func (m Model) renderStoryRow(story domain.Story, isCursor bool) string {
	t := theme.Current

	// Selection indicator
	selIndicator := "  "
	if m.selected[story.Key] {
		selIndicator = lipgloss.NewStyle().
			Foreground(t.Success).
			Bold(true).
			Render("* ")
	}

	// Cursor indicator
	cursor := "  "
	if isCursor {
		cursor = lipgloss.NewStyle().
			Foreground(t.Primary).
			Bold(true).
			Render("> ")
	}

	// Status badge
	var badge string
	switch story.Status {
	case domain.StatusInProgress:
		badge = m.styles.BadgeInProgress.Render(" IN PROGRESS ")
	case domain.StatusReadyForDev:
		badge = m.styles.BadgeReadyForDev.Render(" READY ")
	case domain.StatusBacklog:
		badge = m.styles.BadgeBacklog.Render(" BACKLOG ")
	case domain.StatusDone:
		badge = m.styles.BadgeDone.Render(" DONE ")
	default:
		badge = lipgloss.NewStyle().
			Foreground(t.Subtle).
			Render(fmt.Sprintf(" %s ", story.Status))
	}

	// Story key
	keyStyle := lipgloss.NewStyle().Foreground(t.Foreground)
	if isCursor {
		keyStyle = keyStyle.Foreground(t.Highlight).Bold(true)
	}
	key := keyStyle.Width(30).Render(story.Key)

	// File exists indicator
	fileIndicator := ""
	if story.FileExists {
		fileIndicator = lipgloss.NewStyle().
			Foreground(t.Info).
			Render(" [file exists]")
	}

	row := cursor + selIndicator + badge + "  " + key + fileIndicator

	// Highlight entire row if cursor
	if isCursor {
		row = lipgloss.NewStyle().
			Background(t.Selection).
			Width(m.width - 6).
			Render(row)
	}

	return row
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
