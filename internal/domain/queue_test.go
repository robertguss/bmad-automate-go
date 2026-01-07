package domain

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func createTestStory(key string, status StoryStatus) Story {
	epic := 0
	if len(key) >= 3 && key[1] == '-' {
		epic = int(key[0] - '0')
	}
	return Story{
		Key:    key,
		Epic:   epic,
		Status: status,
	}
}

func TestNewQueue(t *testing.T) {
	q := NewQueue()

	assert.NotNil(t, q.Items)
	assert.Len(t, q.Items, 0)
	assert.Equal(t, QueueIdle, q.Status)
	assert.Equal(t, -1, q.Current)
	assert.NotNil(t, q.StepAverages)
	assert.Len(t, q.StepAverages, 0)
}

func TestQueue_Add(t *testing.T) {
	tests := []struct {
		name          string
		existingKeys  []string
		addKey        string
		expectedCount int
	}{
		{
			name:          "add to empty queue",
			existingKeys:  []string{},
			addKey:        "3-1-test",
			expectedCount: 1,
		},
		{
			name:          "add to non-empty queue",
			existingKeys:  []string{"3-1-first"},
			addKey:        "3-2-second",
			expectedCount: 2,
		},
		{
			name:          "duplicate prevention",
			existingKeys:  []string{"3-1-test"},
			addKey:        "3-1-test",
			expectedCount: 1,
		},
		{
			name:          "add multiple unique",
			existingKeys:  []string{"3-1-first", "3-2-second"},
			addKey:        "3-3-third",
			expectedCount: 3,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			q := NewQueue()

			for _, key := range tt.existingKeys {
				q.Add(createTestStory(key, StatusInProgress))
			}

			q.Add(createTestStory(tt.addKey, StatusInProgress))

			assert.Equal(t, tt.expectedCount, q.TotalCount())
		})
	}
}

func TestQueue_Add_SetsCorrectFields(t *testing.T) {
	q := NewQueue()
	story := createTestStory("3-1-test", StatusInProgress)

	q.Add(story)

	require.Len(t, q.Items, 1)
	item := q.Items[0]

	assert.Equal(t, story.Key, item.Story.Key)
	assert.Equal(t, ExecutionPending, item.Status)
	assert.Equal(t, 1, item.Position)
	assert.False(t, item.AddedAt.IsZero())
	assert.Nil(t, item.Execution)
}

func TestQueue_AddMultiple(t *testing.T) {
	tests := []struct {
		name          string
		stories       []Story
		expectedCount int
	}{
		{
			name:          "add empty slice",
			stories:       []Story{},
			expectedCount: 0,
		},
		{
			name: "add single story",
			stories: []Story{
				createTestStory("3-1-test", StatusInProgress),
			},
			expectedCount: 1,
		},
		{
			name: "add multiple stories",
			stories: []Story{
				createTestStory("3-1-first", StatusInProgress),
				createTestStory("3-2-second", StatusReadyForDev),
				createTestStory("3-3-third", StatusBacklog),
			},
			expectedCount: 3,
		},
		{
			name: "add with duplicates filtered",
			stories: []Story{
				createTestStory("3-1-test", StatusInProgress),
				createTestStory("3-1-test", StatusInProgress),
				createTestStory("3-2-other", StatusInProgress),
			},
			expectedCount: 2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			q := NewQueue()
			q.AddMultiple(tt.stories)

			assert.Equal(t, tt.expectedCount, q.TotalCount())
		})
	}
}

func TestQueue_Remove(t *testing.T) {
	tests := []struct {
		name           string
		existingKeys   []string
		removeKey      string
		queueStatus    QueueStatus
		currentIndex   int
		expectedResult bool
		expectedCount  int
	}{
		{
			name:           "remove existing item",
			existingKeys:   []string{"3-1-first", "3-2-second"},
			removeKey:      "3-1-first",
			queueStatus:    QueueIdle,
			currentIndex:   -1,
			expectedResult: true,
			expectedCount:  1,
		},
		{
			name:           "remove non-existent item",
			existingKeys:   []string{"3-1-first"},
			removeKey:      "3-2-second",
			queueStatus:    QueueIdle,
			currentIndex:   -1,
			expectedResult: false,
			expectedCount:  1,
		},
		{
			name:           "remove from empty queue",
			existingKeys:   []string{},
			removeKey:      "3-1-test",
			queueStatus:    QueueIdle,
			currentIndex:   -1,
			expectedResult: false,
			expectedCount:  0,
		},
		{
			name:           "cannot remove currently running item",
			existingKeys:   []string{"3-1-first", "3-2-second"},
			removeKey:      "3-1-first",
			queueStatus:    QueueRunning,
			currentIndex:   0,
			expectedResult: false,
			expectedCount:  2,
		},
		{
			name:           "can remove non-running item when queue running",
			existingKeys:   []string{"3-1-first", "3-2-second"},
			removeKey:      "3-2-second",
			queueStatus:    QueueRunning,
			currentIndex:   0,
			expectedResult: true,
			expectedCount:  1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			q := NewQueue()
			for _, key := range tt.existingKeys {
				q.Add(createTestStory(key, StatusInProgress))
			}
			q.Status = tt.queueStatus
			q.Current = tt.currentIndex

			result := q.Remove(tt.removeKey)

			assert.Equal(t, tt.expectedResult, result)
			assert.Equal(t, tt.expectedCount, q.TotalCount())
		})
	}
}

func TestQueue_Remove_AdjustsCurrentIndex(t *testing.T) {
	q := NewQueue()
	q.Add(createTestStory("3-1-first", StatusInProgress))
	q.Add(createTestStory("3-2-second", StatusInProgress))
	q.Add(createTestStory("3-3-third", StatusInProgress))
	q.Current = 2 // Point to third item

	q.Remove("3-1-first") // Remove first item

	assert.Equal(t, 1, q.Current, "Current index should be adjusted when removing item before it")
}

func TestQueue_Clear(t *testing.T) {
	t.Run("clears idle queue completely", func(t *testing.T) {
		q := NewQueue()
		q.Add(createTestStory("3-1-first", StatusInProgress))
		q.Add(createTestStory("3-2-second", StatusInProgress))
		q.Status = QueueIdle

		q.Clear()

		assert.Equal(t, 0, q.TotalCount())
		assert.Equal(t, -1, q.Current)
	})

	t.Run("keeps current item when running", func(t *testing.T) {
		q := NewQueue()
		q.Add(createTestStory("3-1-first", StatusInProgress))
		q.Add(createTestStory("3-2-second", StatusInProgress))
		q.Add(createTestStory("3-3-third", StatusInProgress))
		q.Status = QueueRunning
		q.Current = 1 // Running second item

		q.Clear()

		assert.Equal(t, 2, q.TotalCount()) // First and second remain
	})

	t.Run("clears empty queue without error", func(t *testing.T) {
		q := NewQueue()
		q.Clear()

		assert.Equal(t, 0, q.TotalCount())
	})
}

func TestQueue_MoveUp(t *testing.T) {
	tests := []struct {
		name           string
		setupStatuses  []ExecutionStatus
		moveIndex      int
		expectedResult bool
		expectedOrder  []string
	}{
		{
			name:           "move second item up",
			setupStatuses:  []ExecutionStatus{ExecutionPending, ExecutionPending, ExecutionPending},
			moveIndex:      1,
			expectedResult: true,
			expectedOrder:  []string{"3-2-second", "3-1-first", "3-3-third"},
		},
		{
			name:           "cannot move first item up",
			setupStatuses:  []ExecutionStatus{ExecutionPending, ExecutionPending},
			moveIndex:      0,
			expectedResult: false,
			expectedOrder:  []string{"3-1-first", "3-2-second"},
		},
		{
			name:           "cannot move negative index",
			setupStatuses:  []ExecutionStatus{ExecutionPending, ExecutionPending},
			moveIndex:      -1,
			expectedResult: false,
			expectedOrder:  []string{"3-1-first", "3-2-second"},
		},
		{
			name:           "cannot move index beyond length",
			setupStatuses:  []ExecutionStatus{ExecutionPending, ExecutionPending},
			moveIndex:      5,
			expectedResult: false,
			expectedOrder:  []string{"3-1-first", "3-2-second"},
		},
		{
			name:           "cannot move non-pending item",
			setupStatuses:  []ExecutionStatus{ExecutionPending, ExecutionRunning},
			moveIndex:      1,
			expectedResult: false,
			expectedOrder:  []string{"3-1-first", "3-2-second"},
		},
		{
			name:           "cannot swap with non-pending item above",
			setupStatuses:  []ExecutionStatus{ExecutionCompleted, ExecutionPending},
			moveIndex:      1,
			expectedResult: false,
			expectedOrder:  []string{"3-1-first", "3-2-second"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			q := NewQueue()
			keys := []string{"3-1-first", "3-2-second", "3-3-third"}
			for i := 0; i < len(tt.setupStatuses) && i < len(keys); i++ {
				q.Add(createTestStory(keys[i], StatusInProgress))
				q.Items[i].Status = tt.setupStatuses[i]
			}

			result := q.MoveUp(tt.moveIndex)

			assert.Equal(t, tt.expectedResult, result)
			for i, expectedKey := range tt.expectedOrder {
				if i < len(q.Items) {
					assert.Equal(t, expectedKey, q.Items[i].Story.Key)
				}
			}
		})
	}
}

func TestQueue_MoveDown(t *testing.T) {
	tests := []struct {
		name           string
		setupStatuses  []ExecutionStatus
		moveIndex      int
		expectedResult bool
		expectedOrder  []string
	}{
		{
			name:           "move first item down",
			setupStatuses:  []ExecutionStatus{ExecutionPending, ExecutionPending, ExecutionPending},
			moveIndex:      0,
			expectedResult: true,
			expectedOrder:  []string{"3-2-second", "3-1-first", "3-3-third"},
		},
		{
			name:           "cannot move last item down",
			setupStatuses:  []ExecutionStatus{ExecutionPending, ExecutionPending},
			moveIndex:      1,
			expectedResult: false,
			expectedOrder:  []string{"3-1-first", "3-2-second"},
		},
		{
			name:           "cannot move negative index",
			setupStatuses:  []ExecutionStatus{ExecutionPending, ExecutionPending},
			moveIndex:      -1,
			expectedResult: false,
			expectedOrder:  []string{"3-1-first", "3-2-second"},
		},
		{
			name:           "cannot move non-pending item",
			setupStatuses:  []ExecutionStatus{ExecutionRunning, ExecutionPending},
			moveIndex:      0,
			expectedResult: false,
			expectedOrder:  []string{"3-1-first", "3-2-second"},
		},
		{
			name:           "cannot swap with non-pending item below",
			setupStatuses:  []ExecutionStatus{ExecutionPending, ExecutionCompleted},
			moveIndex:      0,
			expectedResult: false,
			expectedOrder:  []string{"3-1-first", "3-2-second"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			q := NewQueue()
			keys := []string{"3-1-first", "3-2-second", "3-3-third"}
			for i := 0; i < len(tt.setupStatuses) && i < len(keys); i++ {
				q.Add(createTestStory(keys[i], StatusInProgress))
				q.Items[i].Status = tt.setupStatuses[i]
			}

			result := q.MoveDown(tt.moveIndex)

			assert.Equal(t, tt.expectedResult, result)
			for i, expectedKey := range tt.expectedOrder {
				if i < len(q.Items) {
					assert.Equal(t, expectedKey, q.Items[i].Story.Key)
				}
			}
		})
	}
}

func TestQueue_GetPending(t *testing.T) {
	tests := []struct {
		name          string
		statuses      []ExecutionStatus
		expectedCount int
	}{
		{
			name:          "all pending",
			statuses:      []ExecutionStatus{ExecutionPending, ExecutionPending, ExecutionPending},
			expectedCount: 3,
		},
		{
			name:          "mixed statuses",
			statuses:      []ExecutionStatus{ExecutionCompleted, ExecutionPending, ExecutionFailed},
			expectedCount: 1,
		},
		{
			name:          "none pending",
			statuses:      []ExecutionStatus{ExecutionCompleted, ExecutionFailed},
			expectedCount: 0,
		},
		{
			name:          "empty queue",
			statuses:      []ExecutionStatus{},
			expectedCount: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			q := NewQueue()
			for i, status := range tt.statuses {
				q.Add(createTestStory("3-"+string(rune('1'+i))+"-test", StatusInProgress))
				q.Items[i].Status = status
			}

			pending := q.GetPending()

			assert.Len(t, pending, tt.expectedCount)
			for _, item := range pending {
				assert.Equal(t, ExecutionPending, item.Status)
			}
		})
	}
}

func TestQueue_GetCompleted(t *testing.T) {
	tests := []struct {
		name          string
		statuses      []ExecutionStatus
		expectedCount int
	}{
		{
			name:          "all completed",
			statuses:      []ExecutionStatus{ExecutionCompleted, ExecutionCompleted},
			expectedCount: 2,
		},
		{
			name:          "includes failed",
			statuses:      []ExecutionStatus{ExecutionCompleted, ExecutionFailed},
			expectedCount: 2,
		},
		{
			name:          "excludes pending",
			statuses:      []ExecutionStatus{ExecutionCompleted, ExecutionPending},
			expectedCount: 1,
		},
		{
			name:          "excludes running",
			statuses:      []ExecutionStatus{ExecutionCompleted, ExecutionRunning},
			expectedCount: 1,
		},
		{
			name:          "none completed",
			statuses:      []ExecutionStatus{ExecutionPending, ExecutionRunning},
			expectedCount: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			q := NewQueue()
			for i, status := range tt.statuses {
				q.Add(createTestStory("3-"+string(rune('1'+i))+"-test", StatusInProgress))
				q.Items[i].Status = status
			}

			completed := q.GetCompleted()

			assert.Len(t, completed, tt.expectedCount)
		})
	}
}

func TestQueue_CurrentItem(t *testing.T) {
	t.Run("returns nil when current is -1", func(t *testing.T) {
		q := NewQueue()
		q.Add(createTestStory("3-1-test", StatusInProgress))
		q.Current = -1

		assert.Nil(t, q.CurrentItem())
	})

	t.Run("returns nil when current is beyond length", func(t *testing.T) {
		q := NewQueue()
		q.Add(createTestStory("3-1-test", StatusInProgress))
		q.Current = 5

		assert.Nil(t, q.CurrentItem())
	})

	t.Run("returns correct item", func(t *testing.T) {
		q := NewQueue()
		q.Add(createTestStory("3-1-first", StatusInProgress))
		q.Add(createTestStory("3-2-second", StatusInProgress))
		q.Current = 1

		item := q.CurrentItem()
		require.NotNil(t, item)
		assert.Equal(t, "3-2-second", item.Story.Key)
	})
}

func TestQueue_NextPending(t *testing.T) {
	t.Run("returns first pending item", func(t *testing.T) {
		q := NewQueue()
		q.Add(createTestStory("3-1-first", StatusInProgress))
		q.Add(createTestStory("3-2-second", StatusInProgress))
		q.Items[0].Status = ExecutionCompleted

		next := q.NextPending()
		require.NotNil(t, next)
		assert.Equal(t, "3-2-second", next.Story.Key)
	})

	t.Run("returns nil when no pending", func(t *testing.T) {
		q := NewQueue()
		q.Add(createTestStory("3-1-test", StatusInProgress))
		q.Items[0].Status = ExecutionCompleted

		assert.Nil(t, q.NextPending())
	})

	t.Run("returns nil for empty queue", func(t *testing.T) {
		q := NewQueue()
		assert.Nil(t, q.NextPending())
	})
}

func TestQueue_ProgressPercent(t *testing.T) {
	tests := []struct {
		name             string
		statuses         []ExecutionStatus
		currentExecution *Execution
		currentIndex     int
		expectedPercent  float64
	}{
		{
			name:            "empty queue",
			statuses:        []ExecutionStatus{},
			expectedPercent: 0,
		},
		{
			name:            "all pending",
			statuses:        []ExecutionStatus{ExecutionPending, ExecutionPending},
			expectedPercent: 0,
		},
		{
			name:            "half completed",
			statuses:        []ExecutionStatus{ExecutionCompleted, ExecutionPending},
			expectedPercent: 50,
		},
		{
			name:            "all completed",
			statuses:        []ExecutionStatus{ExecutionCompleted, ExecutionCompleted},
			expectedPercent: 100,
		},
		{
			name:            "failed counts as complete",
			statuses:        []ExecutionStatus{ExecutionCompleted, ExecutionFailed},
			expectedPercent: 100,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			q := NewQueue()
			for i, status := range tt.statuses {
				q.Add(createTestStory("3-"+string(rune('1'+i))+"-test", StatusInProgress))
				q.Items[i].Status = status
			}

			percent := q.ProgressPercent()

			assert.Equal(t, tt.expectedPercent, percent)
		})
	}
}

func TestQueue_ProgressPercent_WithCurrentItemProgress(t *testing.T) {
	q := NewQueue()
	q.Add(createTestStory("3-1-first", StatusInProgress))
	q.Add(createTestStory("3-2-second", StatusInProgress))

	// Set up current item with 50% progress (2 of 4 steps complete)
	exec := NewExecution(q.Items[0].Story)
	exec.Steps[0].Status = StepSuccess
	exec.Steps[1].Status = StepSuccess
	q.Items[0].Execution = exec
	q.Items[0].Status = ExecutionRunning
	q.Current = 0

	percent := q.ProgressPercent()

	// Expected: (0 completed + 0.5 current progress) / 2 items * 100 = 25%
	assert.Equal(t, 25.0, percent)
}

func TestQueue_EstimatedTimeRemaining(t *testing.T) {
	t.Run("default estimate when no history", func(t *testing.T) {
		q := NewQueue()
		q.Add(createTestStory("3-1-test", StatusInProgress))

		eta := q.EstimatedTimeRemaining()

		// 1 pending item * 20 minutes = 20 minutes
		assert.Equal(t, 20*time.Minute, eta)
	})

	t.Run("calculates from step averages", func(t *testing.T) {
		q := NewQueue()
		q.Add(createTestStory("3-1-test", StatusInProgress))
		q.Add(createTestStory("3-2-test", StatusInProgress))

		// Set step averages: 1 min per step * 4 steps = 4 min per story
		for _, step := range AllSteps() {
			q.StepAverages[step] = time.Minute
		}

		eta := q.EstimatedTimeRemaining()

		// 2 pending items * 4 minutes = 8 minutes
		assert.Equal(t, 8*time.Minute, eta)
	})

	t.Run("returns zero for completed queue", func(t *testing.T) {
		q := NewQueue()
		q.Add(createTestStory("3-1-test", StatusInProgress))
		q.Items[0].Status = ExecutionCompleted

		for _, step := range AllSteps() {
			q.StepAverages[step] = time.Minute
		}

		eta := q.EstimatedTimeRemaining()

		assert.Equal(t, time.Duration(0), eta)
	})
}

func TestQueue_UpdateStepAverage(t *testing.T) {
	t.Run("sets first value", func(t *testing.T) {
		q := NewQueue()
		q.UpdateStepAverage(StepCreateStory, 10*time.Second)

		assert.Equal(t, 10*time.Second, q.StepAverages[StepCreateStory])
	})

	t.Run("calculates moving average", func(t *testing.T) {
		q := NewQueue()
		q.StepAverages[StepCreateStory] = 10 * time.Second

		q.UpdateStepAverage(StepCreateStory, 20*time.Second)

		// (10 + 20) / 2 = 15 seconds
		assert.Equal(t, 15*time.Second, q.StepAverages[StepCreateStory])
	})

	t.Run("handles multiple updates", func(t *testing.T) {
		q := NewQueue()

		q.UpdateStepAverage(StepCreateStory, 10*time.Second)
		q.UpdateStepAverage(StepCreateStory, 20*time.Second)
		q.UpdateStepAverage(StepCreateStory, 30*time.Second)

		// (10+20)/2 = 15, then (15+30)/2 = 22.5 -> truncated to 22s
		expected := (15*time.Second + 30*time.Second) / 2
		assert.Equal(t, expected, q.StepAverages[StepCreateStory])
	})
}

func TestQueue_Contains(t *testing.T) {
	tests := []struct {
		name         string
		existingKeys []string
		checkKey     string
		expected     bool
	}{
		{
			name:         "exists in queue",
			existingKeys: []string{"3-1-first", "3-2-second"},
			checkKey:     "3-1-first",
			expected:     true,
		},
		{
			name:         "does not exist",
			existingKeys: []string{"3-1-first"},
			checkKey:     "3-2-second",
			expected:     false,
		},
		{
			name:         "empty queue",
			existingKeys: []string{},
			checkKey:     "3-1-test",
			expected:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			q := NewQueue()
			for _, key := range tt.existingKeys {
				q.Add(createTestStory(key, StatusInProgress))
			}

			assert.Equal(t, tt.expected, q.Contains(tt.checkKey))
		})
	}
}

func TestQueue_CountMethods(t *testing.T) {
	q := NewQueue()
	q.Add(createTestStory("3-1-pending", StatusInProgress))
	q.Add(createTestStory("3-2-completed", StatusInProgress))
	q.Add(createTestStory("3-3-failed", StatusInProgress))
	q.Add(createTestStory("3-4-running", StatusInProgress))

	q.Items[0].Status = ExecutionPending
	q.Items[1].Status = ExecutionCompleted
	q.Items[2].Status = ExecutionFailed
	q.Items[3].Status = ExecutionRunning

	t.Run("TotalCount", func(t *testing.T) {
		assert.Equal(t, 4, q.TotalCount())
	})

	t.Run("PendingCount", func(t *testing.T) {
		assert.Equal(t, 1, q.PendingCount())
	})

	t.Run("CompletedCount", func(t *testing.T) {
		assert.Equal(t, 1, q.CompletedCount())
	})

	t.Run("FailedCount", func(t *testing.T) {
		assert.Equal(t, 1, q.FailedCount())
	})
}

func TestQueue_IsEmpty(t *testing.T) {
	t.Run("empty queue", func(t *testing.T) {
		q := NewQueue()
		assert.True(t, q.IsEmpty())
	})

	t.Run("non-empty queue", func(t *testing.T) {
		q := NewQueue()
		q.Add(createTestStory("3-1-test", StatusInProgress))
		assert.False(t, q.IsEmpty())
	})
}

func TestQueue_HasPending(t *testing.T) {
	t.Run("has pending", func(t *testing.T) {
		q := NewQueue()
		q.Add(createTestStory("3-1-test", StatusInProgress))
		assert.True(t, q.HasPending())
	})

	t.Run("no pending", func(t *testing.T) {
		q := NewQueue()
		q.Add(createTestStory("3-1-test", StatusInProgress))
		q.Items[0].Status = ExecutionCompleted
		assert.False(t, q.HasPending())
	})

	t.Run("empty queue", func(t *testing.T) {
		q := NewQueue()
		assert.False(t, q.HasPending())
	})
}

func TestQueue_GetItem(t *testing.T) {
	q := NewQueue()
	q.Add(createTestStory("3-1-first", StatusInProgress))
	q.Add(createTestStory("3-2-second", StatusInProgress))

	t.Run("valid index", func(t *testing.T) {
		item := q.GetItem(0)
		require.NotNil(t, item)
		assert.Equal(t, "3-1-first", item.Story.Key)
	})

	t.Run("negative index", func(t *testing.T) {
		assert.Nil(t, q.GetItem(-1))
	})

	t.Run("index beyond length", func(t *testing.T) {
		assert.Nil(t, q.GetItem(10))
	})
}

func TestQueue_IndexOf(t *testing.T) {
	q := NewQueue()
	q.Add(createTestStory("3-1-first", StatusInProgress))
	q.Add(createTestStory("3-2-second", StatusInProgress))

	t.Run("existing key", func(t *testing.T) {
		assert.Equal(t, 0, q.IndexOf("3-1-first"))
		assert.Equal(t, 1, q.IndexOf("3-2-second"))
	})

	t.Run("non-existing key", func(t *testing.T) {
		assert.Equal(t, -1, q.IndexOf("3-3-third"))
	})

	t.Run("empty queue", func(t *testing.T) {
		emptyQ := NewQueue()
		assert.Equal(t, -1, emptyQ.IndexOf("3-1-test"))
	})
}

func TestQueueStatus_Constants(t *testing.T) {
	tests := []struct {
		name     string
		status   QueueStatus
		expected string
	}{
		{
			name:     "QueueIdle value",
			status:   QueueIdle,
			expected: "idle",
		},
		{
			name:     "QueueRunning value",
			status:   QueueRunning,
			expected: "running",
		},
		{
			name:     "QueuePaused value",
			status:   QueuePaused,
			expected: "paused",
		},
		{
			name:     "QueueCompleted value",
			status:   QueueCompleted,
			expected: "completed",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, string(tt.status))
		})
	}
}
