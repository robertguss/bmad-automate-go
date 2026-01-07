# BMAD Automate

[![Go Version](https://img.shields.io/badge/Go-1.24+-00ADD8?style=flat&logo=go)](https://go.dev/)
[![License](https://img.shields.io/badge/License-MIT-blue.svg)](LICENSE)

A beautiful terminal UI application for automating BMAD (Business Method for Agile Development) workflows. Built with Go and the [Charm](https://charm.sh/) ecosystem, BMAD Automate orchestrates Claude AI CLI commands to automate story development through an interactive terminal interface.

## Features

- **Dashboard** - Overview of stories by status, recent activity, quick stats
- **Story List** - Browse and filter stories by epic and status with multi-select
- **Queue Manager** - Batch process multiple stories with reordering and ETA
- **Live Execution** - Watch Claude work in real-time with streaming output
- **Timeline View** - Visual step duration bars for performance analysis
- **History & Stats** - Track execution history with SQLite persistence
- **REST API** - Control BMAD via HTTP endpoints with WebSocket support
- **Profiles** - Multiple project configurations for different environments
- **Custom Workflows** - Define your own step sequences with templates
- **Theming** - Built-in themes (Catppuccin, Dracula, Nord) plus custom themes

## Installation

### From Source

```bash
git clone https://github.com/robertguss/bmad-automate-go.git
cd bmad-automate-go
make build
```

The binary will be in `./bin/bmad`.

### Using Go Install

```bash
go install github.com/robertguss/bmad-automate-go/cmd/bmad@latest
```

### Docker

```bash
docker build -t bmad:latest .
docker run -it -v $(pwd):/project bmad:latest
```

### Requirements

- Go 1.24+
- Claude CLI installed and configured
- A project with `sprint-status.yaml`

## Quick Start

1. **Navigate to your project directory** (where `sprint-status.yaml` is located):

```bash
cd /path/to/your/project
```

2. **Run BMAD Automate**:

```bash
bmad
```

3. **Select stories** from the Story List view and add them to the queue

4. **Start execution** to watch Claude work through each story

## Workflow Steps

BMAD Automate executes stories through a 4-step workflow:

| Step           | Description                                                |
| -------------- | ---------------------------------------------------------- |
| `create-story` | Generate story file from template (auto-skipped if exists) |
| `dev-story`    | Implement the story with Claude CLI                        |
| `code-review`  | Review and auto-fix issues                                 |
| `git-commit`   | Commit and push changes                                    |

## Keyboard Navigation

### Global Keys

| Key      | Action          |
| -------- | --------------- |
| `d`      | Dashboard       |
| `s`      | Story List      |
| `q`      | Queue Manager   |
| `e`      | Execution View  |
| `t`      | Timeline        |
| `h`      | History         |
| `a`      | Statistics      |
| `g`      | Git Diff        |
| `o`      | Settings        |
| `Ctrl+P` | Command Palette |
| `Esc`    | Go back         |
| `Ctrl+C` | Quit            |

### Story List Keys

| Key                | Action                |
| ------------------ | --------------------- |
| `Up/Down` or `j/k` | Navigate              |
| `Space`            | Select/deselect story |
| `Enter`            | Execute single story  |
| `a`                | Select all            |
| `n`                | Deselect all          |
| `e`                | Cycle epic filter     |
| `f`                | Cycle status filter   |
| `q`                | Add selected to queue |

### Queue Manager Keys

| Key             | Action            |
| --------------- | ----------------- |
| `Up/Down`       | Navigate          |
| `Shift+Up/Down` | Reorder items     |
| `Delete`        | Remove from queue |
| `c`             | Clear queue       |
| `Enter`         | Start execution   |

### Execution View Keys

| Key | Action            |
| --- | ----------------- |
| `p` | Pause/Resume      |
| `s` | Skip current step |
| `c` | Cancel execution  |

## Configuration

BMAD Automate stores configuration and data in `.bmad/` within your project directory.

### Default Paths

| Setting         | Default                                                    |
| --------------- | ---------------------------------------------------------- |
| Sprint Status   | `_bmad-output/implementation-artifacts/sprint-status.yaml` |
| Story Directory | `_bmad-output/implementation-artifacts`                    |
| Database        | `.bmad/bmad.db`                                            |
| Profiles        | `.bmad/profiles/`                                          |
| Workflows       | `.bmad/workflows/`                                         |

### Configuration Options

```yaml
# Example profile configuration (.bmad/profiles/myproject.yaml)
name: myproject
description: My Project Configuration
sprint_status_path: ./sprint-status.yaml
story_dir: ./stories
timeout: 900 # 15 minutes
retries: 2
theme: dracula
workflow: quick-dev
max_workers: 2
```

See [docs/configuration.md](docs/configuration.md) for complete configuration reference.

## REST API

Enable the REST API to control BMAD from external tools:

```bash
bmad --api --port 8080
```

Or use make:

```bash
make run-api
```

### API Endpoints

| Method | Endpoint               | Description          |
| ------ | ---------------------- | -------------------- |
| `GET`  | `/health`              | Health check         |
| `GET`  | `/api/stories`         | List all stories     |
| `GET`  | `/api/queue`           | Get queue status     |
| `POST` | `/api/queue/add`       | Add stories to queue |
| `POST` | `/api/execution/start` | Start execution      |
| `GET`  | `/api/stats`           | Get statistics       |
| `GET`  | `/api/ws`              | WebSocket endpoint   |

See [docs/api.md](docs/api.md) for complete API documentation.

## Themes

BMAD Automate includes three built-in themes:

- **Catppuccin Mocha** (default) - Soothing pastel theme
- **Dracula** - Dark theme with vibrant colors
- **Nord** - Arctic, bluish theme

You can also create custom themes via YAML configuration. See [docs/configuration.md](docs/configuration.md#custom-themes) for details.

## Architecture

BMAD Automate uses The Elm Architecture (TEA) via [Bubble Tea](https://github.com/charmbracelet/bubbletea):

```
┌─────────────────────────────────────────────────────────┐
│                    Application                          │
├─────────────┬─────────────┬─────────────┬──────────────┤
│  Dashboard  │  StoryList  │   Queue     │  Execution   │
│    View     │    View     │   View      │    View      │
├─────────────┴─────────────┴─────────────┴──────────────┤
│                    Domain Models                        │
│         (Story, Execution, Queue, StepExecution)       │
├─────────────────────────────────────────────────────────┤
│   Executor  │   Storage   │    API     │   Watcher    │
│   Engine    │   (SQLite)  │   Server   │   (fsnotify) │
└─────────────────────────────────────────────────────────┘
```

See [docs/architecture.md](docs/architecture.md) for detailed architecture documentation.

## Development

```bash
# Run the app
make run

# Build
make build

# Format code
make fmt

# Run linter
make lint

# Run tests
make test

# Run tests with coverage
make test-coverage

# Run with live reload (requires air)
make dev
```

### Project Structure

```
bmad-automate-go/
├── cmd/bmad/              # Entry point
├── internal/
│   ├── api/               # REST API server
│   ├── app/               # Main application model
│   ├── components/        # Reusable UI components
│   ├── config/            # Configuration
│   ├── domain/            # Domain models
│   ├── executor/          # Execution engine
│   ├── git/               # Git integration
│   ├── messages/          # Message types
│   ├── notify/            # Desktop notifications
│   ├── parser/            # YAML parsing
│   ├── preflight/         # Pre-flight checks
│   ├── profile/           # Profile management
│   ├── sound/             # Sound feedback
│   ├── storage/           # SQLite persistence
│   ├── theme/             # Color themes
│   ├── views/             # View models
│   ├── watcher/           # File watching
│   └── workflow/          # Custom workflows
├── docs/                  # Documentation
├── Makefile
└── go.mod
```

## Documentation

- [Architecture Overview](docs/architecture.md)
- [API Reference](docs/api.md)
- [Configuration Guide](docs/configuration.md)
- [Workflow Customization](docs/workflows.md)
- [Contributing Guide](docs/contributing.md)

## License

MIT
