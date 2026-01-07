package executor

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/robertguss/bmad-automate-go/internal/config"
	"github.com/robertguss/bmad-automate-go/internal/domain"
)

func createTestConfig() *config.Config {
	return &config.Config{
		Timeout:  600,
		Retries:  1,
		StoryDir: "/test/stories",
	}
}

func createTestStory() domain.Story {
	return domain.Story{
		Key:        "3-1-test-story",
		Epic:       3,
		Status:     domain.StatusInProgress,
		Title:      "Test Story",
		FilePath:   "/test/stories/3-1-test-story.md",
		FileExists: false,
	}
}

func TestNew(t *testing.T) {
	cfg := createTestConfig()
	e := New(cfg)

	require.NotNil(t, e)
	assert.NotNil(t, e.config)
	assert.NotNil(t, e.pauseCh)
	assert.NotNil(t, e.resumeCh)
	assert.NotNil(t, e.cancelCh)
	assert.NotNil(t, e.skipCh)
	assert.False(t, e.paused)
	assert.False(t, e.canceled)
}

func TestExecutor_SetProgram(t *testing.T) {
	cfg := createTestConfig()
	e := New(cfg)

	// SetProgram should work without panicking
	e.SetProgram(nil)
	assert.Nil(t, e.program)
}

func TestExecutor_BuildCommand(t *testing.T) {
	cfg := createTestConfig()
	e := New(cfg)
	e.execution = domain.NewExecution(createTestStory())

	tests := []struct {
		name     string
		stepName domain.StepName
		contains string
	}{
		{
			name:     "create-story command",
			stepName: domain.StepCreateStory,
			contains: "create-story",
		},
		{
			name:     "dev-story command",
			stepName: domain.StepDevStory,
			contains: "dev-story",
		},
		{
			name:     "code-review command",
			stepName: domain.StepCodeReview,
			contains: "code-review",
		},
		{
			name:     "git-commit command",
			stepName: domain.StepGitCommit,
			contains: "Commit",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := e.buildCommand(tt.stepName, e.execution.Story)
			assert.Contains(t, cmd, tt.contains)
			assert.Contains(t, cmd, "claude")
		})
	}

	t.Run("unknown step returns empty", func(t *testing.T) {
		cmd := e.buildCommand("unknown-step", e.execution.Story)
		assert.Empty(t, cmd)
	})

	t.Run("includes story key", func(t *testing.T) {
		cmd := e.buildCommand(domain.StepCreateStory, e.execution.Story)
		assert.Contains(t, cmd, "3-1-test-story")
	})
}

func TestExecutor_Pause(t *testing.T) {
	cfg := createTestConfig()
	e := New(cfg)

	t.Run("pause without execution does nothing", func(t *testing.T) {
		e.Pause()
		assert.False(t, e.paused)
	})

	t.Run("pause with execution sets paused state", func(t *testing.T) {
		e.execution = domain.NewExecution(createTestStory())
		e.execution.Status = domain.ExecutionRunning

		e.Pause()

		assert.True(t, e.paused)
		assert.Equal(t, domain.ExecutionPaused, e.execution.Status)
	})

	t.Run("double pause does not change state", func(t *testing.T) {
		e.execution = domain.NewExecution(createTestStory())
		e.execution.Status = domain.ExecutionRunning

		e.Pause()
		e.Pause()

		assert.True(t, e.paused)
	})
}

func TestExecutor_Resume(t *testing.T) {
	cfg := createTestConfig()
	e := New(cfg)
	e.execution = domain.NewExecution(createTestStory())
	e.execution.Status = domain.ExecutionRunning

	t.Run("resume when not paused does nothing", func(t *testing.T) {
		e.paused = false
		e.Resume()
		assert.False(t, e.paused)
	})

	t.Run("resume when paused clears paused state", func(t *testing.T) {
		e.paused = true
		e.execution.Status = domain.ExecutionPaused

		e.Resume()

		assert.False(t, e.paused)
		assert.Equal(t, domain.ExecutionRunning, e.execution.Status)
	})
}

func TestExecutor_Cancel(t *testing.T) {
	cfg := createTestConfig()
	e := New(cfg)

	t.Run("cancel sets canceled state", func(t *testing.T) {
		e.Cancel()
		assert.True(t, e.canceled)
	})

	t.Run("cancel calls context cancel if set", func(t *testing.T) {
		// This just verifies it doesn't panic when cancel is nil
		e.canceled = false
		e.cancel = nil
		e.Cancel()
		assert.True(t, e.canceled)
	})
}

func TestExecutor_Skip(t *testing.T) {
	cfg := createTestConfig()
	e := New(cfg)

	t.Run("skip sends to channel without blocking", func(t *testing.T) {
		// This should not block even if no one is receiving
		done := make(chan bool)
		go func() {
			e.Skip()
			done <- true
		}()

		select {
		case <-done:
			// Success
		case <-time.After(100 * time.Millisecond):
			t.Fatal("Skip blocked")
		}
	})
}

func TestExecutor_IsPaused(t *testing.T) {
	cfg := createTestConfig()
	e := New(cfg)

	t.Run("returns false when not paused", func(t *testing.T) {
		e.paused = false
		assert.False(t, e.IsPaused())
	})

	t.Run("returns true when paused", func(t *testing.T) {
		e.paused = true
		assert.True(t, e.IsPaused())
	})
}

func TestExecutor_GetExecution(t *testing.T) {
	cfg := createTestConfig()
	e := New(cfg)

	t.Run("returns nil when no execution", func(t *testing.T) {
		assert.Nil(t, e.GetExecution())
	})

	t.Run("returns execution when set", func(t *testing.T) {
		exec := domain.NewExecution(createTestStory())
		e.execution = exec

		result := e.GetExecution()
		require.NotNil(t, result)
		assert.Equal(t, exec.Story.Key, result.Story.Key)
	})
}

func TestExecutor_WaitIfPaused(t *testing.T) {
	cfg := createTestConfig()
	e := New(cfg)

	t.Run("returns immediately when not paused", func(t *testing.T) {
		e.paused = false

		done := make(chan bool)
		go func() {
			e.waitIfPaused()
			done <- true
		}()

		select {
		case <-done:
			// Success
		case <-time.After(200 * time.Millisecond):
			t.Fatal("waitIfPaused blocked when not paused")
		}
	})

	t.Run("returns immediately when canceled", func(t *testing.T) {
		e.paused = true
		e.canceled = true

		done := make(chan bool)
		go func() {
			e.waitIfPaused()
			done <- true
		}()

		select {
		case <-done:
			// Success
		case <-time.After(200 * time.Millisecond):
			t.Fatal("waitIfPaused blocked when canceled")
		}
	})

	t.Run("returns when resumed", func(t *testing.T) {
		e.paused = true
		e.canceled = false

		done := make(chan bool)
		go func() {
			e.waitIfPaused()
			done <- true
		}()

		// Give it time to start waiting
		time.Sleep(50 * time.Millisecond)

		// Resume
		e.mu.Lock()
		e.paused = false
		e.mu.Unlock()
		e.resumeCh <- struct{}{}

		select {
		case <-done:
			// Success
		case <-time.After(500 * time.Millisecond):
			t.Fatal("waitIfPaused didn't return after resume")
		}
	})
}

func TestExecutor_SendMsg(t *testing.T) {
	cfg := createTestConfig()
	e := New(cfg)

	t.Run("does not panic when program is nil", func(t *testing.T) {
		e.program = nil
		// Should not panic
		e.sendMsg(nil)
	})
}

func TestExecutor_Concurrency(t *testing.T) {
	cfg := createTestConfig()
	e := New(cfg)
	e.execution = domain.NewExecution(createTestStory())
	e.execution.Status = domain.ExecutionRunning

	// Test that concurrent access doesn't cause races
	t.Run("concurrent pause/resume", func(t *testing.T) {
		done := make(chan bool)

		go func() {
			for i := 0; i < 100; i++ {
				e.Pause()
				e.Resume()
			}
			done <- true
		}()

		go func() {
			for i := 0; i < 100; i++ {
				_ = e.IsPaused()
				_ = e.GetExecution()
			}
			done <- true
		}()

		<-done
		<-done
	})
}
