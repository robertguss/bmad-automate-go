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

## Progress Tracker

| Phase                          | Status      | Completed  |
| ------------------------------ | ----------- | ---------- |
| Phase 1: Foundation            | ✅ Complete | 2025-01-07 |
| Phase 2: Execution Engine      | ✅ Complete | 2025-01-07 |
| Phase 3: Queue & Batch         | ✅ Complete | 2025-01-07 |
| Phase 4: Persistence & History | ✅ Complete | 2026-01-07 |
| Phase 5: Polish & UX           | ✅ Complete | 2026-01-07 |
| Phase 6: Advanced Features     | ✅ Complete | 2026-01-07 |

---

## Implementation Phases

### Phase 1: Foundation ✅ COMPLETE

**Goal:** Runnable TUI shell with navigation and story display

- [x] Initialize Go module and directory structure
- [x] Set up Makefile with build/run/lint targets
- [x] Create main app model with view routing
- [x] Implement header + status bar components
- [x] Implement YAML parser for sprint-status.yaml
- [x] Create Dashboard view (story counts, navigation)
- [x] Create Story List view (filter by epic/status)
- [x] Add arrow-key navigation throughout
- [x] Set up basic Catppuccin theme

**Deliverable:** TUI that displays stories from YAML with navigation

**Completed:** 2025-01-07

**Files Created:**

- `cmd/bmad/main.go` - Entry point
- `internal/app/app.go` - Main application model
- `internal/components/header/header.go` - Navigation header
- `internal/components/statusbar/statusbar.go` - Status bar
- `internal/config/config.go` - Configuration
- `internal/domain/story.go` - Story domain model
- `internal/domain/view.go` - View enum
- `internal/messages/messages.go` - Shared message types
- `internal/parser/yaml.go` - YAML parser
- `internal/theme/theme.go` - Catppuccin theme
- `internal/views/dashboard/dashboard.go` - Dashboard view
- `internal/views/storylist/storylist.go` - Story list view

### Phase 2: Execution Engine ✅ COMPLETE

**Goal:** Execute workflows with live output

- [x] Implement executor with Claude CLI command builder
- [x] Add subprocess management with context/timeout
- [x] Implement live stdout/stderr streaming via goroutines
- [x] Create Execution view with split-pane layout
- [x] Add step progress indicators
- [x] Implement step controls (pause, skip, retry, cancel)
- [x] Add pre-flight checks module
- [x] Implement retry logic

**Deliverable:** Execute single story through full workflow with live output

**Completed:** 2025-01-07

**Files Created:**

- `internal/domain/execution.go` - Execution and StepExecution domain models
- `internal/executor/executor.go` - Claude CLI command builder with subprocess management
- `internal/preflight/preflight.go` - Pre-flight checks (Claude CLI, paths, git)
- `internal/views/execution/execution.go` - Split-pane execution view with live output

**Features Implemented:**

- Live stdout/stderr streaming via goroutines
- Step progress with status indicators (pending, running, success, failed, skipped)
- Execution controls: pause (p), resume (r), skip step (k), cancel (c)
- Auto-skip create-story if story file already exists
- Retry logic with configurable attempts
- Timeout handling per step
- Pre-flight checks before execution
- Real-time duration display with tick updates
- Output scrolling with keyboard navigation

### Phase 3: Queue & Batch ✅ COMPLETE

**Goal:** Process multiple stories

- [x] Create Queue Manager view
- [x] Implement multi-select in Story List
- [x] Add queue reordering (move up/down)
- [x] Implement sequential batch processing
- [x] Add pause/resume queue functionality
- [x] Implement auto-skip intelligence
- [x] Create Timeline view with duration bars
- [x] Add progress estimation (ETA)

**Deliverable:** Select and process multiple stories in batch

**Completed:** 2025-01-07

**Files Created:**

- `internal/domain/queue.go` - Queue and QueueItem domain models with reordering
- `internal/executor/batch.go` - BatchExecutor for sequential multi-story execution
- `internal/views/queue/queue.go` - Queue Manager view with status, controls, progress
- `internal/views/timeline/timeline.go` - Timeline view with visual duration bars

**Files Modified:**

- `internal/messages/messages.go` - Added queue-related messages (QueueAddMsg, QueueItemStartedMsg, etc.)
- `internal/views/storylist/storylist.go` - Added 'Q' key to add selected stories to queue
- `internal/app/app.go` - Integrated queue view, timeline view, batch executor, and queue message handling

**Features Implemented:**

- Queue Manager with item display showing status, position, and progress
- Multi-select stories with Space, add to queue with Shift+Q
- Queue reordering with Shift+K (up) and Shift+J (down)
- Remove from queue with x/delete, clear pending with Shift+C
- Sequential batch execution through BatchExecutor
- Queue controls: Start (Enter), Pause (p), Resume (r), Cancel (c)
- Auto-skip create-story step for stories with existing files
- Timeline view with colored step bars showing relative durations
- ETA calculation based on historical step averages
- Real-time progress updates for both individual steps and overall queue

### Phase 4: Persistence & History ✅ COMPLETE

**Goal:** SQLite storage and analytics

- [x] Set up SQLite with modernc.org/sqlite
- [x] Create database schema and migrations
- [x] Implement execution history storage
- [x] Create History view with search/filter
- [x] Create Statistics view with charts
- [x] Create Diff Preview view
- [x] Add historical averages for ETA

**Deliverable:** Full history tracking and statistics

**Completed:** 2026-01-07

**Files Created:**

- `internal/storage/storage.go` - Storage interface and record types
- `internal/storage/sqlite.go` - SQLite implementation with CGO-free driver
- `internal/views/history/history.go` - History view with search/filter
- `internal/views/stats/stats.go` - Statistics view with ASCII charts
- `internal/views/diff/diff.go` - Diff preview with syntax highlighting
- `migrations/001_initial.sql` - Database schema

**Files Modified:**

- `internal/config/config.go` - Added DataDir and DatabasePath settings
- `internal/messages/messages.go` - Added History, Stats, and Diff messages
- `internal/app/app.go` - Integrated storage, new views, and message handling

**Features Implemented:**

- SQLite storage using modernc.org/sqlite (CGO-free)
- Execution persistence with step details and output
- History view with scrolling, search/filter by story key
- Statistics view with success rates, step performance, and charts
- Diff preview view with syntax highlighting
- Historical step averages for ETA calculation
- Automatic saving of executions when queue completes
- Data loading on view navigation

### Phase 5: Polish & UX ✅ COMPLETE

**Goal:** Enhanced user experience

- [x] Implement Command Palette (Ctrl+P)
- [x] Add Dracula and Nord themes
- [x] Add custom theme loading (YAML)
- [x] Implement Settings view
- [x] Add desktop notifications (macOS)
- [x] Add optional sound feedback
- [x] Add confetti animation on success
- [x] Add git status awareness to UI

**Deliverable:** Polished, themeable application

**Completed:** 2026-01-07

**Files Created:**

- `internal/theme/theme.go` - Extended with Dracula, Nord themes, and YAML loading
- `internal/views/settings/settings.go` - Settings view with theme, timeout, retry, notification, sound toggles
- `internal/components/commandpalette/palette.go` - Command palette with fuzzy search
- `internal/components/confetti/confetti.go` - Confetti animation overlay
- `internal/notify/notify.go` - Desktop notifications (macOS/Linux)
- `internal/sound/sound.go` - Sound feedback player (macOS/Linux)
- `internal/git/git.go` - Git status awareness

**Files Modified:**

- `internal/config/config.go` - Added CustomThemePath setting
- `internal/app/app.go` - Integrated all Phase 5 features
- `internal/views/*/` - Added RefreshStyles() method to all views

**Features Implemented:**

- Command Palette with Ctrl+P for quick navigation and theme switching
- Dracula and Nord themes in addition to Catppuccin
- Custom theme loading from YAML files
- Settings view with interactive controls (select, toggle, number)
- Desktop notifications on queue completion (macOS osascript, Linux notify-send)
- Sound feedback for success/error/completion events
- Confetti animation overlay on successful queue completion
- Real-time git status awareness in status bar (branch, clean/modified)
- Theme hot-switching with full style refresh across all views

### Phase 6: Advanced Features ✅ COMPLETE

**Goal:** Power user capabilities

- [x] Implement Profile system
- [x] Add custom workflow definitions
- [x] Implement Watch mode (fsnotify)
- [x] Add parallel execution with worker pool
- [x] Implement REST API server
- [x] Add WebSocket for live output via API
- [x] Set up goreleaser for releases
- [x] Add Homebrew formula

**Deliverable:** Feature-complete application ready for distribution

**Completed:** 2026-01-07

**Files Created:**

- `internal/profile/profile.go` - Profile system for multiple project configurations
- `internal/workflow/workflow.go` - Custom workflow definitions with template support
- `internal/watcher/watcher.go` - File system watcher with fsnotify
- `internal/executor/parallel.go` - Parallel execution with worker pool
- `internal/api/server.go` - REST API server with go-chi
- `internal/api/websocket.go` - WebSocket hub for live output streaming
- `.goreleaser.yaml` - GoReleaser configuration for releases
- `Formula/bmad.rb` - Homebrew formula template
- `Dockerfile` - Docker image for containerized deployment

**Files Modified:**

- `internal/config/config.go` - Added Phase 6 configuration options
- `internal/messages/messages.go` - Added Phase 6 message types
- `internal/app/app.go` - Integrated all Phase 6 features
- `go.mod` - Added fsnotify, go-chi, nhooyr.io/websocket dependencies
- `Makefile` - Added release, snapshot, docker, and version targets

**Features Implemented:**

- **Profile System**: Multiple project configurations with YAML persistence
  - Switch between profiles for different projects
  - Override settings per profile (paths, timeout, retries, theme)
  - Automatic profile loading and saving

- **Custom Workflows**: Define custom step sequences in YAML
  - Template-based prompt rendering with Go templates
  - Per-step timeout and retry overrides
  - Skip conditions (e.g., file_exists)
  - Default workflow with standard 4 steps

- **Watch Mode**: Auto-refresh on file changes
  - Monitor sprint-status.yaml for changes
  - Debounced refresh to prevent excessive reloads
  - Toggle via command palette

- **Parallel Execution**: Worker pool for independent stories
  - Configurable number of parallel workers (1-10)
  - Per-worker job queue with result collection
  - Progress tracking across all workers
  - Pause/resume/cancel across all workers

- **REST API Server**: Full HTTP API with go-chi
  - GET /api/stories - List stories with filtering
  - GET /api/queue - Get queue status
  - POST /api/execution/start - Start execution
  - POST /api/execution/pause - Pause execution
  - GET /api/history - List execution history
  - GET /api/stats - Get statistics
  - CORS support for frontend integration

- **WebSocket Support**: Real-time live output streaming
  - Execution updates in real-time
  - Step output streaming
  - Queue progress updates
  - Client-side command support

- **Release Infrastructure**:
  - GoReleaser for automated releases
  - Multi-platform builds (Linux, macOS, Windows)
  - Multi-arch builds (amd64, arm64)
  - Docker image support
  - Homebrew formula for easy installation
  - Makefile with release, snapshot, and docker targets

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
