package stats

import (
	"fmt"
	"sort"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/robertguss/bmad-automate-go/internal/domain"
	"github.com/robertguss/bmad-automate-go/internal/messages"
	"github.com/robertguss/bmad-automate-go/internal/theme"
)

// Model represents the statistics view state
type Model struct {
	width    int
	height   int
	styles   theme.Styles
	stats    *messages.StatsData
	loading  bool
	errorMsg string
	scroll   int
}

// New creates a new statistics view model
func New() Model {
	return Model{
		styles:  theme.NewStyles(),
		loading: true,
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

	case messages.StatsLoadedMsg:
		m.loading = false
		if msg.Error != nil {
			m.errorMsg = msg.Error.Error()
			return m, nil
		}
		m.stats = msg.Stats
		m.errorMsg = ""
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
		m.scroll++

	case "home":
		m.scroll = 0

	case "r":
		m.loading = true
		return m, func() tea.Msg {
			return messages.StatsRefreshMsg{}
		}
	}

	return m, nil
}

// View renders the statistics view
func (m Model) View() string {
	if m.loading {
		return m.renderLoading()
	}

	if m.errorMsg != "" {
		return m.renderError()
	}

	if m.stats == nil {
		return m.renderNoData()
	}

	var sections []string

	// Title
	sections = append(sections, m.renderTitle())

	// Overview stats
	sections = append(sections, m.renderOverview())

	// Success rate chart
	sections = append(sections, m.renderSuccessRateChart())

	// Step statistics
	sections = append(sections, m.renderStepStats())

	// Activity by day chart
	sections = append(sections, m.renderActivityChart())

	// Executions by epic
	sections = append(sections, m.renderEpicChart())

	// Help footer
	sections = append(sections, m.renderFooter())

	content := lipgloss.JoinVertical(lipgloss.Left, sections...)

	// Apply scrolling if needed
	lines := strings.Split(content, "\n")
	if m.scroll > 0 && m.scroll < len(lines) {
		lines = lines[m.scroll:]
	}
	if len(lines) > m.height-2 {
		lines = lines[:m.height-2]
	}

	return strings.Join(lines, "\n")
}

func (m Model) renderLoading() string {
	t := theme.Current
	return lipgloss.NewStyle().
		Foreground(t.Subtle).
		Padding(2, 0).
		Render("Loading statistics...")
}

func (m Model) renderError() string {
	t := theme.Current
	return lipgloss.NewStyle().
		Foreground(t.Error).
		Padding(2, 0).
		Render(fmt.Sprintf("Error: %s", m.errorMsg))
}

func (m Model) renderNoData() string {
	t := theme.Current
	return lipgloss.NewStyle().
		Foreground(t.Subtle).
		Padding(2, 0).
		Render("No statistics available. Run some executions first!")
}

func (m Model) renderTitle() string {
	t := theme.Current
	return lipgloss.NewStyle().
		Foreground(t.Primary).
		Bold(true).
		Padding(0, 0, 1, 0).
		Render("Execution Statistics")
}

func (m Model) renderOverview() string {
	t := theme.Current
	s := m.stats

	// Build overview box
	titleStyle := lipgloss.NewStyle().Foreground(t.Secondary).Bold(true)
	valueStyle := lipgloss.NewStyle().Foreground(t.Foreground)
	mutedStyle := lipgloss.NewStyle().Foreground(t.Subtle)

	var rows []string

	// Total executions
	rows = append(rows, fmt.Sprintf("%s %s",
		titleStyle.Render("Total Executions:"),
		valueStyle.Render(fmt.Sprintf("%d", s.TotalExecutions)),
	))

	// Success/Fail/Cancelled breakdown
	successStyle := lipgloss.NewStyle().Foreground(t.Success)
	failStyle := lipgloss.NewStyle().Foreground(t.Error)
	cancelStyle := lipgloss.NewStyle().Foreground(t.Warning)

	breakdown := fmt.Sprintf("%s %s | %s %s | %s %s",
		successStyle.Render(fmt.Sprintf("%d", s.SuccessfulCount)), mutedStyle.Render("success"),
		failStyle.Render(fmt.Sprintf("%d", s.FailedCount)), mutedStyle.Render("failed"),
		cancelStyle.Render(fmt.Sprintf("%d", s.CancelledCount)), mutedStyle.Render("cancelled"),
	)
	rows = append(rows, breakdown)

	// Success rate with visual bar
	rateBar := m.renderProgressBar(s.SuccessRate, 20)
	rows = append(rows, fmt.Sprintf("%s %s %.1f%%",
		titleStyle.Render("Success Rate:"),
		rateBar,
		s.SuccessRate,
	))

	// Average duration
	rows = append(rows, fmt.Sprintf("%s %s",
		titleStyle.Render("Avg Duration:"),
		valueStyle.Render(formatDuration(s.AvgDuration)),
	))

	// Total time
	rows = append(rows, fmt.Sprintf("%s %s",
		titleStyle.Render("Total Time:"),
		valueStyle.Render(formatDuration(s.TotalDuration)),
	))

	box := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(t.Border).
		Padding(0, 2).
		Render(strings.Join(rows, "\n"))

	return box
}

func (m Model) renderSuccessRateChart() string {
	t := theme.Current
	s := m.stats

	title := lipgloss.NewStyle().
		Foreground(t.Secondary).
		Bold(true).
		Padding(1, 0, 0, 0).
		Render("Overall Success Rate")

	// Large visual representation
	bar := m.renderLargeProgressBar(s.SuccessRate, 40)

	return lipgloss.JoinVertical(lipgloss.Left, title, bar)
}

func (m Model) renderStepStats() string {
	t := theme.Current
	s := m.stats

	if len(s.StepStats) == 0 {
		return ""
	}

	title := lipgloss.NewStyle().
		Foreground(t.Secondary).
		Bold(true).
		Padding(1, 0, 0, 0).
		Render("Step Performance")

	// Sort steps by workflow order
	stepOrder := []domain.StepName{
		domain.StepCreateStory,
		domain.StepDevStory,
		domain.StepCodeReview,
		domain.StepGitCommit,
	}

	var rows []string
	headerStyle := lipgloss.NewStyle().Foreground(t.Subtle).Bold(true)
	header := fmt.Sprintf("%-15s %8s %8s %10s %10s",
		headerStyle.Render("Step"),
		headerStyle.Render("Success"),
		headerStyle.Render("Failed"),
		headerStyle.Render("Rate"),
		headerStyle.Render("Avg Time"),
	)
	rows = append(rows, header)
	rows = append(rows, strings.Repeat("-", 55))

	for _, stepName := range stepOrder {
		ss, ok := s.StepStats[stepName]
		if !ok {
			continue
		}

		nameStyle := lipgloss.NewStyle().Foreground(t.Primary)
		successStyle := lipgloss.NewStyle().Foreground(t.Success)
		failStyle := lipgloss.NewStyle().Foreground(t.Error)

		var rateStyle lipgloss.Style
		if ss.SuccessRate >= 80 {
			rateStyle = lipgloss.NewStyle().Foreground(t.Success)
		} else if ss.SuccessRate >= 50 {
			rateStyle = lipgloss.NewStyle().Foreground(t.Warning)
		} else {
			rateStyle = lipgloss.NewStyle().Foreground(t.Error)
		}

		row := fmt.Sprintf("%-15s %8s %8s %10s %10s",
			nameStyle.Render(string(ss.StepName)),
			successStyle.Render(fmt.Sprintf("%d", ss.SuccessCount)),
			failStyle.Render(fmt.Sprintf("%d", ss.FailureCount)),
			rateStyle.Render(fmt.Sprintf("%.1f%%", ss.SuccessRate)),
			formatDuration(ss.AvgDuration),
		)
		rows = append(rows, row)
	}

	table := strings.Join(rows, "\n")
	return lipgloss.JoinVertical(lipgloss.Left, title, table)
}

func (m Model) renderActivityChart() string {
	t := theme.Current
	s := m.stats

	if len(s.ExecutionsByDay) == 0 {
		return ""
	}

	title := lipgloss.NewStyle().
		Foreground(t.Secondary).
		Bold(true).
		Padding(1, 0, 0, 0).
		Render("Activity (Last 7 Days)")

	// Get last 7 days
	var days []string
	for i := 6; i >= 0; i-- {
		day := time.Now().AddDate(0, 0, -i).Format("2006-01-02")
		days = append(days, day)
	}

	// Find max for scaling
	maxCount := 1
	for _, count := range s.ExecutionsByDay {
		if count > maxCount {
			maxCount = count
		}
	}

	var rows []string
	for _, day := range days {
		count := s.ExecutionsByDay[day]
		barLen := int(float64(count) / float64(maxCount) * 30)
		if barLen < 0 {
			barLen = 0
		}

		bar := lipgloss.NewStyle().
			Foreground(t.Accent).
			Render(strings.Repeat("=", barLen))

		dayLabel := lipgloss.NewStyle().
			Foreground(t.Subtle).
			Width(12).
			Render(day[5:]) // Show MM-DD only

		countLabel := lipgloss.NewStyle().
			Foreground(t.Foreground).
			Width(4).
			Align(lipgloss.Right).
			Render(fmt.Sprintf("%d", count))

		row := lipgloss.JoinHorizontal(lipgloss.Left, dayLabel, bar, " ", countLabel)
		rows = append(rows, row)
	}

	chart := strings.Join(rows, "\n")
	return lipgloss.JoinVertical(lipgloss.Left, title, chart)
}

func (m Model) renderEpicChart() string {
	t := theme.Current
	s := m.stats

	if len(s.ExecutionsByEpic) == 0 {
		return ""
	}

	title := lipgloss.NewStyle().
		Foreground(t.Secondary).
		Bold(true).
		Padding(1, 0, 0, 0).
		Render("Executions by Epic")

	// Sort epics
	var epics []int
	for epic := range s.ExecutionsByEpic {
		epics = append(epics, epic)
	}
	sort.Ints(epics)

	// Find max for scaling
	maxCount := 1
	for _, count := range s.ExecutionsByEpic {
		if count > maxCount {
			maxCount = count
		}
	}

	var rows []string
	for _, epic := range epics {
		count := s.ExecutionsByEpic[epic]
		barLen := int(float64(count) / float64(maxCount) * 30)
		if barLen < 0 {
			barLen = 0
		}

		bar := lipgloss.NewStyle().
			Foreground(t.Secondary).
			Render(strings.Repeat("=", barLen))

		epicLabel := lipgloss.NewStyle().
			Foreground(t.Primary).
			Width(8).
			Render(fmt.Sprintf("Epic %d", epic))

		countLabel := lipgloss.NewStyle().
			Foreground(t.Foreground).
			Width(4).
			Align(lipgloss.Right).
			Render(fmt.Sprintf("%d", count))

		row := lipgloss.JoinHorizontal(lipgloss.Left, epicLabel, bar, " ", countLabel)
		rows = append(rows, row)
	}

	chart := strings.Join(rows, "\n")
	return lipgloss.JoinVertical(lipgloss.Left, title, chart)
}

func (m Model) renderFooter() string {
	t := theme.Current

	help := lipgloss.NewStyle().
		Foreground(t.Subtle).
		Padding(1, 0, 0, 0).
		Render("Up/Down: Scroll | r: Refresh")

	return help
}

// renderProgressBar creates a simple progress bar
func (m Model) renderProgressBar(percent float64, width int) string {
	t := theme.Current

	filled := int(percent / 100 * float64(width))
	if filled > width {
		filled = width
	}
	if filled < 0 {
		filled = 0
	}
	empty := width - filled

	var color lipgloss.Color
	if percent >= 80 {
		color = t.Success
	} else if percent >= 50 {
		color = t.Warning
	} else {
		color = t.Error
	}

	filledBar := lipgloss.NewStyle().Foreground(color).Render(strings.Repeat("=", filled))
	emptyBar := lipgloss.NewStyle().Foreground(t.Subtle).Render(strings.Repeat("-", empty))

	return fmt.Sprintf("[%s%s]", filledBar, emptyBar)
}

// renderLargeProgressBar creates a larger visual progress bar
func (m Model) renderLargeProgressBar(percent float64, width int) string {
	t := theme.Current

	filled := int(percent / 100 * float64(width))
	if filled > width {
		filled = width
	}
	if filled < 0 {
		filled = 0
	}
	empty := width - filled

	var color lipgloss.Color
	if percent >= 80 {
		color = t.Success
	} else if percent >= 50 {
		color = t.Warning
	} else {
		color = t.Error
	}

	filledBar := lipgloss.NewStyle().
		Background(color).
		Foreground(t.Background).
		Render(strings.Repeat(" ", filled))

	emptyBar := lipgloss.NewStyle().
		Background(t.Border).
		Render(strings.Repeat(" ", empty))

	percentLabel := lipgloss.NewStyle().
		Foreground(color).
		Bold(true).
		Render(fmt.Sprintf(" %.1f%%", percent))

	return lipgloss.JoinHorizontal(lipgloss.Left, filledBar, emptyBar, percentLabel)
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

// SetStats updates the statistics data
func (m *Model) SetStats(stats *messages.StatsData) {
	m.stats = stats
	m.loading = false
}

// SetLoading sets the loading state
func (m *Model) SetLoading(loading bool) {
	m.loading = loading
}

// Helper functions

func formatDuration(d time.Duration) string {
	if d < time.Second {
		return fmt.Sprintf("%dms", d.Milliseconds())
	}
	if d < time.Minute {
		return fmt.Sprintf("%.1fs", d.Seconds())
	}
	if d < time.Hour {
		mins := int(d.Minutes())
		secs := int(d.Seconds()) % 60
		return fmt.Sprintf("%dm%ds", mins, secs)
	}
	hours := int(d.Hours())
	mins := int(d.Minutes()) % 60
	return fmt.Sprintf("%dh%dm", hours, mins)
}
