// Package testutil provides test utilities and helpers for the bmad-automate-go tests.
package testutil

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/robertguss/bmad-automate-go/internal/config"
	"github.com/robertguss/bmad-automate-go/internal/domain"
	"github.com/robertguss/bmad-automate-go/internal/storage"
)

// NewTestConfig creates a Config with temp directories for testing.
// All temp directories are automatically cleaned up when the test completes.
func NewTestConfig(t *testing.T) *config.Config {
	t.Helper()

	tempDir := CreateTempDir(t)

	cfg := &config.Config{
		SprintStatusPath:     filepath.Join(tempDir, "sprint-status.yaml"),
		StoryDir:             filepath.Join(tempDir, "stories"),
		WorkingDir:           tempDir,
		DataDir:              filepath.Join(tempDir, "data"),
		DatabasePath:         filepath.Join(tempDir, "data", "test.db"),
		Timeout:              600,
		Retries:              1,
		Theme:                "catppuccin",
		SoundEnabled:         false,
		NotificationsEnabled: false,
		ActiveProfile:        "",
		ActiveWorkflow:       "default",
		WatchEnabled:         false,
		WatchDebounce:        500,
		MaxWorkers:           1,
		ParallelEnabled:      false,
		APIEnabled:           false,
		APIPort:              8080,
	}

	// Create necessary directories
	if err := os.MkdirAll(cfg.StoryDir, 0755); err != nil {
		t.Fatalf("failed to create story dir: %v", err)
	}
	if err := os.MkdirAll(cfg.DataDir, 0755); err != nil {
		t.Fatalf("failed to create data dir: %v", err)
	}

	return cfg
}

// NewTestStorage creates an in-memory SQLite storage for testing.
// The storage is automatically closed when the test completes.
func NewTestStorage(t *testing.T) *storage.SQLiteStorage {
	t.Helper()

	s, err := storage.NewInMemoryStorage()
	if err != nil {
		t.Fatalf("failed to create in-memory storage: %v", err)
	}

	t.Cleanup(func() {
		s.Close()
	})

	return s
}

// CreateTempDir creates a temporary directory for testing.
// The directory is automatically removed when the test completes.
func CreateTempDir(t *testing.T) string {
	t.Helper()

	dir, err := os.MkdirTemp("", "bmad-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}

	t.Cleanup(func() {
		os.RemoveAll(dir)
	})

	return dir
}

// CreateTempFile creates a temporary file with the given content.
// The file is automatically removed when the test completes.
func CreateTempFile(t *testing.T, content string) string {
	t.Helper()

	f, err := os.CreateTemp("", "bmad-test-*.yaml")
	if err != nil {
		t.Fatalf("failed to create temp file: %v", err)
	}

	if _, err := f.WriteString(content); err != nil {
		f.Close()
		t.Fatalf("failed to write temp file: %v", err)
	}

	if err := f.Close(); err != nil {
		t.Fatalf("failed to close temp file: %v", err)
	}

	t.Cleanup(func() {
		os.Remove(f.Name())
	})

	return f.Name()
}

// CreateTempFileInDir creates a file with given content in the specified directory.
func CreateTempFileInDir(t *testing.T, dir, filename, content string) string {
	t.Helper()

	path := filepath.Join(dir, filename)
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatalf("failed to write file %s: %v", path, err)
	}

	return path
}

// CreateTestStory creates a Story for testing with the given key and status.
func CreateTestStory(key string, status domain.StoryStatus) domain.Story {
	epic := 0
	// Extract epic from key if it matches pattern like "3-1-story-name"
	if len(key) >= 3 && key[1] == '-' {
		epic = int(key[0] - '0')
	}

	return domain.Story{
		Key:        key,
		Epic:       epic,
		Status:     status,
		Title:      "Test Story: " + key,
		FilePath:   "/test/stories/" + key + ".md",
		FileExists: true,
	}
}

// CreateTestStoryWithEpic creates a Story for testing with explicit epic number.
func CreateTestStoryWithEpic(key string, epic int, status domain.StoryStatus) domain.Story {
	return domain.Story{
		Key:        key,
		Epic:       epic,
		Status:     status,
		Title:      "Test Story: " + key,
		FilePath:   "/test/stories/" + key + ".md",
		FileExists: true,
	}
}

// CreateTestExecution creates an Execution for testing with the given story.
func CreateTestExecution(story domain.Story) *domain.Execution {
	return domain.NewExecution(story)
}

// CreateTestExecutionWithStatus creates an Execution with a specific status.
func CreateTestExecutionWithStatus(story domain.Story, status domain.ExecutionStatus) *domain.Execution {
	exec := domain.NewExecution(story)
	exec.Status = status
	exec.StartTime = time.Now()
	return exec
}

// CreateCompletedExecution creates a completed Execution for testing.
func CreateCompletedExecution(story domain.Story) *domain.Execution {
	exec := domain.NewExecution(story)
	exec.Status = domain.ExecutionCompleted
	exec.StartTime = time.Now().Add(-5 * time.Minute)
	exec.EndTime = time.Now()
	exec.Duration = exec.EndTime.Sub(exec.StartTime)

	// Mark all steps as successful
	for _, step := range exec.Steps {
		step.Status = domain.StepSuccess
		step.StartTime = exec.StartTime
		step.EndTime = exec.EndTime
		step.Duration = time.Minute
	}

	return exec
}

// CreateFailedExecution creates a failed Execution for testing.
func CreateFailedExecution(story domain.Story, failedStepIndex int) *domain.Execution {
	exec := domain.NewExecution(story)
	exec.Status = domain.ExecutionFailed
	exec.StartTime = time.Now().Add(-2 * time.Minute)
	exec.EndTime = time.Now()
	exec.Duration = exec.EndTime.Sub(exec.StartTime)
	exec.Error = "test failure"

	// Mark steps before failure as successful
	for i := 0; i < failedStepIndex && i < len(exec.Steps); i++ {
		exec.Steps[i].Status = domain.StepSuccess
		exec.Steps[i].Duration = 30 * time.Second
	}

	// Mark the failed step
	if failedStepIndex < len(exec.Steps) {
		exec.Steps[failedStepIndex].Status = domain.StepFailed
		exec.Steps[failedStepIndex].Error = "step failed"
		exec.Steps[failedStepIndex].Duration = 10 * time.Second
	}

	return exec
}

// CreateTestQueue creates a Queue with the given stories.
func CreateTestQueue(stories ...domain.Story) *domain.Queue {
	q := domain.NewQueue()
	for _, story := range stories {
		q.Add(story)
	}
	return q
}

// CreateQueueWithAverages creates a Queue with pre-populated step averages.
func CreateQueueWithAverages(avgDuration time.Duration, stories ...domain.Story) *domain.Queue {
	q := CreateTestQueue(stories...)
	for _, step := range domain.AllSteps() {
		q.StepAverages[step] = avgDuration
	}
	return q
}

// SprintStatusYAML helpers for common test fixtures

// ValidSprintStatusYAML returns a valid sprint-status.yaml content.
func ValidSprintStatusYAML() string {
	return `development_status:
  3-1-user-auth: in-progress
  3-2-user-profile: ready-for-dev
  4-1-dashboard: backlog
  4-2-reports: done
  5-1-api-gateway: blocked
`
}

// EmptySprintStatusYAML returns an empty sprint-status.yaml content.
func EmptySprintStatusYAML() string {
	return `development_status:
`
}

// InvalidSprintStatusYAML returns sprint-status.yaml with invalid story keys.
func InvalidSprintStatusYAML() string {
	return `development_status:
  invalid-key-format: in-progress
  : empty-key
  3-1-valid: ready-for-dev
`
}

// MalformedYAML returns malformed YAML content.
func MalformedYAML() string {
	return `development_status
  missing: colon
  - invalid: structure
`
}

// TestWorkflowYAML returns a valid workflow YAML content.
func TestWorkflowYAML() string {
	return `name: test-workflow
description: Test workflow for unit tests
version: "1.0"
steps:
  - name: create-story
    prompt_template: "Create story: {{.Story.Key}}"
  - name: dev-story
    prompt_template: "Develop story: {{.StoryPath}}"
  - name: code-review
    prompt_template: "Review story: {{.Story.Key}}"
  - name: git-commit
    prompt_template: "Commit story: {{.Story.Key}}"
`
}
