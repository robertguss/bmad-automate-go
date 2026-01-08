package workflow

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/robertguss/bmad-automate-go/internal/domain"
)

func TestNewWorkflowStore(t *testing.T) {
	tempDir := t.TempDir()

	store := NewWorkflowStore(tempDir)

	require.NotNil(t, store)
	assert.Equal(t, filepath.Join(tempDir, "workflows"), store.workflowDir)
	assert.NotNil(t, store.workflows)
	assert.Len(t, store.workflows, 0)
}

func TestWorkflowStore_Load(t *testing.T) {
	t.Run("creates workflow directory if not exists", func(t *testing.T) {
		tempDir := t.TempDir()
		store := NewWorkflowStore(tempDir)

		err := store.Load()
		require.NoError(t, err)

		info, err := os.Stat(filepath.Join(tempDir, "workflows"))
		require.NoError(t, err)
		assert.True(t, info.IsDir())
	})

	t.Run("loads default workflow", func(t *testing.T) {
		tempDir := t.TempDir()
		store := NewWorkflowStore(tempDir)

		err := store.Load()
		require.NoError(t, err)

		w, ok := store.Get("default")
		assert.True(t, ok)
		assert.NotNil(t, w)
		assert.Equal(t, "default", w.Name)
	})

	t.Run("loads custom workflows from files", func(t *testing.T) {
		tempDir := t.TempDir()
		workflowDir := filepath.Join(tempDir, "workflows")
		_ = os.MkdirAll(workflowDir, 0755)

		customWorkflow := `name: custom
description: Custom workflow
version: "1.0"
steps:
  - name: dev-story
    prompt_template: "Work on {{.Story.Key}}"
`
		_ = os.WriteFile(filepath.Join(workflowDir, "custom.yaml"), []byte(customWorkflow), 0644)

		store := NewWorkflowStore(tempDir)
		err := store.Load()
		require.NoError(t, err)

		w, ok := store.Get("custom")
		assert.True(t, ok)
		assert.NotNil(t, w)
		assert.Equal(t, "custom", w.Name)
		assert.Equal(t, "Custom workflow", w.Description)
	})

	t.Run("skips invalid workflow files", func(t *testing.T) {
		tempDir := t.TempDir()
		workflowDir := filepath.Join(tempDir, "workflows")
		_ = os.MkdirAll(workflowDir, 0755)

		// Write an invalid YAML file
		_ = os.WriteFile(filepath.Join(workflowDir, "invalid.yaml"), []byte("invalid: yaml: here"), 0644)

		store := NewWorkflowStore(tempDir)
		err := store.Load()
		require.NoError(t, err)

		// Should still have default workflow
		_, ok := store.Get("default")
		assert.True(t, ok)
	})

	t.Run("uses filename as name if not specified", func(t *testing.T) {
		tempDir := t.TempDir()
		workflowDir := filepath.Join(tempDir, "workflows")
		_ = os.MkdirAll(workflowDir, 0755)

		workflowNoName := `description: Workflow without name
steps:
  - name: dev-story
    prompt_template: "Work"
`
		_ = os.WriteFile(filepath.Join(workflowDir, "unnamed.yaml"), []byte(workflowNoName), 0644)

		store := NewWorkflowStore(tempDir)
		err := store.Load()
		require.NoError(t, err)

		w, ok := store.Get("unnamed")
		assert.True(t, ok)
		assert.NotNil(t, w)
	})
}

func TestWorkflowStore_Get(t *testing.T) {
	tempDir := t.TempDir()
	store := NewWorkflowStore(tempDir)
	_ = store.Load()

	t.Run("returns workflow when found", func(t *testing.T) {
		w, ok := store.Get("default")
		assert.True(t, ok)
		assert.NotNil(t, w)
	})

	t.Run("returns false when not found", func(t *testing.T) {
		w, ok := store.Get("nonexistent")
		assert.False(t, ok)
		assert.Nil(t, w)
	})
}

func TestWorkflowStore_List(t *testing.T) {
	tempDir := t.TempDir()
	store := NewWorkflowStore(tempDir)
	_ = store.Load()

	names := store.List()

	assert.Contains(t, names, "default")
}

func TestWorkflowStore_GetAll(t *testing.T) {
	tempDir := t.TempDir()
	store := NewWorkflowStore(tempDir)
	_ = store.Load()

	workflows := store.GetAll()

	assert.Len(t, workflows, 1)
	assert.Equal(t, "default", workflows[0].Name)
}

func TestWorkflowStore_Save(t *testing.T) {
	t.Run("saves workflow to disk", func(t *testing.T) {
		tempDir := t.TempDir()
		store := NewWorkflowStore(tempDir)
		_ = store.Load()

		workflow := &Workflow{
			Name:        "test-workflow",
			Description: "Test workflow",
			Steps: []*StepDefinition{
				{
					Name:           "dev-story",
					PromptTemplate: "Work on {{.Story.Key}}",
				},
			},
		}

		err := store.Save(workflow)
		require.NoError(t, err)

		// Verify file exists
		path := filepath.Join(tempDir, "workflows", "test-workflow.yaml")
		_, err = os.Stat(path)
		assert.NoError(t, err)

		// Verify workflow is in store
		w, ok := store.Get("test-workflow")
		assert.True(t, ok)
		assert.NotNil(t, w)
	})

	t.Run("creates directory if not exists", func(t *testing.T) {
		tempDir := t.TempDir()
		store := NewWorkflowStore(tempDir)
		// Don't call Load - directory doesn't exist yet

		workflow := &Workflow{
			Name: "new-workflow",
		}

		err := store.Save(workflow)
		require.NoError(t, err)

		// Verify directory was created
		info, err := os.Stat(filepath.Join(tempDir, "workflows"))
		require.NoError(t, err)
		assert.True(t, info.IsDir())
	})
}

func TestWorkflowStore_Delete(t *testing.T) {
	t.Run("cannot delete default workflow", func(t *testing.T) {
		tempDir := t.TempDir()
		store := NewWorkflowStore(tempDir)
		_ = store.Load()

		err := store.Delete("default")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "cannot delete default workflow")

		// Default should still exist
		_, ok := store.Get("default")
		assert.True(t, ok)
	})

	t.Run("deletes custom workflow", func(t *testing.T) {
		tempDir := t.TempDir()
		store := NewWorkflowStore(tempDir)
		_ = store.Load()

		// First save a workflow
		workflow := &Workflow{Name: "to-delete"}
		_ = store.Save(workflow)

		// Verify it exists
		_, ok := store.Get("to-delete")
		require.True(t, ok)

		// Delete it
		err := store.Delete("to-delete")
		require.NoError(t, err)

		// Verify it's gone
		_, ok = store.Get("to-delete")
		assert.False(t, ok)
	})

	t.Run("succeeds for non-existent workflow", func(t *testing.T) {
		tempDir := t.TempDir()
		store := NewWorkflowStore(tempDir)
		_ = store.Load()

		err := store.Delete("nonexistent")
		assert.NoError(t, err)
	})
}

func TestDefaultWorkflow(t *testing.T) {
	w := DefaultWorkflow()

	t.Run("has correct name", func(t *testing.T) {
		assert.Equal(t, "default", w.Name)
	})

	t.Run("has description", func(t *testing.T) {
		assert.NotEmpty(t, w.Description)
	})

	t.Run("has version", func(t *testing.T) {
		assert.Equal(t, "1.0", w.Version)
	})

	t.Run("has 4 steps", func(t *testing.T) {
		assert.Len(t, w.Steps, 4)
	})

	t.Run("steps have correct order", func(t *testing.T) {
		expectedSteps := []domain.StepName{
			domain.StepCreateStory,
			domain.StepDevStory,
			domain.StepCodeReview,
			domain.StepGitCommit,
		}

		for i, expected := range expectedSteps {
			assert.Equal(t, expected, w.Steps[i].StepName)
		}
	})

	t.Run("create-story has skip_if condition", func(t *testing.T) {
		assert.Equal(t, "file_exists", w.Steps[0].SkipIf)
	})
}

func TestMapStepName(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected domain.StepName
	}{
		{
			name:     "create-story hyphen",
			input:    "create-story",
			expected: domain.StepCreateStory,
		},
		{
			name:     "create_story underscore",
			input:    "create_story",
			expected: domain.StepCreateStory,
		},
		{
			name:     "createstory camelcase",
			input:    "createstory",
			expected: domain.StepCreateStory,
		},
		{
			name:     "dev-story hyphen",
			input:    "dev-story",
			expected: domain.StepDevStory,
		},
		{
			name:     "develop alias",
			input:    "develop",
			expected: domain.StepDevStory,
		},
		{
			name:     "code-review hyphen",
			input:    "code-review",
			expected: domain.StepCodeReview,
		},
		{
			name:     "review alias",
			input:    "review",
			expected: domain.StepCodeReview,
		},
		{
			name:     "git-commit hyphen",
			input:    "git-commit",
			expected: domain.StepGitCommit,
		},
		{
			name:     "commit alias",
			input:    "commit",
			expected: domain.StepGitCommit,
		},
		{
			name:     "case insensitive",
			input:    "CREATE-STORY",
			expected: domain.StepCreateStory,
		},
		{
			name:     "unknown returns as-is",
			input:    "custom-step",
			expected: domain.StepName("custom-step"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := mapStepName(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestStepDefinition_RenderPrompt(t *testing.T) {
	t.Run("renders story key", func(t *testing.T) {
		step := &StepDefinition{
			PromptTemplate: "Work on story: {{.Story.Key}}",
		}
		ctx := &TemplateContext{
			Story: StoryContext{Key: "3-1-test"},
		}

		result, err := step.RenderPrompt(ctx)
		require.NoError(t, err)
		assert.Equal(t, "Work on story: 3-1-test", result)
	})

	t.Run("renders multiple fields", func(t *testing.T) {
		step := &StepDefinition{
			PromptTemplate: "Story {{.Story.Key}} (Epic {{.Story.Epic}}) - Path: {{.StoryPath}}",
		}
		ctx := &TemplateContext{
			Story:     StoryContext{Key: "3-1-test", Epic: 3},
			StoryPath: "/path/to/story.md",
		}

		result, err := step.RenderPrompt(ctx)
		require.NoError(t, err)
		assert.Equal(t, "Story 3-1-test (Epic 3) - Path: /path/to/story.md", result)
	})

	t.Run("renders variables", func(t *testing.T) {
		step := &StepDefinition{
			PromptTemplate: "Run {{.Variables.test_command}} for {{.Story.Key}}",
		}
		ctx := &TemplateContext{
			Story:     StoryContext{Key: "3-1-test"},
			Variables: map[string]string{"test_command": "npm test"},
		}

		result, err := step.RenderPrompt(ctx)
		require.NoError(t, err)
		assert.Equal(t, "Run npm test for 3-1-test", result)
	})

	t.Run("returns error for invalid template", func(t *testing.T) {
		step := &StepDefinition{
			PromptTemplate: "Invalid {{.Unclosed",
		}
		ctx := &TemplateContext{}

		_, err := step.RenderPrompt(ctx)
		assert.Error(t, err)
	})

	t.Run("returns error for missing field", func(t *testing.T) {
		step := &StepDefinition{
			PromptTemplate: "{{.NonExistent.Field}}",
		}
		ctx := &TemplateContext{}

		_, err := step.RenderPrompt(ctx)
		assert.Error(t, err)
	})
}

func TestCreateExampleWorkflow(t *testing.T) {
	tempDir := t.TempDir()

	err := CreateExampleWorkflow(tempDir)
	require.NoError(t, err)

	// Verify file was created
	examplePath := filepath.Join(tempDir, "workflows", "quick-dev.yaml.example")
	_, err = os.Stat(examplePath)
	assert.NoError(t, err)

	// Verify content
	data, err := os.ReadFile(examplePath)
	require.NoError(t, err)
	assert.Contains(t, string(data), "quick-dev")
	assert.Contains(t, string(data), "Quick development workflow")
}
