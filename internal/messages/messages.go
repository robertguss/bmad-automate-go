package messages

import (
	"time"

	"github.com/robertguss/bmad-automate-go/internal/domain"
)

// Navigation messages
type NavigateMsg struct {
	View domain.View
}

type NavigateBackMsg struct{}

// Story messages
type StoriesLoadedMsg struct {
	Stories []domain.Story
	Error   error
}

type StorySelectedMsg struct {
	Story domain.Story
}

type StoriesFilteredMsg struct {
	Epic   int
	Status domain.StoryStatus
}

// Window size message
type WindowSizeMsg struct {
	Width  int
	Height int
}

// Error message
type ErrorMsg struct {
	Error error
}

// Tick message for animations/updates
type TickMsg struct{}

// ========== Execution Messages ==========

// ExecutionStartMsg is sent when a story execution is requested
type ExecutionStartMsg struct {
	Story domain.Story
}

// ExecutionStartedMsg is sent when execution actually begins
type ExecutionStartedMsg struct {
	Execution *domain.Execution
}

// StepStartedMsg is sent when a step begins execution
type StepStartedMsg struct {
	StepIndex int
	StepName  domain.StepName
	Command   string
	Attempt   int
}

// StepOutputMsg is sent when a step produces output
type StepOutputMsg struct {
	StepIndex int
	Line      string
	IsStderr  bool
}

// StepCompletedMsg is sent when a step finishes
type StepCompletedMsg struct {
	StepIndex int
	Status    domain.StepStatus
	Duration  time.Duration
	Error     string
}

// ExecutionCompletedMsg is sent when all steps are done
type ExecutionCompletedMsg struct {
	Status   domain.ExecutionStatus
	Duration time.Duration
	Error    string
}

// ExecutionPauseMsg requests pausing the current execution
type ExecutionPauseMsg struct{}

// ExecutionResumeMsg requests resuming a paused execution
type ExecutionResumeMsg struct{}

// ExecutionCancelMsg requests cancelling the current execution
type ExecutionCancelMsg struct{}

// StepSkipMsg requests skipping the current step
type StepSkipMsg struct{}

// StepRetryMsg requests retrying the current/failed step
type StepRetryMsg struct{}

// ExecutionTickMsg is sent periodically to update duration display
type ExecutionTickMsg struct {
	Time time.Time
}

// ========== Queue Messages ==========

// QueueAddMsg requests adding stories to the queue
type QueueAddMsg struct {
	Stories []domain.Story
}

// QueueRemoveMsg requests removing a story from the queue
type QueueRemoveMsg struct {
	Key string
}

// QueueClearMsg requests clearing all pending items from the queue
type QueueClearMsg struct{}

// QueueMoveUpMsg requests moving an item up in the queue
type QueueMoveUpMsg struct {
	Index int
}

// QueueMoveDownMsg requests moving an item down in the queue
type QueueMoveDownMsg struct {
	Index int
}

// QueueStartMsg requests starting queue execution
type QueueStartMsg struct{}

// QueuePauseMsg requests pausing queue execution
type QueuePauseMsg struct{}

// QueueResumeMsg requests resuming queue execution
type QueueResumeMsg struct{}

// QueueCancelMsg requests cancelling queue execution
type QueueCancelMsg struct{}

// QueueItemStartedMsg is sent when a queue item starts executing
type QueueItemStartedMsg struct {
	Index     int
	Story     domain.Story
	Execution *domain.Execution
}

// QueueItemCompletedMsg is sent when a queue item finishes
type QueueItemCompletedMsg struct {
	Index    int
	Story    domain.Story
	Status   domain.ExecutionStatus
	Duration time.Duration
	Error    string
}

// QueueCompletedMsg is sent when the entire queue finishes
type QueueCompletedMsg struct {
	TotalItems    int
	SuccessCount  int
	FailedCount   int
	TotalDuration time.Duration
}

// QueueUpdatedMsg is sent when queue state changes
type QueueUpdatedMsg struct {
	Queue *domain.Queue
}

// ========== History Messages ==========

// HistoryLoadedMsg is sent when history data is loaded
type HistoryLoadedMsg struct {
	Executions []*HistoryExecution
	TotalCount int
	Error      error
}

// HistoryExecution represents a stored execution for display
type HistoryExecution struct {
	ID         string
	StoryKey   string
	StoryEpic  int
	Status     domain.ExecutionStatus
	StartTime  time.Time
	Duration   time.Duration
	StepCount  int
	ErrorMsg   string
}

// HistoryFilterMsg requests filtering history
type HistoryFilterMsg struct {
	Query  string
	Epic   *int
	Status domain.ExecutionStatus
}

// HistoryRefreshMsg requests refreshing history data
type HistoryRefreshMsg struct{}

// HistoryDetailMsg requests viewing execution details
type HistoryDetailMsg struct {
	ID string
}

// ========== Statistics Messages ==========

// StatsLoadedMsg is sent when statistics are loaded
type StatsLoadedMsg struct {
	Stats *StatsData
	Error error
}

// StatsData contains all statistics for display
type StatsData struct {
	TotalExecutions  int
	SuccessfulCount  int
	FailedCount      int
	CancelledCount   int
	SuccessRate      float64
	AvgDuration      time.Duration
	TotalDuration    time.Duration
	StepStats        map[domain.StepName]*StepStatsData
	ExecutionsByDay  map[string]int
	ExecutionsByEpic map[int]int
}

// StepStatsData contains statistics for a single step
type StepStatsData struct {
	StepName     domain.StepName
	TotalCount   int
	SuccessCount int
	FailureCount int
	SkippedCount int
	SuccessRate  float64
	AvgDuration  time.Duration
	MinDuration  time.Duration
	MaxDuration  time.Duration
}

// StatsRefreshMsg requests refreshing statistics
type StatsRefreshMsg struct{}

// ========== Diff Messages ==========

// DiffLoadedMsg is sent when diff content is loaded
type DiffLoadedMsg struct {
	StoryKey string
	Content  string
	Error    error
}

// DiffRequestMsg requests loading diff for a story
type DiffRequestMsg struct {
	StoryKey string
}
