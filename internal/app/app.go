package app

import (
	"fmt"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/robertguss/bmad-automate-go/internal/components/header"
	"github.com/robertguss/bmad-automate-go/internal/components/statusbar"
	"github.com/robertguss/bmad-automate-go/internal/config"
	"github.com/robertguss/bmad-automate-go/internal/domain"
	"github.com/robertguss/bmad-automate-go/internal/executor"
	"github.com/robertguss/bmad-automate-go/internal/messages"
	"github.com/robertguss/bmad-automate-go/internal/parser"
	"github.com/robertguss/bmad-automate-go/internal/preflight"
	"github.com/robertguss/bmad-automate-go/internal/theme"
	"github.com/robertguss/bmad-automate-go/internal/views/dashboard"
	"github.com/robertguss/bmad-automate-go/internal/views/execution"
	queueview "github.com/robertguss/bmad-automate-go/internal/views/queue"
	"github.com/robertguss/bmad-automate-go/internal/views/storylist"
	"github.com/robertguss/bmad-automate-go/internal/views/timeline"
)

// Model is the main application model
type Model struct {
	// Dimensions
	width  int
	height int
	ready  bool

	// Navigation
	activeView domain.View
	prevView   domain.View

	// Configuration
	config *config.Config

	// Data
	stories []domain.Story
	err     error

	// Executors
	executor      *executor.Executor
	batchExecutor *executor.BatchExecutor

	// Components
	header    header.Model
	statusbar statusbar.Model

	// Views
	dashboard dashboard.Model
	storylist storylist.Model
	execution execution.Model
	queue     queueview.Model
	timeline  timeline.Model

	// Styles
	styles theme.Styles

	// Pre-flight check results
	preflightResults *preflight.Results
}

// New creates a new application model
func New(cfg *config.Config) Model {
	exec := executor.New(cfg)
	batchExec := executor.NewBatchExecutor(cfg)

	return Model{
		activeView:       domain.ViewDashboard,
		config:           cfg,
		executor:         exec,
		batchExecutor:    batchExec,
		header:           header.New(),
		statusbar:        statusbar.New(),
		dashboard:        dashboard.New(),
		storylist:        storylist.New(),
		execution:        execution.New(),
		queue:            queueview.New(),
		timeline:         timeline.New(),
		styles:           theme.NewStyles(),
		preflightResults: nil,
	}
}

// SetProgram sets the tea.Program on the executor for async messages
func (m *Model) SetProgram(p *tea.Program) {
	m.executor.SetProgram(p)
	m.batchExecutor.SetProgram(p)
}

// Init initializes the application
func (m Model) Init() tea.Cmd {
	return tea.Batch(
		m.loadStories,
		m.runPreflightChecks,
	)
}

// loadStories loads stories from sprint-status.yaml
func (m Model) loadStories() tea.Msg {
	stories, err := parser.ParseSprintStatus(m.config)
	return messages.StoriesLoadedMsg{Stories: stories, Error: err}
}

// runPreflightChecks runs pre-flight checks
func (m Model) runPreflightChecks() tea.Msg {
	results := preflight.RunAll(m.config)
	return preflightResultsMsg{Results: results}
}

// preflightResultsMsg carries pre-flight check results
type preflightResultsMsg struct {
	Results *preflight.Results
}

// Update handles all messages
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		// Handle execution-specific keys when in execution view
		if m.activeView == domain.ViewExecution {
			switch msg.String() {
			case "p": // Pause
				if m.executor.GetExecution() != nil &&
					m.executor.GetExecution().Status == domain.ExecutionRunning {
					m.executor.Pause()
					m.statusbar.SetMessage("Execution paused")
					return m, nil
				}
			case "r": // Resume
				if m.executor.GetExecution() != nil &&
					m.executor.GetExecution().Status == domain.ExecutionPaused {
					m.executor.Resume()
					m.statusbar.SetMessage("Execution resumed")
					return m, nil
				}
			case "c": // Cancel
				exec := m.executor.GetExecution()
				if exec != nil && (exec.Status == domain.ExecutionRunning ||
					exec.Status == domain.ExecutionPaused) {
					m.executor.Cancel()
					m.statusbar.SetMessage("Execution cancelled")
					return m, nil
				}
			case "k": // Skip current step
				exec := m.executor.GetExecution()
				if exec != nil && exec.Status == domain.ExecutionRunning {
					m.executor.Skip()
					m.statusbar.SetMessage("Skipping current step...")
					return m, nil
				}
			case "enter":
				// Return to story list if execution is complete
				exec := m.executor.GetExecution()
				if exec != nil && (exec.Status == domain.ExecutionCompleted ||
					exec.Status == domain.ExecutionFailed ||
					exec.Status == domain.ExecutionCancelled) {
					m.prevView = m.activeView
					m.activeView = domain.ViewStoryList
					m.header.SetActiveView(m.activeView)
					return m, nil
				}
			case "esc":
				// Only allow escape if not running
				exec := m.executor.GetExecution()
				if exec == nil || exec.Status == domain.ExecutionCompleted ||
					exec.Status == domain.ExecutionFailed ||
					exec.Status == domain.ExecutionCancelled {
					m.activeView = m.prevView
					m.header.SetActiveView(m.activeView)
					return m, nil
				}
				m.statusbar.SetMessage("Cancel execution first (c) before leaving")
				return m, nil
			}
		}

		// Handle story list specific keys
		if m.activeView == domain.ViewStoryList {
			switch msg.String() {
			case "enter":
				// Execute the currently selected story
				story := m.storylist.GetCurrent()
				if story != nil {
					return m, m.startExecution(*story)
				}
			case "Q": // Add selected stories to queue (Shift+Q)
				selected := m.storylist.GetSelected()
				if len(selected) > 0 {
					m.batchExecutor.AddToQueue(selected)
					m.statusbar.SetMessage(fmt.Sprintf("Added %d stories to queue", len(selected)))
					m.statusbar.SetStoryCounts(len(m.stories), m.batchExecutor.GetQueue().TotalCount())
					// Navigate to queue view
					m.prevView = m.activeView
					m.activeView = domain.ViewQueue
					m.header.SetActiveView(m.activeView)
					m.queue.SetQueue(m.batchExecutor.GetQueue())
					return m, nil
				} else {
					// Add current story if none selected
					story := m.storylist.GetCurrent()
					if story != nil {
						m.batchExecutor.AddToQueue([]domain.Story{*story})
						m.statusbar.SetMessage(fmt.Sprintf("Added %s to queue", story.Key))
						m.statusbar.SetStoryCounts(len(m.stories), m.batchExecutor.GetQueue().TotalCount())
					}
				}
			case "x": // Execute selected stories immediately
				selected := m.storylist.GetSelected()
				if len(selected) > 0 {
					m.batchExecutor.AddToQueue(selected)
					m.queue.SetQueue(m.batchExecutor.GetQueue())
					m.prevView = m.activeView
					m.activeView = domain.ViewExecution
					m.header.SetActiveView(m.activeView)
					return m, m.batchExecutor.Start()
				}
			}
		}

		// Handle queue view specific keys
		if m.activeView == domain.ViewQueue {
			switch msg.String() {
			case "enter":
				// Start queue execution if idle and has pending items
				queue := m.batchExecutor.GetQueue()
				if queue.Status == domain.QueueIdle && queue.HasPending() {
					m.prevView = m.activeView
					m.activeView = domain.ViewExecution
					m.header.SetActiveView(m.activeView)
					return m, m.batchExecutor.Start()
				}
			case "p": // Pause queue
				if m.batchExecutor.IsRunning() && !m.batchExecutor.IsPaused() {
					m.batchExecutor.Pause()
					m.statusbar.SetMessage("Queue paused")
				}
			case "r": // Resume queue
				if m.batchExecutor.IsPaused() {
					m.batchExecutor.Resume()
					m.statusbar.SetMessage("Queue resumed")
				}
			case "c": // Cancel queue
				if m.batchExecutor.IsRunning() {
					m.batchExecutor.Cancel()
					m.statusbar.SetMessage("Queue cancelled")
				}
			case "t": // Navigate to timeline
				if m.canNavigate() {
					m.prevView = m.activeView
					m.activeView = domain.ViewTimeline
					m.header.SetActiveView(m.activeView)
					return m, nil
				}
			}
		}

		// Global key handling
		switch msg.String() {
		case "ctrl+c", "ctrl+q":
			// Cancel any running execution before quitting
			if m.executor.GetExecution() != nil {
				m.executor.Cancel()
			}
			if m.batchExecutor.IsRunning() {
				m.batchExecutor.Cancel()
			}
			return m, tea.Quit

		case "?":
			// TODO: Show help modal
			return m, nil

		// View navigation (disabled during execution)
		case "d":
			if m.canNavigate() {
				m.prevView = m.activeView
				m.activeView = domain.ViewDashboard
				m.header.SetActiveView(m.activeView)
			}
			return m, nil

		case "s":
			if m.canNavigate() {
				m.prevView = m.activeView
				m.activeView = domain.ViewStoryList
				m.header.SetActiveView(m.activeView)
			}
			return m, nil

		case "q":
			if m.canNavigate() {
				m.prevView = m.activeView
				m.activeView = domain.ViewQueue
				m.header.SetActiveView(m.activeView)
			}
			return m, nil

		case "h":
			if m.canNavigate() {
				m.prevView = m.activeView
				m.activeView = domain.ViewHistory
				m.header.SetActiveView(m.activeView)
			}
			return m, nil

		case "a":
			// Only navigate to stats if not in storylist (where 'a' means select all)
			if m.activeView != domain.ViewStoryList && m.canNavigate() {
				m.prevView = m.activeView
				m.activeView = domain.ViewStats
				m.header.SetActiveView(m.activeView)
				return m, nil
			}

		case "o":
			if m.canNavigate() {
				m.prevView = m.activeView
				m.activeView = domain.ViewSettings
				m.header.SetActiveView(m.activeView)
			}
			return m, nil

		case "esc":
			// Go back to previous view or dashboard (if not in execution)
			if m.activeView != domain.ViewDashboard && m.activeView != domain.ViewExecution {
				m.activeView = m.prevView
				if m.activeView == m.activeView { // same view
					m.activeView = domain.ViewDashboard
				}
				m.header.SetActiveView(m.activeView)
			}
			return m, nil
		}

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.ready = true

		// Update component sizes
		m.header.SetWidth(msg.Width)
		m.statusbar.SetWidth(msg.Width)

		// Calculate content height (total - header - statusbar)
		contentHeight := msg.Height - 4 // header(2) + statusbar(2)

		m.dashboard.SetSize(msg.Width, contentHeight)
		m.storylist.SetSize(msg.Width, contentHeight)
		m.execution.SetSize(msg.Width, contentHeight)
		m.queue.SetSize(msg.Width, contentHeight)
		m.timeline.SetSize(msg.Width, contentHeight)

		// Propagate to views
		sizeMsg := messages.WindowSizeMsg{Width: msg.Width, Height: contentHeight}
		m.dashboard, _ = m.dashboard.Update(sizeMsg)
		m.storylist, _ = m.storylist.Update(sizeMsg)
		m.execution, _ = m.execution.Update(sizeMsg)
		m.queue, _ = m.queue.Update(sizeMsg)
		m.timeline, _ = m.timeline.Update(sizeMsg)

	case messages.StoriesLoadedMsg:
		if msg.Error != nil {
			m.err = msg.Error
			m.statusbar.SetMessage(fmt.Sprintf("Error: %v", msg.Error))
		} else {
			m.stories = msg.Stories
			m.statusbar.SetStoryCounts(len(m.stories), 0)

			// Update git info in status bar
			branch := preflight.GetGitBranch(m.config.WorkingDir)
			clean := preflight.IsGitClean(m.config.WorkingDir)
			m.statusbar.SetGitInfo(branch, clean)

			// Update views with stories
			m.dashboard.SetStories(m.stories)
			m.storylist.SetStories(m.stories)
		}

	case preflightResultsMsg:
		m.preflightResults = msg.Results
		if !msg.Results.AllPass {
			failed := msg.Results.FailedChecks()
			if len(failed) > 0 {
				m.statusbar.SetMessage(fmt.Sprintf("Pre-flight warning: %s", failed[0].Error))
			}
		}

	// Execution messages
	case messages.ExecutionStartMsg:
		return m, m.startExecution(msg.Story)

	case messages.ExecutionStartedMsg:
		m.execution.SetExecution(msg.Execution)
		m.prevView = m.activeView
		m.activeView = domain.ViewExecution
		m.header.SetActiveView(m.activeView)
		m.statusbar.SetMessage(fmt.Sprintf("Executing: %s", msg.Execution.Story.Key))

	case messages.StepStartedMsg:
		m.execution, _ = m.execution.Update(msg)

	case messages.StepOutputMsg:
		m.execution, _ = m.execution.Update(msg)

	case messages.StepCompletedMsg:
		m.execution, _ = m.execution.Update(msg)
		if msg.Status == domain.StepSuccess {
			m.statusbar.SetMessage(fmt.Sprintf("Step completed: %d/%d", msg.StepIndex+1, 4))
		} else if msg.Status == domain.StepFailed {
			m.statusbar.SetMessage(fmt.Sprintf("Step failed: %s", msg.Error))
		}

	case messages.ExecutionCompletedMsg:
		m.execution, _ = m.execution.Update(msg)
		switch msg.Status {
		case domain.ExecutionCompleted:
			m.statusbar.SetMessage(fmt.Sprintf("Execution completed in %s", formatDuration(msg.Duration)))
		case domain.ExecutionFailed:
			m.statusbar.SetMessage(fmt.Sprintf("Execution failed: %s", msg.Error))
		case domain.ExecutionCancelled:
			m.statusbar.SetMessage("Execution cancelled")
		}

	case messages.ExecutionTickMsg:
		m.execution, _ = m.execution.Update(msg)

	// Queue messages
	case messages.QueueUpdatedMsg:
		m.queue.SetQueue(msg.Queue)
		m.statusbar.SetStoryCounts(len(m.stories), msg.Queue.TotalCount())

	case messages.QueueItemStartedMsg:
		m.queue, _ = m.queue.Update(msg)
		m.execution.SetExecution(msg.Execution)
		m.statusbar.SetMessage(fmt.Sprintf("Executing: %s (%d/%d)",
			msg.Story.Key, msg.Index+1, m.batchExecutor.GetQueue().TotalCount()))

	case messages.QueueItemCompletedMsg:
		m.queue, _ = m.queue.Update(msg)
		m.timeline.AddExecution(m.batchExecutor.GetQueue().GetItem(msg.Index).Execution)
		if msg.Status == domain.ExecutionCompleted {
			m.statusbar.SetMessage(fmt.Sprintf("Completed: %s", msg.Story.Key))
		} else if msg.Status == domain.ExecutionFailed {
			m.statusbar.SetMessage(fmt.Sprintf("Failed: %s - %s", msg.Story.Key, msg.Error))
		}

	case messages.QueueCompletedMsg:
		m.queue, _ = m.queue.Update(messages.QueueUpdatedMsg{Queue: m.batchExecutor.GetQueue()})
		m.statusbar.SetMessage(fmt.Sprintf("Queue completed: %d/%d succeeded in %s",
			msg.SuccessCount, msg.TotalItems, formatDuration(msg.TotalDuration)))
	}

	// Route to active view
	switch m.activeView {
	case domain.ViewDashboard:
		var cmd tea.Cmd
		m.dashboard, cmd = m.dashboard.Update(msg)
		cmds = append(cmds, cmd)

	case domain.ViewStoryList:
		var cmd tea.Cmd
		m.storylist, cmd = m.storylist.Update(msg)
		cmds = append(cmds, cmd)

	case domain.ViewExecution:
		var cmd tea.Cmd
		m.execution, cmd = m.execution.Update(msg)
		cmds = append(cmds, cmd)

	case domain.ViewQueue:
		var cmd tea.Cmd
		m.queue, cmd = m.queue.Update(msg)
		cmds = append(cmds, cmd)

	case domain.ViewTimeline:
		var cmd tea.Cmd
		m.timeline, cmd = m.timeline.Update(msg)
		cmds = append(cmds, cmd)
	}

	return m, tea.Batch(cmds...)
}

// startExecution begins execution of a story
func (m *Model) startExecution(story domain.Story) tea.Cmd {
	// Check pre-flight first
	if m.preflightResults != nil && !m.preflightResults.AllPass {
		// Find first blocking failure (not Git Clean which is just a warning)
		for _, check := range m.preflightResults.FailedChecks() {
			if check.Name != "Git Clean" {
				m.statusbar.SetMessage(fmt.Sprintf("Cannot execute: %s - %s", check.Name, check.Error))
				return nil
			}
		}
	}

	return m.executor.Execute(story)
}

// canNavigate returns true if view navigation is allowed
func (m Model) canNavigate() bool {
	// Check single story executor
	exec := m.executor.GetExecution()
	if exec != nil && (exec.Status == domain.ExecutionRunning || exec.Status == domain.ExecutionPaused) {
		return false
	}
	// Check batch executor
	if m.batchExecutor.IsRunning() {
		return false
	}
	return true
}

// View renders the application
func (m Model) View() string {
	if !m.ready {
		return "\n  Initializing BMAD Automate..."
	}

	// Header
	m.header.SetActiveView(m.activeView)
	headerView := m.header.View()

	// Content based on active view
	var content string
	switch m.activeView {
	case domain.ViewDashboard:
		content = m.dashboard.View()
	case domain.ViewStoryList:
		content = m.storylist.View()
	case domain.ViewExecution:
		content = m.execution.View()
	case domain.ViewQueue:
		content = m.queue.View()
	case domain.ViewTimeline:
		content = m.timeline.View()
	case domain.ViewDiff:
		content = m.renderPlaceholder("Diff Preview", "Coming in Phase 4")
	case domain.ViewHistory:
		content = m.renderPlaceholder("History", "Coming in Phase 4")
	case domain.ViewStats:
		content = m.renderPlaceholder("Statistics", "Coming in Phase 4")
	case domain.ViewSettings:
		content = m.renderPlaceholder("Settings", "Coming in Phase 5")
	default:
		content = m.renderPlaceholder("Unknown View", "")
	}

	// Status bar
	statusView := m.statusbar.View()

	// Combine all sections
	return lipgloss.JoinVertical(lipgloss.Left,
		headerView,
		content,
		statusView,
	)
}

func (m Model) renderPlaceholder(title, subtitle string) string {
	t := theme.Current

	titleStyle := lipgloss.NewStyle().
		Foreground(t.Primary).
		Bold(true).
		MarginBottom(1)

	subtitleStyle := lipgloss.NewStyle().
		Foreground(t.Subtle).
		Italic(true)

	content := titleStyle.Render(title)
	if subtitle != "" {
		content = lipgloss.JoinVertical(lipgloss.Left,
			content,
			subtitleStyle.Render(subtitle),
		)
	}

	box := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(t.Border).
		Padding(2, 4).
		Render(content)

	// Center in available space
	return lipgloss.NewStyle().
		Width(m.width).
		Height(m.height-4).
		Align(lipgloss.Center, lipgloss.Center).
		Render(box)
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
