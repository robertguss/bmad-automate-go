package app

import (
	"context"
	"fmt"
	"os/exec"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/robertguss/bmad-automate-go/internal/api"
	"github.com/robertguss/bmad-automate-go/internal/components/commandpalette"
	"github.com/robertguss/bmad-automate-go/internal/components/confetti"
	"github.com/robertguss/bmad-automate-go/internal/components/header"
	"github.com/robertguss/bmad-automate-go/internal/components/statusbar"
	"github.com/robertguss/bmad-automate-go/internal/config"
	"github.com/robertguss/bmad-automate-go/internal/domain"
	"github.com/robertguss/bmad-automate-go/internal/executor"
	"github.com/robertguss/bmad-automate-go/internal/git"
	"github.com/robertguss/bmad-automate-go/internal/messages"
	"github.com/robertguss/bmad-automate-go/internal/notify"
	"github.com/robertguss/bmad-automate-go/internal/parser"
	"github.com/robertguss/bmad-automate-go/internal/preflight"
	"github.com/robertguss/bmad-automate-go/internal/profile"
	"github.com/robertguss/bmad-automate-go/internal/sound"
	"github.com/robertguss/bmad-automate-go/internal/storage"
	"github.com/robertguss/bmad-automate-go/internal/theme"
	"github.com/robertguss/bmad-automate-go/internal/views/dashboard"
	"github.com/robertguss/bmad-automate-go/internal/views/diff"
	"github.com/robertguss/bmad-automate-go/internal/views/execution"
	"github.com/robertguss/bmad-automate-go/internal/views/history"
	queueview "github.com/robertguss/bmad-automate-go/internal/views/queue"
	"github.com/robertguss/bmad-automate-go/internal/views/settings"
	"github.com/robertguss/bmad-automate-go/internal/views/stats"
	"github.com/robertguss/bmad-automate-go/internal/views/storylist"
	"github.com/robertguss/bmad-automate-go/internal/views/timeline"
	"github.com/robertguss/bmad-automate-go/internal/watcher"
	"github.com/robertguss/bmad-automate-go/internal/workflow"
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

	// Storage
	storage storage.Storage

	// Executors
	executor         *executor.Executor
	batchExecutor    *executor.BatchExecutor
	parallelExecutor *executor.ParallelExecutor

	// Components
	header    header.Model
	statusbar statusbar.Model

	// Phase 5: New components
	commandPalette commandpalette.Model
	confetti       confetti.Model

	// Phase 5: Services
	notifier    *notify.Notifier
	soundPlayer *sound.Player
	gitStatus   git.Status

	// Phase 6: Profile and Workflow
	profileStore  *profile.ProfileStore
	workflowStore *workflow.WorkflowStore

	// Phase 6: Watcher
	watcher *watcher.Watcher

	// Phase 6: API Server
	apiServer *api.Server

	// Views
	dashboard dashboard.Model
	storylist storylist.Model
	execution execution.Model
	queue     queueview.Model
	timeline  timeline.Model
	history   history.Model
	stats     stats.Model
	diff      diff.Model
	settings  settings.Model

	// Styles
	styles theme.Styles

	// Pre-flight check results
	preflightResults *preflight.Results
}

// New creates a new application model
func New(cfg *config.Config) Model {
	exec := executor.New(cfg)
	batchExec := executor.NewBatchExecutor(cfg)
	parallelExec := executor.NewParallelExecutor(cfg, cfg.MaxWorkers)

	// Initialize storage
	var store storage.Storage
	if err := cfg.EnsureDataDir(); err == nil {
		store, _ = storage.NewSQLiteStorage(cfg.DatabasePath)
	}

	// Apply theme from config
	theme.SetTheme(cfg.Theme)

	// Initialize Phase 6: Profile store
	profileStore := profile.NewProfileStore(cfg.DataDir)
	profileStore.Load()

	// Initialize Phase 6: Workflow store
	workflowStore := workflow.NewWorkflowStore(cfg.DataDir)
	workflowStore.Load()

	// Initialize Phase 6: File watcher
	fileWatcher := watcher.New(time.Duration(cfg.WatchDebounce) * time.Millisecond)
	fileWatcher.AddPath(cfg.SprintStatusPath)

	// Initialize Phase 6: API server
	apiServer := api.NewServer(cfg, store, exec, batchExec)

	return Model{
		activeView:       domain.ViewDashboard,
		config:           cfg,
		storage:          store,
		executor:         exec,
		batchExecutor:    batchExec,
		parallelExecutor: parallelExec,
		header:           header.New(),
		statusbar:        statusbar.New(),
		commandPalette:   commandpalette.New(),
		confetti:         confetti.New(),
		notifier:         notify.New(cfg.NotificationsEnabled),
		soundPlayer:      sound.New(cfg.SoundEnabled),
		profileStore:     profileStore,
		workflowStore:    workflowStore,
		watcher:          fileWatcher,
		apiServer:        apiServer,
		dashboard:        dashboard.New(),
		storylist:        storylist.New(),
		execution:        execution.New(),
		queue:            queueview.New(),
		timeline:         timeline.New(),
		history:          history.New(),
		stats:            stats.New(),
		diff:             diff.New(),
		settings:         settings.New(cfg),
		styles:           theme.NewStyles(),
		preflightResults: nil,
	}
}

// SetProgram sets the tea.Program on the executor for async messages
func (m *Model) SetProgram(p *tea.Program) {
	m.executor.SetProgram(p)
	m.batchExecutor.SetProgram(p)
	m.parallelExecutor.SetProgram(p)
	m.watcher.SetProgram(p)
}

// Init initializes the application
func (m Model) Init() tea.Cmd {
	cmds := []tea.Cmd{
		m.loadStories,
		m.runPreflightChecks,
		m.loadHistoricalAverages,
		git.GetStatusCmd(m.config.WorkingDir),
	}

	// Phase 6: Start watcher if enabled
	if m.config.WatchEnabled {
		cmds = append(cmds, m.startWatcher)
	}

	// Phase 6: Start API server if enabled
	if m.config.APIEnabled {
		cmds = append(cmds, m.startAPIServer)
	}

	return tea.Batch(cmds...)
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

// loadHistoricalAverages loads step averages from storage for ETA calculation
func (m Model) loadHistoricalAverages() tea.Msg {
	if m.storage == nil {
		return nil
	}

	averages, err := m.storage.GetStepAverages(context.Background())
	if err != nil {
		return nil
	}

	return historicalAveragesMsg{Averages: averages}
}

// historicalAveragesMsg carries loaded step averages
type historicalAveragesMsg struct {
	Averages map[domain.StepName]*storage.StepAverage
}

// Update handles all messages
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	// Handle command palette messages first if active
	if m.commandPalette.IsActive() {
		switch msg := msg.(type) {
		case tea.KeyMsg:
			var cmd tea.Cmd
			m.commandPalette, cmd = m.commandPalette.Update(msg)
			cmds = append(cmds, cmd)
			return m, tea.Batch(cmds...)
		case commandpalette.SelectCommandMsg:
			// Execute the selected command
			if msg.Command.Action != nil {
				cmds = append(cmds, func() tea.Msg { return msg.Command.Action() })
			}
			return m, tea.Batch(cmds...)
		case commandpalette.CloseMsg:
			return m, nil
		case commandpalette.NavigateMsg:
			m.prevView = m.activeView
			m.activeView = msg.View
			m.header.SetActiveView(m.activeView)
			return m, nil
		case commandpalette.ThemeChangeMsg:
			theme.SetTheme(msg.Theme)
			m.config.Theme = msg.Theme
			m.refreshAllStyles()
			m.statusbar.SetMessage("Theme changed to " + msg.Theme)
			return m, nil
		case commandpalette.ActionMsg:
			return m.handlePaletteAction(msg.Action)
		}
	}

	// Handle confetti animation
	if m.confetti.IsActive() {
		var cmd tea.Cmd
		m.confetti, cmd = m.confetti.Update(msg)
		cmds = append(cmds, cmd)
	}

	switch msg := msg.(type) {
	case tea.KeyMsg:
		// Command palette activation
		if msg.String() == "ctrl+p" {
			m.commandPalette.Open()
			m.commandPalette.SetSize(m.width, m.height)
			return m, nil
		}

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
				m.history.SetLoading(true)
				return m, m.loadHistory()
			}
			return m, nil

		case "a":
			// Only navigate to stats if not in storylist (where 'a' means select all)
			if m.activeView != domain.ViewStoryList && m.canNavigate() {
				m.prevView = m.activeView
				m.activeView = domain.ViewStats
				m.header.SetActiveView(m.activeView)
				m.stats.SetLoading(true)
				return m, m.loadStats()
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

	case historicalAveragesMsg:
		// Update queue with historical averages for ETA calculation
		if msg.Averages != nil {
			queue := m.batchExecutor.GetQueue()
			for stepName, avg := range msg.Averages {
				queue.UpdateStepAverage(stepName, avg.AvgDuration)
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

		// Save executions to storage and update step averages
		if m.storage != nil {
			queue := m.batchExecutor.GetQueue()
			for _, item := range queue.Items {
				if item.Execution != nil {
					m.storage.SaveExecution(context.Background(), item.Execution)
				}
			}
			m.storage.UpdateStepAverages(context.Background())
		}

		// Phase 5: Notifications, sound, and confetti on completion
		failedCount := msg.TotalItems - msg.SuccessCount
		m.notifier.NotifyQueueComplete(msg.TotalItems, msg.SuccessCount, failedCount)

		if failedCount == 0 {
			// All succeeded - play success sound and show confetti
			m.soundPlayer.PlayComplete()
			cmds = append(cmds, m.confetti.Start(m.width, m.height))
		} else {
			// Some failed - play warning sound
			m.soundPlayer.PlayWarning()
		}

	// Phase 5: Git status handling
	case git.StatusMsg:
		m.gitStatus = msg.Status
		m.statusbar.SetGitInfo(m.gitStatus.Branch, m.gitStatus.IsClean)

	// Phase 5: Settings messages
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

	// Phase 5: Confetti animation
	case confetti.TickMsg:
		var cmd tea.Cmd
		m.confetti, cmd = m.confetti.Update(msg)
		cmds = append(cmds, cmd)

	// History messages
	case messages.HistoryRefreshMsg:
		cmds = append(cmds, m.loadHistory())

	case messages.HistoryFilterMsg:
		cmds = append(cmds, m.loadHistoryFiltered(msg.Query, msg.Epic, msg.Status))

	case messages.HistoryLoadedMsg:
		m.history.SetExecutions(msg.Executions, msg.TotalCount)

	case messages.HistoryDetailMsg:
		// Load full execution with output and show in execution view
		if m.storage != nil {
			cmds = append(cmds, m.loadExecutionDetail(msg.ID))
		}

	// Stats messages
	case messages.StatsRefreshMsg:
		cmds = append(cmds, m.loadStats())

	case messages.StatsLoadedMsg:
		m.stats.SetStats(msg.Stats)

	// Diff messages
	case messages.DiffRequestMsg:
		cmds = append(cmds, m.loadDiff(msg.StoryKey))

	case messages.DiffLoadedMsg:
		m.diff.SetDiff(msg.StoryKey, msg.Content)

	// ========== Phase 6: Message Handlers ==========

	// Profile messages
	case messages.ProfileSwitchMsg:
		m.statusbar.SetMessage(fmt.Sprintf("Switched to profile: %s", msg.ProfileName))
		// Reload stories with new config
		cmds = append(cmds, m.loadStories)

	case messages.ProfileLoadedMsg:
		if msg.Error != nil {
			m.statusbar.SetMessage(fmt.Sprintf("Profile error: %v", msg.Error))
		}

	// Workflow messages
	case messages.WorkflowSwitchMsg:
		m.statusbar.SetMessage(fmt.Sprintf("Switched to workflow: %s", msg.WorkflowName))

	case messages.WorkflowLoadedMsg:
		if msg.Error != nil {
			m.statusbar.SetMessage(fmt.Sprintf("Workflow error: %v", msg.Error))
		}

	// Watch mode messages
	case watcher.RefreshMsg:
		m.statusbar.SetMessage("Files changed, refreshing stories...")
		cmds = append(cmds, m.loadStories)

	case messages.WatchStatusMsg:
		if msg.Running {
			m.statusbar.SetMessage("Watch mode enabled")
		} else {
			m.statusbar.SetMessage("Watch mode disabled")
		}

	// Parallel execution messages
	case messages.ParallelProgressMsg:
		m.statusbar.SetMessage(fmt.Sprintf("Parallel: %d/%d completed, %d active",
			msg.Completed, msg.Total, msg.Active))

	// API server messages
	case messages.APIServerStatusMsg:
		if msg.Running {
			m.statusbar.SetMessage(fmt.Sprintf("API server running at %s", msg.URL))
		} else {
			m.statusbar.SetMessage("API server stopped")
		}

	// Stories refresh (from watcher or manual)
	case messages.StoriesRefreshMsg:
		cmds = append(cmds, m.loadStories)
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

	case domain.ViewHistory:
		var cmd tea.Cmd
		m.history, cmd = m.history.Update(msg)
		cmds = append(cmds, cmd)

	case domain.ViewStats:
		var cmd tea.Cmd
		m.stats, cmd = m.stats.Update(msg)
		cmds = append(cmds, cmd)

	case domain.ViewDiff:
		var cmd tea.Cmd
		m.diff, cmd = m.diff.Update(msg)
		cmds = append(cmds, cmd)

	case domain.ViewSettings:
		var cmd tea.Cmd
		m.settings, cmd = m.settings.Update(msg)
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
		content = m.diff.View()
	case domain.ViewHistory:
		content = m.history.View()
	case domain.ViewStats:
		content = m.stats.View()
	case domain.ViewSettings:
		content = m.settings.View()
	default:
		content = m.renderPlaceholder("Unknown View", "")
	}

	// Status bar
	statusView := m.statusbar.View()

	// Combine all sections
	mainView := lipgloss.JoinVertical(lipgloss.Left,
		headerView,
		content,
		statusView,
	)

	// Overlay confetti if active
	if m.confetti.IsActive() {
		mainView = m.confetti.Overlay(mainView, m.width, m.height)
	}

	// Overlay command palette if active
	if m.commandPalette.IsActive() {
		return m.commandPalette.Overlay(mainView)
	}

	return mainView
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

// loadHistory loads execution history from storage
func (m Model) loadHistory() tea.Cmd {
	return func() tea.Msg {
		if m.storage == nil {
			return messages.HistoryLoadedMsg{
				Executions: nil,
				TotalCount: 0,
				Error:      fmt.Errorf("storage not available"),
			}
		}

		records, err := m.storage.ListExecutions(context.Background(), &storage.ExecutionFilter{
			Limit: 100,
		})
		if err != nil {
			return messages.HistoryLoadedMsg{Error: err}
		}

		count, _ := m.storage.CountExecutions(context.Background(), nil)

		executions := make([]*messages.HistoryExecution, 0, len(records))
		for _, rec := range records {
			executions = append(executions, &messages.HistoryExecution{
				ID:        rec.ID,
				StoryKey:  rec.StoryKey,
				StoryEpic: rec.StoryEpic,
				Status:    rec.Status,
				StartTime: rec.StartTime,
				Duration:  rec.Duration,
				StepCount: len(rec.Steps),
				ErrorMsg:  rec.Error,
			})
		}

		return messages.HistoryLoadedMsg{
			Executions: executions,
			TotalCount: count,
		}
	}
}

// loadHistoryFiltered loads filtered execution history
func (m Model) loadHistoryFiltered(query string, epic *int, status domain.ExecutionStatus) tea.Cmd {
	return func() tea.Msg {
		if m.storage == nil {
			return messages.HistoryLoadedMsg{Error: fmt.Errorf("storage not available")}
		}

		filter := &storage.ExecutionFilter{
			StoryKey: query,
			Epic:     epic,
			Status:   status,
			Limit:    100,
		}

		records, err := m.storage.ListExecutions(context.Background(), filter)
		if err != nil {
			return messages.HistoryLoadedMsg{Error: err}
		}

		count, _ := m.storage.CountExecutions(context.Background(), filter)

		executions := make([]*messages.HistoryExecution, 0, len(records))
		for _, rec := range records {
			executions = append(executions, &messages.HistoryExecution{
				ID:        rec.ID,
				StoryKey:  rec.StoryKey,
				StoryEpic: rec.StoryEpic,
				Status:    rec.Status,
				StartTime: rec.StartTime,
				Duration:  rec.Duration,
				StepCount: len(rec.Steps),
				ErrorMsg:  rec.Error,
			})
		}

		return messages.HistoryLoadedMsg{
			Executions: executions,
			TotalCount: count,
		}
	}
}

// loadExecutionDetail loads full execution details
func (m Model) loadExecutionDetail(id string) tea.Cmd {
	return func() tea.Msg {
		if m.storage == nil {
			return nil
		}

		record, err := m.storage.GetExecutionWithOutput(context.Background(), id)
		if err != nil {
			return messages.ErrorMsg{Error: err}
		}

		// Convert storage record to domain execution for viewing
		execution := &domain.Execution{
			Story: domain.Story{
				Key:    record.StoryKey,
				Epic:   record.StoryEpic,
				Status: domain.StoryStatus(record.StoryStatus),
			},
			Status:    record.Status,
			StartTime: record.StartTime,
			EndTime:   record.EndTime,
			Duration:  record.Duration,
			Error:     record.Error,
			Steps:     make([]*domain.StepExecution, 0, len(record.Steps)),
		}

		for _, step := range record.Steps {
			execution.Steps = append(execution.Steps, &domain.StepExecution{
				Name:      step.StepName,
				Status:    step.Status,
				StartTime: step.StartTime,
				EndTime:   step.EndTime,
				Duration:  step.Duration,
				Output:    step.Output,
				Error:     step.Error,
				Attempt:   step.Attempt,
				Command:   step.Command,
			})
		}

		return messages.ExecutionStartedMsg{Execution: execution}
	}
}

// loadStats loads statistics from storage
func (m Model) loadStats() tea.Cmd {
	return func() tea.Msg {
		if m.storage == nil {
			return messages.StatsLoadedMsg{Error: fmt.Errorf("storage not available")}
		}

		storageStats, err := m.storage.GetStats(context.Background())
		if err != nil {
			return messages.StatsLoadedMsg{Error: err}
		}

		// Convert storage stats to messages stats
		statsData := &messages.StatsData{
			TotalExecutions:  storageStats.TotalExecutions,
			SuccessfulCount:  storageStats.SuccessfulCount,
			FailedCount:      storageStats.FailedCount,
			CancelledCount:   storageStats.CancelledCount,
			SuccessRate:      storageStats.SuccessRate,
			AvgDuration:      storageStats.AvgDuration,
			TotalDuration:    storageStats.TotalDuration,
			ExecutionsByDay:  storageStats.ExecutionsByDay,
			ExecutionsByEpic: storageStats.ExecutionsByEpic,
			StepStats:        make(map[domain.StepName]*messages.StepStatsData),
		}

		for name, ss := range storageStats.StepStats {
			statsData.StepStats[name] = &messages.StepStatsData{
				StepName:     ss.StepName,
				TotalCount:   ss.TotalCount,
				SuccessCount: ss.SuccessCount,
				FailureCount: ss.FailureCount,
				SkippedCount: ss.SkippedCount,
				SuccessRate:  ss.SuccessRate,
				AvgDuration:  ss.AvgDuration,
				MinDuration:  ss.MinDuration,
				MaxDuration:  ss.MaxDuration,
			}
		}

		return messages.StatsLoadedMsg{Stats: statsData}
	}
}

// loadDiff loads git diff for a story
func (m Model) loadDiff(storyKey string) tea.Cmd {
	return func() tea.Msg {
		// Run git diff command
		cmd := exec.Command("git", "diff")
		cmd.Dir = m.config.WorkingDir

		output, err := cmd.Output()
		if err != nil {
			return messages.DiffLoadedMsg{
				StoryKey: storyKey,
				Error:    err,
			}
		}

		return messages.DiffLoadedMsg{
			StoryKey: storyKey,
			Content:  strings.TrimSpace(string(output)),
		}
	}
}

// refreshAllStyles rebuilds all styles after a theme change
func (m *Model) refreshAllStyles() {
	m.styles = theme.NewStyles()
	m.header = header.New()
	m.statusbar = statusbar.New()
	m.dashboard = dashboard.New()
	m.storylist.RefreshStyles()
	m.execution.RefreshStyles()
	m.queue.RefreshStyles()
	m.timeline.RefreshStyles()
	m.history.RefreshStyles()
	m.stats.RefreshStyles()
	m.diff.RefreshStyles()
	m.settings.RefreshStyles()
	m.commandPalette = commandpalette.New()

	// Re-propagate data to views
	m.header.SetWidth(m.width)
	m.header.SetActiveView(m.activeView)
	m.statusbar.SetWidth(m.width)
	m.statusbar.SetGitInfo(m.gitStatus.Branch, m.gitStatus.IsClean)
	m.statusbar.SetStoryCounts(len(m.stories), m.batchExecutor.GetQueue().TotalCount())
	m.dashboard.SetStories(m.stories)
	m.storylist.SetStories(m.stories)
}

// handlePaletteAction handles actions from the command palette
func (m Model) handlePaletteAction(action string) (Model, tea.Cmd) {
	switch action {
	case "start_queue":
		queue := m.batchExecutor.GetQueue()
		if queue.Status == domain.QueueIdle && queue.HasPending() {
			m.prevView = m.activeView
			m.activeView = domain.ViewExecution
			m.header.SetActiveView(m.activeView)
			return m, m.batchExecutor.Start()
		}
	case "pause_queue":
		if m.batchExecutor.IsRunning() && !m.batchExecutor.IsPaused() {
			m.batchExecutor.Pause()
			m.statusbar.SetMessage("Queue paused")
		}
	case "clear_queue":
		if !m.batchExecutor.IsRunning() {
			m.batchExecutor.GetQueue().Clear()
			m.queue.SetQueue(m.batchExecutor.GetQueue())
			m.statusbar.SetStoryCounts(len(m.stories), 0)
			m.statusbar.SetMessage("Queue cleared")
		}
	case "refresh":
		return m, m.loadStories
	// Phase 6: Watch mode actions
	case "toggle_watch":
		if m.watcher.IsRunning() {
			m.watcher.Stop()
			m.statusbar.SetMessage("Watch mode disabled")
		} else {
			m.watcher.Start()
			m.statusbar.SetMessage("Watch mode enabled")
		}
	// Phase 6: API server actions
	case "toggle_api":
		if m.apiServer.IsRunning() {
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			m.apiServer.Stop(ctx)
			m.statusbar.SetMessage("API server stopped")
		} else {
			go m.apiServer.Start(m.config.APIPort)
			m.statusbar.SetMessage(fmt.Sprintf("API server started on port %d", m.config.APIPort))
		}
	// Phase 6: Parallel execution
	case "parallel_mode":
		m.config.ParallelEnabled = !m.config.ParallelEnabled
		if m.config.ParallelEnabled {
			m.statusbar.SetMessage(fmt.Sprintf("Parallel mode enabled (%d workers)", m.config.MaxWorkers))
		} else {
			m.statusbar.SetMessage("Sequential mode enabled")
		}
	}
	return m, nil
}

// ========== Phase 6: Helper Functions ==========

// startWatcher starts the file watcher
func (m Model) startWatcher() tea.Msg {
	if err := m.watcher.Start(); err != nil {
		return messages.ErrorMsg{Error: err}
	}
	return messages.WatchStatusMsg{Running: true, Paths: []string{m.config.SprintStatusPath}}
}

// startAPIServer starts the API server
func (m Model) startAPIServer() tea.Msg {
	go func() {
		m.apiServer.SetStories(m.stories)
		m.apiServer.Start(m.config.APIPort)
	}()
	return messages.APIServerStatusMsg{
		Running: true,
		Port:    m.config.APIPort,
		URL:     fmt.Sprintf("http://localhost:%d", m.config.APIPort),
	}
}

// switchProfile switches to a different profile
func (m *Model) switchProfile(profileName string) tea.Cmd {
	return func() tea.Msg {
		p, ok := m.profileStore.Get(profileName)
		if !ok {
			return messages.ErrorMsg{Error: fmt.Errorf("profile not found: %s", profileName)}
		}

		// Apply profile settings to config
		if p.SprintStatusPath != "" {
			m.config.SprintStatusPath = p.SprintStatusPath
		}
		if p.StoryDir != "" {
			m.config.StoryDir = p.StoryDir
		}
		if p.WorkingDir != "" {
			m.config.WorkingDir = p.WorkingDir
		}
		if p.Timeout > 0 {
			m.config.Timeout = p.Timeout
		}
		if p.Retries > 0 {
			m.config.Retries = p.Retries
		}
		if p.Theme != "" {
			m.config.Theme = p.Theme
			theme.SetTheme(p.Theme)
		}
		if p.MaxWorkers > 0 {
			m.config.MaxWorkers = p.MaxWorkers
			m.parallelExecutor.SetWorkers(p.MaxWorkers)
		}

		m.profileStore.SetActive(profileName)
		m.config.ActiveProfile = profileName

		return messages.ProfileSwitchMsg{ProfileName: profileName}
	}
}

// switchWorkflow switches to a different workflow
func (m *Model) switchWorkflow(workflowName string) tea.Cmd {
	return func() tea.Msg {
		_, ok := m.workflowStore.Get(workflowName)
		if !ok {
			return messages.ErrorMsg{Error: fmt.Errorf("workflow not found: %s", workflowName)}
		}

		m.config.ActiveWorkflow = workflowName
		return messages.WorkflowSwitchMsg{WorkflowName: workflowName}
	}
}

// GetActiveWorkflow returns the currently active workflow
func (m Model) GetActiveWorkflow() *workflow.Workflow {
	w, _ := m.workflowStore.Get(m.config.ActiveWorkflow)
	return w
}

// GetActiveProfile returns the currently active profile
func (m Model) GetActiveProfile() *profile.Profile {
	return m.profileStore.GetActiveProfile()
}

// Cleanup performs cleanup when the application exits
func (m *Model) Cleanup() {
	// Stop watcher if running
	if m.watcher != nil && m.watcher.IsRunning() {
		m.watcher.Stop()
	}

	// Stop API server if running
	if m.apiServer != nil && m.apiServer.IsRunning() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		m.apiServer.Stop(ctx)
	}

	// Close storage
	if m.storage != nil {
		m.storage.Close()
	}
}
