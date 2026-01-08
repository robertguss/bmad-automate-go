package app

// handlers.go - QUAL-001: Extracted message handlers from monolithic Update()
// This file contains focused handler methods for different message categories

import (
	"context"
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/robertguss/bmad-automate-go/internal/components/commandpalette"
	"github.com/robertguss/bmad-automate-go/internal/components/confetti"
	"github.com/robertguss/bmad-automate-go/internal/domain"
	"github.com/robertguss/bmad-automate-go/internal/git"
	"github.com/robertguss/bmad-automate-go/internal/messages"
	"github.com/robertguss/bmad-automate-go/internal/preflight"
	"github.com/robertguss/bmad-automate-go/internal/theme"
	"github.com/robertguss/bmad-automate-go/internal/views/settings"
	"github.com/robertguss/bmad-automate-go/internal/watcher"
)

// handleCommandPaletteMsg handles messages when command palette is active
// Returns (model, cmd, handled) where handled=true means the message was fully processed
func (m Model) handleCommandPaletteMsg(msg tea.Msg) (Model, tea.Cmd, bool) {
	if !m.commandPalette.IsActive() {
		return m, nil, false
	}

	switch msg := msg.(type) {
	case tea.KeyMsg:
		var cmd tea.Cmd
		m.commandPalette, cmd = m.commandPalette.Update(msg)
		return m, cmd, true
	case commandpalette.SelectCommandMsg:
		if msg.Command.Action != nil {
			return m, func() tea.Msg { return msg.Command.Action() }, true
		}
		return m, nil, true
	case commandpalette.CloseMsg:
		return m, nil, true
	case commandpalette.NavigateMsg:
		m.prevView = m.activeView
		m.activeView = msg.View
		m.header.SetActiveView(m.activeView)
		return m, nil, true
	case commandpalette.ThemeChangeMsg:
		theme.SetTheme(msg.Theme)
		m.config.Theme = msg.Theme
		m.refreshAllStyles()
		m.statusbar.SetMessage("Theme changed to " + msg.Theme)
		return m, nil, true
	case commandpalette.ActionMsg:
		m, cmd := m.handlePaletteAction(msg.Action)
		return m, cmd, true
	}

	return m, nil, false
}

// handleKeyMsg handles keyboard input messages
// Returns (model, cmd, handled)
func (m Model) handleKeyMsg(msg tea.KeyMsg) (Model, tea.Cmd, bool) {
	// Command palette activation
	if msg.String() == "ctrl+p" {
		m.commandPalette.Open()
		m.commandPalette.SetSize(m.width, m.height)
		return m, nil, true
	}

	// View-specific key handling
	if handled, result := m.handleViewSpecificKeys(msg); handled {
		return result.model, result.cmd, true
	}

	// Global key handling
	return m.handleGlobalKeys(msg)
}

// keyResult holds the result of a key handler
type keyResult struct {
	model Model
	cmd   tea.Cmd
}

// handleViewSpecificKeys handles keys specific to the current view
func (m Model) handleViewSpecificKeys(msg tea.KeyMsg) (bool, keyResult) {
	switch m.activeView {
	case domain.ViewExecution:
		return m.handleExecutionViewKeys(msg)
	case domain.ViewStoryList:
		return m.handleStoryListViewKeys(msg)
	case domain.ViewQueue:
		return m.handleQueueViewKeys(msg)
	}
	return false, keyResult{}
}

// handleExecutionViewKeys handles keys when in execution view
func (m Model) handleExecutionViewKeys(msg tea.KeyMsg) (bool, keyResult) {
	switch msg.String() {
	case "p": // Pause
		if m.executor.GetExecution() != nil &&
			m.executor.GetExecution().Status == domain.ExecutionRunning {
			m.executor.Pause()
			m.statusbar.SetMessage("Execution paused")
			return true, keyResult{m, nil}
		}
	case "r": // Resume
		if m.executor.GetExecution() != nil &&
			m.executor.GetExecution().Status == domain.ExecutionPaused {
			m.executor.Resume()
			m.statusbar.SetMessage("Execution resumed")
			return true, keyResult{m, nil}
		}
	case "c": // Cancel
		exec := m.executor.GetExecution()
		if exec != nil && (exec.Status == domain.ExecutionRunning ||
			exec.Status == domain.ExecutionPaused) {
			m.executor.Cancel()
			m.statusbar.SetMessage("Execution cancelled")
			return true, keyResult{m, nil}
		}
	case "k": // Skip current step
		exec := m.executor.GetExecution()
		if exec != nil && exec.Status == domain.ExecutionRunning {
			m.executor.Skip()
			m.statusbar.SetMessage("Skipping current step...")
			return true, keyResult{m, nil}
		}
	case "enter":
		exec := m.executor.GetExecution()
		if exec != nil && (exec.Status == domain.ExecutionCompleted ||
			exec.Status == domain.ExecutionFailed ||
			exec.Status == domain.ExecutionCancelled) {
			m.prevView = m.activeView
			m.activeView = domain.ViewStoryList
			m.header.SetActiveView(m.activeView)
			return true, keyResult{m, nil}
		}
	case "esc":
		exec := m.executor.GetExecution()
		if exec == nil || exec.Status == domain.ExecutionCompleted ||
			exec.Status == domain.ExecutionFailed ||
			exec.Status == domain.ExecutionCancelled {
			m.activeView = m.prevView
			m.header.SetActiveView(m.activeView)
			return true, keyResult{m, nil}
		}
		m.statusbar.SetMessage("Cancel execution first (c) before leaving")
		return true, keyResult{m, nil}
	}
	return false, keyResult{}
}

// handleStoryListViewKeys handles keys when in story list view
func (m Model) handleStoryListViewKeys(msg tea.KeyMsg) (bool, keyResult) {
	switch msg.String() {
	case "enter":
		story := m.storylist.GetCurrent()
		if story != nil {
			return true, keyResult{m, m.startExecution(*story)}
		}
	case "q": // Add selected stories to queue
		selected := m.storylist.GetSelected()
		if len(selected) > 0 {
			m.batchExecutor.AddToQueue(selected)
			m.statusbar.SetMessage(fmt.Sprintf("Added %d stories to queue", len(selected)))
			m.statusbar.SetStoryCounts(len(m.stories), m.batchExecutor.GetQueue().TotalCount())
			m.prevView = m.activeView
			m.activeView = domain.ViewQueue
			m.header.SetActiveView(m.activeView)
			m.queue.SetQueue(m.batchExecutor.GetQueue())
			return true, keyResult{m, nil}
		}
		story := m.storylist.GetCurrent()
		if story != nil {
			m.batchExecutor.AddToQueue([]domain.Story{*story})
			m.statusbar.SetMessage(fmt.Sprintf("Added %s to queue", story.Key))
			m.statusbar.SetStoryCounts(len(m.stories), m.batchExecutor.GetQueue().TotalCount())
			m.prevView = m.activeView
			m.activeView = domain.ViewQueue
			m.header.SetActiveView(m.activeView)
			m.queue.SetQueue(m.batchExecutor.GetQueue())
			return true, keyResult{m, nil}
		}
	case "x": // Execute selected stories immediately
		selected := m.storylist.GetSelected()
		if len(selected) > 0 {
			m.batchExecutor.AddToQueue(selected)
			m.queue.SetQueue(m.batchExecutor.GetQueue())
			m.prevView = m.activeView
			m.activeView = domain.ViewExecution
			m.header.SetActiveView(m.activeView)
			return true, keyResult{m, m.batchExecutor.Start()}
		}
	}
	return false, keyResult{}
}

// handleQueueViewKeys handles keys when in queue view
func (m Model) handleQueueViewKeys(msg tea.KeyMsg) (bool, keyResult) {
	switch msg.String() {
	case "enter":
		queue := m.batchExecutor.GetQueue()
		if queue.Status == domain.QueueIdle && queue.HasPending() {
			m.prevView = m.activeView
			m.activeView = domain.ViewExecution
			m.header.SetActiveView(m.activeView)
			return true, keyResult{m, m.batchExecutor.Start()}
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
			return true, keyResult{m, nil}
		}
	}
	return false, keyResult{}
}

// handleGlobalKeys handles global keyboard shortcuts
func (m Model) handleGlobalKeys(msg tea.KeyMsg) (Model, tea.Cmd, bool) {
	switch msg.String() {
	case "ctrl+c", "ctrl+q":
		if m.executor.GetExecution() != nil {
			m.executor.Cancel()
		}
		if m.batchExecutor.IsRunning() {
			m.batchExecutor.Cancel()
		}
		return m, tea.Quit, true

	case "?":
		return m, nil, true

	case "d":
		if m.canNavigate() {
			m.prevView = m.activeView
			m.activeView = domain.ViewDashboard
			m.header.SetActiveView(m.activeView)
		}
		return m, nil, true

	case "s":
		if m.canNavigate() {
			m.prevView = m.activeView
			m.activeView = domain.ViewStoryList
			m.header.SetActiveView(m.activeView)
		}
		return m, nil, true

	case "q":
		if m.canNavigate() {
			m.prevView = m.activeView
			m.activeView = domain.ViewQueue
			m.header.SetActiveView(m.activeView)
		}
		return m, nil, true

	case "h":
		if m.canNavigate() {
			m.prevView = m.activeView
			m.activeView = domain.ViewHistory
			m.header.SetActiveView(m.activeView)
			m.history.SetLoading(true)
			return m, m.loadHistory(), true
		}
		return m, nil, true

	case "a":
		if m.activeView != domain.ViewStoryList && m.canNavigate() {
			m.prevView = m.activeView
			m.activeView = domain.ViewStats
			m.header.SetActiveView(m.activeView)
			m.stats.SetLoading(true)
			return m, m.loadStats(), true
		}
		return m, nil, false // Don't mark as handled to allow storylist to handle 'a'

	case "o":
		if m.canNavigate() {
			m.prevView = m.activeView
			m.activeView = domain.ViewSettings
			m.header.SetActiveView(m.activeView)
		}
		return m, nil, true

	case "esc":
		if m.activeView != domain.ViewDashboard && m.activeView != domain.ViewExecution {
			if m.prevView == m.activeView {
				m.activeView = domain.ViewDashboard
			} else {
				m.activeView = m.prevView
			}
			m.header.SetActiveView(m.activeView)
		}
		return m, nil, true
	}

	return m, nil, false
}

// handleWindowSizeMsg handles window resize messages
func (m Model) handleWindowSizeMsg(msg tea.WindowSizeMsg) Model {
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
	m.history.SetSize(msg.Width, contentHeight)
	m.stats.SetSize(msg.Width, contentHeight)
	m.diff.SetSize(msg.Width, contentHeight)

	// Propagate to views
	sizeMsg := messages.WindowSizeMsg{Width: msg.Width, Height: contentHeight}
	m.dashboard, _ = m.dashboard.Update(sizeMsg)
	m.storylist, _ = m.storylist.Update(sizeMsg)
	m.execution, _ = m.execution.Update(sizeMsg)
	m.queue, _ = m.queue.Update(sizeMsg)
	m.timeline, _ = m.timeline.Update(sizeMsg)
	m.history, _ = m.history.Update(sizeMsg)
	m.stats, _ = m.stats.Update(sizeMsg)
	m.diff, _ = m.diff.Update(sizeMsg)

	return m
}

// handleStoriesMsg handles stories-related messages
func (m Model) handleStoriesMsg(msg messages.StoriesLoadedMsg) Model {
	if msg.Error != nil {
		m.err = msg.Error
		m.statusbar.SetMessage(fmt.Sprintf("Error: %v", msg.Error))
	} else {
		m.stories = msg.Stories
		m.statusbar.SetStoryCounts(len(m.stories), 0)

		branch := preflight.GetGitBranch(m.config.WorkingDir)
		clean := preflight.IsGitClean(m.config.WorkingDir)
		m.statusbar.SetGitInfo(branch, clean)

		m.dashboard.SetStories(m.stories)
		m.storylist.SetStories(m.stories)
	}
	return m
}

// handleExecutionMsgs handles execution-related messages
// Returns (model, cmds) where cmds are any additional commands to run
func (m Model) handleExecutionMsgs(msg tea.Msg) (Model, []tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case messages.ExecutionStartMsg:
		cmds = append(cmds, m.startExecution(msg.Story))

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
	}

	return m, cmds
}

// handleQueueMsgs handles queue-related messages
func (m Model) handleQueueMsgs(msg tea.Msg) (Model, []tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
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
		if msg.Execution != nil {
			m.timeline.AddExecution(msg.Execution)
		}
		if msg.Status == domain.ExecutionCompleted {
			m.statusbar.SetMessage(fmt.Sprintf("Completed: %s", msg.Story.Key))
		} else if msg.Status == domain.ExecutionFailed {
			m.statusbar.SetMessage(fmt.Sprintf("Failed: %s - %s", msg.Story.Key, msg.Error))
		}

	case messages.QueueCompletedMsg:
		m.queue, _ = m.queue.Update(messages.QueueUpdatedMsg{Queue: m.batchExecutor.GetQueue()})
		m.statusbar.SetMessage(fmt.Sprintf("Queue completed: %d/%d succeeded in %s",
			msg.SuccessCount, msg.TotalItems, formatDuration(msg.TotalDuration)))

		// Save executions to storage
		if m.storage != nil {
			queue := m.batchExecutor.GetQueue()
			for _, item := range queue.Items {
				if item.Execution != nil {
					_ = m.storage.SaveExecution(context.Background(), item.Execution)
				}
			}
			_ = m.storage.UpdateStepAverages(context.Background())
		}

		// Notifications and feedback
		failedCount := msg.TotalItems - msg.SuccessCount
		_ = m.notifier.NotifyQueueComplete(msg.TotalItems, msg.SuccessCount, failedCount)

		if failedCount == 0 {
			_ = m.soundPlayer.PlayComplete()
			cmds = append(cmds, m.confetti.Start(m.width, m.height))
		} else {
			_ = m.soundPlayer.PlayWarning()
		}
	}

	return m, cmds
}

// handleSettingsMsgs handles settings and git status messages
func (m Model) handleSettingsMsgs(msg tea.Msg) Model {
	switch msg := msg.(type) {
	case git.StatusMsg:
		m.gitStatus = msg.Status
		m.statusbar.SetGitInfo(m.gitStatus.Branch, m.gitStatus.IsClean)

	case settings.ThemeChangedMsg:
		m.refreshAllStyles()
		m.statusbar.SetMessage("Theme changed to " + msg.Theme)

	case settings.SettingChangedMsg:
		switch msg.Name {
		case "Notifications":
			m.notifier.SetEnabled(msg.Value.(bool))
		case "Sound":
			m.soundPlayer.SetEnabled(msg.Value.(bool))
		}

	case confetti.TickMsg:
		m.confetti, _ = m.confetti.Update(msg)
	}

	return m
}

// handleHistoryStatsMsgs handles history, stats, and diff messages
func (m Model) handleHistoryStatsMsgs(msg tea.Msg) (Model, []tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case messages.HistoryRefreshMsg:
		cmds = append(cmds, m.loadHistory())

	case messages.HistoryFilterMsg:
		cmds = append(cmds, m.loadHistoryFiltered(msg.Query, msg.Epic, msg.Status))

	case messages.HistoryLoadedMsg:
		m.history.SetExecutions(msg.Executions, msg.TotalCount)

	case messages.HistoryDetailMsg:
		if m.storage != nil {
			cmds = append(cmds, m.loadExecutionDetail(msg.ID))
		}

	case messages.StatsRefreshMsg:
		cmds = append(cmds, m.loadStats())

	case messages.StatsLoadedMsg:
		m.stats.SetStats(msg.Stats)

	case messages.DiffRequestMsg:
		cmds = append(cmds, m.loadDiff(msg.StoryKey))

	case messages.DiffLoadedMsg:
		m.diff.SetDiff(msg.StoryKey, msg.Content)
	}

	return m, cmds
}

// handlePhase6Msgs handles Phase 6 messages (profiles, workflows, watch, parallel, API)
func (m Model) handlePhase6Msgs(msg tea.Msg) (Model, []tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case messages.ProfileSwitchMsg:
		m.statusbar.SetMessage(fmt.Sprintf("Switched to profile: %s", msg.ProfileName))
		cmds = append(cmds, m.loadStories)

	case messages.ProfileLoadedMsg:
		if msg.Error != nil {
			m.statusbar.SetMessage(fmt.Sprintf("Profile error: %v", msg.Error))
		}

	case messages.WorkflowSwitchMsg:
		m.statusbar.SetMessage(fmt.Sprintf("Switched to workflow: %s", msg.WorkflowName))

	case messages.WorkflowLoadedMsg:
		if msg.Error != nil {
			m.statusbar.SetMessage(fmt.Sprintf("Workflow error: %v", msg.Error))
		}

	case watcher.RefreshMsg:
		m.statusbar.SetMessage("Files changed, refreshing stories...")
		cmds = append(cmds, m.loadStories)

	case messages.WatchStatusMsg:
		if msg.Running {
			m.statusbar.SetMessage("Watch mode enabled")
		} else {
			m.statusbar.SetMessage("Watch mode disabled")
		}

	case messages.ParallelProgressMsg:
		m.statusbar.SetMessage(fmt.Sprintf("Parallel: %d/%d completed, %d active",
			msg.Completed, msg.Total, msg.Active))

	case messages.APIServerStatusMsg:
		if msg.Running {
			m.statusbar.SetMessage(fmt.Sprintf("API server running at %s", msg.URL))
		} else {
			m.statusbar.SetMessage("API server stopped")
		}

	case messages.StoriesRefreshMsg:
		cmds = append(cmds, m.loadStories)
	}

	return m, cmds
}

// routeToActiveView routes messages to the currently active view
func (m Model) routeToActiveView(msg tea.Msg) (Model, tea.Cmd) {
	var cmd tea.Cmd

	switch m.activeView {
	case domain.ViewDashboard:
		m.dashboard, cmd = m.dashboard.Update(msg)
	case domain.ViewStoryList:
		m.storylist, cmd = m.storylist.Update(msg)
	case domain.ViewExecution:
		m.execution, cmd = m.execution.Update(msg)
	case domain.ViewQueue:
		m.queue, cmd = m.queue.Update(msg)
	case domain.ViewTimeline:
		m.timeline, cmd = m.timeline.Update(msg)
	case domain.ViewHistory:
		m.history, cmd = m.history.Update(msg)
	case domain.ViewStats:
		m.stats, cmd = m.stats.Update(msg)
	case domain.ViewDiff:
		m.diff, cmd = m.diff.Update(msg)
	case domain.ViewSettings:
		m.settings, cmd = m.settings.Update(msg)
	}

	return m, cmd
}
