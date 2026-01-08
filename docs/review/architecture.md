# Architecture Review Report

## Executive Summary

This is a well-structured Go terminal UI application implementing the Elm Architecture pattern via Bubble Tea. The codebase demonstrates thoughtful package organization with clear separation of concerns.

**Overall Assessment: SOUND with Recommended Improvements**

---

## Architecture Diagram

```
                                    ┌─────────────────────────────────────────────────────────────────┐
                                    │                         cmd/bmad/main.go                        │
                                    │                      (Application Entry Point)                  │
                                    └─────────────────────────────────────────────────────────────────┘
                                                                    │
                                                                    ▼
┌─────────────────────────────────────────────────────────────────────────────────────────────────────────────────┐
│                                              internal/app/app.go                                                │
│                                          (Central Orchestrator - Model)                                         │
└─────────────────────────────────────────────────────────────────────────────────────────────────────────────────┘
         │                      │                      │                      │                      │
         ▼                      ▼                      ▼                      ▼                      ▼
┌────────────────┐    ┌────────────────┐    ┌────────────────┐    ┌────────────────┐    ┌────────────────┐
│   messages/    │    │    views/      │    │   executor/    │    │   storage/     │    │     api/       │
│   messages.go  │    │  (9 Views)     │    │  (3 Patterns)  │    │   (SQLite)     │    │   (REST/WS)    │
└────────────────┘    └────────────────┘    └────────────────┘    └────────────────┘    └────────────────┘
                                                    │
                                                    ▼
                          ┌─────────────────────────────────────────────────────────────┐
                          │                      domain/                                │
                          │              (Core Business Entities)                       │
                          │    Story | Execution | Queue | View | StepExecution        │
                          └─────────────────────────────────────────────────────────────┘

┌─────────────────────────────────────────────────────────────────────────────────────────────────────────────────┐
│                                          Supporting Infrastructure                                              │
│   config | parser | watcher | git | preflight | profile | workflow | theme | sound | notify | testutil         │
│                                                                                                                 │
│   components/: header | statusbar | confetti | commandpalette                                                   │
└─────────────────────────────────────────────────────────────────────────────────────────────────────────────────┘
```

---

## 1. Design Pattern Analysis

### 1.1 Elm Architecture (TEA) Implementation

**Location:** `internal/app/app.go`
**Compliance: GOOD with Minor Issues**

```
Model (app.Model) -> Update(msg) -> (Model, tea.Cmd) -> View() -> string
```

**Strengths:**

- Unidirectional data flow maintained
- Messages defined as distinct types in `internal/messages/messages.go`
- Pure `View()` function that computes UI from state
- Commands returned from `Update()` for side effects

**Issues:**

1. **Model Size Bloat** (Lines 43-107): 30+ fields including views, executors, services
2. **Mixed Concerns in Update**: 600-line function handling 50+ message types
3. **Business Logic in Key Handling**: Navigation mixed with execution control

### 1.2 Repository Pattern

**Location:** `internal/storage/storage.go`
**Compliance: EXCELLENT**

- Clean interface definition with single implementation
- Proper context propagation for cancellation
- Separation of records (DTOs) from domain entities

### 1.3 Command Pattern

**Location:** `internal/executor/executor.go`
**Compliance: GOOD**

- Clear step execution lifecycle
- Retry logic with configurable attempts
- Context-based timeout and cancellation

**Issue:** Command building hardcoded rather than injected

---

## 2. Package Structure

### Dependency Graph

```
cmd/bmad
    └── internal/app
            ├── internal/api
            ├── internal/components/*
            ├── internal/config
            ├── internal/domain
            ├── internal/executor
            ├── internal/messages
            ├── internal/storage
            ├── internal/views/*
            └── (other supporting packages)
```

### Dependency Flow Assessment

| Aspect                 | Status  | Notes                                        |
| ---------------------- | ------- | -------------------------------------------- |
| Inward dependency flow | PARTIAL | Domain is core but app imports everything    |
| Circular dependencies  | NONE    | No import cycles detected                    |
| Domain isolation       | GOOD    | `domain/` has no outward dependencies        |
| View isolation         | GOOD    | Views only depend on domain, messages, theme |

### Package Cohesion

| Package    | Cohesion | Assessment                                    |
| ---------- | -------- | --------------------------------------------- |
| `domain`   | HIGH     | Pure value objects and entities               |
| `storage`  | HIGH     | Single responsibility: persistence            |
| `executor` | MEDIUM   | Three related but distinct execution patterns |
| `app`      | LOW      | Orchestrates too many concerns                |
| `messages` | HIGH     | Message type definitions only                 |
| `views/*`  | HIGH     | Each view is self-contained                   |

---

## 3. Concurrency Architecture

### Executor Patterns

**Single Executor** (`executor.go`):

```go
type Executor struct {
    mu       sync.Mutex
    paused   bool
    canceled bool
    ctx      context.Context
    cancel   context.CancelFunc
}
```

**Batch Executor** (`batch.go`):

- Sequential processing with pause/resume
- Reuses single executor for individual stories
- Proper mutex protection of queue state

**Parallel Executor** (`parallel.go`):

- Worker pool pattern with configurable workers (1-10)
- Job queue and result queue channels
- Independent execution per worker

### Concurrency Issues

| Issue                    | Severity | Location                                        |
| ------------------------ | -------- | ----------------------------------------------- |
| Race condition potential | MEDIUM   | `batch.go:221-230` - accesses after unlock      |
| Channel capacity limits  | LOW      | `parallel.go:71-72` - fixed 100 buffer          |
| Graceful shutdown gaps   | MEDIUM   | `parallel.go:355-375` - collectResults may hang |

---

## 4. Data Layer Design

### Storage Architecture

```
domain/Execution  →  storage/ExecutionRecord  →  SQLite (WAL mode)
   (Runtime)            (Persistence DTO)
```

**Strengths:**

- Separate DTO from domain entity
- Lazy loading of step output
- Proper index coverage
- WAL mode for concurrent access

### Migration Strategy

**Assessment: BASIC**

Migration embedded as const string. Consider:

- Separate migration files in `/migrations/`
- Version tracking beyond single `schema_version` row
- Rollback capability

---

## 5. Coupling Analysis

### Afferent Coupling (Incoming)

| Package    | Coupling     | Risk       |
| ---------- | ------------ | ---------- |
| `domain`   | 15+ packages | LOW (core) |
| `config`   | 10+ packages | MEDIUM     |
| `messages` | 10+ packages | LOW        |

### Efferent Coupling (Outgoing)

| Package      | Coupling     | Risk                |
| ------------ | ------------ | ------------------- |
| `app`        | 20+ packages | HIGH (orchestrator) |
| `api/server` | 8 packages   | MEDIUM              |
| `storage`    | 1 package    | EXCELLENT           |

### Instability Metrics

```
Instability = Efferent / (Afferent + Efferent)

domain:   0.00  (Stable - GOOD)
storage:  0.25  (Stable)
executor: 0.67  (Moderately unstable)
app:      0.95  (Highly unstable - expected for orchestrator)
```

---

## 6. Recommendations

### HIGH Priority: Decompose App Model

**Problem:** Model struct has 30+ fields

**Solution:**

```go
type Model struct {
    dimensions Dimensions
    navigation NavigationState
    services   *Services
    views      *ViewModels
}

type Services struct {
    Config   *config.Config
    Storage  storage.Storage
    Executor ExecutorSet
}
```

### HIGH Priority: Extract Update Handlers

**Problem:** 600-line Update() function

**Solution:**

```go
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
    switch msg := msg.(type) {
    case tea.KeyMsg:
        return m.handleKeyMsg(msg)
    case messages.ExecutionStartMsg, messages.ExecutionStartedMsg:
        return m.handleExecutionMsg(msg)
    case messages.QueueAddMsg, messages.QueueUpdatedMsg:
        return m.handleQueueMsg(msg)
    }
}
```

### MEDIUM Priority: Command Builder Interface

**Problem:** Hardcoded Claude CLI commands

**Solution:**

```go
type CommandBuilder interface {
    Build(stepName domain.StepName, story domain.Story) string
}

type ClaudeCommandBuilder struct {
    config *config.Config
}
```

### MEDIUM Priority: Add Service Layer for API

**Problem:** API handlers directly manipulate executors

**Solution:**

```go
type WorkflowService interface {
    StartQueue() error
    PauseExecution() error
    GetStatus() ExecutionStatus
}
```

---

## 7. Quality Attributes

| Attribute           | Rating  | Evidence                                    |
| ------------------- | ------- | ------------------------------------------- |
| **Reliability**     | GOOD    | Panic recovery, graceful cleanup            |
| **Maintainability** | MEDIUM  | Good packages but app.go monolithic         |
| **Testability**     | MEDIUM  | Interfaces exist but executors hard to mock |
| **Scalability**     | GOOD    | Parallel executor, WAL mode SQLite          |
| **Security**        | CAUTION | `--dangerously-skip-permissions`            |
| **Performance**     | GOOD    | Lazy loading, indexed queries               |

---

## 8. Architecture Debt Summary

| Category               | Items                                      | Effort   |
| ---------------------- | ------------------------------------------ | -------- |
| Structural refactoring | App model decomposition, Update extraction | 2-3 days |
| Interface extraction   | CommandBuilder, ExecutorInterface          | 1 day    |
| Service layer          | API service abstraction                    | 1-2 days |
| Migration system       | Version-tracked migrations                 | 1 day    |
