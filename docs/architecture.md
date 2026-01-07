# Architecture Overview

This document describes the architecture of BMAD Automate, a terminal UI application for automating BMAD workflows.

## High-Level Architecture

BMAD Automate follows **The Elm Architecture (TEA)** pattern via the [Bubble Tea](https://github.com/charmbracelet/bubbletea) framework:

```
┌─────────────────────────────────────────────────────────────────────┐
│                         User Interface                               │
│  ┌──────────┬──────────┬──────────┬──────────┬──────────┬────────┐ │
│  │Dashboard │StoryList │  Queue   │Execution │ Timeline │History │ │
│  │   View   │   View   │   View   │   View   │   View   │  View  │ │
│  └──────────┴──────────┴──────────┴──────────┴──────────┴────────┘ │
├─────────────────────────────────────────────────────────────────────┤
│                      Application Layer                               │
│  ┌─────────────────────────────────────────────────────────────┐   │
│  │                    App Model (app.go)                        │   │
│  │  - Message routing       - View navigation                   │   │
│  │  - State management      - Component coordination            │   │
│  └─────────────────────────────────────────────────────────────┘   │
├─────────────────────────────────────────────────────────────────────┤
│                         Domain Layer                                 │
│  ┌──────────┬──────────┬──────────┬──────────┬──────────────────┐  │
│  │  Story   │Execution │  Queue   │  Step    │  ExecutionStatus │  │
│  │          │          │          │Execution │                  │  │
│  └──────────┴──────────┴──────────┴──────────┴──────────────────┘  │
├─────────────────────────────────────────────────────────────────────┤
│                      Infrastructure Layer                            │
│  ┌──────────┬──────────┬──────────┬──────────┬──────────────────┐  │
│  │ Executor │ Storage  │   API    │ Watcher  │   Parser         │  │
│  │ (Claude) │ (SQLite) │ (HTTP)   │(fsnotify)│   (YAML)         │  │
│  └──────────┴──────────┴──────────┴──────────┴──────────────────┘  │
└─────────────────────────────────────────────────────────────────────┘
```

## The Elm Architecture (TEA)

The application follows TEA with three core concepts:

### Model

The `App` struct in `internal/app/app.go` maintains all application state:

```go
type App struct {
    // View management
    currentView  domain.View
    viewStack    []domain.View

    // Domain data
    stories      []domain.Story
    selected     map[string]bool

    // Sub-components
    executor     *executor.Executor
    storage      storage.Storage
    apiServer    *api.Server
    watcher      *watcher.Watcher

    // View models
    dashboard    dashboard.Model
    storylist    storylist.Model
    queue        queue.Model
    execution    execution.Model
    // ... more views
}
```

### Update

Messages trigger state transitions in the `Update` method:

```go
func (m App) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
    switch msg := msg.(type) {
    case messages.NavigateMsg:
        return m.handleNavigation(msg)
    case messages.ExecutionStartMsg:
        return m.handleExecutionStart(msg)
    case messages.StepOutputMsg:
        return m.handleStepOutput(msg)
    // ... more handlers
    }
}
```

### View

Pure rendering functions produce terminal output:

```go
func (m App) View() string {
    var content string
    switch m.currentView {
    case domain.ViewDashboard:
        content = m.dashboard.View()
    case domain.ViewStoryList:
        content = m.storylist.View()
    // ...
    }
    return m.renderFrame(content)
}
```

## Component Diagram

```
┌─────────────────────────────────────────────────────────────────┐
│                          main.go                                 │
│  Creates tea.Program, initializes App                           │
└─────────────────────────┬───────────────────────────────────────┘
                          │
                          ▼
┌─────────────────────────────────────────────────────────────────┐
│                         App Model                                │
│  ┌─────────────┐  ┌─────────────┐  ┌─────────────┐             │
│  │   Config    │  │   Theme     │  │   Styles    │             │
│  └─────────────┘  └─────────────┘  └─────────────┘             │
│                                                                  │
│  ┌─────────────────────────────────────────────────────────┐   │
│  │                    View Router                           │   │
│  │  Dashboard │ StoryList │ Queue │ Execution │ ...        │   │
│  └─────────────────────────────────────────────────────────┘   │
│                                                                  │
│  ┌──────────┐  ┌──────────┐  ┌──────────┐  ┌──────────┐       │
│  │ Executor │  │ Storage  │  │   API    │  │ Watcher  │       │
│  │  Engine  │  │  Layer   │  │  Server  │  │          │       │
│  └──────────┘  └──────────┘  └──────────┘  └──────────┘       │
└─────────────────────────────────────────────────────────────────┘
```

## Data Flow

### Execution Flow

```
User selects story → ExecutionStartMsg → Executor.Execute()
                                              │
                                              ▼
                          ┌─────────────────────────────────┐
                          │        For each step:           │
                          │  1. StepStartedMsg              │
                          │  2. runCommand() with pipes     │
                          │  3. StepOutputMsg (streaming)   │
                          │  4. StepCompletedMsg            │
                          └─────────────────────────────────┘
                                              │
                                              ▼
                              ExecutionCompletedMsg
                                              │
                                              ▼
                              Storage.SaveExecution()
```

### Message Flow

```
┌─────────┐    KeyPress     ┌─────────┐    Message     ┌─────────┐
│Terminal │ ───────────────▶│   App   │ ──────────────▶│  View   │
│         │                 │ Update  │                │ Update  │
└─────────┘                 └─────────┘                └─────────┘
                                 │
                                 │ tea.Cmd
                                 ▼
                           ┌──────────┐
                           │ Async Op │
                           │(Execute) │
                           └──────────┘
                                 │
                                 │ tea.Msg
                                 ▼
                           ┌──────────┐
                           │   App    │
                           │ Update   │
                           └──────────┘
```

## Package Structure

### Core Packages

| Package             | Purpose                                     |
| ------------------- | ------------------------------------------- |
| `cmd/bmad`          | Application entry point                     |
| `internal/app`      | Main application model and orchestration    |
| `internal/domain`   | Core domain types (Story, Execution, Queue) |
| `internal/messages` | All message types for TEA communication     |
| `internal/config`   | Configuration management                    |
| `internal/theme`    | Theming and styling                         |

### View Packages

| Package                    | Purpose                                 |
| -------------------------- | --------------------------------------- |
| `internal/views/dashboard` | Overview statistics and recent activity |
| `internal/views/storylist` | Story browsing with filtering           |
| `internal/views/queue`     | Queue management and reordering         |
| `internal/views/execution` | Live execution with output streaming    |
| `internal/views/timeline`  | Visual step duration display            |
| `internal/views/history`   | Execution history browser               |
| `internal/views/stats`     | Statistics and trends                   |
| `internal/views/diff`      | Git diff viewer                         |
| `internal/views/settings`  | Settings editor                         |

### Infrastructure Packages

| Package              | Purpose                       |
| -------------------- | ----------------------------- |
| `internal/executor`  | Claude CLI execution engine   |
| `internal/storage`   | SQLite persistence layer      |
| `internal/api`       | REST API and WebSocket server |
| `internal/parser`    | YAML sprint-status parsing    |
| `internal/watcher`   | File system watching          |
| `internal/git`       | Git integration               |
| `internal/profile`   | Profile management            |
| `internal/workflow`  | Custom workflow definitions   |
| `internal/preflight` | Pre-execution checks          |
| `internal/notify`    | Desktop notifications         |
| `internal/sound`     | Audio feedback                |

### Component Packages

| Package                              | Purpose              |
| ------------------------------------ | -------------------- |
| `internal/components/header`         | Top navigation bar   |
| `internal/components/statusbar`      | Bottom status bar    |
| `internal/components/commandpalette` | Fuzzy command finder |
| `internal/components/confetti`       | Success celebration  |

## Domain Models

### Story

Represents a development story from `sprint-status.yaml`:

```go
type Story struct {
    Key        string      // e.g., "3-1-user-auth"
    Epic       int         // Extracted from key
    Status     StoryStatus // in-progress, ready-for-dev, etc.
    Title      string
    FilePath   string
    FileExists bool
}

type StoryStatus string
const (
    StatusInProgress  StoryStatus = "in-progress"
    StatusReadyForDev StoryStatus = "ready-for-dev"
    StatusBacklog     StoryStatus = "backlog"
    StatusDone        StoryStatus = "done"
    StatusBlocked     StoryStatus = "blocked"
)
```

### Execution

Tracks the execution of a story through all workflow steps:

```go
type Execution struct {
    Story     Story
    Status    ExecutionStatus
    Steps     []*StepExecution
    Current   int           // Current step index
    StartTime time.Time
    EndTime   time.Time
    Duration  time.Duration
    Error     string
}

type ExecutionStatus string
const (
    ExecutionPending   ExecutionStatus = "pending"
    ExecutionRunning   ExecutionStatus = "running"
    ExecutionPaused    ExecutionStatus = "paused"
    ExecutionCompleted ExecutionStatus = "completed"
    ExecutionFailed    ExecutionStatus = "failed"
    ExecutionCancelled ExecutionStatus = "cancelled"
)
```

### StepExecution

Tracks individual step state:

```go
type StepExecution struct {
    Name      StepName
    Status    StepStatus
    StartTime time.Time
    EndTime   time.Time
    Duration  time.Duration
    Output    []string      // Lines of output
    Error     string
    Attempt   int           // Retry attempt number
    Command   string        // Claude CLI command
}

type StepName string
const (
    StepCreateStory StepName = "create-story"
    StepDevStory    StepName = "dev-story"
    StepCodeReview  StepName = "code-review"
    StepGitCommit   StepName = "git-commit"
)
```

### Queue

Manages batch execution of multiple stories:

```go
type Queue struct {
    Items       []*QueueItem
    Status      QueueStatus
    Current     int
    StepAverage map[StepName]time.Duration
}

type QueueItem struct {
    Story     Story
    Status    QueueItemStatus
    Position  int
    AddedAt   time.Time
    Execution *Execution
}
```

## Execution Engine

The executor manages Claude CLI command execution:

```
┌─────────────────────────────────────────────────────────────────┐
│                      Executor                                    │
│  ┌─────────────────────────────────────────────────────────┐   │
│  │                   Execute(story)                         │   │
│  │  1. Create Execution                                     │   │
│  │  2. For each step:                                       │   │
│  │     a. Build command (buildCommand)                      │   │
│  │     b. Execute with timeout (runCommand)                 │   │
│  │     c. Stream output via pipes                           │   │
│  │     d. Handle retries on failure                         │   │
│  │  3. Save to storage                                      │   │
│  └─────────────────────────────────────────────────────────┘   │
│                                                                  │
│  ┌──────────┐  ┌──────────┐  ┌──────────┐  ┌──────────┐       │
│  │ pauseCh  │  │ resumeCh │  │ cancelCh │  │  skipCh  │       │
│  │(control) │  │(control) │  │(control) │  │(control) │       │
│  └──────────┘  └──────────┘  └──────────┘  └──────────┘       │
└─────────────────────────────────────────────────────────────────┘
```

### Executor Types

| Type               | Description                        |
| ------------------ | ---------------------------------- |
| `Executor`         | Single-story execution             |
| `BatchExecutor`    | Sequential queue processing        |
| `ParallelExecutor` | Worker pool for parallel execution |

## Storage Layer

SQLite persistence using CGO-free [modernc.org/sqlite](https://modernc.org/sqlite):

```
┌─────────────────────────────────────────────────────────────────┐
│                     Storage Interface                            │
├─────────────────────────────────────────────────────────────────┤
│  SaveExecution(ctx, record) error                               │
│  GetExecution(ctx, id) (*ExecutionRecord, error)                │
│  ListExecutions(ctx, filter) ([]*ExecutionRecord, error)        │
│  GetStats(ctx) (*Stats, error)                                  │
│  GetStepAverages(ctx) (map[StepName]time.Duration, error)       │
└─────────────────────────────────────────────────────────────────┘
                              │
                              ▼
┌─────────────────────────────────────────────────────────────────┐
│                    SQLite Implementation                         │
│  ┌─────────────┐  ┌─────────────┐  ┌─────────────┐             │
│  │ executions  │  │    steps    │  │   indexes   │             │
│  │   table     │  │   table     │  │             │             │
│  └─────────────┘  └─────────────┘  └─────────────┘             │
└─────────────────────────────────────────────────────────────────┘
```

## API Server

REST API with WebSocket support using [go-chi](https://github.com/go-chi/chi):

```
┌─────────────────────────────────────────────────────────────────┐
│                       API Server                                 │
│                                                                  │
│  Routes:                                                         │
│  ├── GET  /health                                               │
│  ├── /api                                                       │
│  │   ├── GET  /stories          - List stories                  │
│  │   ├── GET  /stories/{key}    - Get story                     │
│  │   ├── POST /stories/refresh  - Reload stories                │
│  │   │                                                          │
│  │   ├── GET  /queue            - Queue status                  │
│  │   ├── POST /queue/add        - Add to queue                  │
│  │   ├── DELETE /queue/{key}    - Remove from queue             │
│  │   │                                                          │
│  │   ├── GET  /execution        - Current execution             │
│  │   ├── POST /execution/start  - Start execution               │
│  │   ├── POST /execution/pause  - Pause execution               │
│  │   │                                                          │
│  │   ├── GET  /history          - Execution history             │
│  │   ├── GET  /stats            - Statistics                    │
│  │   └── GET  /ws               - WebSocket                     │
│  │                                                              │
│  WebSocket Hub:                                                 │
│  ├── Broadcast execution events to connected clients            │
│  └── Real-time output streaming                                 │
└─────────────────────────────────────────────────────────────────┘
```

## State Machine Diagrams

### Execution State Machine

```
                    ┌─────────┐
                    │ pending │
                    └────┬────┘
                         │ Start
                         ▼
         ┌────────── running ──────────┐
         │               │             │
    Pause│          Complete      Fail/Cancel
         ▼               │             │
    ┌────────┐           │             │
    │ paused │           │             │
    └───┬────┘           │             │
        │ Resume         │             │
        └────────────────┼─────────────┘
                         │
            ┌────────────┼────────────┐
            │            │            │
            ▼            ▼            ▼
      ┌──────────┐ ┌──────────┐ ┌───────────┐
      │completed │ │  failed  │ │ cancelled │
      └──────────┘ └──────────┘ └───────────┘
```

### Step State Machine

```
    ┌─────────┐
    │ pending │
    └────┬────┘
         │ Start
         ▼
    ┌─────────┐
    │ running │
    └────┬────┘
         │
    ┌────┼────┬─────────┐
    │    │    │         │
    ▼    ▼    ▼         ▼
┌───────┐┌──────┐┌───────┐
│success││failed││skipped│
└───────┘└──────┘└───────┘
```

## Concurrency Model

BMAD Automate uses Go's concurrency primitives:

### Goroutines

- **Output streaming**: Two goroutines per command (stdout, stderr)
- **Duration ticker**: Periodic updates during execution
- **File watcher**: Background monitoring of sprint-status.yaml
- **API server**: HTTP server and WebSocket hub
- **Parallel execution**: Worker pool for multi-story processing

### Synchronization

```go
type Executor struct {
    mu       sync.Mutex    // Protects execution state
    pauseCh  chan struct{} // Pause signal
    resumeCh chan struct{} // Resume signal
    cancelCh chan struct{} // Cancel signal
    skipCh   chan struct{} // Skip signal
}
```

### Message Passing

Bubble Tea's `tea.Program.Send()` enables thread-safe communication between goroutines and the main event loop:

```go
// In a goroutine
e.program.Send(messages.StepOutputMsg{
    StepIndex: index,
    Line:      line,
})
```

## Error Handling

### Execution Errors

1. **Timeout**: Step exceeds configured timeout
2. **Command failure**: Non-zero exit code from Claude CLI
3. **Cancellation**: User-initiated cancel
4. **System errors**: File access, network issues

### Retry Logic

```go
func (e *Executor) executeStep(index int, step *StepExecution) error {
    maxAttempts := e.config.Retries + 1

    for attempt := 1; attempt <= maxAttempts; attempt++ {
        err := e.runCommand(ctx, index, step)
        if err == nil {
            return nil
        }

        if attempt < maxAttempts {
            time.Sleep(2 * time.Second)
            continue
        }

        return err
    }
}
```

## Testing Strategy

### Unit Tests

Located alongside source files with `_test.go` suffix:

- Domain model tests
- Executor tests
- Storage tests
- Parser tests

### Test Utilities

`internal/testutil/` provides:

- Mock fixtures
- Test helpers
- Common assertions

### Running Tests

```bash
# All tests
make test

# With coverage
make test-coverage

# Specific package
go test -v ./internal/executor/...
```

## Performance Considerations

### Output Buffering

Large buffers for streaming Claude CLI output:

```go
buf := make([]byte, 0, 64*1024)    // 64KB initial
scanner.Buffer(buf, 1024*1024)      // 1MB max
```

### ETA Calculation

Moving averages from historical step durations:

```go
func (q *Queue) EstimatedTimeRemaining() time.Duration {
    var total time.Duration
    for _, item := range q.Items {
        if item.Status == QueueItemPending {
            for _, step := range domain.AllSteps() {
                total += q.StepAverage[step]
            }
        }
    }
    return total
}
```

### Database Indexes

SQLite indexes for common queries:

- `idx_executions_story_key`
- `idx_executions_status`
- `idx_executions_start_time`
- `idx_steps_execution_id`
