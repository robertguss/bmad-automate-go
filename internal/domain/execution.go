package domain

import (
	"time"
)

// ExecutionStatus represents the overall status of a story execution
type ExecutionStatus string

const (
	ExecutionPending   ExecutionStatus = "pending"
	ExecutionRunning   ExecutionStatus = "running"
	ExecutionPaused    ExecutionStatus = "paused"
	ExecutionCompleted ExecutionStatus = "completed"
	ExecutionFailed    ExecutionStatus = "failed"
	ExecutionCancelled ExecutionStatus = "cancelled"
)

// StepExecution represents the execution state of a single step
type StepExecution struct {
	Name      StepName
	Status    StepStatus
	StartTime time.Time
	EndTime   time.Time
	Duration  time.Duration
	Output    []string // Lines of output
	Error     string
	Attempt   int // Current attempt number (1-based)
	Command   string
}

// IsComplete returns true if the step has finished (success, failed, or skipped)
func (s *StepExecution) IsComplete() bool {
	return s.Status == StepSuccess || s.Status == StepFailed || s.Status == StepSkipped
}

// Execution represents the full execution state of a story through all steps
type Execution struct {
	Story     Story
	Status    ExecutionStatus
	Steps     []*StepExecution
	Current   int // Index of current step (0-based)
	StartTime time.Time
	EndTime   time.Time
	Duration  time.Duration
	Error     string
}

// NewExecution creates a new Execution for a story with all steps initialized
func NewExecution(story Story) *Execution {
	steps := make([]*StepExecution, len(AllSteps()))
	for i, stepName := range AllSteps() {
		steps[i] = &StepExecution{
			Name:    stepName,
			Status:  StepPending,
			Output:  make([]string, 0),
			Attempt: 0,
		}
	}

	return &Execution{
		Story:   story,
		Status:  ExecutionPending,
		Steps:   steps,
		Current: 0,
	}
}

// CurrentStep returns the current step execution, or nil if none
func (e *Execution) CurrentStep() *StepExecution {
	if e.Current >= 0 && e.Current < len(e.Steps) {
		return e.Steps[e.Current]
	}
	return nil
}

// ProgressPercent returns the execution progress as a percentage (0-100)
func (e *Execution) ProgressPercent() float64 {
	if len(e.Steps) == 0 {
		return 0
	}

	completed := 0
	for _, step := range e.Steps {
		if step.IsComplete() {
			completed++
		}
	}

	return float64(completed) / float64(len(e.Steps)) * 100
}

// SuccessfulSteps returns the count of successful steps
func (e *Execution) SuccessfulSteps() int {
	count := 0
	for _, step := range e.Steps {
		if step.Status == StepSuccess {
			count++
		}
	}
	return count
}

// FailedStep returns the first failed step, or nil if none
func (e *Execution) FailedStep() *StepExecution {
	for _, step := range e.Steps {
		if step.Status == StepFailed {
			return step
		}
	}
	return nil
}

// TotalDuration returns the total duration of completed steps
func (e *Execution) TotalDuration() time.Duration {
	var total time.Duration
	for _, step := range e.Steps {
		total += step.Duration
	}
	return total
}
