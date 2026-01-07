package workflow

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"text/template"

	"github.com/robertguss/bmad-automate-go/internal/domain"
	"gopkg.in/yaml.v3"
)

// StepDefinition defines a single step in a workflow
type StepDefinition struct {
	Name           string            `yaml:"name"`
	Description    string            `yaml:"description,omitempty"`
	PromptTemplate string            `yaml:"prompt_template"`
	Timeout        int               `yaml:"timeout,omitempty"`       // Override default timeout (seconds)
	Retries        int               `yaml:"retries,omitempty"`       // Override default retries
	SkipIf         string            `yaml:"skip_if,omitempty"`       // Condition: "file_exists"
	AllowFailure   bool              `yaml:"allow_failure,omitempty"` // Continue if step fails
	Env            map[string]string `yaml:"env,omitempty"`           // Environment variables
	WorkingDir     string            `yaml:"working_dir,omitempty"`   // Override working directory
	StepName       domain.StepName   `yaml:"-"`                       // Mapped step name for domain integration
}

// Workflow defines a complete workflow with multiple steps
type Workflow struct {
	Name        string            `yaml:"name"`
	Description string            `yaml:"description,omitempty"`
	Version     string            `yaml:"version,omitempty"`
	Steps       []*StepDefinition `yaml:"steps"`
	Variables   map[string]string `yaml:"variables,omitempty"` // Default variables
}

// WorkflowStore manages workflow definitions
type WorkflowStore struct {
	workflowDir string
	workflows   map[string]*Workflow
}

// NewWorkflowStore creates a new workflow store
func NewWorkflowStore(dataDir string) *WorkflowStore {
	workflowDir := filepath.Join(dataDir, "workflows")
	return &WorkflowStore{
		workflowDir: workflowDir,
		workflows:   make(map[string]*Workflow),
	}
}

// Load loads all workflows from disk
func (ws *WorkflowStore) Load() error {
	if err := os.MkdirAll(ws.workflowDir, 0755); err != nil {
		return fmt.Errorf("failed to create workflow directory: %w", err)
	}

	// Load default workflow
	ws.workflows["default"] = DefaultWorkflow()

	// Load custom workflows from YAML files
	files, err := filepath.Glob(filepath.Join(ws.workflowDir, "*.yaml"))
	if err != nil {
		return fmt.Errorf("failed to list workflows: %w", err)
	}

	for _, file := range files {
		workflow, err := ws.loadWorkflow(file)
		if err != nil {
			continue // Skip invalid workflows
		}
		ws.workflows[workflow.Name] = workflow
	}

	return nil
}

// loadWorkflow loads a workflow from a YAML file
func (ws *WorkflowStore) loadWorkflow(path string) (*Workflow, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var workflow Workflow
	if err := yaml.Unmarshal(data, &workflow); err != nil {
		return nil, err
	}

	// Use filename as name if not specified
	if workflow.Name == "" {
		base := filepath.Base(path)
		workflow.Name = base[:len(base)-5] // Remove .yaml extension
	}

	// Map step names to domain step names
	for i, step := range workflow.Steps {
		workflow.Steps[i].StepName = mapStepName(step.Name)
	}

	return &workflow, nil
}

// mapStepName converts a string step name to domain.StepName
func mapStepName(name string) domain.StepName {
	switch strings.ToLower(name) {
	case "create-story", "create_story", "createstory":
		return domain.StepCreateStory
	case "dev-story", "dev_story", "devstory", "develop":
		return domain.StepDevStory
	case "code-review", "code_review", "codereview", "review":
		return domain.StepCodeReview
	case "git-commit", "git_commit", "gitcommit", "commit":
		return domain.StepGitCommit
	default:
		return domain.StepName(name)
	}
}

// Save saves a workflow to disk
func (ws *WorkflowStore) Save(workflow *Workflow) error {
	if err := os.MkdirAll(ws.workflowDir, 0755); err != nil {
		return fmt.Errorf("failed to create workflow directory: %w", err)
	}

	path := filepath.Join(ws.workflowDir, workflow.Name+".yaml")
	data, err := yaml.Marshal(workflow)
	if err != nil {
		return fmt.Errorf("failed to marshal workflow: %w", err)
	}

	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("failed to write workflow: %w", err)
	}

	ws.workflows[workflow.Name] = workflow
	return nil
}

// Get returns a workflow by name
func (ws *WorkflowStore) Get(name string) (*Workflow, bool) {
	w, ok := ws.workflows[name]
	return w, ok
}

// List returns all workflow names
func (ws *WorkflowStore) List() []string {
	names := make([]string, 0, len(ws.workflows))
	for name := range ws.workflows {
		names = append(names, name)
	}
	return names
}

// GetAll returns all workflows
func (ws *WorkflowStore) GetAll() []*Workflow {
	workflows := make([]*Workflow, 0, len(ws.workflows))
	for _, w := range ws.workflows {
		workflows = append(workflows, w)
	}
	return workflows
}

// Delete removes a workflow from disk
func (ws *WorkflowStore) Delete(name string) error {
	if name == "default" {
		return fmt.Errorf("cannot delete default workflow")
	}

	path := filepath.Join(ws.workflowDir, name+".yaml")
	if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to delete workflow: %w", err)
	}
	delete(ws.workflows, name)
	return nil
}

// TemplateContext provides data for prompt template rendering
type TemplateContext struct {
	Story     StoryContext
	StoryDir  string
	StoryPath string
	WorkDir   string
	Variables map[string]string
}

// StoryContext provides story data for templates
type StoryContext struct {
	Key        string
	Epic       int
	Status     string
	Title      string
	FilePath   string
	FileExists bool
}

// RenderPrompt renders a step's prompt template with the given context
func (s *StepDefinition) RenderPrompt(ctx *TemplateContext) (string, error) {
	tmpl, err := template.New("prompt").Parse(s.PromptTemplate)
	if err != nil {
		return "", fmt.Errorf("failed to parse prompt template: %w", err)
	}

	var buf strings.Builder
	if err := tmpl.Execute(&buf, ctx); err != nil {
		return "", fmt.Errorf("failed to render prompt template: %w", err)
	}

	return buf.String(), nil
}

// DefaultWorkflow returns the default workflow with standard steps
func DefaultWorkflow() *Workflow {
	return &Workflow{
		Name:        "default",
		Description: "Default BMAD workflow with 4 standard steps",
		Version:     "1.0",
		Steps: []*StepDefinition{
			{
				Name:           "create-story",
				Description:    "Create story file from template",
				PromptTemplate: `/bmad:bmm:workflows:create-story - Create story: {{.Story.Key}}`,
				SkipIf:         "file_exists",
				StepName:       domain.StepCreateStory,
			},
			{
				Name:        "dev-story",
				Description: "Implement the story",
				PromptTemplate: `/bmad:bmm:workflows:dev-story - Work on story file: {{.StoryPath}}. ` +
					`Complete all tasks. Run tests after each implementation. ` +
					`Do not ask clarifying questions - use best judgment based on existing patterns.`,
				StepName: domain.StepDevStory,
			},
			{
				Name:        "code-review",
				Description: "Review code changes",
				PromptTemplate: `/bmad:bmm:workflows:code-review - Review story: {{.StoryPath}}. ` +
					`IMPORTANT: When presenting options, always choose option 1 to ` +
					`auto-fix all issues immediately. Do not wait for user input.`,
				StepName: domain.StepCodeReview,
			},
			{
				Name:        "git-commit",
				Description: "Commit and push changes",
				PromptTemplate: `Commit all changes for story {{.Story.Key}} with a descriptive message. ` +
					`Then push to the current branch.`,
				StepName: domain.StepGitCommit,
			},
		},
	}
}

// CreateExampleWorkflow creates an example custom workflow file
func CreateExampleWorkflow(dataDir string) error {
	workflowDir := filepath.Join(dataDir, "workflows")
	if err := os.MkdirAll(workflowDir, 0755); err != nil {
		return err
	}

	example := &Workflow{
		Name:        "quick-dev",
		Description: "Quick development workflow without code review",
		Version:     "1.0",
		Variables: map[string]string{
			"test_command": "npm test",
		},
		Steps: []*StepDefinition{
			{
				Name:           "create-story",
				Description:    "Create story file if it doesn't exist",
				PromptTemplate: `/bmad:bmm:workflows:create-story - Create story: {{.Story.Key}}`,
				SkipIf:         "file_exists",
			},
			{
				Name:        "dev-story",
				Description: "Implement the story with testing",
				PromptTemplate: `/bmad:bmm:workflows:dev-story - Work on story file: {{.StoryPath}}. ` +
					`Complete all tasks. Run "{{.Variables.test_command}}" after each implementation.`,
				Timeout: 900, // 15 minutes
			},
			{
				Name:           "git-commit",
				Description:    "Commit and push changes",
				PromptTemplate: `Commit all changes for story {{.Story.Key}} with a descriptive message. Then push to the current branch.`,
			},
		},
	}

	data, err := yaml.Marshal(example)
	if err != nil {
		return err
	}

	examplePath := filepath.Join(workflowDir, "quick-dev.yaml.example")
	return os.WriteFile(examplePath, data, 0644)
}
