package executor

import (
	"bufio"
	"context"
	"fmt"
	"os/exec"
	"sync"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/robertguss/bmad-automate-go/internal/config"
	"github.com/robertguss/bmad-automate-go/internal/domain"
	"github.com/robertguss/bmad-automate-go/internal/messages"
)

// Executor manages the execution of story workflows
type Executor struct {
	config    *config.Config
	program   *tea.Program
	execution *domain.Execution

	// Control channels
	pauseCh  chan struct{}
	resumeCh chan struct{}
	cancelCh chan struct{}
	skipCh   chan struct{}

	// State
	mu       sync.Mutex
	paused   bool
	canceled bool
	ctx      context.Context
	cancel   context.CancelFunc
}

// New creates a new Executor
func New(cfg *config.Config) *Executor {
	return &Executor{
		config:   cfg,
		pauseCh:  make(chan struct{}),
		resumeCh: make(chan struct{}),
		cancelCh: make(chan struct{}),
		skipCh:   make(chan struct{}),
	}
}

// SetProgram sets the tea.Program for sending messages
func (e *Executor) SetProgram(p *tea.Program) {
	e.program = p
}

// Execute starts the execution of a story through all workflow steps
func (e *Executor) Execute(story domain.Story) tea.Cmd {
	return func() tea.Msg {
		e.mu.Lock()
		e.execution = domain.NewExecution(story)
		e.execution.Status = domain.ExecutionRunning
		e.execution.StartTime = time.Now()
		e.paused = false
		e.canceled = false
		e.ctx, e.cancel = context.WithCancel(context.Background())
		e.mu.Unlock()

		// Send execution started message
		e.sendMsg(messages.ExecutionStartedMsg{Execution: e.execution})

		// Start the execution tick for updating duration display
		go e.runTicker()

		// Execute each step
		for i, step := range e.execution.Steps {
			e.mu.Lock()
			canceled := e.canceled
			e.mu.Unlock()

			if canceled {
				e.execution.Status = domain.ExecutionCancelled
				break
			}

			// Wait if paused
			e.waitIfPaused()

			// Check for skip request
			select {
			case <-e.skipCh:
				step.Status = domain.StepSkipped
				e.sendMsg(messages.StepCompletedMsg{
					StepIndex: i,
					Status:    domain.StepSkipped,
				})
				continue
			default:
			}

			// Check if we should auto-skip create-story
			if step.Name == domain.StepCreateStory && story.FileExists {
				step.Status = domain.StepSkipped
				e.sendMsg(messages.StepCompletedMsg{
					StepIndex: i,
					Status:    domain.StepSkipped,
				})
				continue
			}

			// Execute the step with retries
			e.execution.Current = i
			err := e.executeStep(i, step)

			if err != nil && step.Status == domain.StepFailed {
				e.execution.Status = domain.ExecutionFailed
				e.execution.Error = err.Error()
				break
			}
		}

		// Mark completion
		e.execution.EndTime = time.Now()
		e.execution.Duration = e.execution.EndTime.Sub(e.execution.StartTime)

		if e.execution.Status == domain.ExecutionRunning {
			e.execution.Status = domain.ExecutionCompleted
		}

		return messages.ExecutionCompletedMsg{
			Status:   e.execution.Status,
			Duration: e.execution.Duration,
			Error:    e.execution.Error,
		}
	}
}

// executeStep runs a single step with retry logic
func (e *Executor) executeStep(index int, step *domain.StepExecution) error {
	maxAttempts := e.config.Retries + 1

	for attempt := 1; attempt <= maxAttempts; attempt++ {
		e.mu.Lock()
		if e.canceled {
			e.mu.Unlock()
			return fmt.Errorf("cancelled")
		}
		e.mu.Unlock()

		step.Attempt = attempt
		step.Status = domain.StepRunning
		step.StartTime = time.Now()
		step.Output = make([]string, 0)
		step.Command = e.buildCommand(step.Name, e.execution.Story)

		e.sendMsg(messages.StepStartedMsg{
			StepIndex: index,
			StepName:  step.Name,
			Command:   step.Command,
			Attempt:   attempt,
		})

		// Execute with timeout
		ctx, cancel := context.WithTimeout(e.ctx, time.Duration(e.config.Timeout)*time.Second)
		err := e.runCommand(ctx, index, step)
		cancel()

		step.EndTime = time.Now()
		step.Duration = step.EndTime.Sub(step.StartTime)

		if err == nil {
			step.Status = domain.StepSuccess
			e.sendMsg(messages.StepCompletedMsg{
				StepIndex: index,
				Status:    domain.StepSuccess,
				Duration:  step.Duration,
			})
			return nil
		}

		// Check if this was a context cancellation (timeout or user cancel)
		if ctx.Err() == context.DeadlineExceeded {
			step.Error = fmt.Sprintf("timeout after %ds", e.config.Timeout)
		} else if ctx.Err() == context.Canceled {
			step.Error = "cancelled"
		} else {
			step.Error = err.Error()
		}

		// If we have retries left, wait before retrying
		if attempt < maxAttempts {
			e.sendMsg(messages.StepOutputMsg{
				StepIndex: index,
				Line:      fmt.Sprintf("Retrying in 2 seconds (attempt %d/%d)...", attempt+1, maxAttempts),
				IsStderr:  true,
			})
			time.Sleep(2 * time.Second)
		} else {
			step.Status = domain.StepFailed
			e.sendMsg(messages.StepCompletedMsg{
				StepIndex: index,
				Status:    domain.StepFailed,
				Duration:  step.Duration,
				Error:     step.Error,
			})
		}
	}

	return fmt.Errorf("%s", step.Error)
}

// runCommand executes a command and streams output
func (e *Executor) runCommand(ctx context.Context, stepIndex int, step *domain.StepExecution) error {
	cmd := exec.CommandContext(ctx, "sh", "-c", step.Command)
	cmd.Dir = e.config.WorkingDir

	// Create pipes for stdout and stderr
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return fmt.Errorf("failed to create stdout pipe: %w", err)
	}

	stderr, err := cmd.StderrPipe()
	if err != nil {
		return fmt.Errorf("failed to create stderr pipe: %w", err)
	}

	// Start the command
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("failed to start command: %w", err)
	}

	// Stream output in goroutines
	var wg sync.WaitGroup
	wg.Add(2)

	go func() {
		defer wg.Done()
		scanner := bufio.NewScanner(stdout)
		// Increase buffer size for long lines
		buf := make([]byte, 0, 64*1024)
		scanner.Buffer(buf, 1024*1024)
		for scanner.Scan() {
			line := scanner.Text()
			e.mu.Lock()
			step.Output = append(step.Output, line)
			e.mu.Unlock()
			e.sendMsg(messages.StepOutputMsg{
				StepIndex: stepIndex,
				Line:      line,
				IsStderr:  false,
			})
		}
	}()

	go func() {
		defer wg.Done()
		scanner := bufio.NewScanner(stderr)
		buf := make([]byte, 0, 64*1024)
		scanner.Buffer(buf, 1024*1024)
		for scanner.Scan() {
			line := scanner.Text()
			e.mu.Lock()
			step.Output = append(step.Output, "[stderr] "+line)
			e.mu.Unlock()
			e.sendMsg(messages.StepOutputMsg{
				StepIndex: stepIndex,
				Line:      line,
				IsStderr:  true,
			})
		}
	}()

	// Wait for output streams to finish
	wg.Wait()

	// Wait for command to complete
	return cmd.Wait()
}

// buildCommand creates the Claude CLI command for a step
func (e *Executor) buildCommand(stepName domain.StepName, story domain.Story) string {
	storyPath := e.config.StoryFilePath(story.Key)

	switch stepName {
	case domain.StepCreateStory:
		return fmt.Sprintf(
			`claude --dangerously-skip-permissions -p "/bmad:bmm:workflows:create-story - Create story: %s"`,
			story.Key,
		)

	case domain.StepDevStory:
		prompt := fmt.Sprintf(
			"/bmad:bmm:workflows:dev-story - Work on story file: %s. "+
				"Complete all tasks. Run tests after each implementation. "+
				"Do not ask clarifying questions - use best judgment based on existing patterns.",
			storyPath,
		)
		return fmt.Sprintf(`claude --dangerously-skip-permissions -p "%s"`, prompt)

	case domain.StepCodeReview:
		prompt := fmt.Sprintf(
			"/bmad:bmm:workflows:code-review - Review story: %s. "+
				"IMPORTANT: When presenting options, always choose option 1 to "+
				"auto-fix all issues immediately. Do not wait for user input.",
			storyPath,
		)
		return fmt.Sprintf(`claude --dangerously-skip-permissions -p "%s"`, prompt)

	case domain.StepGitCommit:
		prompt := fmt.Sprintf(
			"Commit all changes for story %s with a descriptive message. "+
				"Then push to the current branch.",
			story.Key,
		)
		return fmt.Sprintf(`claude --dangerously-skip-permissions -p "%s"`, prompt)

	default:
		return ""
	}
}

// Pause pauses the execution
func (e *Executor) Pause() {
	e.mu.Lock()
	defer e.mu.Unlock()
	if !e.paused && e.execution != nil {
		e.paused = true
		e.execution.Status = domain.ExecutionPaused
	}
}

// Resume resumes a paused execution
func (e *Executor) Resume() {
	e.mu.Lock()
	if e.paused {
		e.paused = false
		if e.execution != nil {
			e.execution.Status = domain.ExecutionRunning
		}
	}
	e.mu.Unlock()

	// Signal to waitIfPaused
	select {
	case e.resumeCh <- struct{}{}:
	default:
	}
}

// Cancel cancels the current execution
func (e *Executor) Cancel() {
	e.mu.Lock()
	e.canceled = true
	if e.cancel != nil {
		e.cancel()
	}
	e.mu.Unlock()
}

// Skip requests skipping the current step
func (e *Executor) Skip() {
	select {
	case e.skipCh <- struct{}{}:
	default:
	}
}

// IsPaused returns true if execution is paused
func (e *Executor) IsPaused() bool {
	e.mu.Lock()
	defer e.mu.Unlock()
	return e.paused
}

// GetExecution returns the current execution state
func (e *Executor) GetExecution() *domain.Execution {
	e.mu.Lock()
	defer e.mu.Unlock()
	return e.execution
}

// waitIfPaused blocks until execution is resumed
func (e *Executor) waitIfPaused() {
	for {
		e.mu.Lock()
		paused := e.paused
		canceled := e.canceled
		e.mu.Unlock()

		if canceled || !paused {
			return
		}

		select {
		case <-e.resumeCh:
			return
		case <-time.After(100 * time.Millisecond):
			// Check again
		}
	}
}

// runTicker sends periodic tick messages for updating duration display
func (e *Executor) runTicker() {
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case t := <-ticker.C:
			e.mu.Lock()
			execution := e.execution
			e.mu.Unlock()

			if execution == nil || execution.Status == domain.ExecutionCompleted ||
				execution.Status == domain.ExecutionFailed ||
				execution.Status == domain.ExecutionCancelled {
				return
			}

			e.sendMsg(messages.ExecutionTickMsg{Time: t})
		}
	}
}

// sendMsg safely sends a message to the tea.Program
func (e *Executor) sendMsg(msg tea.Msg) {
	if e.program != nil {
		e.program.Send(msg)
	}
}
