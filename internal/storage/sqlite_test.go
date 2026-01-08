package storage

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/robertguss/bmad-automate-go/internal/domain"
)

func createTestStory(key string, epic int, status domain.StoryStatus) domain.Story {
	return domain.Story{
		Key:    key,
		Epic:   epic,
		Status: status,
		Title:  "Test Story: " + key,
	}
}

func createMinimalExecution(story domain.Story) *domain.Execution {
	exec := domain.NewExecution(story)
	exec.StartTime = time.Now()
	return exec
}

func createCompletedExecution(story domain.Story) *domain.Execution {
	exec := domain.NewExecution(story)
	exec.Status = domain.ExecutionCompleted
	exec.StartTime = time.Now().Add(-5 * time.Minute)
	exec.EndTime = time.Now()
	exec.Duration = exec.EndTime.Sub(exec.StartTime)

	for i, step := range exec.Steps {
		step.Status = domain.StepSuccess
		step.StartTime = exec.StartTime.Add(time.Duration(i) * time.Minute)
		step.EndTime = step.StartTime.Add(time.Minute)
		step.Duration = time.Minute
		step.Attempt = 1
		step.Command = "test command"
		// Leave Output empty by default
	}

	return exec
}

func TestNewSQLiteStorage(t *testing.T) {
	t.Run("creates in-memory storage", func(t *testing.T) {
		s, err := NewSQLiteStorage(":memory:")
		require.NoError(t, err)
		require.NotNil(t, s)
		defer s.Close()
	})

	t.Run("creates file-based storage", func(t *testing.T) {
		tempDir := t.TempDir()
		dbPath := tempDir + "/test.db"

		s, err := NewSQLiteStorage(dbPath)
		require.NoError(t, err)
		require.NotNil(t, s)
		defer s.Close()
	})
}

func TestNewInMemoryStorage(t *testing.T) {
	s, err := NewInMemoryStorage()
	require.NoError(t, err)
	require.NotNil(t, s)
	defer s.Close()
}

func TestSQLiteStorage_SaveExecution(t *testing.T) {
	t.Run("saves minimal execution", func(t *testing.T) {
		s, _ := NewInMemoryStorage()
		defer s.Close()

		story := createTestStory("3-1-test", 3, domain.StatusInProgress)
		exec := createMinimalExecution(story)

		err := s.SaveExecution(context.Background(), exec)
		assert.NoError(t, err)

		// Verify it was saved
		count, err := s.CountExecutions(context.Background(), &ExecutionFilter{})
		assert.NoError(t, err)
		assert.Equal(t, 1, count)
	})

	t.Run("saves execution with steps", func(t *testing.T) {
		s, _ := NewInMemoryStorage()
		defer s.Close()

		story := createTestStory("3-1-test", 3, domain.StatusInProgress)
		exec := createCompletedExecution(story)

		err := s.SaveExecution(context.Background(), exec)
		assert.NoError(t, err)

		// Verify steps were saved
		records, err := s.ListExecutions(context.Background(), &ExecutionFilter{})
		require.NoError(t, err)
		require.Len(t, records, 1)
		assert.Len(t, records[0].Steps, 4)
	})

	t.Run("saves execution with output", func(t *testing.T) {
		s, _ := NewInMemoryStorage()
		defer s.Close()

		story := createTestStory("3-1-test", 3, domain.StatusInProgress)
		exec := createCompletedExecution(story)
		// Set output on the create-story step
		exec.Steps[0].Output = []string{"line 1", "line 2", "line 3"}

		err := s.SaveExecution(context.Background(), exec)
		assert.NoError(t, err)

		// Get with output
		records, err := s.ListExecutions(context.Background(), &ExecutionFilter{})
		require.NoError(t, err)
		require.Len(t, records, 1)

		rec, err := s.GetExecutionWithOutput(context.Background(), records[0].ID)
		require.NoError(t, err)

		// Find the step with output (steps may be in different order)
		var foundOutput bool
		for _, step := range rec.Steps {
			if len(step.Output) == 3 {
				foundOutput = true
				break
			}
		}
		assert.True(t, foundOutput, "Should find a step with 3 output lines")
	})

	t.Run("saves multiple executions", func(t *testing.T) {
		s, _ := NewInMemoryStorage()
		defer s.Close()

		ctx := context.Background()
		for i := 0; i < 5; i++ {
			story := createTestStory("3-"+string(rune('1'+i))+"-test", 3, domain.StatusInProgress)
			exec := createCompletedExecution(story)
			err := s.SaveExecution(ctx, exec)
			require.NoError(t, err)
		}

		count, err := s.CountExecutions(ctx, &ExecutionFilter{})
		assert.NoError(t, err)
		assert.Equal(t, 5, count)
	})
}

func TestSQLiteStorage_GetExecution(t *testing.T) {
	s, _ := NewInMemoryStorage()
	defer s.Close()
	ctx := context.Background()

	// Save an execution first
	story := createTestStory("3-1-test", 3, domain.StatusInProgress)
	exec := createCompletedExecution(story)
	err := s.SaveExecution(ctx, exec)
	require.NoError(t, err)

	// Get the saved execution's ID
	records, err := s.ListExecutions(ctx, &ExecutionFilter{})
	require.NoError(t, err)
	require.Len(t, records, 1)
	execID := records[0].ID

	t.Run("retrieves existing execution", func(t *testing.T) {
		rec, err := s.GetExecution(ctx, execID)
		require.NoError(t, err)
		require.NotNil(t, rec)

		assert.Equal(t, "3-1-test", rec.StoryKey)
		assert.Equal(t, 3, rec.StoryEpic)
		assert.Equal(t, domain.ExecutionCompleted, rec.Status)
	})

	t.Run("returns error for non-existent ID", func(t *testing.T) {
		_, err := s.GetExecution(ctx, "non-existent-id")
		assert.Error(t, err)
	})

	t.Run("includes steps without output", func(t *testing.T) {
		rec, err := s.GetExecution(ctx, execID)
		require.NoError(t, err)

		assert.Len(t, rec.Steps, 4)
		for _, step := range rec.Steps {
			assert.Nil(t, step.Output) // Output not loaded by GetExecution
		}
	})
}

func TestSQLiteStorage_GetExecutionWithOutput(t *testing.T) {
	s, _ := NewInMemoryStorage()
	defer s.Close()
	ctx := context.Background()

	// Save execution with specific output
	story := createTestStory("3-1-test", 3, domain.StatusInProgress)
	exec := createCompletedExecution(story)
	exec.Steps[0].Output = []string{"first output", "second output"}
	err := s.SaveExecution(ctx, exec)
	require.NoError(t, err)

	records, err := s.ListExecutions(ctx, &ExecutionFilter{})
	require.NoError(t, err)
	execID := records[0].ID

	t.Run("retrieves execution with output", func(t *testing.T) {
		rec, err := s.GetExecutionWithOutput(ctx, execID)
		require.NoError(t, err)

		// Find the step with output (step order may vary)
		var stepWithOutput *StepRecord
		for _, step := range rec.Steps {
			if len(step.Output) > 0 {
				stepWithOutput = step
				break
			}
		}
		require.NotNil(t, stepWithOutput, "Should find a step with output")
		assert.Len(t, stepWithOutput.Output, 2)
		assert.Equal(t, "first output", stepWithOutput.Output[0])
		assert.Equal(t, "second output", stepWithOutput.Output[1])
	})

	t.Run("returns empty output for steps without output", func(t *testing.T) {
		rec, err := s.GetExecutionWithOutput(ctx, execID)
		require.NoError(t, err)

		// Count steps without output (should be 3 out of 4)
		emptyOutputCount := 0
		for _, step := range rec.Steps {
			if len(step.Output) == 0 {
				emptyOutputCount++
			}
		}
		assert.Equal(t, 3, emptyOutputCount, "3 steps should have no output")
	})
}

func TestSQLiteStorage_ListExecutions(t *testing.T) {
	s, _ := NewInMemoryStorage()
	defer s.Close()
	ctx := context.Background()

	// Save multiple executions
	testCases := []struct {
		key    string
		epic   int
		status domain.ExecutionStatus
	}{
		{"3-1-first", 3, domain.ExecutionCompleted},
		{"3-2-second", 3, domain.ExecutionFailed},
		{"4-1-third", 4, domain.ExecutionCompleted},
		{"4-2-fourth", 4, domain.ExecutionCancelled},
		{"5-1-fifth", 5, domain.ExecutionCompleted},
	}

	for _, tc := range testCases {
		story := createTestStory(tc.key, tc.epic, domain.StatusInProgress)
		exec := createCompletedExecution(story)
		exec.Status = tc.status
		err := s.SaveExecution(ctx, exec)
		require.NoError(t, err)
	}

	t.Run("lists all executions without filter", func(t *testing.T) {
		records, err := s.ListExecutions(ctx, &ExecutionFilter{})
		require.NoError(t, err)
		assert.Len(t, records, 5)
	})

	t.Run("filters by story key partial match", func(t *testing.T) {
		filter := &ExecutionFilter{StoryKey: "3-1"}
		records, err := s.ListExecutions(ctx, filter)
		require.NoError(t, err)
		assert.Len(t, records, 1)
		assert.Equal(t, "3-1-first", records[0].StoryKey)
	})

	t.Run("filters by epic", func(t *testing.T) {
		epic := 4
		filter := &ExecutionFilter{Epic: &epic}
		records, err := s.ListExecutions(ctx, filter)
		require.NoError(t, err)
		assert.Len(t, records, 2)
	})

	t.Run("filters by status", func(t *testing.T) {
		filter := &ExecutionFilter{Status: domain.ExecutionCompleted}
		records, err := s.ListExecutions(ctx, filter)
		require.NoError(t, err)
		assert.Len(t, records, 3)
	})

	t.Run("respects limit", func(t *testing.T) {
		filter := &ExecutionFilter{Limit: 2}
		records, err := s.ListExecutions(ctx, filter)
		require.NoError(t, err)
		assert.Len(t, records, 2)
	})

	t.Run("respects offset", func(t *testing.T) {
		filter := &ExecutionFilter{Limit: 2, Offset: 2}
		records, err := s.ListExecutions(ctx, filter)
		require.NoError(t, err)
		assert.Len(t, records, 2)
	})
}

func TestSQLiteStorage_CountExecutions(t *testing.T) {
	s, _ := NewInMemoryStorage()
	defer s.Close()
	ctx := context.Background()

	t.Run("returns zero for empty database", func(t *testing.T) {
		count, err := s.CountExecutions(ctx, &ExecutionFilter{})
		require.NoError(t, err)
		assert.Equal(t, 0, count)
	})

	// Add some executions
	for i := 0; i < 3; i++ {
		story := createTestStory("3-"+string(rune('1'+i))+"-test", 3, domain.StatusInProgress)
		exec := createCompletedExecution(story)
		if i == 2 {
			exec.Status = domain.ExecutionFailed
		}
		_ = s.SaveExecution(ctx, exec)
	}

	t.Run("counts all executions", func(t *testing.T) {
		count, err := s.CountExecutions(ctx, &ExecutionFilter{})
		require.NoError(t, err)
		assert.Equal(t, 3, count)
	})

	t.Run("counts filtered by status", func(t *testing.T) {
		filter := &ExecutionFilter{Status: domain.ExecutionCompleted}
		count, err := s.CountExecutions(ctx, filter)
		require.NoError(t, err)
		assert.Equal(t, 2, count)
	})

	t.Run("counts filtered by epic", func(t *testing.T) {
		epic := 3
		filter := &ExecutionFilter{Epic: &epic}
		count, err := s.CountExecutions(ctx, filter)
		require.NoError(t, err)
		assert.Equal(t, 3, count)
	})
}

func TestSQLiteStorage_DeleteExecution(t *testing.T) {
	s, _ := NewInMemoryStorage()
	defer s.Close()
	ctx := context.Background()

	// Save an execution
	story := createTestStory("3-1-test", 3, domain.StatusInProgress)
	exec := createCompletedExecution(story)
	_ = s.SaveExecution(ctx, exec)

	records, _ := s.ListExecutions(ctx, &ExecutionFilter{})
	execID := records[0].ID

	t.Run("deletes existing execution", func(t *testing.T) {
		err := s.DeleteExecution(ctx, execID)
		assert.NoError(t, err)

		count, _ := s.CountExecutions(ctx, &ExecutionFilter{})
		assert.Equal(t, 0, count)
	})

	t.Run("no error for non-existent ID", func(t *testing.T) {
		err := s.DeleteExecution(ctx, "non-existent-id")
		assert.NoError(t, err) // DELETE doesn't error on missing rows
	})
}

func TestSQLiteStorage_GetStats(t *testing.T) {
	s, _ := NewInMemoryStorage()
	defer s.Close()
	ctx := context.Background()

	// Add executions with different statuses
	for _, status := range []domain.ExecutionStatus{
		domain.ExecutionCompleted,
		domain.ExecutionCompleted,
		domain.ExecutionFailed,
		domain.ExecutionCancelled,
	} {
		story := createTestStory("3-1-test", 3, domain.StatusInProgress)
		exec := createCompletedExecution(story)
		exec.Status = status
		_ = s.SaveExecution(ctx, exec)
	}

	t.Run("calculates overall stats", func(t *testing.T) {
		stats, err := s.GetStats(ctx)
		require.NoError(t, err)

		assert.Equal(t, 4, stats.TotalExecutions)
		assert.Equal(t, 2, stats.SuccessfulCount)
		assert.Equal(t, 1, stats.FailedCount)
		assert.Equal(t, 1, stats.CancelledCount)
		assert.Equal(t, float64(50), stats.SuccessRate)
	})

	t.Run("includes step stats", func(t *testing.T) {
		stats, err := s.GetStats(ctx)
		require.NoError(t, err)

		assert.NotEmpty(t, stats.StepStats)
	})

	t.Run("includes executions by epic", func(t *testing.T) {
		stats, err := s.GetStats(ctx)
		require.NoError(t, err)

		assert.NotEmpty(t, stats.ExecutionsByEpic)
		assert.Equal(t, 4, stats.ExecutionsByEpic[3])
	})
}

func TestSQLiteStorage_GetStepAverages(t *testing.T) {
	s, _ := NewInMemoryStorage()
	defer s.Close()
	ctx := context.Background()

	t.Run("returns empty map for empty database", func(t *testing.T) {
		averages, err := s.GetStepAverages(ctx)
		require.NoError(t, err)
		assert.Empty(t, averages)
	})

	// Save execution and update averages
	story := createTestStory("3-1-test", 3, domain.StatusInProgress)
	exec := createCompletedExecution(story)
	_ = s.SaveExecution(ctx, exec)
	_ = s.UpdateStepAverages(ctx)

	t.Run("returns averages after update", func(t *testing.T) {
		averages, err := s.GetStepAverages(ctx)
		require.NoError(t, err)

		assert.NotEmpty(t, averages)
	})
}

func TestSQLiteStorage_UpdateStepAverages(t *testing.T) {
	s, _ := NewInMemoryStorage()
	defer s.Close()
	ctx := context.Background()

	// Save multiple executions
	for i := 0; i < 3; i++ {
		story := createTestStory("3-"+string(rune('1'+i))+"-test", 3, domain.StatusInProgress)
		exec := createCompletedExecution(story)
		_ = s.SaveExecution(ctx, exec)
	}

	t.Run("updates averages successfully", func(t *testing.T) {
		err := s.UpdateStepAverages(ctx)
		assert.NoError(t, err)

		averages, err := s.GetStepAverages(ctx)
		require.NoError(t, err)
		assert.NotEmpty(t, averages)
	})
}

func TestSQLiteStorage_GetRecentExecutions(t *testing.T) {
	s, _ := NewInMemoryStorage()
	defer s.Close()
	ctx := context.Background()

	// Save 5 executions
	for i := 0; i < 5; i++ {
		story := createTestStory("3-"+string(rune('1'+i))+"-test", 3, domain.StatusInProgress)
		exec := createCompletedExecution(story)
		_ = s.SaveExecution(ctx, exec)
	}

	t.Run("returns limited recent executions", func(t *testing.T) {
		records, err := s.GetRecentExecutions(ctx, 3)
		require.NoError(t, err)
		assert.Len(t, records, 3)
	})

	t.Run("returns all if limit exceeds count", func(t *testing.T) {
		records, err := s.GetRecentExecutions(ctx, 10)
		require.NoError(t, err)
		assert.Len(t, records, 5)
	})
}

func TestSQLiteStorage_GetExecutionsByStory(t *testing.T) {
	s, _ := NewInMemoryStorage()
	defer s.Close()
	ctx := context.Background()

	// Save executions for different stories
	story1 := createTestStory("3-1-story-a", 3, domain.StatusInProgress)
	story2 := createTestStory("3-2-story-b", 3, domain.StatusInProgress)

	_ = s.SaveExecution(ctx, createCompletedExecution(story1))
	_ = s.SaveExecution(ctx, createCompletedExecution(story1))
	_ = s.SaveExecution(ctx, createCompletedExecution(story2))

	t.Run("returns executions for specific story", func(t *testing.T) {
		records, err := s.GetExecutionsByStory(ctx, "3-1-story-a")
		require.NoError(t, err)
		assert.Len(t, records, 2)
	})

	t.Run("returns empty for non-existent story", func(t *testing.T) {
		records, err := s.GetExecutionsByStory(ctx, "non-existent")
		require.NoError(t, err)
		assert.Len(t, records, 0)
	})
}

func TestSQLiteStorage_GetStepOutput(t *testing.T) {
	s, _ := NewInMemoryStorage()
	defer s.Close()
	ctx := context.Background()

	// Save execution with output
	story := createTestStory("3-1-test", 3, domain.StatusInProgress)
	exec := createCompletedExecution(story)
	exec.Steps[0].Output = []string{"output 1", "output 2", "output 3"}
	_ = s.SaveExecution(ctx, exec)

	// Get the step ID - find the step that was assigned output
	records, _ := s.ListExecutions(ctx, &ExecutionFilter{})
	rec, _ := s.GetExecution(ctx, records[0].ID)

	// Find step with output by checking output_size
	var stepWithOutputID string
	var stepWithoutOutputID string
	for _, step := range rec.Steps {
		if step.OutputSize == 3 {
			stepWithOutputID = step.ID
		} else if step.OutputSize == 0 && stepWithoutOutputID == "" {
			stepWithoutOutputID = step.ID
		}
	}

	t.Run("returns step output", func(t *testing.T) {
		require.NotEmpty(t, stepWithOutputID, "Should have a step with output")
		output, err := s.GetStepOutput(ctx, stepWithOutputID)
		require.NoError(t, err)
		assert.Len(t, output, 3)
		assert.Equal(t, "output 1", output[0])
	})

	t.Run("returns empty for step without output", func(t *testing.T) {
		require.NotEmpty(t, stepWithoutOutputID, "Should have a step without output")
		output, err := s.GetStepOutput(ctx, stepWithoutOutputID)
		require.NoError(t, err)
		assert.Empty(t, output)
	})
}

func TestSQLiteStorage_Close(t *testing.T) {
	s, _ := NewInMemoryStorage()

	err := s.Close()
	assert.NoError(t, err)
}

func TestGetDatabasePath(t *testing.T) {
	path := GetDatabasePath("/test/data")
	assert.Equal(t, "/test/data/bmad.db", path)
}

func TestExecutionFilter_DateFiltering(t *testing.T) {
	s, _ := NewInMemoryStorage()
	defer s.Close()
	ctx := context.Background()

	now := time.Now()

	// Save execution
	story := createTestStory("3-1-test", 3, domain.StatusInProgress)
	exec := createCompletedExecution(story)
	exec.StartTime = now
	_ = s.SaveExecution(ctx, exec)

	t.Run("filters by start_after", func(t *testing.T) {
		yesterday := now.Add(-24 * time.Hour)
		filter := &ExecutionFilter{StartAfter: &yesterday}
		records, err := s.ListExecutions(ctx, filter)
		require.NoError(t, err)
		assert.Len(t, records, 1)
	})

	t.Run("filters by start_before", func(t *testing.T) {
		tomorrow := now.Add(24 * time.Hour)
		filter := &ExecutionFilter{StartBefore: &tomorrow}
		records, err := s.ListExecutions(ctx, filter)
		require.NoError(t, err)
		assert.Len(t, records, 1)
	})

	t.Run("excludes when outside range", func(t *testing.T) {
		yesterday := now.Add(-24 * time.Hour)
		filter := &ExecutionFilter{StartBefore: &yesterday}
		records, err := s.ListExecutions(ctx, filter)
		require.NoError(t, err)
		assert.Len(t, records, 0)
	})
}
