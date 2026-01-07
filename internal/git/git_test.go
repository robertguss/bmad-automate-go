package git

import (
	"os"
	"os/exec"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseIntOrZero(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected int
	}{
		{
			name:     "simple number",
			input:    "123",
			expected: 123,
		},
		{
			name:     "zero",
			input:    "0",
			expected: 0,
		},
		{
			name:     "single digit",
			input:    "5",
			expected: 5,
		},
		{
			name:     "large number",
			input:    "999",
			expected: 999,
		},
		{
			name:     "empty string",
			input:    "",
			expected: 0,
		},
		{
			name:     "non-numeric",
			input:    "abc",
			expected: 0,
		},
		{
			name:     "mixed content",
			input:    "12abc34",
			expected: 1234,
		},
		{
			name:     "leading zeros",
			input:    "007",
			expected: 7,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parseIntOrZero(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestStatus_FormatStatus(t *testing.T) {
	tests := []struct {
		name     string
		status   Status
		expected string
	}{
		{
			name:     "not a git repo",
			status:   Status{IsGitRepo: false},
			expected: "Not a git repo",
		},
		{
			name:     "clean repo",
			status:   Status{IsGitRepo: true, IsClean: true},
			expected: "Clean",
		},
		{
			name:     "modified only",
			status:   Status{IsGitRepo: true, IsClean: false, HasUncommitted: true, HasUntracked: false},
			expected: "Modified",
		},
		{
			name:     "untracked only",
			status:   Status{IsGitRepo: true, IsClean: false, HasUncommitted: false, HasUntracked: true},
			expected: "Untracked",
		},
		{
			name:     "modified and untracked",
			status:   Status{IsGitRepo: true, IsClean: false, HasUncommitted: true, HasUntracked: true},
			expected: "Modified, Untracked",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.status.FormatStatus()
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestStatus_FormatBranch(t *testing.T) {
	tests := []struct {
		name     string
		status   Status
		expected string
	}{
		{
			name:     "not a git repo",
			status:   Status{IsGitRepo: false, Branch: "main"},
			expected: "",
		},
		{
			name:     "simple branch",
			status:   Status{IsGitRepo: true, Branch: "main", Ahead: 0, Behind: 0},
			expected: "main",
		},
		{
			name:     "ahead only",
			status:   Status{IsGitRepo: true, Branch: "feature", Ahead: 3, Behind: 0},
			expected: "feature ↑3",
		},
		{
			name:     "behind only",
			status:   Status{IsGitRepo: true, Branch: "feature", Ahead: 0, Behind: 2},
			expected: "feature ↓2",
		},
		{
			name:     "ahead and behind",
			status:   Status{IsGitRepo: true, Branch: "feature", Ahead: 1, Behind: 2},
			expected: "feature ↑1↓2",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.status.FormatBranch()
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestStatus_Structure(t *testing.T) {
	status := Status{
		Branch:           "main",
		IsClean:          true,
		HasUncommitted:   false,
		HasUntracked:     false,
		Ahead:            0,
		Behind:           0,
		IsGitRepo:        true,
		UncommittedCount: 0,
		UntrackedCount:   0,
	}

	assert.Equal(t, "main", status.Branch)
	assert.True(t, status.IsClean)
	assert.False(t, status.HasUncommitted)
	assert.False(t, status.HasUntracked)
	assert.Equal(t, 0, status.Ahead)
	assert.Equal(t, 0, status.Behind)
	assert.True(t, status.IsGitRepo)
	assert.Equal(t, 0, status.UncommittedCount)
	assert.Equal(t, 0, status.UntrackedCount)
}

func TestIsGitRepo(t *testing.T) {
	t.Run("returns true in git repository", func(t *testing.T) {
		wd, err := os.Getwd()
		require.NoError(t, err)

		// Verify we're in a git repo first
		cmd := exec.Command("git", "rev-parse", "--is-inside-work-tree")
		cmd.Dir = wd
		if err := cmd.Run(); err != nil {
			t.Skip("Not running in a git repository")
		}

		result := isGitRepo(wd)
		assert.True(t, result)
	})

	t.Run("returns false outside git repository", func(t *testing.T) {
		tempDir := t.TempDir()

		result := isGitRepo(tempDir)
		assert.False(t, result)
	})
}

func TestGetBranch(t *testing.T) {
	t.Run("returns branch name in git repo", func(t *testing.T) {
		wd, err := os.Getwd()
		require.NoError(t, err)

		// Verify we're in a git repo first
		cmd := exec.Command("git", "rev-parse", "--is-inside-work-tree")
		cmd.Dir = wd
		if err := cmd.Run(); err != nil {
			t.Skip("Not running in a git repository")
		}

		branch := getBranch(wd)
		assert.NotEqual(t, "unknown", branch)
		assert.NotEmpty(t, branch)
	})

	t.Run("returns unknown outside git repo", func(t *testing.T) {
		tempDir := t.TempDir()

		branch := getBranch(tempDir)
		assert.Equal(t, "unknown", branch)
	})
}

func TestGetStatus(t *testing.T) {
	t.Run("returns status for git repository", func(t *testing.T) {
		wd, err := os.Getwd()
		require.NoError(t, err)

		// Verify we're in a git repo first
		cmd := exec.Command("git", "rev-parse", "--is-inside-work-tree")
		cmd.Dir = wd
		if err := cmd.Run(); err != nil {
			t.Skip("Not running in a git repository")
		}

		status := GetStatus(wd)

		assert.True(t, status.IsGitRepo)
		assert.NotEmpty(t, status.Branch)
	})

	t.Run("returns not a git repo for non-git directory", func(t *testing.T) {
		tempDir := t.TempDir()

		status := GetStatus(tempDir)

		assert.False(t, status.IsGitRepo)
		assert.Empty(t, status.Branch)
	})
}

func TestGetStatusCmd(t *testing.T) {
	t.Run("returns a tea.Cmd", func(t *testing.T) {
		wd, err := os.Getwd()
		require.NoError(t, err)

		cmd := GetStatusCmd(wd)

		require.NotNil(t, cmd)

		// Execute the command
		msg := cmd()
		statusMsg, ok := msg.(StatusMsg)

		assert.True(t, ok)
		assert.Nil(t, statusMsg.Error)
	})
}

func TestHasUntracked(t *testing.T) {
	t.Run("returns false for non-git directory", func(t *testing.T) {
		tempDir := t.TempDir()

		has, count := hasUntracked(tempDir)

		assert.False(t, has)
		assert.Equal(t, 0, count)
	})
}

func TestHasUncommitted(t *testing.T) {
	t.Run("returns false for non-git directory", func(t *testing.T) {
		tempDir := t.TempDir()

		has, count := hasUncommitted(tempDir)

		assert.False(t, has)
		assert.Equal(t, 0, count)
	})
}

func TestGetAheadBehind(t *testing.T) {
	t.Run("returns zeros for non-git directory", func(t *testing.T) {
		tempDir := t.TempDir()

		ahead, behind := getAheadBehind(tempDir)

		assert.Equal(t, 0, ahead)
		assert.Equal(t, 0, behind)
	})
}
