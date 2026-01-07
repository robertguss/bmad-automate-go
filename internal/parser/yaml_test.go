package parser

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/robertguss/bmad-automate-go/internal/config"
	"github.com/robertguss/bmad-automate-go/internal/domain"
)

func createTestConfig(t *testing.T, sprintStatusContent string) *config.Config {
	t.Helper()

	tempDir := t.TempDir()
	sprintStatusPath := filepath.Join(tempDir, "sprint-status.yaml")
	storyDir := filepath.Join(tempDir, "stories")

	if err := os.WriteFile(sprintStatusPath, []byte(sprintStatusContent), 0644); err != nil {
		t.Fatalf("failed to write sprint status file: %v", err)
	}

	if err := os.MkdirAll(storyDir, 0755); err != nil {
		t.Fatalf("failed to create story dir: %v", err)
	}

	return &config.Config{
		SprintStatusPath: sprintStatusPath,
		StoryDir:         storyDir,
		WorkingDir:       tempDir,
	}
}

func TestParseSprintStatus(t *testing.T) {
	t.Run("parses valid file", func(t *testing.T) {
		cfg := createTestConfig(t, `development_status:
  3-1-user-auth: in-progress
  3-2-user-profile: ready-for-dev
  4-1-dashboard: backlog
`)

		stories, err := ParseSprintStatus(cfg)
		require.NoError(t, err)
		assert.Len(t, stories, 3)
	})

	t.Run("parses empty file", func(t *testing.T) {
		cfg := createTestConfig(t, `development_status:
`)

		stories, err := ParseSprintStatus(cfg)
		require.NoError(t, err)
		assert.Len(t, stories, 0)
	})

	t.Run("returns error for missing file", func(t *testing.T) {
		cfg := &config.Config{
			SprintStatusPath: "/nonexistent/path/sprint-status.yaml",
		}

		_, err := ParseSprintStatus(cfg)
		assert.Error(t, err)
	})

	t.Run("returns error for malformed YAML", func(t *testing.T) {
		cfg := createTestConfig(t, `development_status
  invalid yaml here
`)

		_, err := ParseSprintStatus(cfg)
		assert.Error(t, err)
	})

	t.Run("skips invalid story keys", func(t *testing.T) {
		cfg := createTestConfig(t, `development_status:
  invalid-key: in-progress
  another-bad: ready-for-dev
  3-1-valid-key: backlog
`)

		stories, err := ParseSprintStatus(cfg)
		require.NoError(t, err)
		assert.Len(t, stories, 1)
		assert.Equal(t, "3-1-valid-key", stories[0].Key)
	})

	t.Run("sorts stories by epic then key", func(t *testing.T) {
		cfg := createTestConfig(t, `development_status:
  4-2-later: in-progress
  3-1-first: ready-for-dev
  4-1-third: backlog
  3-2-second: done
`)

		stories, err := ParseSprintStatus(cfg)
		require.NoError(t, err)
		require.Len(t, stories, 4)

		assert.Equal(t, "3-1-first", stories[0].Key)
		assert.Equal(t, "3-2-second", stories[1].Key)
		assert.Equal(t, "4-1-third", stories[2].Key)
		assert.Equal(t, "4-2-later", stories[3].Key)
	})

	t.Run("extracts epic number correctly", func(t *testing.T) {
		cfg := createTestConfig(t, `development_status:
  10-5-large-epic: in-progress
`)

		stories, err := ParseSprintStatus(cfg)
		require.NoError(t, err)
		require.Len(t, stories, 1)
		assert.Equal(t, 10, stories[0].Epic)
	})

	t.Run("maps status correctly", func(t *testing.T) {
		cfg := createTestConfig(t, `development_status:
  3-1-test: in-progress
`)

		stories, err := ParseSprintStatus(cfg)
		require.NoError(t, err)
		require.Len(t, stories, 1)
		assert.Equal(t, domain.StatusInProgress, stories[0].Status)
	})
}

func TestExtractEpic(t *testing.T) {
	tests := []struct {
		name     string
		key      string
		expected int
	}{
		{
			name:     "simple epic",
			key:      "3-1-story",
			expected: 3,
		},
		{
			name:     "double digit epic",
			key:      "10-2-story",
			expected: 10,
		},
		{
			name:     "single digit story number",
			key:      "5-1-my-feature",
			expected: 5,
		},
		{
			name:     "invalid format returns 0",
			key:      "invalid",
			expected: 0,
		},
		{
			name:     "empty string returns 0",
			key:      "",
			expected: 0,
		},
		{
			name:     "non-numeric epic returns 0",
			key:      "abc-1-story",
			expected: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := extractEpic(tt.key)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestFilterStoriesByStatus(t *testing.T) {
	stories := []domain.Story{
		{Key: "3-1-test", Status: domain.StatusInProgress},
		{Key: "3-2-test", Status: domain.StatusReadyForDev},
		{Key: "3-3-test", Status: domain.StatusInProgress},
		{Key: "3-4-test", Status: domain.StatusDone},
	}

	t.Run("filters matching status", func(t *testing.T) {
		filtered := FilterStoriesByStatus(stories, domain.StatusInProgress)
		assert.Len(t, filtered, 2)
	})

	t.Run("returns empty for no matches", func(t *testing.T) {
		filtered := FilterStoriesByStatus(stories, domain.StatusBlocked)
		assert.Len(t, filtered, 0)
	})

	t.Run("handles empty input", func(t *testing.T) {
		filtered := FilterStoriesByStatus([]domain.Story{}, domain.StatusInProgress)
		assert.Len(t, filtered, 0)
	})
}

func TestFilterStoriesByEpic(t *testing.T) {
	stories := []domain.Story{
		{Key: "3-1-test", Epic: 3},
		{Key: "3-2-test", Epic: 3},
		{Key: "4-1-test", Epic: 4},
		{Key: "5-1-test", Epic: 5},
	}

	t.Run("filters matching epic", func(t *testing.T) {
		filtered := FilterStoriesByEpic(stories, 3)
		assert.Len(t, filtered, 2)
	})

	t.Run("returns all for epic 0", func(t *testing.T) {
		filtered := FilterStoriesByEpic(stories, 0)
		assert.Len(t, filtered, 4)
	})

	t.Run("returns empty for no matches", func(t *testing.T) {
		filtered := FilterStoriesByEpic(stories, 99)
		assert.Len(t, filtered, 0)
	})

	t.Run("handles empty input", func(t *testing.T) {
		filtered := FilterStoriesByEpic([]domain.Story{}, 3)
		assert.Len(t, filtered, 0)
	})
}

func TestGetActionableStories(t *testing.T) {
	stories := []domain.Story{
		{Key: "3-1-test", Status: domain.StatusBacklog},
		{Key: "3-2-test", Status: domain.StatusInProgress},
		{Key: "3-3-test", Status: domain.StatusDone},
		{Key: "3-4-test", Status: domain.StatusReadyForDev},
		{Key: "3-5-test", Status: domain.StatusBlocked},
		{Key: "3-6-test", Status: domain.StatusInProgress},
	}

	t.Run("returns actionable in priority order", func(t *testing.T) {
		actionable := GetActionableStories(stories)
		require.Len(t, actionable, 4)

		// in-progress should come first
		assert.Equal(t, domain.StatusInProgress, actionable[0].Status)
		assert.Equal(t, domain.StatusInProgress, actionable[1].Status)
		// then ready-for-dev
		assert.Equal(t, domain.StatusReadyForDev, actionable[2].Status)
		// then backlog
		assert.Equal(t, domain.StatusBacklog, actionable[3].Status)
	})

	t.Run("excludes done and blocked", func(t *testing.T) {
		actionable := GetActionableStories(stories)
		for _, s := range actionable {
			assert.NotEqual(t, domain.StatusDone, s.Status)
			assert.NotEqual(t, domain.StatusBlocked, s.Status)
		}
	})

	t.Run("handles empty input", func(t *testing.T) {
		actionable := GetActionableStories([]domain.Story{})
		assert.Len(t, actionable, 0)
	})
}

func TestCountByStatus(t *testing.T) {
	stories := []domain.Story{
		{Key: "3-1-test", Status: domain.StatusInProgress},
		{Key: "3-2-test", Status: domain.StatusInProgress},
		{Key: "3-3-test", Status: domain.StatusDone},
		{Key: "3-4-test", Status: domain.StatusBacklog},
	}

	t.Run("counts by status", func(t *testing.T) {
		counts := CountByStatus(stories)
		assert.Equal(t, 2, counts[domain.StatusInProgress])
		assert.Equal(t, 1, counts[domain.StatusDone])
		assert.Equal(t, 1, counts[domain.StatusBacklog])
		assert.Equal(t, 0, counts[domain.StatusBlocked])
	})

	t.Run("handles empty input", func(t *testing.T) {
		counts := CountByStatus([]domain.Story{})
		assert.Empty(t, counts)
	})
}

func TestGetUniqueEpics(t *testing.T) {
	stories := []domain.Story{
		{Key: "3-1-test", Epic: 3},
		{Key: "3-2-test", Epic: 3},
		{Key: "5-1-test", Epic: 5},
		{Key: "4-1-test", Epic: 4},
	}

	t.Run("returns unique epics sorted", func(t *testing.T) {
		epics := GetUniqueEpics(stories)
		require.Len(t, epics, 3)
		assert.Equal(t, 3, epics[0])
		assert.Equal(t, 4, epics[1])
		assert.Equal(t, 5, epics[2])
	})

	t.Run("handles empty input", func(t *testing.T) {
		epics := GetUniqueEpics([]domain.Story{})
		assert.Len(t, epics, 0)
	})

	t.Run("handles single epic", func(t *testing.T) {
		singleEpicStories := []domain.Story{
			{Key: "3-1-test", Epic: 3},
			{Key: "3-2-test", Epic: 3},
		}
		epics := GetUniqueEpics(singleEpicStories)
		assert.Len(t, epics, 1)
		assert.Equal(t, 3, epics[0])
	})
}

func TestStoryKeyPattern(t *testing.T) {
	tests := []struct {
		name     string
		key      string
		expected bool
	}{
		{
			name:     "valid simple key",
			key:      "3-1-story",
			expected: true,
		},
		{
			name:     "valid with longer name",
			key:      "3-1-user-authentication",
			expected: true,
		},
		{
			name:     "valid double digit epic",
			key:      "10-5-feature",
			expected: true,
		},
		{
			name:     "invalid no numbers",
			key:      "invalid-key",
			expected: false,
		},
		{
			name:     "invalid single number",
			key:      "3-story",
			expected: false,
		},
		{
			name:     "invalid empty",
			key:      "",
			expected: false,
		},
		{
			name:     "invalid missing name",
			key:      "3-1-",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := storyKeyPattern.MatchString(tt.key)
			assert.Equal(t, tt.expected, result)
		})
	}
}
