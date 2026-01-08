package storage

import (
	"context"
	"time"

	"github.com/robertguss/bmad-automate-go/internal/domain"
)

// ExecutionRecord represents a stored execution
type ExecutionRecord struct {
	ID          string
	StoryKey    string
	StoryEpic   int
	StoryStatus string
	StoryTitle  string
	Status      domain.ExecutionStatus
	StartTime   time.Time
	EndTime     time.Time
	Duration    time.Duration
	Error       string
	CreatedAt   time.Time
	Steps       []*StepRecord
}

// StepRecord represents a stored step execution
type StepRecord struct {
	ID          string
	ExecutionID string
	StepName    domain.StepName
	Status      domain.StepStatus
	StartTime   time.Time
	EndTime     time.Time
	Duration    time.Duration
	Attempt     int
	Command     string
	Error       string
	OutputSize  int
	Output      []string // Loaded on demand
}

// StepAverage represents historical averages for a step
type StepAverage struct {
	StepName     domain.StepName
	AvgDuration  time.Duration
	SuccessCount int
	FailureCount int
	TotalCount   int
	LastUpdated  time.Time
}

// ExecutionFilter provides filtering options for listing executions
type ExecutionFilter struct {
	StoryKey    string                 // Filter by story key (partial match)
	Epic        *int                   // Filter by epic number
	Status      domain.ExecutionStatus // Filter by status
	StartAfter  *time.Time             // Filter by start time
	StartBefore *time.Time             // Filter by start time
	Limit       int                    // Max results (default 100)
	Offset      int                    // Pagination offset
}

// Stats represents aggregate statistics
type Stats struct {
	TotalExecutions  int
	SuccessfulCount  int
	FailedCount      int
	CancelledCount   int
	SuccessRate      float64
	AvgDuration      time.Duration
	TotalDuration    time.Duration
	StepStats        map[domain.StepName]*StepStats
	RecentExecutions []*ExecutionRecord
	ExecutionsByDay  map[string]int
	ExecutionsByEpic map[int]int
}

// StepStats represents statistics for a specific step
type StepStats struct {
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

// Storage defines the interface for persistence operations
type Storage interface {
	// Lifecycle
	Close() error

	// Executions
	SaveExecution(ctx context.Context, exec *domain.Execution) error
	GetExecution(ctx context.Context, id string) (*ExecutionRecord, error)
	GetExecutionWithOutput(ctx context.Context, id string) (*ExecutionRecord, error)
	ListExecutions(ctx context.Context, filter *ExecutionFilter) ([]*ExecutionRecord, error)
	CountExecutions(ctx context.Context, filter *ExecutionFilter) (int, error)
	DeleteExecution(ctx context.Context, id string) error

	// Step output (loaded separately for performance)
	GetStepOutput(ctx context.Context, stepID string) ([]string, error)

	// Statistics
	GetStats(ctx context.Context) (*Stats, error)
	GetStepAverages(ctx context.Context) (map[domain.StepName]*StepAverage, error)
	UpdateStepAverages(ctx context.Context) error

	// Recent activity
	GetRecentExecutions(ctx context.Context, limit int) ([]*ExecutionRecord, error)
	GetExecutionsByStory(ctx context.Context, storyKey string) ([]*ExecutionRecord, error)
}
