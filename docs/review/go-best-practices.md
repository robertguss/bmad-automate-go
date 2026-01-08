# Go Best Practices Compliance Report

## Summary Scores

| Category                   | Score        | Status            |
| -------------------------- | ------------ | ----------------- |
| Go Idioms Compliance       | 72/100       | Needs Improvement |
| Concurrency Best Practices | 78/100       | Good              |
| Modern Go Features         | 55/100       | Needs Improvement |
| Code Organization          | 85/100       | Good              |
| Tool Compatibility         | 68/100       | Needs Improvement |
| **Overall**                | **71.6/100** | Needs Improvement |

---

## 1. Go Idioms Compliance (72/100)

### Error Handling Issues

| File:Line               | Issue                                    | Severity |
| ----------------------- | ---------------------------------------- | -------- |
| `storage/storage.go`    | No sentinel errors defined               | Medium   |
| `storage/sqlite.go:579` | Returns string error instead of sentinel | Medium   |
| `api/server.go`         | Returns string errors in handlers        | Low      |
| `app/app.go:652-668`    | Multiple unchecked error returns         | High     |

**Missing Pattern:**

```go
// Recommended sentinel errors (not present):
var (
    ErrNotFound      = errors.New("not found")
    ErrAlreadyExists = errors.New("already exists")
)
```

**Remediation:**

```go
// internal/domain/errors.go
package domain

import "errors"

var (
    ErrExecutionNotFound = errors.New("execution not found")
    ErrQueueEmpty        = errors.New("queue is empty")
    ErrAlreadyRunning    = errors.New("execution already running")
)
```

### Interface Design Issues

| File:Line                          | Issue                                  | Severity |
| ---------------------------------- | -------------------------------------- | -------- |
| `views/settings/settings.go:30-32` | Uses `interface{}` instead of generics | Medium   |
| `api/server.go:181`                | `interface{}` should be `any`          | Low      |
| N/A                                | No executor interface - tight coupling | Medium   |

**Recommendation:** Define Executor interface:

```go
type Executor interface {
    Execute(ctx context.Context, story domain.Story) tea.Cmd
    Pause()
    Resume()
    Cancel()
    GetExecution() *domain.Execution
    IsPaused() bool
}
```

### Naming Conventions

| Status               | Notes                         |
| -------------------- | ----------------------------- |
| Package names        | GOOD - lowercase, single word |
| Exported types       | GOOD - PascalCase             |
| Unexported functions | GOOD - camelCase              |

---

## 2. Concurrency Best Practices (78/100)

### Context Propagation Issues

| File:Line                       | Issue                                     | Severity |
| ------------------------------- | ----------------------------------------- | -------- |
| `executor/executor.go:62`       | Uses `context.Background()` not parameter | High     |
| `executor/batch.go:151`         | Creates root context                      | High     |
| `executor/parallel.go:108`      | Creates root context                      | High     |
| `app/app.go:971,1015,1050,1095` | `context.Background()` in storage calls   | Medium   |

**Current (incorrect):**

```go
func (e *Executor) Execute(story domain.Story) tea.Cmd {
    e.ctx, e.cancel = context.WithCancel(context.Background())
}
```

**Recommended:**

```go
func (e *Executor) Execute(ctx context.Context, story domain.Story) tea.Cmd {
    e.ctx, e.cancel = context.WithCancel(ctx)
}
```

### Mutex vs Channel Usage

**Good Patterns:**

- Non-blocking channel sends with `select/default`
- Worker pool in parallel executor
- Hub broadcast pattern in WebSocket

**Issues:**
| File:Line | Issue |
|-----------|-------|
| `executor/executor.go:380-398` | Polling with `time.After(100ms)` |
| `executor/batch.go:400-418` | Duplicated polling pattern |

**Recommendation:** Use condition variables:

```go
e.cond.L.Lock()
for e.paused && !e.canceled {
    e.cond.Wait()
}
e.cond.L.Unlock()
```

### Goroutine Lifecycle Issues

| File:Line                 | Issue                                         | Severity |
| ------------------------- | --------------------------------------------- | -------- |
| `app/app.go:1249-1252`    | Fire-and-forget API server goroutine          | High     |
| `api/server.go:434,464`   | `go s.batchExecutor.Start()` without tracking | Medium   |
| `executor/executor.go:69` | `go e.runTicker()` no completion wait         | Low      |

**Recommendation:**

```go
type Executor struct {
    done chan struct{}
    wg   sync.WaitGroup
}

func (e *Executor) runTicker() {
    e.wg.Add(1)
    defer e.wg.Done()
    // ...
}

func (e *Executor) Wait() {
    e.wg.Wait()
}
```

---

## 3. Modern Go Features (55/100)

### Missing Features

| Feature                   | Current       | Recommendation     |
| ------------------------- | ------------- | ------------------ |
| **slog** (Go 1.21)        | `log.Printf`  | Structured logging |
| **Generics** (Go 1.18)    | Not used      | Type-safe patterns |
| **any** keyword           | `interface{}` | Modern syntax      |
| **errors.Join** (Go 1.20) | Not used      | Combine errors     |

### Specific Violations

| File:Line                       | Issue                 |
| ------------------------------- | --------------------- |
| `api/websocket.go:141,172,212`  | Uses `log.Printf`     |
| `views/settings/settings.go:30` | Uses `interface{}`    |
| `api/server.go`                 | No structured logging |

**Remediation for slog:**

```go
// Replace:
log.Printf("WebSocket accept error: %v", err)

// With:
slog.Error("WebSocket accept failed", "error", err, "remote", r.RemoteAddr)
```

---

## 4. Code Organization (85/100)

### Package Structure (Excellent)

```
internal/
├── api/           - REST API and WebSocket
├── app/           - Main application logic
├── components/    - Reusable UI components
├── config/        - Configuration handling
├── domain/        - Pure domain models
├── executor/      - Execution logic
├── storage/       - Persistence layer
└── views/         - UI views
```

**Strengths:**

- All app code in `internal/`
- No cyclic dependencies
- Clean domain package with no external dependencies

### Interface Location Issue

The `Storage` interface is defined in `storage` package (provider-side) instead of `app` package (consumer-side).

**Recommendation:**

```go
// internal/app/storage.go (consumer-side)
type StorageReader interface {
    GetStats(ctx context.Context) (*storage.Stats, error)
    ListExecutions(ctx context.Context, filter *storage.ExecutionFilter) ([]*storage.ExecutionRecord, error)
}
```

---

## 5. Tool Compatibility (68/100)

### golangci-lint Issues

**Total: 34 errcheck violations**

| File:Line                     | Unchecked Call                    |
| ----------------------------- | --------------------------------- |
| `storage/sqlite.go:146`       | `tx.Rollback()`                   |
| `api/server.go:184`           | `json.NewEncoder(w).Encode(data)` |
| `app/app.go:126`              | `profileStore.Load()`             |
| `app/app.go:130`              | `workflowStore.Load()`            |
| `app/app.go:652`              | `m.storage.SaveExecution()`       |
| `watcher/watcher.go:61,73,97` | `w.watcher.Add(path)`             |

### go vet

**Status: PASS**

### gofmt

**Status: Minor alignment issues in 2 files**

Files needing formatting:

- `internal/storage/storage.go`
- `internal/views/settings/settings.go`

---

## 6. Modernization Recommendations

### Priority 1: Critical

1. **Check All Error Returns**

   ```bash
   golangci-lint run --enable=errcheck
   ```

2. **Propagate Context**
   - Pass `context.Context` to `Execute()` methods
   - Use context from HTTP requests in API

3. **Define Sentinel Errors**
   ```go
   // internal/domain/errors.go
   var (
       ErrExecutionNotFound = errors.New("execution not found")
       ErrQueueEmpty        = errors.New("queue is empty")
   )
   ```

### Priority 2: High

4. **Adopt slog**

   ```go
   import "log/slog"

   slog.Error("operation failed", "error", err, "context", ctx)
   ```

5. **Replace interface{} with any**
   - Global find/replace

6. **Define Executor Interface**
   - Enable mocking and testing

### Priority 3: Medium

7. **Use Generics**

   ```go
   func Filter[T any](slice []T, predicate func(T) bool) []T
   ```

8. **Replace Polling with Condition Variables**

9. **Add Build Constraints**
   ```go
   //go:build darwin
   ```

---

## 7. Test Coverage Analysis

| Package    | Has Tests | Notes             |
| ---------- | --------- | ----------------- |
| `config`   | Yes       | Good coverage     |
| `domain`   | Yes       | Good coverage     |
| `executor` | Yes       | Needs improvement |
| `storage`  | Yes       | Excellent         |
| `api`      | **No**    | Critical gap      |
| `app`      | **No**    | Critical gap      |
| `views/*`  | **No**    | Medium gap        |

---

## 8. Checklist for PR Reviews

### Must Have

- [ ] All errors checked (`errcheck` passes)
- [ ] Context propagated (no `context.Background()` in library code)
- [ ] Tests for new functionality
- [ ] `gofmt` applied

### Should Have

- [ ] Structured logging with `slog`
- [ ] Uses `any` instead of `interface{}`
- [ ] Interface defined by consumer
- [ ] Goroutines trackable (WaitGroup or done channel)

### Nice to Have

- [ ] Generics where applicable
- [ ] Benchmarks for performance-critical code
- [ ] Table-driven tests
