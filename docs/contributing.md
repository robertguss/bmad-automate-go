# Contributing Guide

Thank you for your interest in contributing to BMAD Automate! This guide will help you get started with development.

## Getting Started

### Prerequisites

- Go 1.24 or later
- Git
- Make (optional but recommended)
- golangci-lint (for linting)

### Setup

1. **Clone the repository**

```bash
git clone https://github.com/robertguss/bmad-automate-go.git
cd bmad-automate-go
```

2. **Install dependencies**

```bash
make deps
# or
go mod download && go mod tidy
```

3. **Build the project**

```bash
make build
# Binary will be at ./bin/bmad
```

4. **Run the application**

```bash
make run
# or
go run ./cmd/bmad
```

### Development Tools

Install recommended tools:

```bash
# Linter
go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest

# Live reload (optional)
go install github.com/cosmtrek/air@latest

# Release tool (for maintainers)
go install github.com/goreleaser/goreleaser@latest
```

## Project Structure

```
bmad-automate-go/
├── cmd/bmad/              # Application entry point
│   └── main.go
├── internal/              # Private application code
│   ├── api/               # REST API server
│   ├── app/               # Main application model
│   ├── components/        # Reusable UI components
│   ├── config/            # Configuration management
│   ├── domain/            # Core domain types
│   ├── executor/          # Execution engine
│   ├── git/               # Git integration
│   ├── messages/          # Message types
│   ├── notify/            # Desktop notifications
│   ├── parser/            # YAML parsing
│   ├── preflight/         # Pre-flight checks
│   ├── profile/           # Profile management
│   ├── sound/             # Audio feedback
│   ├── storage/           # SQLite persistence
│   ├── testutil/          # Test utilities
│   ├── theme/             # Theming system
│   ├── views/             # View models
│   ├── watcher/           # File watching
│   └── workflow/          # Custom workflows
├── docs/                  # Documentation
├── Makefile               # Build automation
├── go.mod                 # Go module definition
└── go.sum                 # Dependency checksums
```

## Development Workflow

### Making Changes

1. **Create a feature branch**

```bash
git checkout -b feature/my-feature
```

2. **Make your changes**

3. **Format and lint**

```bash
make fmt
make lint
```

4. **Run tests**

```bash
make test
```

5. **Build and test manually**

```bash
make build
./bin/bmad
```

6. **Commit your changes**

```bash
git add .
git commit -m "Add feature: description"
```

7. **Push and create a pull request**

```bash
git push origin feature/my-feature
```

### Using Live Reload

For faster development iteration:

```bash
make dev
# or
air
```

This will automatically rebuild and restart the application when files change.

## Code Guidelines

### Go Style

Follow the [Effective Go](https://golang.org/doc/effective_go.html) guidelines:

- Use `gofmt` for formatting
- Keep functions focused and small
- Document exported types and functions
- Use meaningful variable names

### Project Conventions

#### Naming

- Package names: lowercase, single word when possible
- Types: PascalCase (e.g., `StoryList`, `ExecutionStatus`)
- Functions: PascalCase for exported, camelCase for private
- Variables: camelCase
- Constants: PascalCase

#### File Organization

- One type per file when the type is substantial
- Group related small types in a single file
- Test files: `*_test.go` alongside source

#### Error Handling

```go
// Return errors, don't panic
func DoSomething() error {
    if err := something(); err != nil {
        return fmt.Errorf("failed to do something: %w", err)
    }
    return nil
}
```

### Message Types

When adding new functionality, define message types:

```go
// internal/messages/messages.go

// MyFeatureMsg is sent when...
type MyFeatureMsg struct {
    Data string
}

// MyFeatureResultMsg is sent when...
type MyFeatureResultMsg struct {
    Result string
    Error  error
}
```

### View Components

Follow the Bubble Tea pattern:

```go
// internal/views/myview/myview.go
package myview

import (
    tea "github.com/charmbracelet/bubbletea"
    "github.com/charmbracelet/lipgloss"
)

type Model struct {
    // State
    width  int
    height int
    // ...
}

func New() Model {
    return Model{}
}

func (m Model) Init() tea.Cmd {
    return nil
}

func (m Model) Update(msg tea.Msg) (Model, tea.Cmd) {
    switch msg := msg.(type) {
    case tea.WindowSizeMsg:
        m.width = msg.Width
        m.height = msg.Height
    }
    return m, nil
}

func (m Model) View() string {
    // Render view
    return "My View"
}
```

## Testing

### Running Tests

```bash
# All tests
make test

# With verbose output
go test -v ./...

# Specific package
go test -v ./internal/executor/...

# With coverage
make test-coverage
```

### Writing Tests

```go
// internal/domain/story_test.go
package domain

import (
    "testing"
    "github.com/stretchr/testify/assert"
)

func TestStory_IsActionable(t *testing.T) {
    tests := []struct {
        name     string
        status   StoryStatus
        expected bool
    }{
        {"in-progress is actionable", StatusInProgress, true},
        {"done is not actionable", StatusDone, false},
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            story := Story{Status: tt.status}
            assert.Equal(t, tt.expected, story.IsActionable())
        })
    }
}
```

### Test Utilities

Use the test utilities in `internal/testutil/`:

```go
import "github.com/robertguss/bmad-automate-go/internal/testutil"

func TestSomething(t *testing.T) {
    // Use test fixtures, helpers, etc.
}
```

## Adding Features

### New View

1. Create the view package:

```bash
mkdir -p internal/views/myview
```

2. Implement the view model:

```go
// internal/views/myview/myview.go
package myview

// Implement Model with Init, Update, View methods
```

3. Add to domain views:

```go
// internal/domain/view.go
const ViewMyView View = "myview"
```

4. Register in app:

```go
// internal/app/app.go
import "github.com/robertguss/bmad-automate-go/internal/views/myview"

type App struct {
    // ...
    myview myview.Model
}
```

5. Add navigation:

```go
// internal/app/app.go
case "m": // Key for my view
    return m.navigate(domain.ViewMyView)
```

### New Message Type

1. Define in messages package:

```go
// internal/messages/messages.go
type MyNewMsg struct {
    // Fields
}
```

2. Handle in app or relevant view:

```go
// internal/app/app.go
case messages.MyNewMsg:
    // Handle message
```

### New Executor Feature

1. Add to executor:

```go
// internal/executor/executor.go
func (e *Executor) NewFeature() tea.Cmd {
    // Implementation
}
```

2. Add message types if needed

3. Wire up to views

## Architecture Decisions

### Why Bubble Tea?

- Clean TEA architecture
- Cross-platform terminal support
- Active community and ecosystem
- Composable components

### Why SQLite?

- No external database required
- CGO-free via modernc.org/sqlite
- Portable data files
- Fast for our use case

### Why Go?

- Single binary deployment
- Cross-platform compilation
- Good concurrency primitives
- Fast build times

## Pull Request Guidelines

### Before Submitting

1. Run all checks:

```bash
make fmt
make lint
make test
```

2. Ensure your code builds:

```bash
make build
```

3. Test manually:

```bash
./bin/bmad
```

### PR Description

Include:

- What the change does
- Why it's needed
- How to test it
- Screenshots for UI changes

### Code Review

- Respond to feedback constructively
- Make requested changes promptly
- Keep discussions focused on code

## Release Process

### Versioning

We use semantic versioning (SemVer):

- **Major**: Breaking changes
- **Minor**: New features (backward compatible)
- **Patch**: Bug fixes

### Creating a Release

1. Update version in relevant files
2. Update CHANGELOG.md
3. Create and push tag:

```bash
git tag v1.2.3
git push origin v1.2.3
```

4. GoReleaser will create the release automatically

### Snapshot Releases

For testing:

```bash
make snapshot
```

## Getting Help

### Resources

- [Go Documentation](https://golang.org/doc/)
- [Bubble Tea Docs](https://github.com/charmbracelet/bubbletea)
- [Lipgloss Docs](https://github.com/charmbracelet/lipgloss)

### Questions

- Open an issue for bugs or feature requests
- Use discussions for questions
- Tag issues appropriately

## Code of Conduct

Be respectful and constructive. We welcome contributions from everyone.

## License

By contributing, you agree that your contributions will be licensed under the MIT License.
