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
	"github.com/robertguss/bmad-automate-go/internal/util"
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
	_ = profileStore.Load()

	// Initialize Phase 6: Workflow store
	workflowStore := workflow.NewWorkflowStore(cfg.DataDir)
	_ = workflowStore.Load()

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
// QUAL-001: Refactored to use extracted handlers for better maintainability
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	// Handle command palette messages first if active
	if newModel, cmd, handled := m.handleCommandPaletteMsg(msg); handled {
		return newModel, cmd
	}

	// Handle confetti animation
	if m.confetti.IsActive() {
		var cmd tea.Cmd
		m.confetti, cmd = m.confetti.Update(msg)
		cmds = append(cmds, cmd)
	}

	// Process messages by type
	switch msg := msg.(type) {
	case tea.KeyMsg:
		if newModel, cmd, handled := m.handleKeyMsg(msg); handled {
			return newModel, cmd
		}

	case tea.WindowSizeMsg:
		m = m.handleWindowSizeMsg(msg)

	case messages.StoriesLoadedMsg:
		m = m.handleStoriesMsg(msg)

	case preflightResultsMsg:
		m.preflightResults = msg.Results
		if !msg.Results.AllPass {
			failed := msg.Results.FailedChecks()
			if len(failed) > 0 {
				m.statusbar.SetMessage(fmt.Sprintf("Pre-flight warning: %s", failed[0].Error))
			}
		}

	case historicalAveragesMsg:
		if msg.Averages != nil {
			queue := m.batchExecutor.GetQueue()
			for stepName, avg := range msg.Averages {
				queue.UpdateStepAverage(stepName, avg.AvgDuration)
			}
		}

	// Execution messages
	case messages.ExecutionStartMsg, messages.ExecutionStartedMsg, messages.StepStartedMsg,
		messages.StepOutputMsg, messages.StepCompletedMsg, messages.ExecutionCompletedMsg,
		messages.ExecutionTickMsg:
		var execCmds []tea.Cmd
		m, execCmds = m.handleExecutionMsgs(msg)
		cmds = append(cmds, execCmds...)

	// Queue messages
	case messages.QueueUpdatedMsg, messages.QueueItemStartedMsg, messages.QueueItemCompletedMsg,
		messages.QueueCompletedMsg:
		var queueCmds []tea.Cmd
		m, queueCmds = m.handleQueueMsgs(msg)
		cmds = append(cmds, queueCmds...)

	// Settings and git messages
	case git.StatusMsg, settings.ThemeChangedMsg, settings.SettingChangedMsg, confetti.TickMsg:
		m = m.handleSettingsMsgs(msg)

	// History, stats, and diff messages
	case messages.HistoryRefreshMsg, messages.HistoryFilterMsg, messages.HistoryLoadedMsg,
		messages.HistoryDetailMsg, messages.StatsRefreshMsg, messages.StatsLoadedMsg,
		messages.DiffRequestMsg, messages.DiffLoadedMsg:
		var histCmds []tea.Cmd
		m, histCmds = m.handleHistoryStatsMsgs(msg)
		cmds = append(cmds, histCmds...)

	// Phase 6 messages
	case messages.ProfileSwitchMsg, messages.ProfileLoadedMsg, messages.WorkflowSwitchMsg,
		messages.WorkflowLoadedMsg, watcher.RefreshMsg, messages.WatchStatusMsg,
		messages.ParallelProgressMsg, messages.APIServerStatusMsg, messages.StoriesRefreshMsg:
		var p6Cmds []tea.Cmd
		m, p6Cmds = m.handlePhase6Msgs(msg)
		cmds = append(cmds, p6Cmds...)
	}

	// Route to active view
	var viewCmd tea.Cmd
	m, viewCmd = m.routeToActiveView(msg)
	cmds = append(cmds, viewCmd)

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

// formatDuration is an alias to the shared utility function
// QUAL-002: Using shared utility instead of duplicated code
var formatDuration = util.FormatDuration

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
			_ = m.watcher.Stop()
			m.statusbar.SetMessage("Watch mode disabled")
		} else {
			_ = m.watcher.Start()
			m.statusbar.SetMessage("Watch mode enabled")
		}
	// Phase 6: API server actions
	case "toggle_api":
		if m.apiServer.IsRunning() {
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			_ = m.apiServer.Stop(ctx)
			m.statusbar.SetMessage("API server stopped")
		} else {
			go func() { _ = m.apiServer.Start(m.config.APIPort) }()
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
		_ = m.apiServer.Start(m.config.APIPort)
	}()
	return messages.APIServerStatusMsg{
		Running: true,
		Port:    m.config.APIPort,
		URL:     fmt.Sprintf("http://localhost:%d", m.config.APIPort),
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
		_ = m.watcher.Stop()
	}

	// Stop API server if running
	if m.apiServer != nil && m.apiServer.IsRunning() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = m.apiServer.Stop(ctx)
	}

	// Close storage
	if m.storage != nil {
		m.storage.Close()
	}
}
