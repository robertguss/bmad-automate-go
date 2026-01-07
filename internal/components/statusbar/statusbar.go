package statusbar

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/robertguss/bmad-automate-go/internal/theme"
)

// Model represents the status bar component
type Model struct {
	width      int
	gitBranch  string
	gitClean   bool
	storyCount int
	queueCount int
	message    string
	styles     theme.Styles
}

// New creates a new status bar model
func New() Model {
	return Model{
		gitBranch: "main",
		gitClean:  true,
		styles:    theme.NewStyles(),
	}
}

// SetWidth sets the status bar width
func (m *Model) SetWidth(width int) {
	m.width = width
}

// SetGitInfo sets the git branch and clean status
func (m *Model) SetGitInfo(branch string, clean bool) {
	m.gitBranch = branch
	m.gitClean = clean
}

// SetStoryCounts sets the story and queue counts
func (m *Model) SetStoryCounts(stories, queue int) {
	m.storyCount = stories
	m.queueCount = queue
}

// SetMessage sets a temporary status message
func (m *Model) SetMessage(msg string) {
	m.message = msg
}

// ClearMessage clears the status message
func (m *Model) ClearMessage() {
	m.message = ""
}

// View renders the status bar
func (m Model) View() string {
	t := theme.Current

	// Top border
	border := lipgloss.NewStyle().
		Foreground(t.Border).
		Width(m.width).
		Render(strings.Repeat("─", m.width))

	// Git info
	gitStatus := "Clean"
	gitColor := t.Success
	if !m.gitClean {
		gitStatus = "Modified"
		gitColor = t.Warning
	}

	gitInfo := fmt.Sprintf("Git: %s | %s",
		lipgloss.NewStyle().Foreground(t.Info).Render(m.gitBranch),
		lipgloss.NewStyle().Foreground(gitColor).Render(gitStatus),
	)

	// Counts
	counts := fmt.Sprintf("Stories: %s | Queue: %s",
		lipgloss.NewStyle().Foreground(t.Foreground).Bold(true).Render(fmt.Sprintf("%d", m.storyCount)),
		lipgloss.NewStyle().Foreground(t.Foreground).Bold(true).Render(fmt.Sprintf("%d", m.queueCount)),
	)

	// Message or help
	var rightContent string
	if m.message != "" {
		rightContent = lipgloss.NewStyle().Foreground(t.Warning).Render(m.message)
	} else {
		rightContent = lipgloss.NewStyle().Foreground(t.Subtle).Render("Press ? for help | Ctrl+C to quit")
	}

	// Calculate spacing
	leftWidth := lipgloss.Width(gitInfo)
	centerWidth := lipgloss.Width(counts)
	rightWidth := lipgloss.Width(rightContent)
	totalContent := leftWidth + centerWidth + rightWidth

	var content string
	if m.width > totalContent+4 {
		gap := (m.width - totalContent - 4) / 2
		content = gitInfo + strings.Repeat(" ", gap) + counts + strings.Repeat(" ", gap) + rightContent
	} else {
		content = gitInfo + "  " + counts
	}

	// Status bar container
	bar := lipgloss.NewStyle().
		Background(t.StatusBar).
		Foreground(t.Subtle).
		Width(m.width).
		Padding(0, 2).
		Render(content)

	return lipgloss.JoinVertical(lipgloss.Left, border, bar)
}
