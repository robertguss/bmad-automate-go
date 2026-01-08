package executor

import (
	"context"
	"fmt"
	"sync"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/robertguss/bmad-automate-go/internal/config"
	"github.com/robertguss/bmad-automate-go/internal/domain"
	"github.com/robertguss/bmad-automate-go/internal/messages"
)

// ParallelExecutor manages parallel execution of multiple stories
type ParallelExecutor struct {
	config  *config.Config
	program *tea.Program
	workers int

	// Job management
	jobQueue    chan *parallelJob
	resultQueue chan *parallelResult
	activeJobs  map[string]*parallelJob

	// Control
	mu       sync.Mutex
	ctx      context.Context
	cancel   context.CancelFunc
	running  bool
	paused   bool
	pauseCh  chan struct{}
	resumeCh chan struct{}

	// Statistics
	completed int
	failed    int
	total     int
	startTime time.Time
}

// parallelJob represents a job to be executed
type parallelJob struct {
	index     int
	story     domain.Story
	execution *domain.Execution
}

// parallelResult represents the result of a job
type parallelResult struct {
	index     int
	story     domain.Story
	status    domain.ExecutionStatus
	duration  time.Duration
	error     string
	execution *domain.Execution
}

// NewParallelExecutor creates a new parallel executor
func NewParallelExecutor(cfg *config.Config, workers int) *ParallelExecutor {
	if workers < 1 {
		workers = 1
	}
	if workers > 10 {
		workers = 10 // Cap at 10 workers
	}

	return &ParallelExecutor{
		config:      cfg,
		workers:     workers,
		jobQueue:    make(chan *parallelJob, 100),
		resultQueue: make(chan *parallelResult, 100),
		activeJobs:  make(map[string]*parallelJob),
		pauseCh:     make(chan struct{}),
		resumeCh:    make(chan struct{}),
	}
}

// SetProgram sets the tea.Program for sending messages
func (p *ParallelExecutor) SetProgram(prog *tea.Program) {
	p.program = prog
}

// SetWorkers sets the number of parallel workers
func (p *ParallelExecutor) SetWorkers(n int) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if n < 1 {
		n = 1
	}
	if n > 10 {
		n = 10
	}
	p.workers = n
}

// GetWorkers returns the current number of workers
func (p *ParallelExecutor) GetWorkers() int {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.workers
}

// Execute starts parallel execution of stories
func (p *ParallelExecutor) Execute(stories []domain.Story) tea.Cmd {
	return func() tea.Msg {
		p.mu.Lock()
		p.ctx, p.cancel = context.WithCancel(context.Background())
		p.running = true
		p.paused = false
		p.total = len(stories)
		p.completed = 0
		p.failed = 0
		p.startTime = time.Now()
		p.activeJobs = make(map[string]*parallelJob)
		p.mu.Unlock()

		// Start worker pool
		var wg sync.WaitGroup
		for i := 0; i < p.workers; i++ {
			wg.Add(1)
			go p.worker(i, &wg)
		}

		// Start result collector
		go p.collectResults()

		// Queue all jobs
		for i, story := range stories {
			job := &parallelJob{
				index:     i,
				story:     story,
				execution: domain.NewExecution(story),
			}

			p.mu.Lock()
			p.activeJobs[story.Key] = job
			p.mu.Unlock()

			p.sendMsg(messages.QueueItemStartedMsg{
				Index:     i,
				Story:     story,
				Execution: job.execution,
			})

			select {
			case p.jobQueue <- job:
			case <-p.ctx.Done():
				p.mu.Lock()
				p.running = false
				p.mu.Unlock()
				close(p.jobQueue)
				wg.Wait()
				return p.completionMsg()
			}
		}

		close(p.jobQueue)
		wg.Wait()

		p.mu.Lock()
		p.running = false
		p.mu.Unlock()

		return p.completionMsg()
	}
}

// worker processes jobs from the queue
func (p *ParallelExecutor) worker(id int, wg *sync.WaitGroup) {
	defer wg.Done()

	for job := range p.jobQueue {
		// Check if paused
		p.waitIfPaused()

		// Check if cancelled
		select {
		case <-p.ctx.Done():
			p.resultQueue <- &parallelResult{
				index:  job.index,
				story:  job.story,
				status: domain.ExecutionCancelled,
				error:  "cancelled",
			}
			continue
		default:
		}

		// Execute the story
		result := p.executeStory(job)
		p.resultQueue <- result
	}
}

// executeStory executes a single story through all steps
func (p *ParallelExecutor) executeStory(job *parallelJob) *parallelResult {
	job.execution.Status = domain.ExecutionRunning
	job.execution.StartTime = time.Now()

	// Execute each step
	for i, step := range job.execution.Steps {
		// Check for cancellation
		select {
		case <-p.ctx.Done():
			job.execution.Status = domain.ExecutionCancelled
			job.execution.EndTime = time.Now()
			job.execution.Duration = job.execution.EndTime.Sub(job.execution.StartTime)
			return &parallelResult{
				index:     job.index,
				story:     job.story,
				status:    domain.ExecutionCancelled,
				duration:  job.execution.Duration,
				error:     "cancelled",
				execution: job.execution,
			}
		default:
		}

		// Check if paused
		p.waitIfPaused()

		// Auto-skip create-story if file exists
		if step.Name == domain.StepCreateStory && job.story.FileExists {
			step.Status = domain.StepSkipped
			p.sendMsg(messages.StepCompletedMsg{
				StepIndex: i,
				Status:    domain.StepSkipped,
			})
			continue
		}

		// Execute step
		job.execution.Current = i
		err := p.executeStep(job, i, step)

		if err != nil && step.Status == domain.StepFailed {
			job.execution.Status = domain.ExecutionFailed
			job.execution.Error = err.Error()
			job.execution.EndTime = time.Now()
			job.execution.Duration = job.execution.EndTime.Sub(job.execution.StartTime)

			return &parallelResult{
				index:     job.index,
				story:     job.story,
				status:    domain.ExecutionFailed,
				duration:  job.execution.Duration,
				error:     err.Error(),
				execution: job.execution,
			}
		}
	}

	job.execution.Status = domain.ExecutionCompleted
	job.execution.EndTime = time.Now()
	job.execution.Duration = job.execution.EndTime.Sub(job.execution.StartTime)

	return &parallelResult{
		index:     job.index,
		story:     job.story,
		status:    domain.ExecutionCompleted,
		duration:  job.execution.Duration,
		execution: job.execution,
	}
}

// executeStep executes a single step with retry logic
func (p *ParallelExecutor) executeStep(job *parallelJob, index int, step *domain.StepExecution) error {
	maxAttempts := p.config.Retries + 1

	for attempt := 1; attempt <= maxAttempts; attempt++ {
		select {
		case <-p.ctx.Done():
			return fmt.Errorf("cancelled")
		default:
		}

		step.Attempt = attempt
		step.Status = domain.StepRunning
		step.StartTime = time.Now()
		step.Output = make([]string, 0)
		step.Command = p.buildCommand(step.Name, job.story)

		p.sendMsg(messages.StepStartedMsg{
			StepIndex: index,
			StepName:  step.Name,
			Command:   step.Command,
			Attempt:   attempt,
		})

		// Execute with timeout
		ctx, cancel := context.WithTimeout(p.ctx, time.Duration(p.config.Timeout)*time.Second)
		err := p.runCommand(ctx, job, index, step)
		cancel()

		step.EndTime = time.Now()
		step.Duration = step.EndTime.Sub(step.StartTime)

		if err == nil {
			step.Status = domain.StepSuccess
			p.sendMsg(messages.StepCompletedMsg{
				StepIndex: index,
				Status:    domain.StepSuccess,
				Duration:  step.Duration,
			})
			return nil
		}

		// Handle errors
		if ctx.Err() == context.DeadlineExceeded {
			step.Error = fmt.Sprintf("timeout after %ds", p.config.Timeout)
		} else if ctx.Err() == context.Canceled {
			step.Error = "cancelled"
		} else {
			step.Error = err.Error()
		}

		// Retry or fail
		if attempt < maxAttempts {
			p.sendMsg(messages.StepOutputMsg{
				StepIndex: index,
				Line:      fmt.Sprintf("[%s] Retrying in 2s (attempt %d/%d)...", job.story.Key, attempt+1, maxAttempts),
				IsStderr:  true,
			})
			time.Sleep(2 * time.Second)
		} else {
			step.Status = domain.StepFailed
			p.sendMsg(messages.StepCompletedMsg{
				StepIndex: index,
				Status:    domain.StepFailed,
				Duration:  step.Duration,
				Error:     step.Error,
			})
		}
	}

	return fmt.Errorf("%s", step.Error)
}

// runCommand executes a command and streams output (similar to Executor.runCommand)
func (p *ParallelExecutor) runCommand(ctx context.Context, job *parallelJob, stepIndex int, step *domain.StepExecution) error {
	// Use the same implementation as the regular executor
	exec := New(p.config)
	exec.program = p.program
	return exec.runCommand(ctx, stepIndex, step)
}

// buildCommand creates the Claude CLI command for a step
func (p *ParallelExecutor) buildCommand(stepName domain.StepName, story domain.Story) string {
	exec := New(p.config)
	return exec.buildCommand(stepName, story)
}

// collectResults processes results from workers
func (p *ParallelExecutor) collectResults() {
	for result := range p.resultQueue {
		p.mu.Lock()
		if result.status == domain.ExecutionCompleted {
			p.completed++
		} else {
			p.failed++
		}
		delete(p.activeJobs, result.story.Key)
		p.mu.Unlock()

		p.sendMsg(messages.QueueItemCompletedMsg{
			Index:     result.index,
			Story:     result.story,
			Status:    result.status,
			Duration:  result.duration,
			Error:     result.error,
			Execution: result.execution,
		})
	}
}

// Pause pauses execution
func (p *ParallelExecutor) Pause() {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.paused = true
}

// Resume resumes execution
func (p *ParallelExecutor) Resume() {
	p.mu.Lock()
	p.paused = false
	p.mu.Unlock()

	select {
	case p.resumeCh <- struct{}{}:
	default:
	}
}

// Cancel cancels execution
func (p *ParallelExecutor) Cancel() {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.cancel != nil {
		p.cancel()
	}
}

// IsRunning returns whether execution is running
func (p *ParallelExecutor) IsRunning() bool {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.running
}

// IsPaused returns whether execution is paused
func (p *ParallelExecutor) IsPaused() bool {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.paused
}

// waitIfPaused blocks until execution is resumed
func (p *ParallelExecutor) waitIfPaused() {
	for {
		p.mu.Lock()
		paused := p.paused
		p.mu.Unlock()

		if !paused {
			return
		}

		select {
		case <-p.resumeCh:
			return
		case <-p.ctx.Done():
			return
		case <-time.After(100 * time.Millisecond):
		}
	}
}

// completionMsg creates the completion message
func (p *ParallelExecutor) completionMsg() messages.QueueCompletedMsg {
	p.mu.Lock()
	defer p.mu.Unlock()

	return messages.QueueCompletedMsg{
		TotalItems:    p.total,
		SuccessCount:  p.completed,
		FailedCount:   p.failed,
		TotalDuration: time.Since(p.startTime),
	}
}

// sendMsg safely sends a message to the tea.Program
func (p *ParallelExecutor) sendMsg(msg tea.Msg) {
	if p.program != nil {
		p.program.Send(msg)
	}
}

// GetProgress returns current progress statistics
func (p *ParallelExecutor) GetProgress() (completed, failed, total int) {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.completed, p.failed, p.total
}

// GetActiveJobs returns the number of currently active jobs
func (p *ParallelExecutor) GetActiveJobs() int {
	p.mu.Lock()
	defer p.mu.Unlock()
	return len(p.activeJobs)
}
