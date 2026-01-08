# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

BMAD Automate is a terminal UI application that automates BMAD (Business Method for Agile Development) workflows by orchestrating Claude CLI commands. Built with Go 1.24+ using the Charm ecosystem (Bubble Tea, Lipgloss).

## Common Commands

```bash
# Build and run
make build              # Build binary to ./bin/bmad
make run                # Run application directly
make run-api            # Run with REST API server on :8080

# Testing
make test               # Run all tests with verbose output
make test-coverage      # Generate coverage report (coverage.html)
go test -v ./internal/executor/...  # Test single package
go test -run TestName -v ./...      # Run specific test

# Code quality
make fmt                # Format code with gofmt
make lint               # Run golangci-lint

# Development
make dev                # Live reload with air
make deps               # Download and tidy dependencies
```

## Architecture

The application follows The Elm Architecture (TEA) via Bubble Tea:

```
App (internal/app/app.go) - Main model, message routing, view orchestration
├── Views (internal/views/) - 9 views: dashboard, storylist, queue, execution, timeline, history, stats, diff, settings
├── Executor (internal/executor/) - Handles story execution
│   ├── executor.go - Single-story execution engine
│   ├── batch.go - Sequential processing
│   └── parallel.go - Worker pool implementation
├── Storage (internal/storage/) - SQLite persistence (CGO-free via modernc.org/sqlite)
├── API Server (internal/api/) - REST + WebSocket (go-chi + coder/websocket)
└── Components (internal/components/) - Reusable UI: header, statusbar, commandpalette, confetti
```

**Message Flow:**

- User/System events → `App.Update()` → Process message → Send `tea.Cmd`
- Async operations return `tea.Msg` → `App.Update()` → Re-render `View()`

## Key Domain Models

**Story** (`internal/domain/story.go`): Story with key format "epic-number-slug" (e.g., "3-1-user-auth")

**Execution** (`internal/domain/execution.go`): Tracks story execution through workflow steps. Status: pending, running, paused, completed, failed, cancelled

**StepExecution** (`internal/domain/execution.go`): Individual step with output streaming, retry support. Steps: create-story, dev-story, code-review, git-commit

**Queue** (`internal/domain/queue.go`): Batch execution management with ETA calculation

## Concurrency Model

The executor uses channels for control signals:

- `pauseCh`, `resumeCh`, `cancelCh`, `skipCh` - Control execution flow
- Goroutines for: output streaming (stdout/stderr), file watching, API server, parallel workers
- `sync.Mutex` protects execution state
- `tea.Program.Send()` for thread-safe app communication

## Configuration

Application stores data in `.bmad/` directory:

- `.bmad/bmad.db` - SQLite database
- `.bmad/profiles/` - Project profiles
- `.bmad/workflows/` - Custom workflow definitions

Default paths:

- Sprint Status: `_bmad-output/implementation-artifacts/sprint-status.yaml`
- Story Directory: `_bmad-output/implementation-artifacts`

## Testing Patterns

- Tests use `testify/assert` for assertions
- Test files alongside source with `_test.go` suffix
- Test utilities in `/internal/testutil/`

## Task Tracking

Use 'bd' for task tracking
