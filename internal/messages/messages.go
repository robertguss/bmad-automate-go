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
