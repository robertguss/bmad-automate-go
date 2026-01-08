package executor

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/robertguss/bmad-automate-go/internal/config"
	"github.com/robertguss/bmad-automate-go/internal/domain"
)

func TestNewBatchExecutor(t *testing.T) {
	cfg := &config.Config{
		Timeout: 600,
		Retries: 1,
	}

	b := NewBatchExecutor(cfg)

	require.NotNil(t, b)
	assert.NotNil(t, b.config)
	assert.NotNil(t, b.queue)
	assert.NotNil(t, b.executor)
	assert.NotNil(t, b.pauseCtrl)
	assert.False(t, b.pauseCtrl.IsPaused())
	assert.False(t, b.pauseCtrl.IsCanceled())
	assert.False(t, b.running)
}

func TestBatchExecutor_SetProgram(t *testing.T) {
	cfg := &config.Config{}
	b := NewBatchExecutor(cfg)

	// Should not panic
	b.SetProgram(nil)
	assert.Nil(t, b.program)
}

func TestBatchExecutor_GetQueue(t *testing.T) {
	cfg := &config.Config{}
	b := NewBatchExecutor(cfg)

	q := b.GetQueue()
	require.NotNil(t, q)
	assert.Equal(t, 0, q.TotalCount())
}

func TestBatchExecutor_SetQueue(t *testing.T) {
	cfg := &config.Config{}
	b := NewBatchExecutor(cfg)

	newQueue := domain.NewQueue()
	newQueue.Add(domain.Story{Key: "3-1-test"})

	b.SetQueue(newQueue)

	q := b.GetQueue()
	assert.Equal(t, 1, q.TotalCount())
}

func TestBatchExecutor_AddToQueue(t *testing.T) {
	cfg := &config.Config{}
	b := NewBatchExecutor(cfg)

	stories := []domain.Story{
		{Key: "3-1-first", Status: domain.StatusInProgress},
		{Key: "3-2-second", Status: domain.StatusReadyForDev},
	}

	b.AddToQueue(stories)

	q := b.GetQueue()
	assert.Equal(t, 2, q.TotalCount())
}

func TestBatchExecutor_RemoveFromQueue(t *testing.T) {
	cfg := &config.Config{}
	b := NewBatchExecutor(cfg)

	b.queue.Add(domain.Story{Key: "3-1-first"})
	b.queue.Add(domain.Story{Key: "3-2-second"})

	t.Run("removes existing item", func(t *testing.T) {
		result := b.RemoveFromQueue("3-1-first")
		assert.True(t, result)
		assert.Equal(t, 1, b.GetQueue().TotalCount())
	})

	t.Run("returns false for non-existent item", func(t *testing.T) {
		result := b.RemoveFromQueue("non-existent")
		assert.False(t, result)
	})
}

func TestBatchExecutor_ClearQueue(t *testing.T) {
	cfg := &config.Config{}
	b := NewBatchExecutor(cfg)

	b.queue.Add(domain.Story{Key: "3-1-first"})
	b.queue.Add(domain.Story{Key: "3-2-second"})

	b.ClearQueue()

	assert.Equal(t, 0, b.GetQueue().TotalCount())
}

func TestBatchExecutor_MoveUp(t *testing.T) {
	cfg := &config.Config{}
	b := NewBatchExecutor(cfg)

	b.queue.Add(domain.Story{Key: "3-1-first"})
	b.queue.Add(domain.Story{Key: "3-2-second"})

	t.Run("moves item up", func(t *testing.T) {
		result := b.MoveUp(1)
		assert.True(t, result)
		assert.Equal(t, "3-2-second", b.GetQueue().Items[0].Story.Key)
	})

	t.Run("returns false for invalid index", func(t *testing.T) {
		result := b.MoveUp(0)
		assert.False(t, result)
	})
}

func TestBatchExecutor_MoveDown(t *testing.T) {
	cfg := &config.Config{}
	b := NewBatchExecutor(cfg)

	b.queue.Add(domain.Story{Key: "3-1-first"})
	b.queue.Add(domain.Story{Key: "3-2-second"})

	t.Run("moves item down", func(t *testing.T) {
		result := b.MoveDown(0)
		assert.True(t, result)
		assert.Equal(t, "3-2-second", b.GetQueue().Items[0].Story.Key)
	})

	t.Run("returns false for invalid index", func(t *testing.T) {
		result := b.MoveDown(1)
		assert.False(t, result)
	})
}

func TestBatchExecutor_IsPaused(t *testing.T) {
	cfg := &config.Config{}
	b := NewBatchExecutor(cfg)

	assert.False(t, b.IsPaused())

	b.pauseCtrl.Pause()
	assert.True(t, b.IsPaused())
}

func TestBatchExecutor_IsRunning(t *testing.T) {
	cfg := &config.Config{}
	b := NewBatchExecutor(cfg)

	assert.False(t, b.IsRunning())

	b.running = true
	assert.True(t, b.IsRunning())
}

func TestBatchExecutor_GetExecutor(t *testing.T) {
	cfg := &config.Config{}
	b := NewBatchExecutor(cfg)

	exec := b.GetExecutor()
	require.NotNil(t, exec)
}

func TestBatchExecutor_Pause(t *testing.T) {
	cfg := &config.Config{}
	b := NewBatchExecutor(cfg)
	b.running = true

	b.Pause()

	assert.True(t, b.pauseCtrl.IsPaused())
	assert.Equal(t, domain.QueuePaused, b.queue.Status)
}

func TestBatchExecutor_Resume(t *testing.T) {
	cfg := &config.Config{}
	b := NewBatchExecutor(cfg)
	b.running = true
	b.pauseCtrl.Pause()
	b.queue.Status = domain.QueuePaused

	b.Resume()

	assert.False(t, b.pauseCtrl.IsPaused())
	assert.Equal(t, domain.QueueRunning, b.queue.Status)
}

func TestBatchExecutor_Cancel(t *testing.T) {
	cfg := &config.Config{}
	b := NewBatchExecutor(cfg)

	b.Cancel()

	assert.True(t, b.pauseCtrl.IsCanceled())
}

func TestBatchExecutor_Concurrency(t *testing.T) {
	cfg := &config.Config{}
	b := NewBatchExecutor(cfg)

	// Test concurrent access doesn't cause races
	done := make(chan bool)

	go func() {
		for i := 0; i < 50; i++ {
			b.Pause()
			b.Resume()
		}
		done <- true
	}()

	go func() {
		for i := 0; i < 50; i++ {
			_ = b.IsPaused()
			_ = b.IsRunning()
			_ = b.GetQueue()
		}
		done <- true
	}()

	<-done
	<-done
}
