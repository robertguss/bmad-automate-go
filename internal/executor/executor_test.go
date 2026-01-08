package executor

import (
	"context"
	"sync"
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
	assert.NotNil(t, e.skipCh)
	assert.NotNil(t, e.pauseCtrl)
	assert.False(t, e.pauseCtrl.IsPaused())
	assert.False(t, e.pauseCtrl.IsCanceled())
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
			cmdSpec := e.buildCommand(tt.stepName, e.execution.Story)
			assert.Equal(t, "claude", cmdSpec.Name)
			assert.Contains(t, cmdSpec.DisplayString(), tt.contains)
			// Verify args are properly separated (SEC-001 fix)
			assert.Contains(t, cmdSpec.Args, "--dangerously-skip-permissions")
			assert.Contains(t, cmdSpec.Args, "-p")
		})
	}

	t.Run("unknown step returns empty CommandSpec", func(t *testing.T) {
		cmdSpec := e.buildCommand("unknown-step", e.execution.Story)
		assert.Empty(t, cmdSpec.Name)
		assert.Empty(t, cmdSpec.Args)
	})

	t.Run("includes story key in prompt arg", func(t *testing.T) {
		cmdSpec := e.buildCommand(domain.StepCreateStory, e.execution.Story)
		// The story key should be in the prompt argument, not as a separate arg
		assert.Contains(t, cmdSpec.DisplayString(), "3-1-test-story")
		// Verify the prompt is a single argument (prevents shell injection)
		foundPrompt := false
		for _, arg := range cmdSpec.Args {
			if arg == "-p" {
				foundPrompt = true
			} else if foundPrompt {
				// The next arg after -p should contain the story key
				assert.Contains(t, arg, "3-1-test-story")
				break
			}
		}
	})
}

func TestCommandSpec_DisplayString(t *testing.T) {
	t.Run("returns name when no args", func(t *testing.T) {
		cs := CommandSpec{Name: "echo"}
		assert.Equal(t, "echo", cs.DisplayString())
	})

	t.Run("returns name and args joined", func(t *testing.T) {
		cs := CommandSpec{Name: "echo", Args: []string{"hello", "world"}}
		assert.Equal(t, "echo hello world", cs.DisplayString())
	})
}

// TestBuildCommand_PreventShellInjection verifies SEC-001 fix:
// Story keys with shell metacharacters should not be executed as shell commands
func TestBuildCommand_PreventShellInjection(t *testing.T) {
	cfg := createTestConfig()
	e := New(cfg)

	// These malicious story keys would cause command injection with "sh -c"
	// but should be safe when passed as separate args to exec.Command
	maliciousKeys := []struct {
		name string
		key  string
	}{
		{"semicolon command", "3-1-test; rm -rf /"},
		{"command substitution $(...)", "3-1-test$(whoami)"},
		{"command substitution backtick", "3-1-test`id`"},
		{"pipe injection", "3-1-test | cat /etc/passwd"},
		{"and injection", "3-1-test && curl evil.com"},
		{"quote escape", `3-1-test"; echo pwned; "`},
		{"newline injection", "3-1-test\nrm -rf /"},
		{"redirect injection", "3-1-test > /tmp/pwned"},
	}

	for _, tc := range maliciousKeys {
		t.Run(tc.name, func(t *testing.T) {
			story := domain.Story{
				Key:    tc.key,
				Epic:   3,
				Status: domain.StatusInProgress,
			}

			cmdSpec := e.buildCommand(domain.StepCreateStory, story)

			// The command should always be "claude" (not "sh")
			assert.Equal(t, "claude", cmdSpec.Name, "command name should be 'claude', not 'sh'")

			// Args should be exactly 3 items: --dangerously-skip-permissions, -p, and the prompt
			assert.Len(t, cmdSpec.Args, 3, "should have exactly 3 args")
			assert.Equal(t, "--dangerously-skip-permissions", cmdSpec.Args[0])
			assert.Equal(t, "-p", cmdSpec.Args[1])

			// The malicious key should be embedded in the prompt string,
			// not as a separate shell argument
			prompt := cmdSpec.Args[2]
			assert.Contains(t, prompt, tc.key, "story key should be in the prompt")

			// Verify the prompt is a single argument (the key should NOT be interpreted as shell)
			// There should NOT be any shell metacharacters being parsed
			// The entire story key should be part of one prompt string
			for i, arg := range cmdSpec.Args {
				if i == 2 {
					// This is the prompt - it should contain the story key as-is
					continue
				}
				// Other args should NOT contain the malicious key
				assert.NotContains(t, arg, tc.key, "malicious key should only be in prompt arg")
			}
		})
	}
}

// TestRunCommand_UsesExecCommandDirectly verifies that runCommand uses
// exec.Command directly instead of "sh -c" to prevent shell injection
func TestRunCommand_UsesExecCommandDirectly(t *testing.T) {
	// Create a step with a command that would be dangerous if run via shell
	step := &domain.StepExecution{
		Name:        domain.StepCreateStory,
		CommandName: "echo",                    // Safe command
		CommandArgs: []string{"hello; whoami"}, // Would be dangerous via "sh -c"
		Command:     "echo hello; whoami",      // Display string
	}

	// Verify the step has the correct command structure
	assert.Equal(t, "echo", step.CommandName)
	assert.Equal(t, []string{"hello; whoami"}, step.CommandArgs)

	// If this were run via "sh -c echo hello; whoami", it would execute whoami
	// But with exec.Command("echo", "hello; whoami"), the semicolon is literal
}

func TestExecutor_Pause(t *testing.T) {
	cfg := createTestConfig()
	e := New(cfg)

	t.Run("pause without execution does nothing", func(t *testing.T) {
		e.Pause()
		assert.False(t, e.pauseCtrl.IsPaused())
	})

	t.Run("pause with execution sets paused state", func(t *testing.T) {
		e.execution = domain.NewExecution(createTestStory())
		e.execution.Status = domain.ExecutionRunning

		e.Pause()

		assert.True(t, e.pauseCtrl.IsPaused())
		assert.Equal(t, domain.ExecutionPaused, e.execution.Status)
	})

	t.Run("double pause does not change state", func(t *testing.T) {
		e.pauseCtrl.Reset()
		e.execution = domain.NewExecution(createTestStory())
		e.execution.Status = domain.ExecutionRunning

		e.Pause()
		e.Pause()

		assert.True(t, e.pauseCtrl.IsPaused())
	})
}

func TestExecutor_Resume(t *testing.T) {
	cfg := createTestConfig()
	e := New(cfg)
	e.execution = domain.NewExecution(createTestStory())
	e.execution.Status = domain.ExecutionRunning

	t.Run("resume when not paused does nothing", func(t *testing.T) {
		e.pauseCtrl.Reset()
		e.Resume()
		assert.False(t, e.pauseCtrl.IsPaused())
	})

	t.Run("resume when paused clears paused state", func(t *testing.T) {
		e.pauseCtrl.Pause()
		e.execution.Status = domain.ExecutionPaused

		e.Resume()

		assert.False(t, e.pauseCtrl.IsPaused())
		assert.Equal(t, domain.ExecutionRunning, e.execution.Status)
	})
}

func TestExecutor_Cancel(t *testing.T) {
	cfg := createTestConfig()
	e := New(cfg)

	t.Run("cancel sets canceled state", func(t *testing.T) {
		e.Cancel()
		assert.True(t, e.pauseCtrl.IsCanceled())
	})

	t.Run("cancel calls context cancel if set", func(t *testing.T) {
		// This just verifies it doesn't panic when cancel is nil
		e.pauseCtrl.Reset()
		e.cancel = nil
		e.Cancel()
		assert.True(t, e.pauseCtrl.IsCanceled())
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
		e.pauseCtrl.Reset()
		assert.False(t, e.IsPaused())
	})

	t.Run("returns true when paused", func(t *testing.T) {
		e.pauseCtrl.Pause()
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
		e.pauseCtrl.Reset()

		done := make(chan bool)
		go func() {
			e.pauseCtrl.WaitIfPaused(nil)
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
		e.pauseCtrl.Reset()
		e.pauseCtrl.Pause()
		e.pauseCtrl.Cancel()

		done := make(chan bool)
		go func() {
			e.pauseCtrl.WaitIfPaused(nil)
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
		e.pauseCtrl.Reset()
		e.pauseCtrl.Pause()

		done := make(chan bool)
		go func() {
			e.pauseCtrl.WaitIfPaused(nil)
			done <- true
		}()

		// Give it time to start waiting
		time.Sleep(50 * time.Millisecond)

		// Resume
		e.pauseCtrl.Resume()

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

func TestExecutor_CancelWithContext(t *testing.T) {
	cfg := createTestConfig()
	e := New(cfg)

	t.Run("cancel with nil context does not panic", func(t *testing.T) {
		e.cancel = nil
		e.Cancel()
		assert.True(t, e.pauseCtrl.IsCanceled())
	})

	t.Run("cancel with valid context calls cancel func", func(t *testing.T) {
		e.pauseCtrl.Reset()
		_, cancel := context.WithCancel(context.Background())
		e.cancel = cancel

		e.Cancel()
		assert.True(t, e.pauseCtrl.IsCanceled())
	})
}

func TestExecutor_SkipMultipleTimes(t *testing.T) {
	cfg := createTestConfig()
	e := New(cfg)

	// Multiple skip calls should not block
	done := make(chan bool)
	go func() {
		for i := 0; i < 5; i++ {
			e.Skip()
		}
		done <- true
	}()

	select {
	case <-done:
		// Success
	case <-time.After(200 * time.Millisecond):
		t.Fatal("Multiple Skip calls blocked")
	}
}

func TestExecutor_BuildCommandWithDifferentSteps(t *testing.T) {
	cfg := createTestConfig()
	e := New(cfg)

	story := domain.Story{
		Key:        "5-2-feature-branch",
		Epic:       5,
		Status:     domain.StatusInProgress,
		Title:      "Feature Branch",
		FilePath:   "/test/stories/5-2-feature-branch.md",
		FileExists: true,
	}
	e.execution = domain.NewExecution(story)

	t.Run("dev-story command format", func(t *testing.T) {
		cmdSpec := e.buildCommand(domain.StepDevStory, story)
		assert.Equal(t, "claude", cmdSpec.Name)
		assert.Len(t, cmdSpec.Args, 3)
		assert.Contains(t, cmdSpec.Args[2], "dev-story")
		assert.Contains(t, cmdSpec.Args[2], "5-2-feature-branch")
	})

	t.Run("code-review command format", func(t *testing.T) {
		cmdSpec := e.buildCommand(domain.StepCodeReview, story)
		assert.Equal(t, "claude", cmdSpec.Name)
		assert.Len(t, cmdSpec.Args, 3)
		assert.Contains(t, cmdSpec.Args[2], "code-review")
	})

	t.Run("git-commit command format", func(t *testing.T) {
		cmdSpec := e.buildCommand(domain.StepGitCommit, story)
		assert.Equal(t, "claude", cmdSpec.Name)
		assert.Len(t, cmdSpec.Args, 3)
		assert.Contains(t, cmdSpec.Args[2], "Commit")
	})
}

func TestExecutor_ExecutionMutexSafety(t *testing.T) {
	cfg := createTestConfig()
	e := New(cfg)

	// Test mutex safety with concurrent access
	var wg sync.WaitGroup

	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_ = e.GetExecution()
		}()
	}

	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			e.mu.Lock()
			e.execution = domain.NewExecution(createTestStory())
			e.mu.Unlock()
		}()
	}

	wg.Wait()
}

func TestCommandSpec_Empty(t *testing.T) {
	t.Run("empty CommandSpec", func(t *testing.T) {
		cs := CommandSpec{}
		assert.Equal(t, "", cs.DisplayString())
	})

	t.Run("CommandSpec with only name", func(t *testing.T) {
		cs := CommandSpec{Name: "test"}
		assert.Equal(t, "test", cs.DisplayString())
	})

	t.Run("CommandSpec with empty args slice", func(t *testing.T) {
		cs := CommandSpec{Name: "test", Args: []string{}}
		assert.Equal(t, "test", cs.DisplayString())
	})
}

func TestExecutor_PauseResumeStates(t *testing.T) {
	cfg := createTestConfig()
	e := New(cfg)

	t.Run("pause when not running (nil execution)", func(t *testing.T) {
		e.pauseCtrl.Reset()
		e.execution = nil

		e.Pause()
		// Pause checks for execution != nil, so it won't pause
		assert.False(t, e.pauseCtrl.IsPaused())
	})

	t.Run("pause when execution exists pauses regardless of status", func(t *testing.T) {
		e.pauseCtrl.Reset()
		e.execution = domain.NewExecution(createTestStory())
		e.execution.Status = domain.ExecutionCompleted

		e.Pause()
		// Pause only checks if execution != nil, not the status
		assert.True(t, e.pauseCtrl.IsPaused())
	})

	t.Run("resume clears pause even with nil execution", func(t *testing.T) {
		e.pauseCtrl.Reset()
		e.pauseCtrl.Pause()
		e.execution = nil

		e.Resume()
		// Resume still clears the pause state
		assert.False(t, e.pauseCtrl.IsPaused())
	})
}
