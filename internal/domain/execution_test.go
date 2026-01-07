package domain

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewExecution(t *testing.T) {
	story := Story{
		Key:    "3-1-test-story",
		Epic:   3,
		Status: StatusInProgress,
		Title:  "Test Story",
	}

	exec := NewExecution(story)

	t.Run("creates execution with correct story", func(t *testing.T) {
		assert.Equal(t, story.Key, exec.Story.Key)
		assert.Equal(t, story.Epic, exec.Story.Epic)
		assert.Equal(t, story.Status, exec.Story.Status)
	})

	t.Run("creates execution with pending status", func(t *testing.T) {
		assert.Equal(t, ExecutionPending, exec.Status)
	})

	t.Run("creates execution with 4 steps", func(t *testing.T) {
		assert.Len(t, exec.Steps, 4)
	})

	t.Run("creates steps in correct order", func(t *testing.T) {
		expectedSteps := AllSteps()
		for i, step := range exec.Steps {
			assert.Equal(t, expectedSteps[i], step.Name)
		}
	})

	t.Run("creates steps with pending status", func(t *testing.T) {
		for _, step := range exec.Steps {
			assert.Equal(t, StepPending, step.Status)
		}
	})

	t.Run("creates steps with empty output slice", func(t *testing.T) {
		for _, step := range exec.Steps {
			assert.NotNil(t, step.Output)
			assert.Len(t, step.Output, 0)
		}
	})

	t.Run("creates steps with zero attempt count", func(t *testing.T) {
		for _, step := range exec.Steps {
			assert.Equal(t, 0, step.Attempt)
		}
	})

	t.Run("sets current step to 0", func(t *testing.T) {
		assert.Equal(t, 0, exec.Current)
	})
}

func TestExecution_CurrentStep(t *testing.T) {
	tests := []struct {
		name     string
		current  int
		expected StepName
		isNil    bool
	}{
		{
			name:     "returns first step when current is 0",
			current:  0,
			expected: StepCreateStory,
			isNil:    false,
		},
		{
			name:     "returns second step when current is 1",
			current:  1,
			expected: StepDevStory,
			isNil:    false,
		},
		{
			name:     "returns last step when current is 3",
			current:  3,
			expected: StepGitCommit,
			isNil:    false,
		},
		{
			name:    "returns nil when current is negative",
			current: -1,
			isNil:   true,
		},
		{
			name:    "returns nil when current is beyond steps",
			current: 4,
			isNil:   true,
		},
		{
			name:    "returns nil when current is way beyond steps",
			current: 100,
			isNil:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			story := Story{Key: "3-1-test"}
			exec := NewExecution(story)
			exec.Current = tt.current

			step := exec.CurrentStep()

			if tt.isNil {
				assert.Nil(t, step)
			} else {
				require.NotNil(t, step)
				assert.Equal(t, tt.expected, step.Name)
			}
		})
	}
}

func TestExecution_ProgressPercent(t *testing.T) {
	tests := []struct {
		name            string
		completedSteps  []int // indices of steps to mark as complete
		stepStatuses    []StepStatus
		expectedPercent float64
	}{
		{
			name:            "0% when no steps completed",
			completedSteps:  []int{},
			stepStatuses:    []StepStatus{StepPending, StepPending, StepPending, StepPending},
			expectedPercent: 0,
		},
		{
			name:            "25% when 1 step completed",
			completedSteps:  []int{0},
			stepStatuses:    []StepStatus{StepSuccess, StepPending, StepPending, StepPending},
			expectedPercent: 25,
		},
		{
			name:            "50% when 2 steps completed",
			completedSteps:  []int{0, 1},
			stepStatuses:    []StepStatus{StepSuccess, StepSuccess, StepPending, StepPending},
			expectedPercent: 50,
		},
		{
			name:            "75% when 3 steps completed",
			completedSteps:  []int{0, 1, 2},
			stepStatuses:    []StepStatus{StepSuccess, StepSuccess, StepSuccess, StepPending},
			expectedPercent: 75,
		},
		{
			name:            "100% when all steps completed",
			completedSteps:  []int{0, 1, 2, 3},
			stepStatuses:    []StepStatus{StepSuccess, StepSuccess, StepSuccess, StepSuccess},
			expectedPercent: 100,
		},
		{
			name:            "counts failed as complete",
			completedSteps:  []int{0, 1},
			stepStatuses:    []StepStatus{StepSuccess, StepFailed, StepPending, StepPending},
			expectedPercent: 50,
		},
		{
			name:            "counts skipped as complete",
			completedSteps:  []int{0, 1, 2},
			stepStatuses:    []StepStatus{StepSuccess, StepSkipped, StepSkipped, StepPending},
			expectedPercent: 75,
		},
		{
			name:            "running step is not complete",
			completedSteps:  []int{0},
			stepStatuses:    []StepStatus{StepSuccess, StepRunning, StepPending, StepPending},
			expectedPercent: 25,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			story := Story{Key: "3-1-test"}
			exec := NewExecution(story)

			for i, status := range tt.stepStatuses {
				exec.Steps[i].Status = status
			}

			assert.Equal(t, tt.expectedPercent, exec.ProgressPercent())
		})
	}
}

func TestExecution_ProgressPercent_EmptySteps(t *testing.T) {
	exec := &Execution{
		Steps: []*StepExecution{},
	}

	assert.Equal(t, float64(0), exec.ProgressPercent())
}

func TestExecution_SuccessfulSteps(t *testing.T) {
	tests := []struct {
		name         string
		stepStatuses []StepStatus
		expected     int
	}{
		{
			name:         "zero when no success",
			stepStatuses: []StepStatus{StepPending, StepPending, StepPending, StepPending},
			expected:     0,
		},
		{
			name:         "counts only success status",
			stepStatuses: []StepStatus{StepSuccess, StepPending, StepPending, StepPending},
			expected:     1,
		},
		{
			name:         "all success",
			stepStatuses: []StepStatus{StepSuccess, StepSuccess, StepSuccess, StepSuccess},
			expected:     4,
		},
		{
			name:         "does not count failed",
			stepStatuses: []StepStatus{StepSuccess, StepFailed, StepPending, StepPending},
			expected:     1,
		},
		{
			name:         "does not count skipped",
			stepStatuses: []StepStatus{StepSuccess, StepSkipped, StepSuccess, StepPending},
			expected:     2,
		},
		{
			name:         "does not count running",
			stepStatuses: []StepStatus{StepSuccess, StepRunning, StepPending, StepPending},
			expected:     1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			story := Story{Key: "3-1-test"}
			exec := NewExecution(story)

			for i, status := range tt.stepStatuses {
				exec.Steps[i].Status = status
			}

			assert.Equal(t, tt.expected, exec.SuccessfulSteps())
		})
	}
}

func TestExecution_FailedStep(t *testing.T) {
	tests := []struct {
		name         string
		stepStatuses []StepStatus
		expectedStep *StepName
	}{
		{
			name:         "returns nil when no failure",
			stepStatuses: []StepStatus{StepSuccess, StepSuccess, StepSuccess, StepSuccess},
			expectedStep: nil,
		},
		{
			name:         "returns nil when all pending",
			stepStatuses: []StepStatus{StepPending, StepPending, StepPending, StepPending},
			expectedStep: nil,
		},
		{
			name:         "returns first failed step",
			stepStatuses: []StepStatus{StepSuccess, StepFailed, StepPending, StepPending},
			expectedStep: func() *StepName { s := StepDevStory; return &s }(),
		},
		{
			name:         "returns first failed when multiple failures",
			stepStatuses: []StepStatus{StepSuccess, StepFailed, StepFailed, StepPending},
			expectedStep: func() *StepName { s := StepDevStory; return &s }(),
		},
		{
			name:         "returns failed at first position",
			stepStatuses: []StepStatus{StepFailed, StepPending, StepPending, StepPending},
			expectedStep: func() *StepName { s := StepCreateStory; return &s }(),
		},
		{
			name:         "returns failed at last position",
			stepStatuses: []StepStatus{StepSuccess, StepSuccess, StepSuccess, StepFailed},
			expectedStep: func() *StepName { s := StepGitCommit; return &s }(),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			story := Story{Key: "3-1-test"}
			exec := NewExecution(story)

			for i, status := range tt.stepStatuses {
				exec.Steps[i].Status = status
			}

			failedStep := exec.FailedStep()

			if tt.expectedStep == nil {
				assert.Nil(t, failedStep)
			} else {
				require.NotNil(t, failedStep)
				assert.Equal(t, *tt.expectedStep, failedStep.Name)
			}
		})
	}
}

func TestExecution_TotalDuration(t *testing.T) {
	tests := []struct {
		name      string
		durations []time.Duration
		expected  time.Duration
	}{
		{
			name:      "zero when no durations",
			durations: []time.Duration{0, 0, 0, 0},
			expected:  0,
		},
		{
			name:      "sums all durations",
			durations: []time.Duration{time.Minute, 2 * time.Minute, 3 * time.Minute, 4 * time.Minute},
			expected:  10 * time.Minute,
		},
		{
			name:      "handles partial durations",
			durations: []time.Duration{time.Minute, 0, time.Minute, 0},
			expected:  2 * time.Minute,
		},
		{
			name:      "single step duration",
			durations: []time.Duration{5 * time.Second, 0, 0, 0},
			expected:  5 * time.Second,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			story := Story{Key: "3-1-test"}
			exec := NewExecution(story)

			for i, dur := range tt.durations {
				exec.Steps[i].Duration = dur
			}

			assert.Equal(t, tt.expected, exec.TotalDuration())
		})
	}
}

func TestStepExecution_IsComplete(t *testing.T) {
	tests := []struct {
		name     string
		status   StepStatus
		expected bool
	}{
		{
			name:     "pending is not complete",
			status:   StepPending,
			expected: false,
		},
		{
			name:     "running is not complete",
			status:   StepRunning,
			expected: false,
		},
		{
			name:     "success is complete",
			status:   StepSuccess,
			expected: true,
		},
		{
			name:     "failed is complete",
			status:   StepFailed,
			expected: true,
		},
		{
			name:     "skipped is complete",
			status:   StepSkipped,
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			step := &StepExecution{
				Name:   StepCreateStory,
				Status: tt.status,
			}
			assert.Equal(t, tt.expected, step.IsComplete())
		})
	}
}

func TestExecutionStatus_Constants(t *testing.T) {
	tests := []struct {
		name     string
		status   ExecutionStatus
		expected string
	}{
		{
			name:     "ExecutionPending value",
			status:   ExecutionPending,
			expected: "pending",
		},
		{
			name:     "ExecutionRunning value",
			status:   ExecutionRunning,
			expected: "running",
		},
		{
			name:     "ExecutionPaused value",
			status:   ExecutionPaused,
			expected: "paused",
		},
		{
			name:     "ExecutionCompleted value",
			status:   ExecutionCompleted,
			expected: "completed",
		},
		{
			name:     "ExecutionFailed value",
			status:   ExecutionFailed,
			expected: "failed",
		},
		{
			name:     "ExecutionCancelled value",
			status:   ExecutionCancelled,
			expected: "cancelled",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, string(tt.status))
		})
	}
}

func TestStepExecution_Fields(t *testing.T) {
	now := time.Now()
	step := StepExecution{
		Name:      StepCreateStory,
		Status:    StepRunning,
		StartTime: now,
		EndTime:   now.Add(time.Minute),
		Duration:  time.Minute,
		Output:    []string{"line1", "line2"},
		Error:     "test error",
		Attempt:   2,
		Command:   "test command",
	}

	assert.Equal(t, StepCreateStory, step.Name)
	assert.Equal(t, StepRunning, step.Status)
	assert.Equal(t, now, step.StartTime)
	assert.Equal(t, now.Add(time.Minute), step.EndTime)
	assert.Equal(t, time.Minute, step.Duration)
	assert.Equal(t, []string{"line1", "line2"}, step.Output)
	assert.Equal(t, "test error", step.Error)
	assert.Equal(t, 2, step.Attempt)
	assert.Equal(t, "test command", step.Command)
}
