package executor

import (
	"context"
	"fmt"
	"os"
	"runtime/debug"
	"sync"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/robertguss/bmad-automate-go/internal/config"
	"github.com/robertguss/bmad-automate-go/internal/domain"
	"github.com/robertguss/bmad-automate-go/internal/messages"
)

// BatchExecutor manages sequential execution of multiple stories
type BatchExecutor struct {
	config  *config.Config
	program *tea.Program
	queue   *domain.Queue

	// Pause/resume/cancel control (QUAL-003: shared utility)
	pauseCtrl *PauseController

	// State
	mu      sync.Mutex
	running bool
	ctx     context.Context
	cancel  context.CancelFunc

	// Child executor for individual stories
	executor *Executor
}

// NewBatchExecutor creates a new BatchExecutor
func NewBatchExecutor(cfg *config.Config) *BatchExecutor {
	return &BatchExecutor{
		config:    cfg,
		queue:     domain.NewQueue(),
		pauseCtrl: NewPauseController(),
		executor:  New(cfg),
	}
}

// SetProgram sets the tea.Program for sending messages
func (b *BatchExecutor) SetProgram(p *tea.Program) {
	b.program = p
	b.executor.SetProgram(p)
}

// GetQueue returns the current queue
func (b *BatchExecutor) GetQueue() *domain.Queue {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.queue
}

// SetQueue sets the queue
func (b *BatchExecutor) SetQueue(q *domain.Queue) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.queue = q
}

// AddToQueue adds stories to the queue
func (b *BatchExecutor) AddToQueue(stories []domain.Story) {
	b.mu.Lock()
	b.queue.AddMultiple(stories)
	b.mu.Unlock()
	// Don't send message here - caller updates UI directly
	// Sending here would deadlock since tea.Program.Send blocks
	// while the program is still in Update processing the keypress
}

// RemoveFromQueue removes a story from the queue
func (b *BatchExecutor) RemoveFromQueue(key string) bool {
	b.mu.Lock()
	result := b.queue.Remove(key)
	queue := b.queue
	b.mu.Unlock()
	b.sendMsg(messages.QueueUpdatedMsg{Queue: queue})
	return result
}

// ClearQueue clears all pending items
func (b *BatchExecutor) ClearQueue() {
	b.mu.Lock()
	b.queue.Clear()
	queue := b.queue
	b.mu.Unlock()
	b.sendMsg(messages.QueueUpdatedMsg{Queue: queue})
}

// MoveUp moves an item up in the queue
func (b *BatchExecutor) MoveUp(index int) bool {
	b.mu.Lock()
	result := b.queue.MoveUp(index)
	queue := b.queue
	b.mu.Unlock()
	if result {
		b.sendMsg(messages.QueueUpdatedMsg{Queue: queue})
	}
	return result
}

// MoveDown moves an item down in the queue
func (b *BatchExecutor) MoveDown(index int) bool {
	b.mu.Lock()
	result := b.queue.MoveDown(index)
	queue := b.queue
	b.mu.Unlock()
	if result {
		b.sendMsg(messages.QueueUpdatedMsg{Queue: queue})
	}
	return result
}

// Start begins batch execution of the queue
func (b *BatchExecutor) Start() tea.Cmd {
	return func() tea.Msg {
		// Capture panic for debugging
		defer func() {
			if r := recover(); r != nil {
				// Log stack trace to file before re-panicking
				if f, err := os.Create("bmad-batch-panic.log"); err == nil {
					fmt.Fprintf(f, "Panic in batch executor: %v\n\nStack trace:\n%s", r, debug.Stack())
					f.Close()
				}
				panic(r)
			}
		}()

		b.mu.Lock()
		if b.running || !b.queue.HasPending() {
			b.mu.Unlock()
			return nil
		}

		b.running = true
		b.pauseCtrl.Reset()
		b.queue.Status = domain.QueueRunning
		b.queue.StartTime = time.Now()
		b.ctx, b.cancel = context.WithCancel(context.Background())
		b.mu.Unlock()

		b.sendMsg(messages.QueueUpdatedMsg{Queue: b.queue})

		// Process each pending item
		for {
			if b.pauseCtrl.IsCanceled() {
				b.mu.Lock()
				b.queue.Status = domain.QueueIdle
				b.running = false
				b.mu.Unlock()
				break
			}

			// Find next pending item
			b.mu.Lock()
			var nextItem *domain.QueueItem
			var nextIndex int = -1
			for i, item := range b.queue.Items {
				if item.Status == domain.ExecutionPending {
					nextItem = item
					nextIndex = i
					break
				}
			}

			if nextItem == nil {
				// No more pending items
				b.queue.Status = domain.QueueCompleted
				b.queue.EndTime = time.Now()
				b.running = false
				b.mu.Unlock()
				break
			}

			b.queue.Current = nextIndex
			b.mu.Unlock()

			// Wait if paused (QUAL-003: using shared utility)
			b.pauseCtrl.WaitIfPaused(nil)

			// Check if cancelled during pause
			if b.pauseCtrl.IsCanceled() {
				b.mu.Lock()
				b.queue.Status = domain.QueueIdle
				b.running = false
				b.mu.Unlock()
				break
			}

			// Execute the story
			b.executeItem(nextIndex, nextItem)
		}

		// Calculate final stats
		b.mu.Lock()
		queue := b.queue
		b.mu.Unlock()

		return messages.QueueCompletedMsg{
			TotalItems:    queue.TotalCount(),
			SuccessCount:  queue.CompletedCount(),
			FailedCount:   queue.FailedCount(),
			TotalDuration: time.Since(queue.StartTime),
		}
	}
}

// executeItem executes a single queue item
func (b *BatchExecutor) executeItem(index int, item *domain.QueueItem) {
	// Create execution for this item
	execution := domain.NewExecution(item.Story)
	execution.Status = domain.ExecutionRunning
	execution.StartTime = time.Now()

	b.mu.Lock()
	item.Status = domain.ExecutionRunning
	item.Execution = execution
	b.mu.Unlock()

	// Send item started message
	b.sendMsg(messages.QueueItemStartedMsg{
		Index:     index,
		Story:     item.Story,
		Execution: execution,
	})

	// Also send ExecutionStartedMsg for the execution view
	b.sendMsg(messages.ExecutionStartedMsg{Execution: execution})

	// Execute each step
	for i, step := range execution.Steps {
		if b.pauseCtrl.IsCanceled() {
			execution.Status = domain.ExecutionCancelled
			break
		}

		// Wait if paused (QUAL-003: using shared utility)
		b.pauseCtrl.WaitIfPaused(nil)

		// Check for cancellation after pause
		if b.pauseCtrl.IsCanceled() {
			execution.Status = domain.ExecutionCancelled
			break
		}

		// Auto-skip create-story if file exists
		if step.Name == domain.StepCreateStory && item.Story.FileExists {
			step.Status = domain.StepSkipped
			b.sendMsg(messages.StepCompletedMsg{
				StepIndex: i,
				Status:    domain.StepSkipped,
			})
			continue
		}

		// Execute the step
		execution.Current = i
		err := b.executor.executeStep(i, step)

		if err != nil && step.Status == domain.StepFailed {
			execution.Status = domain.ExecutionFailed
			execution.Error = err.Error()
			break
		}

		// Update step averages for ETA calculation
		if step.Status == domain.StepSuccess && step.Duration > 0 {
			b.mu.Lock()
			b.queue.UpdateStepAverage(step.Name, step.Duration)
			b.mu.Unlock()
		}
	}

	// Mark completion
	execution.EndTime = time.Now()
	execution.Duration = execution.EndTime.Sub(execution.StartTime)

	if execution.Status == domain.ExecutionRunning {
		execution.Status = domain.ExecutionCompleted
	}

	b.mu.Lock()
	item.Status = execution.Status
	b.mu.Unlock()

	// Send completion messages
	b.sendMsg(messages.ExecutionCompletedMsg{
		Status:   execution.Status,
		Duration: execution.Duration,
		Error:    execution.Error,
	})

	b.sendMsg(messages.QueueItemCompletedMsg{
		Index:     index,
		Story:     item.Story,
		Status:    execution.Status,
		Duration:  execution.Duration,
		Error:     execution.Error,
		Execution: execution,
	})
}

// Pause pauses the batch execution
func (b *BatchExecutor) Pause() {
	b.mu.Lock()
	defer b.mu.Unlock()
	if !b.pauseCtrl.IsPaused() && b.running {
		b.pauseCtrl.Pause()
		b.queue.Status = domain.QueuePaused
		// Also pause the individual executor
		b.executor.Pause()
	}
}

// Resume resumes a paused batch execution
func (b *BatchExecutor) Resume() {
	b.mu.Lock()
	if b.pauseCtrl.IsPaused() {
		b.queue.Status = domain.QueueRunning
		// Also resume the individual executor
		b.executor.Resume()
	}
	b.mu.Unlock()

	// Resume will signal to WaitIfPaused
	b.pauseCtrl.Resume()
}

// Cancel cancels the batch execution
func (b *BatchExecutor) Cancel() {
	b.mu.Lock()
	if b.cancel != nil {
		b.cancel()
	}
	// Also cancel the individual executor
	b.executor.Cancel()
	b.mu.Unlock()
	b.pauseCtrl.Cancel()
}

// Skip skips the current step in the current item
func (b *BatchExecutor) Skip() {
	b.executor.Skip()
}

// IsPaused returns true if batch execution is paused
func (b *BatchExecutor) IsPaused() bool {
	return b.pauseCtrl.IsPaused()
}

// IsRunning returns true if batch execution is running
func (b *BatchExecutor) IsRunning() bool {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.running
}

// GetCurrentExecution returns the execution for the current item
func (b *BatchExecutor) GetCurrentExecution() *domain.Execution {
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.queue.Current >= 0 && b.queue.Current < len(b.queue.Items) {
		return b.queue.Items[b.queue.Current].Execution
	}
	return nil
}

// GetExecutor returns the underlying single-story executor
func (b *BatchExecutor) GetExecutor() *Executor {
	return b.executor
}

// sendMsg safely sends a message to the tea.Program
func (b *BatchExecutor) sendMsg(msg tea.Msg) {
	if b.program != nil {
		b.program.Send(msg)
	}
}
