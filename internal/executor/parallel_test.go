package executor

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/robertguss/bmad-automate-go/internal/config"
	"github.com/robertguss/bmad-automate-go/internal/domain"
)

func TestNewParallelExecutor(t *testing.T) {
	cfg := &config.Config{
		Timeout: 600,
		Retries: 1,
	}

	t.Run("creates with valid workers", func(t *testing.T) {
		p := NewParallelExecutor(cfg, 3)
		require.NotNil(t, p)
		assert.Equal(t, 3, p.workers)
	})

	t.Run("caps workers at 10", func(t *testing.T) {
		p := NewParallelExecutor(cfg, 20)
		assert.Equal(t, 10, p.workers)
	})

	t.Run("minimum workers is 1", func(t *testing.T) {
		p := NewParallelExecutor(cfg, 0)
		assert.Equal(t, 1, p.workers)
	})

	t.Run("minimum workers for negative", func(t *testing.T) {
		p := NewParallelExecutor(cfg, -5)
		assert.Equal(t, 1, p.workers)
	})

	t.Run("initializes all fields", func(t *testing.T) {
		p := NewParallelExecutor(cfg, 3)
		assert.NotNil(t, p.config)
		assert.NotNil(t, p.jobQueue)
		assert.NotNil(t, p.resultQueue)
		assert.NotNil(t, p.activeJobs)
		assert.NotNil(t, p.pauseCtrl)
		assert.False(t, p.running)
		assert.False(t, p.pauseCtrl.IsPaused())
	})
}

func TestParallelExecutor_SetProgram(t *testing.T) {
	cfg := &config.Config{}
	p := NewParallelExecutor(cfg, 2)

	// Should not panic
	p.SetProgram(nil)
	assert.Nil(t, p.program)
}

func TestParallelExecutor_SetWorkers(t *testing.T) {
	cfg := &config.Config{}
	p := NewParallelExecutor(cfg, 2)

	t.Run("sets valid worker count", func(t *testing.T) {
		p.SetWorkers(5)
		assert.Equal(t, 5, p.workers)
	})

	t.Run("caps at 10", func(t *testing.T) {
		p.SetWorkers(15)
		assert.Equal(t, 10, p.workers)
	})

	t.Run("minimum is 1", func(t *testing.T) {
		p.SetWorkers(0)
		assert.Equal(t, 1, p.workers)
	})
}

func TestParallelExecutor_GetWorkers(t *testing.T) {
	cfg := &config.Config{}
	p := NewParallelExecutor(cfg, 5)

	assert.Equal(t, 5, p.GetWorkers())
}

func TestParallelExecutor_GetProgress(t *testing.T) {
	cfg := &config.Config{}
	p := NewParallelExecutor(cfg, 2)

	p.completed = 3
	p.failed = 1
	p.total = 10

	completed, failed, total := p.GetProgress()

	assert.Equal(t, 3, completed)
	assert.Equal(t, 1, failed)
	assert.Equal(t, 10, total)
}

func TestParallelExecutor_GetActiveJobs(t *testing.T) {
	cfg := &config.Config{}
	p := NewParallelExecutor(cfg, 2)

	// Initially empty
	assert.Equal(t, 0, p.GetActiveJobs())

	// Add some jobs
	p.activeJobs["job1"] = &parallelJob{}
	p.activeJobs["job2"] = &parallelJob{}

	assert.Equal(t, 2, p.GetActiveJobs())
}

func TestParallelExecutor_IsPaused(t *testing.T) {
	cfg := &config.Config{}
	p := NewParallelExecutor(cfg, 2)

	assert.False(t, p.IsPaused())

	p.pauseCtrl.Pause()
	assert.True(t, p.IsPaused())
}

func TestParallelExecutor_IsRunning(t *testing.T) {
	cfg := &config.Config{}
	p := NewParallelExecutor(cfg, 2)

	assert.False(t, p.IsRunning())

	p.running = true
	assert.True(t, p.IsRunning())
}

func TestParallelExecutor_Pause(t *testing.T) {
	cfg := &config.Config{}
	p := NewParallelExecutor(cfg, 2)

	p.Pause()

	assert.True(t, p.pauseCtrl.IsPaused())
}

func TestParallelExecutor_Resume(t *testing.T) {
	cfg := &config.Config{}
	p := NewParallelExecutor(cfg, 2)
	p.pauseCtrl.Pause()

	p.Resume()

	assert.False(t, p.pauseCtrl.IsPaused())
}

func TestParallelExecutor_Cancel(t *testing.T) {
	cfg := &config.Config{}
	p := NewParallelExecutor(cfg, 2)

	// Should not panic even with nil context
	p.Cancel()
}

func TestParallelExecutor_Concurrency(t *testing.T) {
	cfg := &config.Config{}
	p := NewParallelExecutor(cfg, 2)

	// Test concurrent access doesn't cause races
	done := make(chan bool)

	go func() {
		for i := 0; i < 50; i++ {
			p.Pause()
			p.Resume()
		}
		done <- true
	}()

	go func() {
		for i := 0; i < 50; i++ {
			_ = p.IsPaused()
			_ = p.IsRunning()
			_, _, _ = p.GetProgress()
			_ = p.GetActiveJobs()
		}
		done <- true
	}()

	<-done
	<-done
}

func TestParallelExecutor_CancelWithContext(t *testing.T) {
	cfg := &config.Config{}
	p := NewParallelExecutor(cfg, 2)

	t.Run("cancel with nil context does not panic", func(t *testing.T) {
		p.cancel = nil
		// ParallelExecutor.Cancel only calls cancel func, doesn't set pauseCtrl
		p.Cancel()
		// Should not panic, but pauseCtrl is not affected
	})

	t.Run("cancel with valid context calls cancel func", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		p.cancel = cancel

		// Verify context is not yet done
		select {
		case <-ctx.Done():
			t.Fatal("context should not be done yet")
		default:
		}

		p.Cancel()

		// Now context should be canceled
		select {
		case <-ctx.Done():
			// Success
		default:
			t.Fatal("context should be done after Cancel")
		}
	})
}

func TestParallelExecutor_ProgressTracking(t *testing.T) {
	cfg := &config.Config{}
	p := NewParallelExecutor(cfg, 2)

	t.Run("initial progress is zero", func(t *testing.T) {
		completed, failed, total := p.GetProgress()
		assert.Equal(t, 0, completed)
		assert.Equal(t, 0, failed)
		assert.Equal(t, 0, total)
	})

	t.Run("progress updates correctly", func(t *testing.T) {
		p.completed = 5
		p.failed = 2
		p.total = 10

		completed, failed, total := p.GetProgress()
		assert.Equal(t, 5, completed)
		assert.Equal(t, 2, failed)
		assert.Equal(t, 10, total)
	})
}

func TestParallelExecutor_ActiveJobsTracking(t *testing.T) {
	cfg := &config.Config{}
	p := NewParallelExecutor(cfg, 2)

	t.Run("active jobs count updates", func(t *testing.T) {
		assert.Equal(t, 0, p.GetActiveJobs())

		// Add jobs
		p.mu.Lock()
		p.activeJobs["job1"] = &parallelJob{story: domain.Story{Key: "1-1-test"}}
		p.activeJobs["job2"] = &parallelJob{story: domain.Story{Key: "1-2-test"}}
		p.activeJobs["job3"] = &parallelJob{story: domain.Story{Key: "1-3-test"}}
		p.mu.Unlock()

		assert.Equal(t, 3, p.GetActiveJobs())

		// Remove a job
		p.mu.Lock()
		delete(p.activeJobs, "job2")
		p.mu.Unlock()

		assert.Equal(t, 2, p.GetActiveJobs())
	})
}

func TestParallelExecutor_SendMsg(t *testing.T) {
	cfg := &config.Config{}
	p := NewParallelExecutor(cfg, 2)

	t.Run("does not panic with nil program", func(t *testing.T) {
		p.program = nil
		p.sendMsg(nil)
	})
}

func TestParallelExecutor_WorkerLimits(t *testing.T) {
	cfg := &config.Config{}

	testCases := []struct {
		name     string
		input    int
		expected int
	}{
		{"negative becomes 1", -10, 1},
		{"zero becomes 1", 0, 1},
		{"1 stays 1", 1, 1},
		{"5 stays 5", 5, 5},
		{"10 stays 10", 10, 10},
		{"11 becomes 10", 11, 10},
		{"100 becomes 10", 100, 10},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			p := NewParallelExecutor(cfg, tc.input)
			assert.Equal(t, tc.expected, p.GetWorkers())
		})
	}
}

func TestParallelExecutor_SetWorkersLimits(t *testing.T) {
	cfg := &config.Config{}
	p := NewParallelExecutor(cfg, 2)

	testCases := []struct {
		name     string
		input    int
		expected int
	}{
		{"negative becomes 1", -5, 1},
		{"zero becomes 1", 0, 1},
		{"15 becomes 10", 15, 10},
		{"10 stays 10", 10, 10},
		{"3 stays 3", 3, 3},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			p.SetWorkers(tc.input)
			assert.Equal(t, tc.expected, p.GetWorkers())
		})
	}
}
