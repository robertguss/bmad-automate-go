package executor

import (
	"bufio"
	"context"
	"fmt"
	"os/exec"
	"strings"
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
	skipCh chan struct{}

	// Pause/resume/cancel control (QUAL-003: shared utility)
	pauseCtrl *PauseController

	// State
	mu     sync.Mutex
	ctx    context.Context
	cancel context.CancelFunc
}

// New creates a new Executor
func New(cfg *config.Config) *Executor {
	return &Executor{
		config:    cfg,
		skipCh:    make(chan struct{}),
		pauseCtrl: NewPauseController(),
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
		e.pauseCtrl.Reset()
		e.ctx, e.cancel = context.WithCancel(context.Background())
		e.mu.Unlock()

		// Send execution started message
		e.sendMsg(messages.ExecutionStartedMsg{Execution: e.execution})

		// Start the execution tick for updating duration display
		go e.runTicker()

		// Execute each step
		for i, step := range e.execution.Steps {
			if e.pauseCtrl.IsCanceled() {
				e.execution.Status = domain.ExecutionCancelled
				break
			}

			// Wait if paused (QUAL-003: using shared utility)
			e.pauseCtrl.WaitIfPaused(nil)

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
		if e.pauseCtrl.IsCanceled() {
			return fmt.Errorf("cancelled")
		}

		step.Attempt = attempt
		step.Status = domain.StepRunning
		step.StartTime = time.Now()
		step.Output = make([]string, 0)

		// Build command with separate name and args (prevents shell injection)
		cmdSpec := e.buildCommand(step.Name, e.execution.Story)
		step.CommandName = cmdSpec.Name
		step.CommandArgs = cmdSpec.Args
		step.Command = cmdSpec.DisplayString() // For logging/display only

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
			time.Sleep(RetryDelayDuration)
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
// Uses exec.CommandContext with separate args to prevent shell injection
func (e *Executor) runCommand(ctx context.Context, stepIndex int, step *domain.StepExecution) error {
	// Execute command directly without shell interpolation (SEC-001 fix)
	cmd := exec.CommandContext(ctx, step.CommandName, step.CommandArgs...)
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
		buf := make([]byte, 0, ScannerInitialBufferSize)
		scanner.Buffer(buf, ScannerMaxBufferSize)
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
		buf := make([]byte, 0, ScannerInitialBufferSize)
		scanner.Buffer(buf, ScannerMaxBufferSize)
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

// CommandSpec holds the command name and arguments for safe execution
type CommandSpec struct {
	Name string   // Executable name (e.g., "claude")
	Args []string // Arguments passed directly to exec.Command (no shell interpolation)
}

// DisplayString returns a human-readable representation of the command for logging
func (c CommandSpec) DisplayString() string {
	if len(c.Args) == 0 {
		return c.Name
	}
	// Build a display string (for logging only, not for execution)
	return fmt.Sprintf("%s %s", c.Name, strings.Join(c.Args, " "))
}

// buildCommand creates the Claude CLI command specification for a step
// Returns command name and args separately to prevent shell injection
func (e *Executor) buildCommand(stepName domain.StepName, story domain.Story) CommandSpec {
	storyPath := e.config.StoryFilePath(story.Key)

	switch stepName {
	case domain.StepCreateStory:
		prompt := fmt.Sprintf("/bmad:bmm:workflows:create-story - Create story: %s", story.Key)
		return CommandSpec{
			Name: "claude",
			Args: []string{"--dangerously-skip-permissions", "-p", prompt},
		}

	case domain.StepDevStory:
		prompt := fmt.Sprintf(
			"/bmad:bmm:workflows:dev-story - Work on story file: %s. "+
				"Complete all tasks. Run tests after each implementation. "+
				"Do not ask clarifying questions - use best judgment based on existing patterns.",
			storyPath,
		)
		return CommandSpec{
			Name: "claude",
			Args: []string{"--dangerously-skip-permissions", "-p", prompt},
		}

	case domain.StepCodeReview:
		prompt := fmt.Sprintf(
			"/bmad:bmm:workflows:code-review - Review story: %s. "+
				"IMPORTANT: When presenting options, always choose option 1 to "+
				"auto-fix all issues immediately. Do not wait for user input.",
			storyPath,
		)
		return CommandSpec{
			Name: "claude",
			Args: []string{"--dangerously-skip-permissions", "-p", prompt},
		}

	case domain.StepGitCommit:
		prompt := fmt.Sprintf(
			"Commit all changes for story %s with a descriptive message. "+
				"Then push to the current branch.",
			story.Key,
		)
		return CommandSpec{
			Name: "claude",
			Args: []string{"--dangerously-skip-permissions", "-p", prompt},
		}

	default:
		return CommandSpec{}
	}
}

// Pause pauses the execution
func (e *Executor) Pause() {
	e.mu.Lock()
	defer e.mu.Unlock()
	if !e.pauseCtrl.IsPaused() && e.execution != nil {
		e.pauseCtrl.Pause()
		e.execution.Status = domain.ExecutionPaused
	}
}

// Resume resumes a paused execution
func (e *Executor) Resume() {
	e.mu.Lock()
	if e.pauseCtrl.IsPaused() {
		if e.execution != nil {
			e.execution.Status = domain.ExecutionRunning
		}
	}
	e.mu.Unlock()

	// Resume will signal to WaitIfPaused
	e.pauseCtrl.Resume()
}

// Cancel cancels the current execution
func (e *Executor) Cancel() {
	e.mu.Lock()
	if e.cancel != nil {
		e.cancel()
	}
	e.mu.Unlock()
	e.pauseCtrl.Cancel()
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
	return e.pauseCtrl.IsPaused()
}

// GetExecution returns the current execution state
func (e *Executor) GetExecution() *domain.Execution {
	e.mu.Lock()
	defer e.mu.Unlock()
	return e.execution
}

// runTicker sends periodic tick messages for updating duration display
func (e *Executor) runTicker() {
	ticker := time.NewTicker(ExecutionTickInterval)
	defer ticker.Stop()

	for t := range ticker.C {
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

// sendMsg safely sends a message to the tea.Program
func (e *Executor) sendMsg(msg tea.Msg) {
	if e.program != nil {
		e.program.Send(msg)
	}
}
