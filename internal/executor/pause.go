package executor

import (
	"sync"
	"time"
)

// QUAL-004: Executor package constants to replace magic numbers

// Timing constants
const (
	// PauseCheckInterval is the interval for checking pause state
	PauseCheckInterval = 100 * time.Millisecond

	// RetryDelayDuration is the wait time between retry attempts
	RetryDelayDuration = 2 * time.Second

	// ExecutionTickInterval is the interval for updating duration display
	ExecutionTickInterval = 1 * time.Second
)

// Buffer size constants for command output streaming
const (
	// ScannerInitialBufferSize is the initial buffer for scanning command output (64KB)
	ScannerInitialBufferSize = 64 * 1024

	// ScannerMaxBufferSize is the maximum buffer for scanning command output (1MB)
	ScannerMaxBufferSize = 1024 * 1024
)

// Parallel executor constants
const (
	// MinParallelWorkers is the minimum number of parallel workers
	MinParallelWorkers = 1

	// MaxParallelWorkers is the maximum number of parallel workers
	MaxParallelWorkers = 10

	// JobQueueBufferSize is the buffer capacity for the job queue
	JobQueueBufferSize = 100

	// ResultQueueBufferSize is the buffer capacity for the result queue
	ResultQueueBufferSize = 100
)

// PauseController manages pause/resume functionality for executors
// QUAL-003: Shared utility to eliminate duplicated waitIfPaused implementations
type PauseController struct {
	mu       sync.Mutex
	paused   bool
	canceled bool
	resumeCh chan struct{}
}

// NewPauseController creates a new PauseController
func NewPauseController() *PauseController {
	return &PauseController{
		resumeCh: make(chan struct{}),
	}
}

// Pause sets the paused state to true
func (pc *PauseController) Pause() {
	pc.mu.Lock()
	defer pc.mu.Unlock()
	pc.paused = true
}

// Resume sets the paused state to false and signals waiting goroutines
func (pc *PauseController) Resume() {
	pc.mu.Lock()
	pc.paused = false
	pc.mu.Unlock()

	// Signal to WaitIfPaused
	select {
	case pc.resumeCh <- struct{}{}:
	default:
	}
}

// Cancel sets the canceled state to true
func (pc *PauseController) Cancel() {
	pc.mu.Lock()
	defer pc.mu.Unlock()
	pc.canceled = true
}

// Reset resets the paused and canceled states
func (pc *PauseController) Reset() {
	pc.mu.Lock()
	defer pc.mu.Unlock()
	pc.paused = false
	pc.canceled = false
}

// IsPaused returns whether the controller is paused
func (pc *PauseController) IsPaused() bool {
	pc.mu.Lock()
	defer pc.mu.Unlock()
	return pc.paused
}

// IsCanceled returns whether the controller is canceled
func (pc *PauseController) IsCanceled() bool {
	pc.mu.Lock()
	defer pc.mu.Unlock()
	return pc.canceled
}

// WaitIfPaused blocks until execution is resumed or canceled.
// If cancelCh is provided, it will also unblock on that channel.
// This consolidates the waitIfPaused logic from Executor, BatchExecutor, and ParallelExecutor.
func (pc *PauseController) WaitIfPaused(cancelCh <-chan struct{}) {
	for {
		pc.mu.Lock()
		paused := pc.paused
		canceled := pc.canceled
		pc.mu.Unlock()

		if canceled || !paused {
			return
		}

		if cancelCh != nil {
			select {
			case <-pc.resumeCh:
				return
			case <-cancelCh:
				return
			case <-time.After(PauseCheckInterval):
				// Check again
			}
		} else {
			select {
			case <-pc.resumeCh:
				return
			case <-time.After(PauseCheckInterval):
				// Check again
			}
		}
	}
}
