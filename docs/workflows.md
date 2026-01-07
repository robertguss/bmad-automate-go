# Workflow Customization

BMAD Automate supports custom workflows that define the sequence of steps executed for each story. This guide explains how to create and use custom workflows.

## Overview

A workflow defines:

- The sequence of steps to execute
- Prompt templates for each step
- Per-step configuration (timeouts, retries)
- Skip conditions and failure handling

## Default Workflow

BMAD includes a default 4-step workflow:

| Step | Name           | Description                       |
| ---- | -------------- | --------------------------------- |
| 1    | `create-story` | Generate story file from template |
| 2    | `dev-story`    | Implement the story               |
| 3    | `code-review`  | Review and fix code               |
| 4    | `git-commit`   | Commit and push changes           |

```yaml
# Default workflow (built-in)
name: default
description: Default BMAD workflow with 4 standard steps
version: "1.0"

steps:
  - name: create-story
    description: Create story file from template
    prompt_template: /bmad:bmm:workflows:create-story - Create story: {{.Story.Key}}
    skip_if: file_exists

  - name: dev-story
    description: Implement the story
    prompt_template: >
      /bmad:bmm:workflows:dev-story - Work on story file: {{.StoryPath}}.
      Complete all tasks. Run tests after each implementation.
      Do not ask clarifying questions - use best judgment based on existing patterns.

  - name: code-review
    description: Review code changes
    prompt_template: >
      /bmad:bmm:workflows:code-review - Review story: {{.StoryPath}}.
      IMPORTANT: When presenting options, always choose option 1 to
      auto-fix all issues immediately. Do not wait for user input.

  - name: git-commit
    description: Commit and push changes
    prompt_template: >
      Commit all changes for story {{.Story.Key}} with a descriptive message.
      Then push to the current branch.
```

## Creating Custom Workflows

### Step 1: Create the Workflow File

Create a YAML file in `.bmad/workflows/`:

```yaml
# .bmad/workflows/quick-dev.yaml
name: quick-dev
description: Quick development workflow without code review
version: "1.0"

variables:
  test_command: npm test

steps:
  - name: create-story
    description: Create story file if it doesn't exist
    prompt_template: /bmad:bmm:workflows:create-story - Create story: {{.Story.Key}}
    skip_if: file_exists

  - name: dev-story
    description: Implement the story with testing
    prompt_template: >
      /bmad:bmm:workflows:dev-story - Work on story file: {{.StoryPath}}.
      Complete all tasks. Run "{{.Variables.test_command}}" after each implementation.
    timeout: 900

  - name: git-commit
    description: Commit and push changes
    prompt_template: >
      Commit all changes for story {{.Story.Key}} with a descriptive message.
      Then push to the current branch.
```

### Step 2: Activate the Workflow

In your profile configuration:

```yaml
# .bmad/profiles/myproject.yaml
workflow: quick-dev
```

Or via the Settings view in the TUI.

## Workflow Configuration

### Basic Structure

```yaml
name: workflow-name # Required: Unique identifier
description: Description # Optional: Human-readable description
version: "1.0" # Optional: Version string

variables: # Optional: Default variables
  key: value

steps: # Required: List of steps
  - name: step-name
    # ... step configuration
```

### Step Configuration

| Field             | Type    | Required | Description                |
| ----------------- | ------- | -------- | -------------------------- |
| `name`            | string  | Yes      | Step identifier            |
| `description`     | string  | No       | Human-readable description |
| `prompt_template` | string  | Yes      | Claude CLI prompt template |
| `timeout`         | integer | No       | Step timeout in seconds    |
| `retries`         | integer | No       | Retry attempts             |
| `skip_if`         | string  | No       | Skip condition             |
| `allow_failure`   | boolean | No       | Continue if step fails     |
| `env`             | map     | No       | Environment variables      |
| `working_dir`     | string  | No       | Override working directory |

### Step Names

Standard step names that integrate with domain models:

| Step Name      | Domain Integration       |
| -------------- | ------------------------ |
| `create-story` | `domain.StepCreateStory` |
| `dev-story`    | `domain.StepDevStory`    |
| `code-review`  | `domain.StepCodeReview`  |
| `git-commit`   | `domain.StepGitCommit`   |

You can also define custom step names for specialized workflows.

## Template Variables

### Available Variables

Templates have access to these variables:

```go
type TemplateContext struct {
    Story     StoryContext          // Story information
    StoryDir  string               // Story directory path
    StoryPath string               // Full path to story file
    WorkDir   string               // Working directory
    Variables map[string]string    // Custom variables
}

type StoryContext struct {
    Key        string    // e.g., "3-1-user-auth"
    Epic       int       // e.g., 3
    Status     string    // e.g., "ready-for-dev"
    Title      string    // e.g., "User Authentication"
    FilePath   string    // Full file path
    FileExists bool      // Whether file exists
}
```

### Template Syntax

Templates use Go's `text/template` syntax:

```yaml
# Simple variable
prompt_template: Create story {{.Story.Key}}

# Conditional
prompt_template: >
  {{if .Story.FileExists}}
  Update story {{.Story.Key}}
  {{else}}
  Create story {{.Story.Key}}
  {{end}}

# Custom variables
prompt_template: Run "{{.Variables.test_command}}" for {{.Story.Key}}

# Multi-line
prompt_template: |
  /bmad:bmm:workflows:dev-story
  Story: {{.Story.Key}}
  Path: {{.StoryPath}}
  Epic: {{.Story.Epic}}
```

## Skip Conditions

### File Exists

Skip the step if the story file already exists:

```yaml
- name: create-story
  skip_if: file_exists
```

### Manual Skip

Users can skip steps during execution using the `s` key.

## Error Handling

### Allow Failure

Continue workflow even if step fails:

```yaml
- name: optional-lint
  prompt_template: Run linting on {{.StoryPath}}
  allow_failure: true
```

### Retries

Automatically retry failed steps:

```yaml
- name: dev-story
  prompt_template: ...
  retries: 3 # Try up to 3 times
```

### Timeout

Set step-specific timeouts:

```yaml
- name: dev-story
  prompt_template: ...
  timeout: 1800 # 30 minutes
```

## Example Workflows

### Minimal Workflow

```yaml
# .bmad/workflows/minimal.yaml
name: minimal
description: Just development and commit
version: "1.0"

steps:
  - name: dev-story
    prompt_template: >
      Implement the story defined in {{.StoryPath}}.
      Follow existing patterns.

  - name: git-commit
    prompt_template: Commit changes for {{.Story.Key}}
```

### Test-First Workflow

```yaml
# .bmad/workflows/tdd.yaml
name: tdd
description: Test-driven development workflow
version: "1.0"

variables:
  test_command: go test ./...

steps:
  - name: write-tests
    description: Write tests first
    prompt_template: >
      Read the requirements in {{.StoryPath}}.
      Write comprehensive tests for the functionality.
      Do not implement the actual code yet.

  - name: dev-story
    description: Implement to make tests pass
    prompt_template: >
      Implement the functionality for {{.Story.Key}}.
      Run "{{.Variables.test_command}}" after each change.
      All tests must pass before moving on.

  - name: code-review
    prompt_template: >
      Review {{.StoryPath}}.
      Ensure test coverage is adequate.
      Auto-fix any issues.

  - name: git-commit
    prompt_template: Commit all changes for {{.Story.Key}}
```

### Documentation Workflow

```yaml
# .bmad/workflows/with-docs.yaml
name: with-docs
description: Development with documentation
version: "1.0"

steps:
  - name: create-story
    prompt_template: Create story {{.Story.Key}}
    skip_if: file_exists

  - name: dev-story
    prompt_template: Implement {{.StoryPath}}
    timeout: 900

  - name: update-docs
    description: Update documentation
    prompt_template: >
      Update documentation for changes made in {{.Story.Key}}.
      Update README if needed.
      Add inline code comments.
    allow_failure: true

  - name: code-review
    prompt_template: Review {{.StoryPath}}

  - name: git-commit
    prompt_template: Commit {{.Story.Key}} with documentation
```

### Multi-Language Workflow

```yaml
# .bmad/workflows/fullstack.yaml
name: fullstack
description: Full-stack development workflow
version: "1.0"

variables:
  backend_test: go test ./...
  frontend_test: npm test

steps:
  - name: create-story
    prompt_template: Create story {{.Story.Key}}
    skip_if: file_exists

  - name: backend-dev
    description: Implement backend
    prompt_template: >
      Implement backend for {{.Story.Key}}.
      Run "{{.Variables.backend_test}}" after changes.
    working_dir: "{{.WorkDir}}/backend"

  - name: frontend-dev
    description: Implement frontend
    prompt_template: >
      Implement frontend for {{.Story.Key}}.
      Run "{{.Variables.frontend_test}}" after changes.
    working_dir: "{{.WorkDir}}/frontend"

  - name: integration-test
    description: Run integration tests
    prompt_template: Run integration tests for {{.Story.Key}}
    allow_failure: true

  - name: git-commit
    prompt_template: Commit {{.Story.Key}}
```

## Environment Variables

Set environment variables for specific steps:

```yaml
- name: dev-story
  prompt_template: ...
  env:
    NODE_ENV: development
    DEBUG: "true"
```

## Working Directory

Override working directory for a step:

```yaml
- name: frontend-dev
  prompt_template: ...
  working_dir: ./frontend
```

## Best Practices

### 1. Clear Prompt Templates

Write clear, specific prompts:

```yaml
# Good
prompt_template: >
  Implement user authentication for {{.Story.Key}}.
  Use JWT tokens. Follow existing auth patterns.
  Run tests after implementation.

# Less helpful
prompt_template: Do {{.Story.Key}}
```

### 2. Appropriate Timeouts

Set realistic timeouts based on step complexity:

```yaml
- name: create-story
  timeout: 60 # Quick task

- name: dev-story
  timeout: 1800 # Complex implementation

- name: code-review
  timeout: 600 # Moderate complexity
```

### 3. Use Skip Conditions

Avoid redundant work:

```yaml
- name: create-story
  skip_if: file_exists # Don't recreate existing files
```

### 4. Allow Failure for Optional Steps

Mark non-critical steps as optional:

```yaml
- name: update-changelog
  prompt_template: Update CHANGELOG.md
  allow_failure: true
```

### 5. Use Variables for Reusability

Define common values as variables:

```yaml
variables:
  test_command: npm test
  build_command: npm run build

steps:
  - name: dev-story
    prompt_template: >
      Implement {{.Story.Key}}.
      Run "{{.Variables.test_command}}" after changes.
```

## Troubleshooting

### Workflow Not Loading

**Problem**: Custom workflow doesn't appear in the list

**Solution**: Ensure the YAML file is in `.bmad/workflows/` and has valid syntax:

```bash
# Validate YAML
python -c "import yaml; yaml.safe_load(open('.bmad/workflows/custom.yaml'))"
```

### Template Errors

**Problem**: Template fails to render

**Solution**: Check template syntax. Common issues:

- Missing closing braces: `{{.Story.Key}` (missing `}}`)
- Invalid variable: `{{.Story.Invalid}}` (field doesn't exist)
- Unclosed conditionals: `{{if ...}}` without `{{end}}`

### Step Not Executing

**Problem**: Step is always skipped

**Solution**: Check skip conditions:

- `skip_if: file_exists` skips if story file exists
- Remove or modify the skip condition if not desired
