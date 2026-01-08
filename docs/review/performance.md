# Performance Analysis Report

## Executive Summary

The codebase demonstrates solid architecture but contains several performance bottlenecks that should be addressed for production use at scale.

**Overall Assessment: MODERATE RISK**

---

## 1. CPU/Memory Hotspots

### 1.1 Update() Function (Critical)

**Location:** `internal/app/app.go:237-826`

**Issue:** Large monolithic switch statement with 40+ message types

**Performance Impact:**

- Type assertion overhead (O(n) worst case)
- Branch prediction failures
- Code locality issues (instruction cache)

**Severity: MEDIUM**

**Recommendation:**

```go
type MessageHandler func(m *Model, msg tea.Msg) (Model, tea.Cmd)

var handlers = map[reflect.Type]MessageHandler{
    reflect.TypeOf(messages.StepOutputMsg{}): handleStepOutput,
    // ...
}
```

**Estimated Impact:** 15-25% improvement in message throughput

### 1.2 View Rendering - Style Recreation

**Location:** Multiple view files in `internal/views/`

**Issue:** Style objects created every render frame:

```go
func (m Model) View() string {
    title := lipgloss.NewStyle().
        Foreground(t.Primary).
        Bold(true).
        Render("Timeline")  // New style per render
}
```

**Severity: MEDIUM**

**Recommendation:** Pre-computed style cache:

```go
type Model struct {
    styles struct {
        title    lipgloss.Style
        subtitle lipgloss.Style
    }
}

func (m *Model) initStyles() {
    m.styles.title = lipgloss.NewStyle().Foreground(t.Primary).Bold(true)
}
```

**Estimated Impact:** 20-30% reduction in GC overhead

### 1.3 Redundant View Updates on Resize

**Location:** `internal/app/app.go:536-544`

**Issue:** Updates ALL 8 views on every resize event:

```go
case tea.WindowSizeMsg:
    m.dashboard, _ = m.dashboard.Update(sizeMsg)
    m.storylist, _ = m.storylist.Update(sizeMsg)
    // ... 6 more views
```

**Recommendation:** Lazy view updates:

```go
case tea.WindowSizeMsg:
    m.width = msg.Width
    m.height = msg.Height
    // Only update active view
    switch m.activeView {
    case domain.ViewExecution:
        m.execution.SetSize(msg.Width, contentHeight)
    }
```

**Estimated Impact:** 7x reduction in resize processing

---

## 2. Database Performance

### 2.1 N+1 Query Pattern (Critical)

**Location:** `internal/storage/sqlite.go:294-300`

**Issue:** GetRecentExecutions makes 1 query per execution for steps:

```go
for _, rec := range records {
    steps, err := s.getSteps(ctx, rec.ID, false)  // N queries!
}
```

**Severity: HIGH**

**Recommendation:** Combined query with JOIN:

```go
func (s *SQLiteStorage) GetRecentExecutionsWithSteps(ctx context.Context, limit int) ([]*ExecutionRecord, error) {
    query := `
        SELECT e.*, s.id, s.step_name, s.status, s.duration_ms
        FROM executions e
        LEFT JOIN step_executions s ON s.execution_id = e.id
        ORDER BY e.created_at DESC
        LIMIT ?
    `
}
```

**Estimated Impact:** 80-90% reduction in GetStats() latency

### 2.2 Per-Line INSERT (High)

**Location:** `internal/storage/sqlite.go:200-209`

**Issue:** Individual INSERT per output line (up to 1000 per step):

```go
for i, line := range outputLines {
    _, err = tx.ExecContext(ctx, `INSERT INTO step_outputs ...`, stepID, i, line, false)
}
```

**Severity: MEDIUM**

**Recommendation:** Batch INSERT:

```go
values := make([]interface{}, 0, len(outputLines)*4)
placeholders := make([]string, 0, len(outputLines))
for i, line := range outputLines {
    placeholders = append(placeholders, "(?, ?, ?, ?)")
    values = append(values, stepID, i, line, false)
}
query := fmt.Sprintf("INSERT INTO step_outputs (...) VALUES %s", strings.Join(placeholders, ","))
tx.ExecContext(ctx, query, values...)
```

**Estimated Impact:** 10-15x improvement in SaveExecution latency

### 2.3 Missing Index

**Issue:** No index on `story_epic` column used in GROUP BY queries

**Recommendation:**

```sql
CREATE INDEX idx_executions_story_epic ON executions(story_epic);
```

### 2.4 Query Scaling Concerns

| Executions | Stats Query Time (est.) |
| ---------- | ----------------------- |
| 1,000      | <100ms                  |
| 10,000     | ~500ms                  |
| 100,000    | ~5s                     |

**Recommendation:** Add time-bounded queries:

```sql
SELECT ... FROM executions
WHERE created_at >= datetime('now', '-90 days')
```

---

## 3. Concurrency Performance

### 3.1 Mutex Contention in Output Streaming

**Location:** `internal/executor/executor.go:239-249`

**Issue:** Lock acquired per output line:

```go
go func() {
    for scanner.Scan() {
        e.mu.Lock()           // Lock per line!
        step.Output = append(step.Output, line)
        e.mu.Unlock()
    }
}()
```

**Severity: MEDIUM**

**Recommendation:** Buffered channel pattern or batch updates

**Estimated Impact:** 30-40% reduction in lock contention

### 3.2 Channel Buffer Sizing

| Component            | Current | Recommendation    |
| -------------------- | ------- | ----------------- |
| jobQueue             | 100     | Adequate          |
| resultQueue          | 100     | Increase to 256   |
| WebSocket broadcast  | 256     | Adequate          |
| WebSocketClient send | 64      | May drop messages |

### 3.3 Polling in waitIfPaused()

**Issue:** Uses 100ms polling instead of condition variables

**Recommendation:** Use `sync.Cond` for immediate wake-up

---

## 4. Memory Management

### 4.1 Buffer Sizing (Good)

**Location:** `internal/executor/executor.go:236-238`

```go
buf := make([]byte, 0, 64*1024)    // 64KB initial
scanner.Buffer(buf, 1024*1024)     // 1MB max
```

**Status: APPROPRIATE**

### 4.2 Output Storage Limits

| Metric        | Value      |
| ------------- | ---------- |
| View buffer   | 500 lines  |
| DB storage    | 1000 lines |
| Per execution | ~400KB     |
| Queue of 50   | ~20MB      |

**Status: ACCEPTABLE for typical use**

### 4.3 Model Memory Footprint

| Component          | Size            |
| ------------------ | --------------- |
| execution view     | 500KB           |
| timeline           | 500KB+          |
| other views        | ~10KB each      |
| **Total baseline** | **700KB-1.5MB** |

---

## 5. I/O Performance

### 5.1 File Watching

**Issue:** Path matching calls `filepath.Abs()` on every event

```go
func (w *Watcher) isWatchedPath(path string) bool {
    absPath, _ := filepath.Abs(path)  // syscall every event
}
```

**Recommendation:** Pre-compute absolute paths at AddPath time

**Severity: LOW (debounce mitigates)**

### 5.2 SQLite Configuration (Good)

```go
"PRAGMA journal_mode = WAL",
"PRAGMA synchronous = NORMAL",
"PRAGMA cache_size = -64000",  // 64MB
```

**Status: WELL OPTIMIZED**

---

## 6. Priority Optimization Matrix

| Issue            | Severity | Impact                 | Effort  | Priority |
| ---------------- | -------- | ---------------------- | ------- | -------- |
| N+1 queries      | HIGH     | 80-90% latency         | Medium  | **P1**   |
| Per-line INSERT  | MEDIUM   | 10-15x speed           | Low     | **P1**   |
| Update() switch  | MEDIUM   | 15-25% throughput      | High    | **P2**   |
| Style recreation | MEDIUM   | 20-30% GC              | Medium  | **P2**   |
| Mutex contention | MEDIUM   | 30-40% less contention | Medium  | **P2**   |
| Missing index    | LOW      | Query optimization     | Trivial | **P3**   |
| View resize      | LOW-MED  | 7x resize perf         | Low     | **P3**   |

---

## 7. Recommended Action Plan

### Phase 1 (Immediate)

1. Fix N+1 query with JOIN
2. Implement bulk INSERT
3. Add `story_epic` index

### Phase 2 (Short-term)

4. Implement style caching in views
5. Add queue size limits
6. Reduce mutex contention

### Phase 3 (Medium-term)

7. Refactor Update() handlers
8. Add time-bounded stats queries
9. Lazy view updates on resize

---

## 8. Benchmarking Recommendations

```go
func BenchmarkUpdateMessageRouting(b *testing.B) {
    // Measure message throughput
}

func BenchmarkGetStatsQuery(b *testing.B) {
    // Measure with varying database sizes
}

func BenchmarkViewRender(b *testing.B) {
    // Measure style creation overhead
}
```

Enable profiling:

```go
import _ "net/http/pprof"

go func() {
    http.ListenAndServe("localhost:6060", nil)
}()
```
