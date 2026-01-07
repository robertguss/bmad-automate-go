package timeline

import (
	"fmt"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/robertguss/bmad-automate-go/internal/domain"
	"github.com/robertguss/bmad-automate-go/internal/messages"
	"github.com/robertguss/bmad-automate-go/internal/theme"
)

// Model represents the timeline view
type Model struct {
	width      int
	height     int
	queue      *domain.Queue
	executions []*domain.Execution // Historical executions for display
	scroll     int
	styles     theme.Styles
}

// New creates a new timeline model
func New() Model {
	return Model{
		executions: make([]*domain.Execution, 0),
		styles:     theme.NewStyles(),
	}
}

// Init initializes the timeline view
func (m Model) Init() tea.Cmd {
	return nil
}

// Update handles messages
func (m Model) Update(msg tea.Msg) (Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
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
		}

	case messages.QueueUpdatedMsg:
		m.queue = msg.Queue

	case messages.ExecutionCompletedMsg:
		// Add completed execution to the list
		if m.queue != nil && m.queue.Current >= 0 && m.queue.Current < len(m.queue.Items) {
			item := m.queue.Items[m.queue.Current]
			if item.Execution != nil {
				m.executions = append(m.executions, item.Execution)
			}
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

// SetQueue sets the queue reference
func (m *Model) SetQueue(q *domain.Queue) {
	m.queue = q
}

// AddExecution adds an execution to the timeline
func (m *Model) AddExecution(exec *domain.Execution) {
	m.executions = append(m.executions, exec)
}

// ClearExecutions clears all executions
func (m *Model) ClearExecutions() {
	m.executions = make([]*domain.Execution, 0)
}

// maxScroll returns the maximum scroll position
func (m Model) maxScroll() int {
	totalRows := len(m.executions)
	visibleRows := m.height - 8
	if totalRows <= visibleRows {
		return 0
	}
	return totalRows - visibleRows
}

// View renders the timeline view
func (m Model) View() string {
	if m.width == 0 || m.height == 0 {
		return ""
	}

	t := theme.Current

	// Header
	title := lipgloss.NewStyle().
		Foreground(t.Primary).
		Bold(true).
		Render("Timeline")

	subtitle := lipgloss.NewStyle().
		Foreground(t.Subtle).
		Render(fmt.Sprintf("  (%d executions)", len(m.executions)))

	header := title + subtitle

	// Summary stats
	summary := m.renderSummary()

	// Timeline content
	content := m.renderTimeline()

	// Help
	help := lipgloss.NewStyle().
		Foreground(t.Subtle).
		Render("[Up/Down] Scroll  [Home/End] Jump")

	// Combine all sections
	view := lipgloss.JoinVertical(lipgloss.Left,
		header,
		summary,
		"",
		content,
		"",
		help,
	)

	return lipgloss.NewStyle().Padding(1, 2).Render(view)
}

// renderSummary renders summary statistics
func (m Model) renderSummary() string {
	t := theme.Current

	if len(m.executions) == 0 {
		return lipgloss.NewStyle().
			Foreground(t.Subtle).
			Italic(true).
			Render("No executions recorded yet")
	}

	// Calculate stats
	var totalDuration time.Duration
	var successCount, failedCount int
	stepDurations := make(map[domain.StepName][]time.Duration)

	for _, exec := range m.executions {
		totalDuration += exec.Duration
		if exec.Status == domain.ExecutionCompleted {
			successCount++
		} else if exec.Status == domain.ExecutionFailed {
			failedCount++
		}

		for _, step := range exec.Steps {
			if step.Duration > 0 {
				stepDurations[step.Name] = append(stepDurations[step.Name], step.Duration)
			}
		}
	}

	avgDuration := totalDuration / time.Duration(len(m.executions))

	// Format summary line
	stats := fmt.Sprintf("Total: %s | Avg: %s | Success: %d | Failed: %d",
		formatDuration(totalDuration),
		formatDuration(avgDuration),
		successCount,
		failedCount,
	)

	return lipgloss.NewStyle().Foreground(t.Subtle).Render(stats)
}

// renderTimeline renders the main timeline visualization
func (m Model) renderTimeline() string {
	t := theme.Current

	if len(m.executions) == 0 {
		return ""
	}

	// Find max duration for scaling
	var maxDuration time.Duration
	for _, exec := range m.executions {
		if exec.Duration > maxDuration {
			maxDuration = exec.Duration
		}
	}

	if maxDuration == 0 {
		maxDuration = time.Minute // Fallback
	}

	// Calculate available width for bars
	keyWidth := 35
	durationWidth := 12
	barWidth := m.width - keyWidth - durationWidth - 10

	if barWidth < 20 {
		barWidth = 20
	}

	var rows []string

	// Column headers
	headerStyle := lipgloss.NewStyle().Foreground(t.Subtle).Bold(true)
	headers := fmt.Sprintf("%s  %s  %s",
		headerStyle.Width(keyWidth).Render("Story"),
		headerStyle.Width(durationWidth).Render("Duration"),
		headerStyle.Render("Timeline"),
	)
	rows = append(rows, headers)
	rows = append(rows, strings.Repeat("-", m.width-6))

	// Visible range
	visibleHeight := m.height - 10
	startIdx := m.scroll
	endIdx := startIdx + visibleHeight
	if endIdx > len(m.executions) {
		endIdx = len(m.executions)
	}

	// Render each execution
	for i := startIdx; i < endIdx; i++ {
		exec := m.executions[i]
		rows = append(rows, m.renderExecutionRow(exec, barWidth, maxDuration))
	}

	return lipgloss.JoinVertical(lipgloss.Left, rows...)
}

// renderExecutionRow renders a single execution as a timeline row
func (m Model) renderExecutionRow(exec *domain.Execution, barWidth int, maxDuration time.Duration) string {
	t := theme.Current

	// Story key
	keyStyle := lipgloss.NewStyle().Foreground(t.Foreground)
	if exec.Status == domain.ExecutionCompleted {
		keyStyle = keyStyle.Foreground(t.Success)
	} else if exec.Status == domain.ExecutionFailed {
		keyStyle = keyStyle.Foreground(t.Error)
	}
	key := keyStyle.Width(35).Render(exec.Story.Key)

	// Duration
	durationStyle := lipgloss.NewStyle().Foreground(t.Subtle)
	duration := durationStyle.Width(12).Render(formatDuration(exec.Duration))

	// Step bars
	bar := m.renderStepBars(exec, barWidth, maxDuration)

	return fmt.Sprintf("%s  %s  %s", key, duration, bar)
}

// renderStepBars renders the colored step duration bars
func (m Model) renderStepBars(exec *domain.Execution, barWidth int, maxDuration time.Duration) string {
	t := theme.Current

	if exec.Duration == 0 {
		return strings.Repeat("-", barWidth)
	}

	// Calculate scale factor
	scale := float64(barWidth) / float64(maxDuration)

	var bar strings.Builder
	totalWidth := 0

	// Step colors
	stepColors := map[domain.StepName]lipgloss.Color{
		domain.StepCreateStory: t.Info,
		domain.StepDevStory:    t.Primary,
		domain.StepCodeReview:  t.Warning,
		domain.StepGitCommit:   t.Success,
	}

	for _, step := range exec.Steps {
		if step.Status == domain.StepSkipped {
			continue
		}

		// Calculate width for this step
		stepWidth := int(float64(step.Duration) * scale)
		if stepWidth < 1 && step.Duration > 0 {
			stepWidth = 1
		}

		// Ensure we don't exceed bar width
		if totalWidth+stepWidth > barWidth {
			stepWidth = barWidth - totalWidth
		}

		if stepWidth <= 0 {
			continue
		}

		// Get color for this step
		color := t.Subtle
		if c, ok := stepColors[step.Name]; ok {
			color = c
		}

		// Add failure indicator if step failed
		char := "="
		if step.Status == domain.StepFailed {
			color = t.Error
			char = "X"
		}

		// Render the bar segment
		style := lipgloss.NewStyle().Foreground(color)
		bar.WriteString(style.Render(strings.Repeat(char, stepWidth)))
		totalWidth += stepWidth
	}

	// Fill remaining space
	if totalWidth < barWidth {
		bar.WriteString(lipgloss.NewStyle().
			Foreground(t.Subtle).
			Render(strings.Repeat("-", barWidth-totalWidth)))
	}

	return bar.String()
}

// formatDuration formats a duration for display
func formatDuration(d time.Duration) string {
	if d < time.Minute {
		return fmt.Sprintf("%ds", int(d.Seconds()))
	}
	if d < time.Hour {
		minutes := int(d.Minutes())
		seconds := int(d.Seconds()) % 60
		return fmt.Sprintf("%dm %02ds", minutes, seconds)
	}
	hours := int(d.Hours())
	minutes := int(d.Minutes()) % 60
	return fmt.Sprintf("%dh %02dm", hours, minutes)
}
