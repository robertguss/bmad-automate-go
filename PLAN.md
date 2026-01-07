# Plan: bmad-automate-go - Full TUI Application

## Overview

Create a new Go TUI application at `/Users/robertguss/Projects/bmad-automate-go` that replicates and significantly enhances the Python `bmad-automate` CLI tool using the Charm ecosystem (Bubble Tea, Lip Gloss, Bubbles).

## Tech Stack

- **Go 1.21+**
- **Bubble Tea** - TUI framework (Elm architecture)
- **Lip Gloss** - Styling and layout
- **Bubbles** - Pre-built components
- **modernc.org/sqlite** - CGO-free SQLite for cross-platform builds
- **gopkg.in/yaml.v3** - YAML parsing
- **fsnotify** - File watching
- **go-chi** + **nhooyr.io/websocket** - REST API mode

## Project Structure

```
bmad-automate-go/
├── cmd/bmad/main.go              # Entry point
├── internal/
│   ├── app/                      # Main app model, routing, messages
│   ├── views/                    # All 9 views (dashboard, storylist, queue, etc.)
│   ├── components/               # Reusable UI (statusbar, header, palette, modal, confetti)
│   ├── domain/                   # Story, Step, Execution, Queue models
│   ├── executor/                 # Claude CLI execution with live streaming
│   ├── parser/                   # YAML + git diff parsing
│   ├── storage/                  # SQLite persistence
│   ├── config/                   # Configuration + profiles
│   ├── theme/                    # Catppuccin, Dracula, Nord themes
│   ├── notify/                   # Desktop notifications + sound
│   ├── preflight/                # Pre-flight checks
│   ├── git/                      # Git operations
│   ├── watcher/                  # File system watcher
│   └── api/                      # REST API server
├── migrations/001_initial.sql
├── go.mod, go.sum
├── Makefile
└── .goreleaser.yaml
```

## Views (9 Total)

| View          | Description                                          |
| ------------- | ---------------------------------------------------- |
| Dashboard     | Overview: story counts, recent activity, quick stats |
| Story List    | Browse/filter stories by status and epic             |
| Queue Manager | Multi-select, reorder, batch process                 |
| Execution     | Split-pane: queue + live output streaming            |
| Timeline      | Visual horizontal bars showing step durations        |
| Diff Preview  | Git diff viewer before commits                       |
| History       | Search past executions (SQLite-backed)               |
| Statistics    | Success rates, average times, trends                 |
| Settings      | Timeouts, retries, paths, themes, sounds             |

## Key Features

- **Live output streaming** - Real-time Claude output via `tea.Program.Send()`
- **Step controls** - Pause/resume, skip step, retry failed, cancel
- **Progress estimation** - ETA based on historical averages
- **Arrow-key navigation** - Standard navigation (no vim bindings)
- **Command palette** - Ctrl+P fuzzy finder for quick actions
- **SQLite history** - All executions logged with full details
- **Theming** - Catppuccin (default), Dracula, Nord + custom YAML
- **Desktop notifications** - macOS osascript when tasks complete
- **Sound feedback** - Optional audio cues
- **Confetti animation** - Celebrate successful completions
- **Pre-flight checks** - Verify Claude CLI, paths, YAML validity
- **Auto-skip intelligence** - Detect existing story files
- **Git status awareness** - Show branch, warn uncommitted changes
- **Dry-run mode** - Preview commands without executing
- **Profile system** - Multiple project configurations
- **Custom workflows** - Define step sequences in YAML
- **Parallel execution** - Worker pool for independent stories
- **Watch mode** - Auto-refresh on sprint-status.yaml changes
- **REST API mode** - HTTP server with WebSocket for live output

---

## Implementation Phases

### Phase 1: Foundation

**Goal:** Runnable TUI shell with navigation and story display

1. Initialize Go module and directory structure
2. Set up Makefile with build/run/lint targets
3. Create main app model with view routing
4. Implement header + status bar components
5. Implement YAML parser for sprint-status.yaml
6. Create Dashboard view (story counts, navigation)
7. Create Story List view (filter by epic/status)
8. Add arrow-key navigation throughout
9. Set up basic Catppuccin theme

**Deliverable:** TUI that displays stories from YAML with navigation

### Phase 2: Execution Engine

**Goal:** Execute workflows with live output

1. Implement executor with Claude CLI command builder
2. Add subprocess management with context/timeout
3. Implement live stdout/stderr streaming via goroutines
4. Create Execution view with split-pane layout
5. Add step progress indicators
6. Implement step controls (pause, skip, retry, cancel)
7. Add pre-flight checks module
8. Implement retry logic

**Deliverable:** Execute single story through full workflow with live output

### Phase 3: Queue & Batch

**Goal:** Process multiple stories

1. Create Queue Manager view
2. Implement multi-select in Story List
3. Add queue reordering (move up/down)
4. Implement sequential batch processing
5. Add pause/resume queue functionality
6. Implement auto-skip intelligence
7. Create Timeline view with duration bars
8. Add progress estimation (ETA)

**Deliverable:** Select and process multiple stories in batch

### Phase 4: Persistence & History

**Goal:** SQLite storage and analytics

1. Set up SQLite with modernc.org/sqlite
2. Create database schema and migrations
3. Implement execution history storage
4. Create History view with search/filter
5. Create Statistics view with charts
6. Create Diff Preview view
7. Add historical averages for ETA

**Deliverable:** Full history tracking and statistics

### Phase 5: Polish & UX

**Goal:** Enhanced user experience

1. Implement Command Palette (Ctrl+P)
2. Add Dracula and Nord themes
3. Add custom theme loading (YAML)
4. Implement Settings view
5. Add desktop notifications (macOS)
6. Add optional sound feedback
7. Add confetti animation on success
8. Add git status awareness to UI

**Deliverable:** Polished, themeable application

### Phase 6: Advanced Features

**Goal:** Power user capabilities

1. Implement Profile system
2. Add custom workflow definitions
3. Implement Watch mode (fsnotify)
4. Add parallel execution with worker pool
5. Implement REST API server
6. Add WebSocket for live output via API
7. Set up goreleaser for releases
8. Add Homebrew formula

**Deliverable:** Feature-complete application ready for distribution

---

## Reference Files

- `/Users/robertguss/Projects/bmad-automate/src/bmad_automate/cli.py` - Python implementation to replicate

## Build Approach

**Iterative** - Build one phase at a time, user tests between phases.

## Key Design Decisions

1. **CGO-free SQLite** - Use `modernc.org/sqlite` for easy cross-compilation
2. **Elm Architecture** - Single root model with embedded view models
3. **Live streaming** - `tea.Program.Send()` from goroutines for async output
4. **Arrow keys only** - Standard navigation, no vim bindings per user request
