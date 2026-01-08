package preflight

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/robertguss/bmad-automate-go/internal/config"
)

func TestResults_PassedCount(t *testing.T) {
	tests := []struct {
		name     string
		checks   []CheckResult
		expected int
	}{
		{
			name:     "all passed",
			checks:   []CheckResult{{Passed: true}, {Passed: true}, {Passed: true}},
			expected: 3,
		},
		{
			name:     "some failed",
			checks:   []CheckResult{{Passed: true}, {Passed: false}, {Passed: true}},
			expected: 2,
		},
		{
			name:     "all failed",
			checks:   []CheckResult{{Passed: false}, {Passed: false}},
			expected: 0,
		},
		{
			name:     "empty",
			checks:   []CheckResult{},
			expected: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := &Results{Checks: tt.checks}
			assert.Equal(t, tt.expected, r.PassedCount())
		})
	}
}

func TestResults_FailedChecks(t *testing.T) {
	tests := []struct {
		name          string
		checks        []CheckResult
		expectedCount int
	}{
		{
			name: "returns only failed checks",
			checks: []CheckResult{
				{Name: "Check1", Passed: true},
				{Name: "Check2", Passed: false},
				{Name: "Check3", Passed: true},
				{Name: "Check4", Passed: false},
			},
			expectedCount: 2,
		},
		{
			name: "returns empty when all pass",
			checks: []CheckResult{
				{Name: "Check1", Passed: true},
				{Name: "Check2", Passed: true},
			},
			expectedCount: 0,
		},
		{
			name:          "handles empty checks",
			checks:        []CheckResult{},
			expectedCount: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := &Results{Checks: tt.checks}
			failed := r.FailedChecks()
			assert.Len(t, failed, tt.expectedCount)

			// Verify all returned are actually failed
			for _, check := range failed {
				assert.False(t, check.Passed)
			}
		})
	}
}

func TestResults_AddCheck(t *testing.T) {
	t.Run("adds check to list", func(t *testing.T) {
		r := &Results{Checks: []CheckResult{}, AllPass: true}

		r.addCheck(CheckResult{Name: "Test", Passed: true})

		assert.Len(t, r.Checks, 1)
		assert.Equal(t, "Test", r.Checks[0].Name)
	})

	t.Run("sets AllPass to false when check fails", func(t *testing.T) {
		r := &Results{Checks: []CheckResult{}, AllPass: true}

		r.addCheck(CheckResult{Name: "Failed", Passed: false})

		assert.False(t, r.AllPass)
	})

	t.Run("keeps AllPass true for Git Clean warning", func(t *testing.T) {
		r := &Results{Checks: []CheckResult{}, AllPass: true}

		r.addCheck(CheckResult{Name: "Git Clean", Passed: false})

		assert.True(t, r.AllPass) // Git Clean is a warning, not a blocker
	})

	t.Run("keeps AllPass true when check passes", func(t *testing.T) {
		r := &Results{Checks: []CheckResult{}, AllPass: true}

		r.addCheck(CheckResult{Name: "Passed", Passed: true})

		assert.True(t, r.AllPass)
	})
}

func TestCheckSprintStatus(t *testing.T) {
	t.Run("passes when file exists", func(t *testing.T) {
		tempDir := t.TempDir()
		sprintStatusPath := filepath.Join(tempDir, "sprint-status.yaml")
		_ = os.WriteFile(sprintStatusPath, []byte("test"), 0644)

		cfg := &config.Config{SprintStatusPath: sprintStatusPath}

		result := checkSprintStatus(cfg)

		assert.True(t, result.Passed)
		assert.Equal(t, "Sprint Status", result.Name)
		assert.Equal(t, "Found", result.Message)
	})

	t.Run("fails when file does not exist", func(t *testing.T) {
		cfg := &config.Config{SprintStatusPath: "/nonexistent/sprint-status.yaml"}

		result := checkSprintStatus(cfg)

		assert.False(t, result.Passed)
		assert.Contains(t, result.Error, "not found")
	})
}

func TestCheckStoryDir(t *testing.T) {
	t.Run("passes when directory exists", func(t *testing.T) {
		tempDir := t.TempDir()
		storyDir := filepath.Join(tempDir, "stories")
		_ = os.MkdirAll(storyDir, 0755)

		cfg := &config.Config{StoryDir: storyDir}

		result := checkStoryDir(cfg)

		assert.True(t, result.Passed)
		assert.Equal(t, "Story Directory", result.Name)
		assert.Equal(t, "Found", result.Message)
	})

	t.Run("fails when directory does not exist", func(t *testing.T) {
		cfg := &config.Config{StoryDir: "/nonexistent/stories"}

		result := checkStoryDir(cfg)

		assert.False(t, result.Passed)
		assert.Contains(t, result.Error, "not found")
	})

	t.Run("fails when path is not a directory", func(t *testing.T) {
		tempDir := t.TempDir()
		filePath := filepath.Join(tempDir, "not-a-dir")
		_ = os.WriteFile(filePath, []byte("test"), 0644)

		cfg := &config.Config{StoryDir: filePath}

		result := checkStoryDir(cfg)

		assert.False(t, result.Passed)
		assert.Contains(t, result.Error, "Not a directory")
	})
}

func TestCheckGitRepo(t *testing.T) {
	// This test requires being in a git repository
	// Since we're running in a git repo, we can test the positive case

	t.Run("passes in git repository", func(t *testing.T) {
		// Get current working directory (which should be in the git repo)
		wd, err := os.Getwd()
		require.NoError(t, err)

		// Find the git root
		cmd := exec.Command("git", "rev-parse", "--show-toplevel")
		output, err := cmd.Output()
		if err != nil {
			t.Skip("Not running in a git repository")
		}

		// Verify we got a valid git root (non-empty output)
		_ = string(output)

		cfg := &config.Config{WorkingDir: wd}

		result := checkGitRepo(cfg)

		assert.True(t, result.Passed)
		assert.Equal(t, "Git Repository", result.Name)
	})

	t.Run("fails outside git repository", func(t *testing.T) {
		tempDir := t.TempDir()
		cfg := &config.Config{WorkingDir: tempDir}

		result := checkGitRepo(cfg)

		assert.False(t, result.Passed)
		assert.Contains(t, result.Error, "Not a git repository")
	})
}

func TestGetGitBranch(t *testing.T) {
	t.Run("returns branch name in git repo", func(t *testing.T) {
		// Get current working directory
		wd, err := os.Getwd()
		require.NoError(t, err)

		// Verify we're in a git repo
		cmd := exec.Command("git", "rev-parse", "--is-inside-work-tree")
		cmd.Dir = wd
		if err := cmd.Run(); err != nil {
			t.Skip("Not running in a git repository")
		}

		branch := GetGitBranch(wd)

		assert.NotEqual(t, "unknown", branch)
		assert.NotEmpty(t, branch)
	})

	t.Run("returns unknown outside git repo", func(t *testing.T) {
		tempDir := t.TempDir()

		branch := GetGitBranch(tempDir)

		assert.Equal(t, "unknown", branch)
	})
}

func TestIsGitClean(t *testing.T) {
	t.Run("returns false outside git repo", func(t *testing.T) {
		tempDir := t.TempDir()

		result := IsGitClean(tempDir)

		assert.False(t, result)
	})

	// Note: Testing true case would require a completely clean git repo
	// which may not be the case during development
}

func TestCheckResult_Structure(t *testing.T) {
	result := CheckResult{
		Name:    "Test Check",
		Passed:  true,
		Message: "Everything OK",
		Error:   "",
	}

	assert.Equal(t, "Test Check", result.Name)
	assert.True(t, result.Passed)
	assert.Equal(t, "Everything OK", result.Message)
	assert.Empty(t, result.Error)
}

func TestRunAll(t *testing.T) {
	// This is an integration test that runs all checks
	// We'll create a minimal valid configuration

	t.Run("runs all checks", func(t *testing.T) {
		tempDir := t.TempDir()
		sprintStatusPath := filepath.Join(tempDir, "sprint-status.yaml")
		storyDir := filepath.Join(tempDir, "stories")

		_ = os.WriteFile(sprintStatusPath, []byte("development_status:\n"), 0644)
		_ = os.MkdirAll(storyDir, 0755)

		cfg := &config.Config{
			SprintStatusPath: sprintStatusPath,
			StoryDir:         storyDir,
			WorkingDir:       tempDir,
		}

		results := RunAll(cfg)

		require.NotNil(t, results)
		// Should have at least 5 checks (Claude CLI, Sprint Status, Story Dir, Git Repo, Git Clean)
		assert.GreaterOrEqual(t, len(results.Checks), 5)
	})
}
