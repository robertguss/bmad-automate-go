# BMAD Automate (Go TUI)

A beautiful terminal UI application for automating BMAD (Business Method for Agile Development) workflows. Built with Go and the [Charm](https://charm.sh/) ecosystem.

## Features

- **Dashboard** - Overview of stories by status, recent activity, quick stats
- **Story List** - Browse and filter stories by epic and status
- **Queue Manager** - Select and batch process multiple stories (coming soon)
- **Live Execution** - Watch Claude work in real-time with streaming output (coming soon)
- **History & Stats** - Track execution history with SQLite (coming soon)
- **Beautiful UI** - Catppuccin theme with smooth navigation

## Installation

### From Source

```bash
git clone https://github.com/robertguss/bmad-automate-go.git
cd bmad-automate-go
make build
```

The binary will be in `./bin/bmad`.

### Requirements

- Go 1.21+
- Claude CLI installed and configured
- A project with `sprint-status.yaml`

## Usage

Run from your project directory (where `sprint-status.yaml` is located):

```bash
bmad
```

### Keyboard Navigation

| Key      | Action        |
| -------- | ------------- |
| `d`      | Dashboard     |
| `s`      | Story List    |
| `q`      | Queue Manager |
| `h`      | History       |
| `a`      | Statistics    |
| `o`      | Settings      |
| `Esc`    | Go back       |
| `Ctrl+C` | Quit          |

### Story List Keys

| Key       | Action                |
| --------- | --------------------- |
| `Up/Down` | Navigate              |
| `Space`   | Select/deselect story |
| `a`       | Select all            |
| `n`       | Deselect all          |
| `e`       | Cycle epic filter     |
| `f`       | Cycle status filter   |

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
```

## Project Structure

```
bmad-automate-go/
├── cmd/bmad/           # Entry point
├── internal/
│   ├── app/            # Main application model
│   ├── components/     # Reusable UI components
│   ├── config/         # Configuration
│   ├── domain/         # Domain models
│   ├── messages/       # Message types
│   ├── parser/         # YAML parsing
│   ├── theme/          # Color themes
│   └── views/          # View models
├── Makefile
└── PLAN.md             # Implementation roadmap
```

## Roadmap

See [PLAN.md](PLAN.md) for the full implementation plan.

- [x] **Phase 1**: Foundation - TUI shell, navigation, story display
- [ ] **Phase 2**: Execution Engine - Live output, step controls
- [ ] **Phase 3**: Queue & Batch - Multi-story processing
- [ ] **Phase 4**: Persistence - SQLite history, statistics
- [ ] **Phase 5**: Polish - Command palette, themes, notifications
- [ ] **Phase 6**: Advanced - Profiles, watch mode, REST API

## License

MIT
