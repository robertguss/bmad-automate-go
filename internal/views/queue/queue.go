package queue

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

// Model represents the queue manager view
type Model struct {
	width  int
	height int
	queue  *domain.Queue
	cursor int
	styles theme.Styles
}

// New creates a new queue manager model
func New() Model {
	return Model{
		queue:  domain.NewQueue(),
		cursor: 0,
		styles: theme.NewStyles(),
	}
}

// Init initializes the queue manager
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
			if m.cursor < len(m.queue.Items)-1 {
				m.cursor++
			}
		case "K": // Shift+K to move up
			if m.queue.MoveUp(m.cursor) {
				m.cursor--
			}
		case "J": // Shift+J to move down
			if m.queue.MoveDown(m.cursor) {
				m.cursor++
			}
		case "delete", "backspace", "x":
			if m.cursor < len(m.queue.Items) {
				item := m.queue.Items[m.cursor]
				if item.Status == domain.ExecutionPending {
					m.queue.Remove(item.Story.Key)
					if m.cursor >= len(m.queue.Items) && m.cursor > 0 {
						m.cursor--
					}
				}
			}
		case "C": // Shift+C to clear pending
			m.queue.Clear()
			m.cursor = 0
		}

	case messages.QueueAddMsg:
		m.queue.AddMultiple(msg.Stories)

	case messages.QueueRemoveMsg:
		m.queue.Remove(msg.Key)
		if m.cursor >= len(m.queue.Items) && m.cursor > 0 {
			m.cursor--
		}

	case messages.QueueClearMsg:
		m.queue.Clear()
		m.cursor = 0

	case messages.QueueMoveUpMsg:
		if m.queue.MoveUp(msg.Index) && m.cursor == msg.Index {
			m.cursor--
		}

	case messages.QueueMoveDownMsg:
		if m.queue.MoveDown(msg.Index) && m.cursor == msg.Index {
			m.cursor++
		}

	case messages.QueueItemStartedMsg:
		m.queue.Current = msg.Index
		m.queue.Status = domain.QueueRunning
		if msg.Index < len(m.queue.Items) {
			m.queue.Items[msg.Index].Status = domain.ExecutionRunning
			m.queue.Items[msg.Index].Execution = msg.Execution
		}

	case messages.QueueItemCompletedMsg:
		if msg.Index < len(m.queue.Items) {
			m.queue.Items[msg.Index].Status = msg.Status
		}

	case messages.QueueCompletedMsg:
		m.queue.Status = domain.QueueCompleted
		m.queue.EndTime = time.Now()

	case messages.QueueUpdatedMsg:
		if msg.Queue != nil {
			m.queue = msg.Queue
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

// RefreshStyles rebuilds styles after theme change
func (m *Model) RefreshStyles() {
	m.styles = theme.NewStyles()
}

// GetQueue returns the current queue
func (m Model) GetQueue() *domain.Queue {
	return m.queue
}

// SetQueue sets the queue
func (m *Model) SetQueue(q *domain.Queue) {
	m.queue = q
}

// AddStories adds stories to the queue
func (m *Model) AddStories(stories []domain.Story) {
	m.queue.AddMultiple(stories)
}

// GetCursor returns the current cursor position
func (m Model) GetCursor() int {
	return m.cursor
}

// GetCurrentItem returns the item at the cursor
func (m Model) GetCurrentItem() *domain.QueueItem {
	if m.cursor >= 0 && m.cursor < len(m.queue.Items) {
		return m.queue.Items[m.cursor]
	}
	return nil
}

// View renders the queue manager
func (m Model) View() string {
	if m.width == 0 || m.height == 0 {
		return ""
	}

	t := theme.Current

	// Header with queue status and counts
	header := m.renderHeader()

	// Progress bar (if running)
	var progressBar string
	if m.queue.Status == domain.QueueRunning {
		progressBar = m.renderProgressBar()
	}

	// Queue list
	queueList := m.renderQueueList()

	// Help/controls
	help := m.renderHelp()

	// Combine all sections
	var sections []string
	sections = append(sections, header)
	if progressBar != "" {
		sections = append(sections, progressBar)
	}
	sections = append(sections, "", queueList, "", help)

	content := lipgloss.JoinVertical(lipgloss.Left, sections...)

	return lipgloss.NewStyle().
		Padding(1, 2).
		Width(m.width).
		Height(m.height).
		Foreground(t.Foreground).
		Render(content)
}

// renderHeader renders the queue header with status and counts
func (m Model) renderHeader() string {
	t := theme.Current

	// Title
	title := lipgloss.NewStyle().
		Foreground(t.Primary).
		Bold(true).
		Render("Queue Manager")

	// Status badge
	var statusBadge string
	switch m.queue.Status {
	case domain.QueueIdle:
		statusBadge = lipgloss.NewStyle().
			Foreground(t.Subtle).
			Render("[IDLE]")
	case domain.QueueRunning:
		statusBadge = lipgloss.NewStyle().
			Foreground(t.Warning).
			Bold(true).
			Render("[RUNNING]")
	case domain.QueuePaused:
		statusBadge = lipgloss.NewStyle().
			Foreground(t.Info).
			Bold(true).
			Render("[PAUSED]")
	case domain.QueueCompleted:
		statusBadge = lipgloss.NewStyle().
			Foreground(t.Success).
			Bold(true).
			Render("[COMPLETED]")
	}

	// Counts
	total := m.queue.TotalCount()
	pending := m.queue.PendingCount()
	completed := m.queue.CompletedCount()
	failed := m.queue.FailedCount()

	counts := lipgloss.NewStyle().
		Foreground(t.Subtle).
		Render(fmt.Sprintf("Total: %d | Pending: %d | Completed: %d | Failed: %d",
			total, pending, completed, failed))

	// ETA (if running)
	var eta string
	if m.queue.Status == domain.QueueRunning && m.queue.HasPending() {
		remaining := m.queue.EstimatedTimeRemaining()
		eta = lipgloss.NewStyle().
			Foreground(t.Info).
			Render(fmt.Sprintf("ETA: %s", formatDuration(remaining)))
	}

	headerLine := fmt.Sprintf("%s  %s", title, statusBadge)
	if eta != "" {
		headerLine = fmt.Sprintf("%s  %s", headerLine, eta)
	}

	return lipgloss.JoinVertical(lipgloss.Left, headerLine, counts)
}

// renderProgressBar renders the overall progress bar
func (m Model) renderProgressBar() string {
	t := theme.Current

	progress := m.queue.ProgressPercent()
	barWidth := m.width - 20

	filled := int(progress / 100 * float64(barWidth))
	if filled > barWidth {
		filled = barWidth
	}

	bar := strings.Repeat("=", filled) + strings.Repeat("-", barWidth-filled)

	return lipgloss.NewStyle().
		Foreground(t.Success).
		Render(fmt.Sprintf("[%s] %.0f%%", bar, progress))
}

// renderQueueList renders the list of queued items
func (m Model) renderQueueList() string {
	t := theme.Current

	if m.queue.IsEmpty() {
		return lipgloss.NewStyle().
			Foreground(t.Subtle).
			Italic(true).
			Render("  Queue is empty. Select stories and press 'Q' to add them.")
	}

	var rows []string
	visibleHeight := m.height - 10 // Account for header, progress, help

	// Calculate scroll offset
	startIdx := 0
	if m.cursor >= visibleHeight {
		startIdx = m.cursor - visibleHeight + 1
	}

	for i := startIdx; i < len(m.queue.Items) && i < startIdx+visibleHeight; i++ {
		item := m.queue.Items[i]
		rows = append(rows, m.renderQueueItem(item, i, i == m.cursor))
	}

	return lipgloss.JoinVertical(lipgloss.Left, rows...)
}

// renderQueueItem renders a single queue item
func (m Model) renderQueueItem(item *domain.QueueItem, index int, isCursor bool) string {
	t := theme.Current

	// Position number
	position := lipgloss.NewStyle().
		Foreground(t.Subtle).
		Width(4).
		Render(fmt.Sprintf("%d.", item.Position))

	// Status indicator
	var indicator string
	var keyStyle lipgloss.Style

	switch item.Status {
	case domain.ExecutionPending:
		indicator = lipgloss.NewStyle().Foreground(t.Subtle).Render("  ")
		keyStyle = lipgloss.NewStyle().Foreground(t.Foreground)
	case domain.ExecutionRunning:
		indicator = lipgloss.NewStyle().Foreground(t.Warning).Bold(true).Render("> ")
		keyStyle = lipgloss.NewStyle().Foreground(t.Warning).Bold(true)
	case domain.ExecutionCompleted:
		indicator = lipgloss.NewStyle().Foreground(t.Success).Render("OK")
		keyStyle = lipgloss.NewStyle().Foreground(t.Success)
	case domain.ExecutionFailed:
		indicator = lipgloss.NewStyle().Foreground(t.Error).Render("XX")
		keyStyle = lipgloss.NewStyle().Foreground(t.Error)
	case domain.ExecutionCancelled:
		indicator = lipgloss.NewStyle().Foreground(t.Warning).Render("--")
		keyStyle = lipgloss.NewStyle().Foreground(t.Subtle).Italic(true)
	case domain.ExecutionPaused:
		indicator = lipgloss.NewStyle().Foreground(t.Info).Render("||")
		keyStyle = lipgloss.NewStyle().Foreground(t.Info)
	}

	// Story key
	key := keyStyle.Width(50).Render(item.Story.Key)

	// Status badge
	var badge string
	switch item.Story.Status {
	case domain.StatusInProgress:
		badge = m.styles.BadgeInProgress.Render(" IN PROGRESS ")
	case domain.StatusReadyForDev:
		badge = m.styles.BadgeReadyForDev.Render(" READY ")
	case domain.StatusBacklog:
		badge = m.styles.BadgeBacklog.Render(" BACKLOG ")
	default:
		badge = ""
	}

	// Duration (if completed)
	var duration string
	if item.Execution != nil && item.Execution.Duration > 0 {
		duration = lipgloss.NewStyle().
			Foreground(t.Subtle).
			Render(fmt.Sprintf(" (%s)", formatDuration(item.Execution.Duration)))
	}

	// Progress (if running)
	var progress string
	if item.Status == domain.ExecutionRunning && item.Execution != nil {
		pct := item.Execution.ProgressPercent()
		progress = lipgloss.NewStyle().
			Foreground(t.Info).
			Render(fmt.Sprintf(" %.0f%%", pct))
	}

	// File exists indicator
	fileIndicator := ""
	if item.Story.FileExists {
		fileIndicator = lipgloss.NewStyle().
			Foreground(t.Info).
			Render(" [file]")
	}

	// Cursor indicator
	cursor := "  "
	if isCursor {
		cursor = lipgloss.NewStyle().
			Foreground(t.Primary).
			Bold(true).
			Render("> ")
	}

	row := fmt.Sprintf("%s%s%s %s %s%s%s%s", cursor, position, indicator, key, badge, fileIndicator, progress, duration)

	// Highlight entire row if cursor
	if isCursor {
		row = lipgloss.NewStyle().
			Background(t.Selection).
			Width(m.width - 6).
			Render(row)
	}

	return row
}

// renderHelp renders the control help line
func (m Model) renderHelp() string {
	t := theme.Current

	var controls []string

	if m.queue.Status == domain.QueueIdle {
		if m.queue.HasPending() {
			controls = append(controls, renderControl("Enter", "Start"))
		}
		controls = append(controls,
			renderControl("K/J", "Move Up/Down"),
			renderControl("x", "Remove"),
			renderControl("C", "Clear"),
		)
	} else if m.queue.Status == domain.QueueRunning {
		controls = append(controls,
			renderControl("p", "Pause"),
			renderControl("c", "Cancel"),
		)
	} else if m.queue.Status == domain.QueuePaused {
		controls = append(controls,
			renderControl("r", "Resume"),
			renderControl("c", "Cancel"),
		)
	}

	controls = append(controls, renderControl("Up/Down", "Navigate"))

	return lipgloss.NewStyle().
		Foreground(t.Subtle).
		Render(strings.Join(controls, "  "))
}

// renderControl renders a single control hint
func renderControl(key, action string) string {
	t := theme.Current
	keyStyle := lipgloss.NewStyle().Foreground(t.Accent).Bold(true)
	actionStyle := lipgloss.NewStyle().Foreground(t.Subtle)
	return fmt.Sprintf("[%s] %s", keyStyle.Render(key), actionStyle.Render(action))
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
