package executor

import (
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestPauseController_BasicOperations(t *testing.T) {
	pc := NewPauseController()

	// Initial state
	assert.False(t, pc.IsPaused())
	assert.False(t, pc.IsCanceled())

	// Pause
	pc.Pause()
	assert.True(t, pc.IsPaused())
	assert.False(t, pc.IsCanceled())

	// Resume
	pc.Resume()
	assert.False(t, pc.IsPaused())
	assert.False(t, pc.IsCanceled())

	// Cancel
	pc.Cancel()
	assert.True(t, pc.IsCanceled())

	// Reset
	pc.Reset()
	assert.False(t, pc.IsPaused())
	assert.False(t, pc.IsCanceled())
}

func TestPauseController_WaitIfPaused_NotPaused(t *testing.T) {
	pc := NewPauseController()

	// Should return immediately when not paused
	done := make(chan struct{})
	go func() {
		pc.WaitIfPaused(nil)
		close(done)
	}()

	select {
	case <-done:
		// Success - returned immediately
	case <-time.After(200 * time.Millisecond):
		t.Fatal("WaitIfPaused did not return immediately when not paused")
	}
}

func TestPauseController_WaitIfPaused_ResumeSignal(t *testing.T) {
	pc := NewPauseController()
	pc.Pause()

	done := make(chan struct{})
	go func() {
		pc.WaitIfPaused(nil)
		close(done)
	}()

	// Wait a bit to ensure goroutine is blocked
	time.Sleep(50 * time.Millisecond)

	// Resume should unblock
	pc.Resume()

	select {
	case <-done:
		// Success
	case <-time.After(300 * time.Millisecond):
		t.Fatal("WaitIfPaused did not unblock after Resume")
	}
}

func TestPauseController_WaitIfPaused_CancelFlag(t *testing.T) {
	pc := NewPauseController()
	pc.Pause()

	done := make(chan struct{})
	go func() {
		pc.WaitIfPaused(nil)
		close(done)
	}()

	// Wait a bit to ensure goroutine is blocked
	time.Sleep(50 * time.Millisecond)

	// Cancel should unblock
	pc.Cancel()

	select {
	case <-done:
		// Success
	case <-time.After(300 * time.Millisecond):
		t.Fatal("WaitIfPaused did not unblock after Cancel")
	}
}

func TestPauseController_WaitIfPaused_CancelChannel(t *testing.T) {
	pc := NewPauseController()
	pc.Pause()

	cancelCh := make(chan struct{})
	done := make(chan struct{})
	go func() {
		pc.WaitIfPaused(cancelCh)
		close(done)
	}()

	// Wait a bit to ensure goroutine is blocked
	time.Sleep(50 * time.Millisecond)

	// Close cancel channel should unblock
	close(cancelCh)

	select {
	case <-done:
		// Success
	case <-time.After(300 * time.Millisecond):
		t.Fatal("WaitIfPaused did not unblock when cancel channel closed")
	}
}

func TestPauseController_ConcurrentAccess(t *testing.T) {
	pc := NewPauseController()

	var wg sync.WaitGroup
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func(n int) {
			defer wg.Done()
			if n%4 == 0 {
				pc.Pause()
			} else if n%4 == 1 {
				pc.Resume()
			} else if n%4 == 2 {
				_ = pc.IsPaused()
			} else {
				_ = pc.IsCanceled()
			}
		}(i)
	}

	wg.Wait()
	// Just verify no race conditions occurred - test passes if no panic
}

func TestPauseController_MultipleResumes(t *testing.T) {
	pc := NewPauseController()
	pc.Pause()

	// Multiple resume calls should not block
	done := make(chan struct{})
	go func() {
		for i := 0; i < 5; i++ {
			pc.Resume()
		}
		close(done)
	}()

	select {
	case <-done:
		// Success
	case <-time.After(500 * time.Millisecond):
		t.Fatal("Multiple Resume calls blocked")
	}
}

func TestPauseCheckInterval(t *testing.T) {
	// Verify the constant is defined correctly
	assert.Equal(t, 100*time.Millisecond, PauseCheckInterval)
}

// QUAL-004: Tests for executor package constants
func TestExecutorConstants(t *testing.T) {
	t.Run("timing constants", func(t *testing.T) {
		assert.Equal(t, 100*time.Millisecond, PauseCheckInterval)
		assert.Equal(t, 2*time.Second, RetryDelayDuration)
		assert.Equal(t, 1*time.Second, ExecutionTickInterval)
	})

	t.Run("buffer size constants", func(t *testing.T) {
		assert.Equal(t, 64*1024, ScannerInitialBufferSize)
		assert.Equal(t, 1024*1024, ScannerMaxBufferSize)
	})

	t.Run("parallel worker constants", func(t *testing.T) {
		assert.Equal(t, 1, MinParallelWorkers)
		assert.Equal(t, 10, MaxParallelWorkers)
		assert.Equal(t, 100, JobQueueBufferSize)
		assert.Equal(t, 100, ResultQueueBufferSize)
	})
}
