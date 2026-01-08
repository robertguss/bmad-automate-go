# Security Documentation

This document describes the security considerations, configurations, and best practices for bmad-automate-go.

## Security Configuration

### Environment Variables

| Variable            | Description                                                   | Default                                 |
| ------------------- | ------------------------------------------------------------- | --------------------------------------- |
| `BMAD_API_KEY`      | API key for authenticating REST API and WebSocket connections | (empty - no auth)                       |
| `BMAD_CORS_ORIGINS` | Comma-separated list of allowed CORS origins                  | `http://localhost:*,http://127.0.0.1:*` |

### API Authentication (SEC-004)

When `BMAD_API_KEY` is set, all `/api/*` endpoints require authentication. Provide the key via:

1. **Header**: `X-API-Key: your-secret-key`
2. **Bearer Token**: `Authorization: Bearer your-secret-key`

Example:

```bash
# Set API key
export BMAD_API_KEY="your-secret-key-here"

# Make authenticated request
curl -H "X-API-Key: your-secret-key-here" http://localhost:8080/api/stories
```

The `/health` endpoint remains public for load balancer health checks.

### CORS Configuration (SEC-003)

By default, CORS is restricted to localhost origins only:

- `http://localhost:*`
- `http://127.0.0.1:*`

To allow additional origins:

```bash
export BMAD_CORS_ORIGINS="http://localhost:*,https://your-app.example.com"
```

### WebSocket Authentication (SEC-005/006)

WebSocket connections at `/api/ws` respect the same authentication as the REST API:

1. **Query parameter**: `ws://localhost:8080/api/ws?api_key=your-secret-key`
2. **Header**: `X-API-Key: your-secret-key` (for clients that support it)

WebSocket origin restrictions match the CORS configuration.

## Important Security Considerations

### Claude CLI `--dangerously-skip-permissions` Flag (SEC-002)

**WARNING**: This application uses the `--dangerously-skip-permissions` flag when invoking Claude CLI for automated story processing.

#### What This Means

The flag bypasses Claude CLI's interactive permission prompts, allowing Claude to:

- Read and write files without confirmation
- Execute shell commands without approval
- Make changes to your codebase automatically

#### Why It's Used

The bmad-automate-go tool is designed for automated batch processing of stories. Interactive prompts would break the automation workflow. The flag is necessary for:

- Unattended execution of story workflows
- Batch processing multiple stories
- CI/CD integration

#### Risk Mitigation

To minimize risks when using this tool:

1. **Run in isolated environments**: Use Docker, VMs, or dedicated development machines
2. **Limit file access**: Run with a restricted user account
3. **Review changes**: Always review generated code before committing
4. **Use version control**: Keep your work in git to easily revert unwanted changes
5. **Set working directory carefully**: The `--working-dir` flag limits Claude's scope
6. **Monitor execution**: Watch the output for unexpected behavior

#### Recommended Setup

For production use, consider:

```bash
# Create a restricted user for running bmad
sudo useradd -m -s /bin/bash bmad-runner

# Run in a Docker container with limited mounts
docker run -v /path/to/project:/workspace bmad-automate \
  --working-dir /workspace

# Use a dedicated development branch
git checkout -b feature/automated-stories
```

## Security Best Practices

### Development Environment

1. **Always set an API key** when exposing the API server:

   ```bash
   export BMAD_API_KEY=$(openssl rand -hex 32)
   ```

2. **Restrict CORS origins** to your frontend application:

   ```bash
   export BMAD_CORS_ORIGINS="http://localhost:3000"
   ```

3. **Use HTTPS** in production (consider a reverse proxy like nginx or Caddy)

### Production Deployment

1. Never expose the API server directly to the internet without:
   - API key authentication enabled
   - HTTPS termination
   - Rate limiting (consider a reverse proxy)

2. Monitor execution logs for suspicious activity

3. Regularly update dependencies:
   ```bash
   go get -u ./...
   go mod tidy
   ```

## Vulnerability Reporting

If you discover a security vulnerability, please report it by:

1. Opening a private security advisory on GitHub
2. Emailing the maintainers directly
3. NOT creating a public issue

## Security Audit History

| Date       | Finding                          | Severity | Status     |
| ---------- | -------------------------------- | -------- | ---------- |
| 2026-01-07 | SEC-001: Shell command injection | CRITICAL | Fixed      |
| 2026-01-07 | SEC-002: Dangerous CLI flag      | CRITICAL | Documented |
| 2026-01-07 | SEC-003: Permissive CORS         | HIGH     | Fixed      |
| 2026-01-07 | SEC-004: No API authentication   | HIGH     | Fixed      |
| 2026-01-07 | SEC-005: No WebSocket auth       | HIGH     | Fixed      |
| 2026-01-07 | SEC-006: WebSocket origin bypass | HIGH     | Fixed      |
| 2026-01-08 | SEC-007: No rate limiting        | MEDIUM   | Fixed      |
| 2026-01-08 | SEC-008: Path traversal profiles | MEDIUM   | Fixed      |
| 2026-01-08 | SEC-011: LIKE wildcard injection | LOW      | Fixed      |
| 2026-01-08 | SEC-012: No API input validation | MEDIUM   | Fixed      |

## Changes Made

### SEC-001: Shell Command Injection Fix

**Before**: Commands were built as strings and executed via `sh -c`:

```go
cmd := exec.CommandContext(ctx, "sh", "-c", step.Command)
```

**After**: Commands use separate arguments preventing shell injection:

```go
cmd := exec.CommandContext(ctx, step.CommandName, step.CommandArgs...)
```

### SEC-003/004/005/006: Authentication and CORS

Added configurable security middleware:

- CORS restricted to configured origins (default: localhost only)
- API key authentication for REST endpoints
- WebSocket authentication with API key validation
- Origin restriction for WebSocket connections

### SEC-007: Rate Limiting

Added per-IP rate limiting middleware using `golang.org/x/time/rate`:

- Token bucket algorithm with 100 requests/second, burst of 200
- Per-IP tracking with automatic cleanup every 10 minutes
- Returns HTTP 429 with `Retry-After` header when limit exceeded
- Location: `internal/api/server.go`

### SEC-008: Path Traversal Validation

Added profile name validation to prevent directory traversal attacks:

- Rejects names containing `/`, `\`, or `..`
- Rejects names starting with `.` (hidden files)
- Applied to both Save and Delete profile operations
- Location: `internal/profile/profile.go`

### SEC-011: LIKE Wildcard Injection

Fixed SQL LIKE wildcard injection in storage queries:

- Added `escapeLikeWildcards()` function to escape `%`, `_`, and `\`
- Uses `ESCAPE '\'` clause in LIKE queries
- Location: `internal/storage/sqlite.go`

### SEC-012: API Input Validation

Added comprehensive input validation for API endpoints:

- Body size limit middleware (1MB max) prevents memory exhaustion
- `validatePathParam()` checks URL parameters for path traversal
- `decodeJSONBody()` safely parses JSON with:
  - Content-Type validation
  - Empty body detection
  - Malformed JSON handling
  - Oversized body protection (via MaxBytesReader)
- Applied to all handlers accepting URL parameters or JSON bodies
- Location: `internal/api/server.go`
