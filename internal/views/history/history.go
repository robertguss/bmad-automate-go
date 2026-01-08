package history

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/robertguss/bmad-automate-go/internal/domain"
	"github.com/robertguss/bmad-automate-go/internal/messages"
	"github.com/robertguss/bmad-automate-go/internal/theme"
	"github.com/robertguss/bmad-automate-go/internal/util"
)

// Model represents the history view state
type Model struct {
	width      int
	height     int
	styles     theme.Styles
	executions []*messages.HistoryExecution
	cursor     int
	scroll     int
	totalCount int
	loading    bool
	errorMsg   string

	// Filter state
	filterQuery  string
	filterEpic   *int
	filterStatus domain.ExecutionStatus
	filtering    bool
}

// New creates a new history view model
func New() Model {
	return Model{
		styles:     theme.NewStyles(),
		executions: make([]*messages.HistoryExecution, 0),
		loading:    true,
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
		if m.filtering {
			return m.handleFilterInput(msg)
		}
		return m.handleKeyMsg(msg)

	case messages.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height

	case messages.HistoryLoadedMsg:
		m.loading = false
		if msg.Error != nil {
			m.errorMsg = msg.Error.Error()
			return m, nil
		}
		m.executions = msg.Executions
		m.totalCount = msg.TotalCount
		m.errorMsg = ""
		// Reset cursor if out of bounds
		if m.cursor >= len(m.executions) {
			m.cursor = 0
			m.scroll = 0
		}
	}

	return m, nil
}

func (m Model) handleKeyMsg(msg tea.KeyMsg) (Model, tea.Cmd) {
	switch msg.String() {
	case "up":
		if m.cursor > 0 {
			m.cursor--
			if m.cursor < m.scroll {
				m.scroll = m.cursor
			}
		}

	case "down":
		if m.cursor < len(m.executions)-1 {
			m.cursor++
			contentHeight := m.contentHeight()
			if m.cursor >= m.scroll+contentHeight {
				m.scroll = m.cursor - contentHeight + 1
			}
		}

	case "home":
		m.cursor = 0
		m.scroll = 0

	case "end":
		if len(m.executions) > 0 {
			m.cursor = len(m.executions) - 1
			contentHeight := m.contentHeight()
			if m.cursor >= contentHeight {
				m.scroll = m.cursor - contentHeight + 1
			}
		}

	case "pgup":
		contentHeight := m.contentHeight()
		m.cursor -= contentHeight
		if m.cursor < 0 {
			m.cursor = 0
		}
		m.scroll -= contentHeight
		if m.scroll < 0 {
			m.scroll = 0
		}

	case "pgdown":
		contentHeight := m.contentHeight()
		m.cursor += contentHeight
		if m.cursor >= len(m.executions) {
			m.cursor = len(m.executions) - 1
		}
		if m.cursor < 0 {
			m.cursor = 0
		}
		maxScroll := m.maxScroll()
		m.scroll += contentHeight
		if m.scroll > maxScroll {
			m.scroll = maxScroll
		}

	case "/":
		m.filtering = true
		m.filterQuery = ""

	case "r":
		m.loading = true
		return m, func() tea.Msg {
			return messages.HistoryRefreshMsg{}
		}

	case "c":
		// Clear filter
		m.filterQuery = ""
		m.filterEpic = nil
		m.filterStatus = ""
		m.loading = true
		return m, func() tea.Msg {
			return messages.HistoryRefreshMsg{}
		}

	case "enter":
		if len(m.executions) > 0 && m.cursor < len(m.executions) {
			exec := m.executions[m.cursor]
			return m, func() tea.Msg {
				return messages.HistoryDetailMsg{ID: exec.ID}
			}
		}
	}

	return m, nil
}

func (m Model) handleFilterInput(msg tea.KeyMsg) (Model, tea.Cmd) {
	switch msg.String() {
	case "enter":
		m.filtering = false
		m.loading = true
		return m, func() tea.Msg {
			return messages.HistoryFilterMsg{
				Query:  m.filterQuery,
				Epic:   m.filterEpic,
				Status: m.filterStatus,
			}
		}

	case "esc":
		m.filtering = false
		m.filterQuery = ""

	case "backspace":
		if len(m.filterQuery) > 0 {
			m.filterQuery = m.filterQuery[:len(m.filterQuery)-1]
		}

	default:
		if len(msg.String()) == 1 {
			m.filterQuery += msg.String()
		}
	}

	return m, nil
}

// View renders the history view
func (m Model) View() string {
	t := theme.Current

	if m.loading {
		return m.renderLoading()
	}

	if m.errorMsg != "" {
		return m.renderError()
	}

	var sections []string

	// Header with count and filter
	header := m.renderHeader()
	sections = append(sections, header)

	// Filter input if active
	if m.filtering {
		filterInput := lipgloss.NewStyle().
			Foreground(t.Accent).
			Render(fmt.Sprintf("Filter: %s_", m.filterQuery))
		sections = append(sections, filterInput)
	} else if m.filterQuery != "" {
		filterInfo := lipgloss.NewStyle().
			Foreground(t.Subtle).
			Render(fmt.Sprintf("Filtered by: %s (c to clear)", m.filterQuery))
		sections = append(sections, filterInfo)
	}

	// Execution list
	list := m.renderExecutionList()
	sections = append(sections, list)

	// Help footer
	footer := m.renderFooter()
	sections = append(sections, footer)

	return lipgloss.JoinVertical(lipgloss.Left, sections...)
}

func (m Model) renderLoading() string {
	t := theme.Current
	return lipgloss.NewStyle().
		Foreground(t.Subtle).
		Padding(2, 0).
		Render("Loading execution history...")
}

func (m Model) renderError() string {
	t := theme.Current
	return lipgloss.NewStyle().
		Foreground(t.Error).
		Padding(2, 0).
		Render(fmt.Sprintf("Error: %s", m.errorMsg))
}

func (m Model) renderHeader() string {
	t := theme.Current

	title := lipgloss.NewStyle().
		Foreground(t.Primary).
		Bold(true).
		Render("Execution History")

	count := lipgloss.NewStyle().
		Foreground(t.Subtle).
		Render(fmt.Sprintf("(%d executions)", m.totalCount))

	return lipgloss.JoinHorizontal(lipgloss.Left, title, " ", count)
}

func (m Model) renderExecutionList() string {
	if len(m.executions) == 0 {
		return lipgloss.NewStyle().
			Foreground(theme.Current.Subtle).
			Padding(1, 0).
			Render("No executions found")
	}

	t := theme.Current
	contentHeight := m.contentHeight()

	// Calculate visible range
	start := m.scroll
	end := start + contentHeight
	if end > len(m.executions) {
		end = len(m.executions)
	}

	var lines []string
	for i := start; i < end; i++ {
		exec := m.executions[i]
		line := m.renderExecutionRow(exec, i == m.cursor)
		lines = append(lines, line)
	}

	// Scroll indicator
	if m.maxScroll() > 0 {
		scrollInfo := lipgloss.NewStyle().
			Foreground(t.Subtle).
			Render(fmt.Sprintf(" [%d-%d of %d]", start+1, end, len(m.executions)))
		lines = append(lines, scrollInfo)
	}

	return strings.Join(lines, "\n")
}

func (m Model) renderExecutionRow(exec *messages.HistoryExecution, selected bool) string {
	t := theme.Current

	// Status indicator
	var statusStyle lipgloss.Style
	var statusIcon string
	switch exec.Status {
	case domain.ExecutionCompleted:
		statusStyle = lipgloss.NewStyle().Foreground(t.Success)
		statusIcon = "[OK]"
	case domain.ExecutionFailed:
		statusStyle = lipgloss.NewStyle().Foreground(t.Error)
		statusIcon = "[X]"
	case domain.ExecutionCancelled:
		statusStyle = lipgloss.NewStyle().Foreground(t.Warning)
		statusIcon = "[!]"
	default:
		statusStyle = lipgloss.NewStyle().Foreground(t.Subtle)
		statusIcon = "[-]"
	}

	// Format time
	timeStr := exec.StartTime.Format("2006-01-02 15:04")

	// Format duration
	durationStr := formatDuration(exec.Duration)

	// Build row
	status := statusStyle.Render(statusIcon)
	storyKey := lipgloss.NewStyle().
		Foreground(t.Primary).
		Width(20).
		Render(truncate(exec.StoryKey, 20))

	timeCol := lipgloss.NewStyle().
		Foreground(t.Subtle).
		Width(16).
		Render(timeStr)

	durationCol := lipgloss.NewStyle().
		Foreground(t.Foreground).
		Width(10).
		Render(durationStr)

	epicCol := lipgloss.NewStyle().
		Foreground(t.Secondary).
		Width(8).
		Render(fmt.Sprintf("E%d", exec.StoryEpic))

	row := lipgloss.JoinHorizontal(lipgloss.Left,
		status, " ",
		storyKey, " ",
		epicCol, " ",
		timeCol, " ",
		durationCol,
	)

	// Apply selection style
	if selected {
		row = lipgloss.NewStyle().
			Background(t.Selection).
			Foreground(t.Foreground).
			Bold(true).
			Width(m.width - 4).
			Render(row)
	}

	return row
}

func (m Model) renderFooter() string {
	t := theme.Current

	help := []string{
		"Up/Down: Navigate",
		"Enter: View Details",
		"/: Filter",
		"r: Refresh",
		"c: Clear Filter",
	}

	helpText := lipgloss.NewStyle().
		Foreground(t.Subtle).
		Render(strings.Join(help, " | "))

	return lipgloss.NewStyle().
		Padding(1, 0, 0, 0).
		Render(helpText)
}

// SetSize updates the view dimensions
func (m *Model) SetSize(width, height int) {
	m.width = width
	m.height = height
}

// RefreshStyles rebuilds styles after theme change
func (m *Model) RefreshStyles() {
	m.styles = theme.NewStyles()
}

// SetExecutions updates the execution list
func (m *Model) SetExecutions(executions []*messages.HistoryExecution, total int) {
	m.executions = executions
	m.totalCount = total
	m.loading = false
}

// SetLoading sets the loading state
func (m *Model) SetLoading(loading bool) {
	m.loading = loading
}

// GetFilter returns the current filter settings
func (m *Model) GetFilter() (string, *int, domain.ExecutionStatus) {
	return m.filterQuery, m.filterEpic, m.filterStatus
}

// contentHeight returns the available height for the execution list
func (m Model) contentHeight() int {
	// Reserve space for header (1), filter (1), footer (2), and some padding
	reserved := 5
	if m.filtering || m.filterQuery != "" {
		reserved++
	}
	height := m.height - reserved
	if height < 1 {
		height = 1
	}
	return height
}

// maxScroll returns the maximum scroll position
func (m Model) maxScroll() int {
	contentHeight := m.contentHeight()
	if len(m.executions) <= contentHeight {
		return 0
	}
	return len(m.executions) - contentHeight
}

// Helper functions

// formatDuration uses the shared compact duration formatter
// QUAL-002: Using shared utility instead of duplicated code
var formatDuration = util.FormatDurationCompact

func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	if maxLen <= 3 {
		return s[:maxLen]
	}
	return s[:maxLen-3] + "..."
}
