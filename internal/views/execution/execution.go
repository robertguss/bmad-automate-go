package execution

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

const (
	maxOutputLines = 500 // Maximum lines to keep in output buffer
	leftPaneWidth  = 35  // Width of the step list pane
)

// Model represents the execution view
type Model struct {
	width     int
	height    int
	execution *domain.Execution
	output    []outputLine
	scroll    int // Current scroll position in output
	styles    theme.Styles
	startTime time.Time
	elapsed   time.Duration
}

type outputLine struct {
	text     string
	isStderr bool
	step     int
}

// New creates a new execution view model
func New() Model {
	return Model{
		output: make([]outputLine, 0, maxOutputLines),
		styles: theme.NewStyles(),
	}
}

// Init initializes the execution view
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
		case "pgup":
			m.scroll -= 10
			if m.scroll < 0 {
				m.scroll = 0
			}
		case "pgdown":
			m.scroll += 10
			maxScroll := m.maxScroll()
			if m.scroll > maxScroll {
				m.scroll = maxScroll
			}
		case "home":
			m.scroll = 0
		case "end":
			m.scroll = m.maxScroll()
		}

	case messages.ExecutionStartedMsg:
		m.execution = msg.Execution
		m.output = make([]outputLine, 0, maxOutputLines)
		m.scroll = 0
		m.startTime = time.Now()
		m.elapsed = 0

	case messages.StepStartedMsg:
		if m.execution != nil && msg.StepIndex < len(m.execution.Steps) {
			step := m.execution.Steps[msg.StepIndex]
			step.Status = domain.StepRunning
			step.Attempt = msg.Attempt
			step.Command = msg.Command
			step.StartTime = time.Now()
			m.execution.Current = msg.StepIndex

			// Add a separator for the new step
			m.addOutput(fmt.Sprintf("--- %s (attempt %d) ---", msg.StepName, msg.Attempt), false, msg.StepIndex)
		}

	case messages.StepOutputMsg:
		m.addOutput(msg.Line, msg.IsStderr, msg.StepIndex)
		// Auto-scroll to bottom when new output arrives
		m.scroll = m.maxScroll()

	case messages.StepCompletedMsg:
		if m.execution != nil && msg.StepIndex < len(m.execution.Steps) {
			step := m.execution.Steps[msg.StepIndex]
			step.Status = msg.Status
			step.Duration = msg.Duration
			step.EndTime = time.Now()
			if msg.Error != "" {
				step.Error = msg.Error
			}
		}

	case messages.ExecutionCompletedMsg:
		if m.execution != nil {
			m.execution.Status = msg.Status
			m.execution.Duration = msg.Duration
			m.execution.EndTime = time.Now()
			if msg.Error != "" {
				m.execution.Error = msg.Error
			}
		}

	case messages.ExecutionTickMsg:
		if m.execution != nil && m.execution.Status == domain.ExecutionRunning {
			m.elapsed = time.Since(m.startTime)
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

// SetExecution sets the current execution
func (m *Model) SetExecution(exec *domain.Execution) {
	m.execution = exec
	m.output = make([]outputLine, 0, maxOutputLines)
	m.scroll = 0
	m.startTime = time.Now()
}

// GetExecution returns the current execution
func (m Model) GetExecution() *domain.Execution {
	return m.execution
}

// addOutput adds a line to the output buffer
func (m *Model) addOutput(line string, isStderr bool, step int) {
	m.output = append(m.output, outputLine{
		text:     line,
		isStderr: isStderr,
		step:     step,
	})

	// Trim if too many lines
	if len(m.output) > maxOutputLines {
		m.output = m.output[len(m.output)-maxOutputLines:]
	}
}

// maxScroll returns the maximum scroll position
func (m Model) maxScroll() int {
	outputHeight := m.height - 8 // Account for header, footer, borders
	if len(m.output) <= outputHeight {
		return 0
	}
	return len(m.output) - outputHeight
}

// View renders the execution view
func (m Model) View() string {
	if m.width == 0 || m.height == 0 {
		return ""
	}

	t := theme.Current

	// Calculate pane dimensions
	rightPaneWidth := m.width - leftPaneWidth - 5 // 5 for borders and padding
	contentHeight := m.height - 4                 // Account for controls at bottom

	// Render left pane (step list)
	leftPane := m.renderStepList(leftPaneWidth, contentHeight)

	// Render right pane (output)
	rightPane := m.renderOutput(rightPaneWidth, contentHeight)

	// Combine panes horizontally
	content := lipgloss.JoinHorizontal(lipgloss.Top, leftPane, rightPane)

	// Render controls at bottom
	controls := m.renderControls()

	// Status line
	var statusLine string
	if m.execution != nil {
		statusText := m.renderStatusBadge()
		elapsed := formatDuration(m.elapsed)
		progress := fmt.Sprintf("%.0f%%", m.execution.ProgressPercent())

		statusLine = lipgloss.NewStyle().
			Foreground(t.Subtle).
			Render(fmt.Sprintf("  %s  |  Elapsed: %s  |  Progress: %s", statusText, elapsed, progress))
	}

	return lipgloss.JoinVertical(lipgloss.Left,
		content,
		statusLine,
		controls,
	)
}

// renderStepList renders the step progress list
func (m Model) renderStepList(width, height int) string {
	t := theme.Current

	// Title
	title := lipgloss.NewStyle().
		Foreground(t.Primary).
		Bold(true).
		Render("Steps")

	var storyInfo string
	if m.execution != nil {
		storyInfo = lipgloss.NewStyle().
			Foreground(t.Info).
			Render(m.execution.Story.Key)
	}

	header := lipgloss.JoinVertical(lipgloss.Left, title, storyInfo, "")

	// Step list
	var steps []string
	if m.execution != nil {
		for i, step := range m.execution.Steps {
			steps = append(steps, m.renderStep(i, step, width-4))
		}
	} else {
		steps = append(steps, lipgloss.NewStyle().
			Foreground(t.Subtle).
			Italic(true).
			Render("No execution in progress"))
	}

	stepList := lipgloss.JoinVertical(lipgloss.Left, steps...)
	content := lipgloss.JoinVertical(lipgloss.Left, header, stepList)

	// Add border
	return lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(t.Border).
		Width(width).
		Height(height).
		Padding(1, 1).
		Render(content)
}

// renderStep renders a single step in the list
func (m Model) renderStep(index int, step *domain.StepExecution, width int) string {
	t := theme.Current

	// Status indicator
	var indicator string
	var nameStyle lipgloss.Style

	switch step.Status {
	case domain.StepPending:
		indicator = lipgloss.NewStyle().Foreground(t.Subtle).Render("  ")
		nameStyle = lipgloss.NewStyle().Foreground(t.Subtle)
	case domain.StepRunning:
		indicator = lipgloss.NewStyle().Foreground(t.Warning).Bold(true).Render("> ")
		nameStyle = lipgloss.NewStyle().Foreground(t.Warning).Bold(true)
	case domain.StepSuccess:
		indicator = lipgloss.NewStyle().Foreground(t.Success).Render("OK")
		nameStyle = lipgloss.NewStyle().Foreground(t.Success)
	case domain.StepFailed:
		indicator = lipgloss.NewStyle().Foreground(t.Error).Render("XX")
		nameStyle = lipgloss.NewStyle().Foreground(t.Error)
	case domain.StepSkipped:
		indicator = lipgloss.NewStyle().Foreground(t.Subtle).Render("--")
		nameStyle = lipgloss.NewStyle().Foreground(t.Subtle).Italic(true)
	}

	// Step name
	name := nameStyle.Render(string(step.Name))

	// Duration (if completed)
	var duration string
	if step.Duration > 0 {
		duration = lipgloss.NewStyle().
			Foreground(t.Subtle).
			Render(" " + formatDuration(step.Duration))
	} else if step.Status == domain.StepRunning && !step.StartTime.IsZero() {
		elapsed := time.Since(step.StartTime)
		duration = lipgloss.NewStyle().
			Foreground(t.Subtle).
			Render(" " + formatDuration(elapsed))
	}

	// Attempt info
	var attempt string
	if step.Attempt > 1 {
		attempt = lipgloss.NewStyle().
			Foreground(t.Warning).
			Render(fmt.Sprintf(" [%d]", step.Attempt))
	}

	// Highlight current step
	row := fmt.Sprintf("%s %s%s%s", indicator, name, attempt, duration)
	if m.execution != nil && index == m.execution.Current && step.Status == domain.StepRunning {
		row = lipgloss.NewStyle().
			Background(t.Selection).
			Width(width).
			Render(row)
	}

	return row
}

// renderOutput renders the output pane
func (m Model) renderOutput(width, height int) string {
	t := theme.Current

	// Title
	title := lipgloss.NewStyle().
		Foreground(t.Primary).
		Bold(true).
		Render("Output")

	scrollInfo := ""
	if len(m.output) > 0 {
		scrollInfo = lipgloss.NewStyle().
			Foreground(t.Subtle).
			Render(fmt.Sprintf(" (%d/%d)", m.scroll+1, len(m.output)))
	}

	header := title + scrollInfo

	// Output lines
	outputHeight := height - 4 // Account for header and padding
	var lines []string

	if len(m.output) == 0 {
		lines = append(lines, lipgloss.NewStyle().
			Foreground(t.Subtle).
			Italic(true).
			Render("Waiting for output..."))
	} else {
		// Get visible lines based on scroll
		startIdx := m.scroll
		endIdx := startIdx + outputHeight
		if endIdx > len(m.output) {
			endIdx = len(m.output)
		}

		for i := startIdx; i < endIdx; i++ {
			line := m.output[i]
			style := lipgloss.NewStyle().Foreground(t.Foreground)
			if line.isStderr {
				style = style.Foreground(t.Error)
			}

			// Truncate long lines
			text := line.text
			if len(text) > width-4 {
				text = text[:width-7] + "..."
			}

			lines = append(lines, style.Render(text))
		}
	}

	content := lipgloss.JoinVertical(lipgloss.Left,
		header,
		"",
		lipgloss.JoinVertical(lipgloss.Left, lines...),
	)

	// Add border
	return lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(t.Border).
		Width(width).
		Height(height).
		Padding(1, 1).
		Render(content)
}

// renderControls renders the control help line
func (m Model) renderControls() string {
	t := theme.Current

	var controls []string

	if m.execution != nil {
		switch m.execution.Status {
		case domain.ExecutionRunning:
			controls = append(controls,
				renderControl("p", "Pause"),
				renderControl("k", "Skip Step"),
				renderControl("c", "Cancel"),
			)
		case domain.ExecutionPaused:
			controls = append(controls,
				renderControl("r", "Resume"),
				renderControl("c", "Cancel"),
			)
		case domain.ExecutionCompleted, domain.ExecutionFailed, domain.ExecutionCancelled:
			controls = append(controls,
				renderControl("Enter", "Back to Stories"),
			)
		}
	}

	controls = append(controls,
		renderControl("Up/Down", "Scroll"),
		renderControl("Home/End", "Jump"),
	)

	return lipgloss.NewStyle().
		Foreground(t.Subtle).
		Padding(0, 2).
		Render(strings.Join(controls, "  "))
}

// renderStatusBadge renders the execution status as a badge
func (m Model) renderStatusBadge() string {
	t := theme.Current

	if m.execution == nil {
		return ""
	}

	var style lipgloss.Style
	var text string

	switch m.execution.Status {
	case domain.ExecutionPending:
		style = lipgloss.NewStyle().Foreground(t.Subtle)
		text = "PENDING"
	case domain.ExecutionRunning:
		style = lipgloss.NewStyle().Foreground(t.Warning).Bold(true)
		text = "RUNNING"
	case domain.ExecutionPaused:
		style = lipgloss.NewStyle().Foreground(t.Info).Bold(true)
		text = "PAUSED"
	case domain.ExecutionCompleted:
		style = lipgloss.NewStyle().Foreground(t.Success).Bold(true)
		text = "COMPLETED"
	case domain.ExecutionFailed:
		style = lipgloss.NewStyle().Foreground(t.Error).Bold(true)
		text = "FAILED"
	case domain.ExecutionCancelled:
		style = lipgloss.NewStyle().Foreground(t.Warning).Bold(true)
		text = "CANCELLED"
	}

	return style.Render(text)
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
	minutes := int(d.Minutes())
	seconds := int(d.Seconds()) % 60
	return fmt.Sprintf("%dm %02ds", minutes, seconds)
}
