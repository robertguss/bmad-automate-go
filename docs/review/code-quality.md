# Code Quality Report

## Summary

| Metric                | Value         | Rating                        |
| --------------------- | ------------- | ----------------------------- |
| Total Lines of Code   | ~17,326       | Medium-sized                  |
| Test Coverage (Core)  | 28-100%       | Needs improvement             |
| Test Coverage (UI)    | 0%            | Critical gap                  |
| Cyclomatic Complexity | Moderate-High | app.go needs attention        |
| SOLID Compliance      | Partial       | SRP violations present        |
| Code Duplication      | Moderate      | 6 files with formatDuration() |
| Magic Numbers         | High          | 50+ hardcoded values          |

**Overall Score: 6.5/10**

---

## 1. Complexity Analysis

### Critical: Update() Method

**Location:** `internal/app/app.go:237-826`

| Issue                 | Value | Recommended      |
| --------------------- | ----- | ---------------- |
| Lines                 | 589   | < 50             |
| Switch cases          | 40+   | Split by concern |
| Cyclomatic complexity | ~45   | < 10             |

**Problem:** Single massive switch statement handling all message types.

**Remediation:**

```go
// Extract message handlers
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
    switch msg := msg.(type) {
    case tea.KeyMsg:
        return m.handleKeyMsg(msg)
    case tea.WindowSizeMsg:
        return m.handleWindowSize(msg)
    default:
        return m.handleAppMessage(msg)
    }
}

func (m Model) handleKeyMsg(msg tea.KeyMsg) (Model, tea.Cmd) {
    switch m.activeView {
    case domain.ViewExecution:
        return m.handleExecutionKeys(msg)
    case domain.ViewStoryList:
        return m.handleStoryListKeys(msg)
    // ...
    }
    return m.handleGlobalKeys(msg)
}
```

### Other Long Methods

| File                    | Method          | Lines | Recommended |
| ----------------------- | --------------- | ----- | ----------- |
| `storage/sqlite.go:349` | `GetStats()`    | 123   | < 50        |
| `executor/batch.go:126` | `Start()`       | 92    | < 50        |
| `executor/batch.go:221` | `executeItem()` | 101   | < 50        |

---

## 2. SOLID Violations

### Single Responsibility Principle (SRP)

**Model struct in app.go (lines 43-107)**

The Model struct holds 25+ fields spanning 7+ responsibilities:

- Dimensions (View concern)
- Navigation (Router concern)
- Configuration (Config concern)
- Data state (State concern)
- Storage (Persistence concern)
- 3 Executor types
- 9 View models
- 4 Component models
- 5 Service instances

**Recommendation:** Extract into focused structs:

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

### Interface Segregation - Good

The `Storage` interface at `internal/storage/storage.go:92-115` is well-designed with logical groupings.

**Suggestion:** Consider splitting into `ExecutionStore` and `StatsReader` interfaces.

---

## 3. Code Duplication

### formatDuration() - 6 Implementations

| File                                    | Lines    |
| --------------------------------------- | -------- |
| `internal/app/app.go`                   | 951-958  |
| `internal/views/queue/queue.go`         | 457-469  |
| `internal/views/history/history.go`     | 454-469  |
| `internal/views/timeline/timeline.go`   | ~similar |
| `internal/views/stats/stats.go`         | ~similar |
| `internal/views/execution/execution.go` | ~similar |

**Remediation:** Create `internal/util/format.go`:

```go
package util

import (
    "fmt"
    "time"
)

func FormatDuration(d time.Duration) string {
    if d < time.Minute {
        return fmt.Sprintf("%ds", int(d.Seconds()))
    }
    if d < time.Hour {
        minutes := int(d.Minutes())
        seconds := int(d.Seconds()) % 60
        return fmt.Sprintf("%dm %02ds", minutes, seconds)
    }
    hours := int(d.Hours())
    minutes := int(d.Minutes()) % 60
    return fmt.Sprintf("%dh %02dm", hours, minutes)
}
```

### waitIfPaused() - 3 Implementations

| File                            | Lines   |
| ------------------------------- | ------- |
| `internal/executor/executor.go` | 380-398 |
| `internal/executor/batch.go`    | 401-418 |
| `internal/executor/parallel.go` | 420-438 |

**Remediation:** Create shared `PauseController`:

```go
// internal/executor/pause.go
package executor

type PauseController struct {
    mu       sync.Mutex
    paused   bool
    canceled bool
    resumeCh chan struct{}
}

func (pc *PauseController) WaitIfPaused(ctx context.Context) error {
    for {
        pc.mu.Lock()
        paused := pc.paused
        canceled := pc.canceled
        pc.mu.Unlock()

        if canceled {
            return context.Canceled
        }
        if !paused {
            return nil
        }

        select {
        case <-ctx.Done():
            return ctx.Err()
        case <-pc.resumeCh:
            return nil
        case <-time.After(100 * time.Millisecond):
            // Continue polling
        }
    }
}
```

---

## 4. Magic Numbers

### High-Severity Instances

| File:Line                       | Value       | Should Be                        |
| ------------------------------- | ----------- | -------------------------------- |
| `app/app.go:523`                | `4`         | `headerHeight + statusBarHeight` |
| `storage/sqlite.go:195`         | `1000`      | `const maxOutputLines = 1000`    |
| `storage/sqlite.go:274`         | `100`       | `const defaultLimit = 100`       |
| `executor/executor.go:237`      | `64*1024`   | `const initialBufferSize`        |
| `executor/executor.go:238`      | `1024*1024` | `const maxBufferSize`            |
| `executor/parallel.go:64-65`    | `10`        | `const maxParallelWorkers = 10`  |
| `executor/parallel.go:71-72`    | `100`       | `const jobQueueSize = 100`       |
| `views/settings/settings.go:82` | `3600`      | `const maxTimeoutSeconds = 3600` |
| `app/app.go:601`                | `4`         | `len(domain.AllSteps())`         |

### Polling Intervals (100ms everywhere)

| File:Line                  |
| -------------------------- |
| `executor/executor.go:394` |
| `executor/batch.go:415`    |
| `executor/parallel.go:435` |

**Remediation:** Create constants file:

```go
// internal/constants/constants.go
package constants

import "time"

const (
    PollInterval       = 100 * time.Millisecond
    MaxOutputLines     = 1000
    DefaultQueryLimit  = 100
    MaxParallelWorkers = 10
    JobQueueSize       = 100
    InitialBufferSize  = 64 * 1024
    MaxBufferSize      = 1024 * 1024
)
```

---

## 5. Error Handling Issues

| File:Line                   | Issue                                                               |
| --------------------------- | ------------------------------------------------------------------- |
| `app/app.go:118`            | `store, _ = storage.NewSQLiteStorage(...)` - Error silently ignored |
| `app/app.go:225-228`        | Returns `nil` on error instead of error message                     |
| `app/app.go:1047`           | Returns `nil` when storage is nil                                   |
| `storage/sqlite.go:590-596` | Time parse errors silently ignored                                  |

**Remediation for app.go:118:**

```go
store, err := storage.NewSQLiteStorage(cfg.DatabasePath)
if err != nil {
    // Log warning but continue - app can work without persistence
    log.Printf("Warning: Could not initialize storage: %v", err)
}
```

---

## 6. Potential Dead Code

| File:Line         | Item                  | Notes               |
| ----------------- | --------------------- | ------------------- |
| `app/app.go:1261` | `switchProfile()`     | Caller path unclear |
| `app/app.go:1301` | `switchWorkflow()`    | Caller path unclear |
| `app/app.go:1314` | `GetActiveWorkflow()` | Potentially unused  |
| `app/app.go:1320` | `GetActiveProfile()`  | Potentially unused  |

---

## 7. Refactoring Priority

### Priority 1: Critical

1. **Split Update() method** - Extract message handlers
2. **Extract formatDuration()** - Create shared utility
3. **Fix error handling** - Don't ignore storage errors

### Priority 2: High

4. **Extract PauseController** - Deduplicate executor code
5. **Define constants** - Replace magic numbers
6. **Add tests for views** - 0% coverage currently

### Priority 3: Medium

7. **Split GetStats()** - Break into smaller queries
8. **Create ViewBase** - Common view functionality
9. **Define sentinel errors** - Better error handling

---

## 8. Files Requiring Attention

| File                            | Lines | Issues                        |
| ------------------------------- | ----- | ----------------------------- |
| `internal/app/app.go`           | 1343  | God object, Update() too long |
| `internal/storage/sqlite.go`    | 743   | GetStats() too long           |
| `internal/executor/batch.go`    | 427   | Duplication with executor.go  |
| `internal/executor/parallel.go` | 473   | Duplication with batch.go     |
