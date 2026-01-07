package domain

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestStory_IsActionable(t *testing.T) {
	tests := []struct {
		name     string
		status   StoryStatus
		expected bool
	}{
		{
			name:     "in-progress is actionable",
			status:   StatusInProgress,
			expected: true,
		},
		{
			name:     "ready-for-dev is actionable",
			status:   StatusReadyForDev,
			expected: true,
		},
		{
			name:     "backlog is actionable",
			status:   StatusBacklog,
			expected: true,
		},
		{
			name:     "done is not actionable",
			status:   StatusDone,
			expected: false,
		},
		{
			name:     "blocked is not actionable",
			status:   StatusBlocked,
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			story := Story{
				Key:    "3-1-test-story",
				Status: tt.status,
			}
			assert.Equal(t, tt.expected, story.IsActionable())
		})
	}
}

func TestStoryStatus_Constants(t *testing.T) {
	tests := []struct {
		name     string
		status   StoryStatus
		expected string
	}{
		{
			name:     "StatusInProgress value",
			status:   StatusInProgress,
			expected: "in-progress",
		},
		{
			name:     "StatusReadyForDev value",
			status:   StatusReadyForDev,
			expected: "ready-for-dev",
		},
		{
			name:     "StatusBacklog value",
			status:   StatusBacklog,
			expected: "backlog",
		},
		{
			name:     "StatusDone value",
			status:   StatusDone,
			expected: "done",
		},
		{
			name:     "StatusBlocked value",
			status:   StatusBlocked,
			expected: "blocked",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, string(tt.status))
		})
	}
}

func TestAllSteps(t *testing.T) {
	steps := AllSteps()

	assert.Len(t, steps, 4, "AllSteps should return 4 steps")

	expectedOrder := []StepName{
		StepCreateStory,
		StepDevStory,
		StepCodeReview,
		StepGitCommit,
	}

	for i, expected := range expectedOrder {
		assert.Equal(t, expected, steps[i], "Step %d should be %s", i, expected)
	}
}

func TestStepName_Constants(t *testing.T) {
	tests := []struct {
		name     string
		step     StepName
		expected string
	}{
		{
			name:     "StepCreateStory value",
			step:     StepCreateStory,
			expected: "create-story",
		},
		{
			name:     "StepDevStory value",
			step:     StepDevStory,
			expected: "dev-story",
		},
		{
			name:     "StepCodeReview value",
			step:     StepCodeReview,
			expected: "code-review",
		},
		{
			name:     "StepGitCommit value",
			step:     StepGitCommit,
			expected: "git-commit",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, string(tt.step))
		})
	}
}

func TestStepStatus_Constants(t *testing.T) {
	tests := []struct {
		name     string
		status   StepStatus
		expected string
	}{
		{
			name:     "StepPending value",
			status:   StepPending,
			expected: "pending",
		},
		{
			name:     "StepRunning value",
			status:   StepRunning,
			expected: "running",
		},
		{
			name:     "StepSuccess value",
			status:   StepSuccess,
			expected: "success",
		},
		{
			name:     "StepFailed value",
			status:   StepFailed,
			expected: "failed",
		},
		{
			name:     "StepSkipped value",
			status:   StepSkipped,
			expected: "skipped",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, string(tt.status))
		})
	}
}

func TestStory_Fields(t *testing.T) {
	story := Story{
		Key:        "3-1-test-story",
		Epic:       3,
		Status:     StatusInProgress,
		Title:      "Test Story Title",
		FilePath:   "/path/to/story.md",
		FileExists: true,
	}

	assert.Equal(t, "3-1-test-story", story.Key)
	assert.Equal(t, 3, story.Epic)
	assert.Equal(t, StatusInProgress, story.Status)
	assert.Equal(t, "Test Story Title", story.Title)
	assert.Equal(t, "/path/to/story.md", story.FilePath)
	assert.True(t, story.FileExists)
}
