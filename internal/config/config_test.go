package config

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNew(t *testing.T) {
	cfg := New()

	t.Run("sets default timeout", func(t *testing.T) {
		assert.Equal(t, DefaultTimeout, cfg.Timeout)
	})

	t.Run("sets default retries", func(t *testing.T) {
		assert.Equal(t, DefaultRetries, cfg.Retries)
	})

	t.Run("sets default theme", func(t *testing.T) {
		assert.Equal(t, "catppuccin", cfg.Theme)
	})

	t.Run("sets default workflow", func(t *testing.T) {
		assert.Equal(t, "default", cfg.ActiveWorkflow)
	})

	t.Run("sets default max workers", func(t *testing.T) {
		assert.Equal(t, DefaultMaxWorkers, cfg.MaxWorkers)
	})

	t.Run("sets default API port", func(t *testing.T) {
		assert.Equal(t, DefaultAPIPort, cfg.APIPort)
	})

	t.Run("sets default watch debounce", func(t *testing.T) {
		assert.Equal(t, DefaultWatchDebounce, cfg.WatchDebounce)
	})

	t.Run("sound disabled by default", func(t *testing.T) {
		assert.False(t, cfg.SoundEnabled)
	})

	t.Run("notifications enabled by default", func(t *testing.T) {
		assert.True(t, cfg.NotificationsEnabled)
	})

	t.Run("watch disabled by default", func(t *testing.T) {
		assert.False(t, cfg.WatchEnabled)
	})

	t.Run("parallel disabled by default", func(t *testing.T) {
		assert.False(t, cfg.ParallelEnabled)
	})

	t.Run("API disabled by default", func(t *testing.T) {
		assert.False(t, cfg.APIEnabled)
	})

	t.Run("sets paths relative to working directory", func(t *testing.T) {
		wd, _ := os.Getwd()
		assert.Contains(t, cfg.SprintStatusPath, wd)
		assert.Contains(t, cfg.StoryDir, wd)
		assert.Contains(t, cfg.DataDir, wd)
		assert.Contains(t, cfg.DatabasePath, wd)
	})
}

func TestConfig_StoryFilePath(t *testing.T) {
	tests := []struct {
		name     string
		storyKey string
		storyDir string
		expected string
	}{
		{
			name:     "simple key",
			storyKey: "3-1-test",
			storyDir: "/stories",
			expected: "/stories/3-1-test.md",
		},
		{
			name:     "key with hyphens",
			storyKey: "3-1-user-authentication-flow",
			storyDir: "/stories",
			expected: "/stories/3-1-user-authentication-flow.md",
		},
		{
			name:     "empty key",
			storyKey: "",
			storyDir: "/stories",
			expected: "/stories/.md",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &Config{StoryDir: tt.storyDir}
			result := cfg.StoryFilePath(tt.storyKey)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestConfig_StoryFileExists(t *testing.T) {
	tempDir := t.TempDir()
	storyDir := filepath.Join(tempDir, "stories")
	os.MkdirAll(storyDir, 0755)

	cfg := &Config{StoryDir: storyDir}

	t.Run("returns true when file exists", func(t *testing.T) {
		storyPath := filepath.Join(storyDir, "3-1-test.md")
		os.WriteFile(storyPath, []byte("test content"), 0644)

		assert.True(t, cfg.StoryFileExists("3-1-test"))
	})

	t.Run("returns false when file does not exist", func(t *testing.T) {
		assert.False(t, cfg.StoryFileExists("3-2-nonexistent"))
	})
}

func TestConfig_EnsureDataDir(t *testing.T) {
	t.Run("creates directory if not exists", func(t *testing.T) {
		tempDir := t.TempDir()
		dataDir := filepath.Join(tempDir, "newdata")

		cfg := &Config{DataDir: dataDir}

		err := cfg.EnsureDataDir()
		require.NoError(t, err)

		info, err := os.Stat(dataDir)
		require.NoError(t, err)
		assert.True(t, info.IsDir())
	})

	t.Run("succeeds if directory already exists", func(t *testing.T) {
		tempDir := t.TempDir()
		dataDir := filepath.Join(tempDir, "existingdata")
		os.MkdirAll(dataDir, 0755)

		cfg := &Config{DataDir: dataDir}

		err := cfg.EnsureDataDir()
		assert.NoError(t, err)
	})

	t.Run("creates nested directories", func(t *testing.T) {
		tempDir := t.TempDir()
		dataDir := filepath.Join(tempDir, "a", "b", "c")

		cfg := &Config{DataDir: dataDir}

		err := cfg.EnsureDataDir()
		require.NoError(t, err)

		info, err := os.Stat(dataDir)
		require.NoError(t, err)
		assert.True(t, info.IsDir())
	})
}

func TestDefaultConstants(t *testing.T) {
	tests := []struct {
		name     string
		constant interface{}
		expected interface{}
	}{
		{
			name:     "DefaultSprintStatus",
			constant: DefaultSprintStatus,
			expected: "_bmad-output/implementation-artifacts/sprint-status.yaml",
		},
		{
			name:     "DefaultStoryDir",
			constant: DefaultStoryDir,
			expected: "_bmad-output/implementation-artifacts",
		},
		{
			name:     "DefaultTimeout",
			constant: DefaultTimeout,
			expected: 600,
		},
		{
			name:     "DefaultRetries",
			constant: DefaultRetries,
			expected: 1,
		},
		{
			name:     "DefaultDataDir",
			constant: DefaultDataDir,
			expected: ".bmad",
		},
		{
			name:     "DefaultDBName",
			constant: DefaultDBName,
			expected: "bmad.db",
		},
		{
			name:     "DefaultAPIPort",
			constant: DefaultAPIPort,
			expected: 8080,
		},
		{
			name:     "DefaultMaxWorkers",
			constant: DefaultMaxWorkers,
			expected: 1,
		},
		{
			name:     "DefaultWatchDebounce",
			constant: DefaultWatchDebounce,
			expected: 500,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.constant)
		})
	}
}
