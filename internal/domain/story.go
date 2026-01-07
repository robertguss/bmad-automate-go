package domain

// StoryStatus represents the development status of a story
type StoryStatus string

const (
	StatusInProgress  StoryStatus = "in-progress"
	StatusReadyForDev StoryStatus = "ready-for-dev"
	StatusBacklog     StoryStatus = "backlog"
	StatusDone        StoryStatus = "done"
	StatusBlocked     StoryStatus = "blocked"
)

// Story represents a development story from sprint-status.yaml
type Story struct {
	Key        string
	Epic       int
	Status     StoryStatus
	Title      string
	FilePath   string
	FileExists bool
}

// IsActionable returns true if the story can be processed
func (s Story) IsActionable() bool {
	return s.Status == StatusInProgress ||
		s.Status == StatusReadyForDev ||
		s.Status == StatusBacklog
}

// StepName represents a workflow step
type StepName string

const (
	StepCreateStory StepName = "create-story"
	StepDevStory    StepName = "dev-story"
	StepCodeReview  StepName = "code-review"
	StepGitCommit   StepName = "git-commit"
)

// AllSteps returns all workflow steps in order
func AllSteps() []StepName {
	return []StepName{
		StepCreateStory,
		StepDevStory,
		StepCodeReview,
		StepGitCommit,
	}
}

// StepStatus represents the execution status of a step
type StepStatus string

const (
	StepPending StepStatus = "pending"
	StepRunning StepStatus = "running"
	StepSuccess StepStatus = "success"
	StepFailed  StepStatus = "failed"
	StepSkipped StepStatus = "skipped"
)
