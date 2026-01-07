# Configuration Guide

This guide covers all configuration options for BMAD Automate.

## Overview

BMAD Automate stores configuration and data in the `.bmad/` directory within your project:

```
.bmad/
├── bmad.db           # SQLite database
├── profiles/         # Profile configurations
│   ├── default.yaml
│   └── production.yaml
├── workflows/        # Custom workflow definitions
│   └── quick-dev.yaml
└── themes/           # Custom theme files
    └── my-theme.yaml
```

## Configuration Defaults

| Setting            | Default Value                                              | Description                |
| ------------------ | ---------------------------------------------------------- | -------------------------- |
| `SprintStatusPath` | `_bmad-output/implementation-artifacts/sprint-status.yaml` | Path to sprint status file |
| `StoryDir`         | `_bmad-output/implementation-artifacts`                    | Directory for story files  |
| `Timeout`          | `600` (10 minutes)                                         | Execution timeout per step |
| `Retries`          | `1`                                                        | Number of retry attempts   |
| `Theme`            | `catppuccin`                                               | Active theme               |
| `APIPort`          | `8080`                                                     | REST API server port       |
| `MaxWorkers`       | `1`                                                        | Parallel execution workers |
| `WatchDebounce`    | `500`                                                      | File watch debounce (ms)   |

## Profiles

Profiles allow you to save and switch between different project configurations.

### Creating a Profile

Create a YAML file in `.bmad/profiles/`:

```yaml
# .bmad/profiles/myproject.yaml
name: myproject
description: My Project Configuration

# Paths
sprint_status_path: ./sprint-status.yaml
story_dir: ./stories
working_dir: /path/to/project

# Execution
timeout: 900 # 15 minutes
retries: 2 # 2 retry attempts

# Appearance
theme: dracula

# Workflow
workflow: quick-dev

# Parallel execution
max_workers: 2
```

### Profile Options

| Field                | Type    | Description                      |
| -------------------- | ------- | -------------------------------- |
| `name`               | string  | Profile identifier               |
| `description`        | string  | Human-readable description       |
| `sprint_status_path` | string  | Path to sprint-status.yaml       |
| `story_dir`          | string  | Directory containing story files |
| `working_dir`        | string  | Working directory for commands   |
| `timeout`            | integer | Step timeout in seconds          |
| `retries`            | integer | Number of retry attempts         |
| `theme`              | string  | Theme name or custom theme path  |
| `workflow`           | string  | Name of workflow to use          |
| `max_workers`        | integer | Number of parallel workers       |

### Switching Profiles

Use the Settings view (`o`) to switch between profiles, or use the API:

```bash
# Via API (when API server is running)
curl -X POST "http://localhost:8080/api/profiles/switch/myproject"
```

## Themes

### Built-in Themes

BMAD Automate includes three themes:

- **catppuccin** (default) - Soothing pastel theme
- **dracula** - Dark theme with vibrant colors
- **nord** - Arctic, bluish theme

### Setting a Theme

In your profile configuration:

```yaml
theme: dracula
```

Or via the Settings view in the TUI.

### Custom Themes

Create a custom theme by defining colors in YAML:

```yaml
# .bmad/themes/my-theme.yaml
name: My Custom Theme

# Base colors
background: "#1a1b26"
foreground: "#c0caf5"
subtle: "#565f89"
highlight: "#bb9af7"

# Status colors
success: "#9ece6a"
warning: "#e0af68"
error: "#f7768e"
info: "#7aa2f7"

# Accent colors
primary: "#7aa2f7"
secondary: "#bb9af7"
accent: "#73daca"

# UI element colors
border: "#3b4261"
selection: "#33467c"
active_tab: "#7aa2f7"
inactive_tab: "#565f89"
status_bar: "#16161e"
header_bg: "#16161e"
```

Then reference it in your profile:

```yaml
theme: my-theme
custom_theme_path: .bmad/themes/my-theme.yaml
```

### Color Reference

| Color          | Usage                       |
| -------------- | --------------------------- |
| `background`   | Main background color       |
| `foreground`   | Primary text color          |
| `subtle`       | Muted/secondary text        |
| `highlight`    | Emphasized text             |
| `success`      | Success states (green)      |
| `warning`      | Warning states (yellow)     |
| `error`        | Error states (red)          |
| `info`         | Informational states (blue) |
| `primary`      | Primary accent color        |
| `secondary`    | Secondary accent color      |
| `accent`       | Tertiary accent color       |
| `border`       | Box borders                 |
| `selection`    | Selected item background    |
| `active_tab`   | Active navigation tab       |
| `inactive_tab` | Inactive navigation tab     |
| `status_bar`   | Status bar background       |
| `header_bg`    | Header background           |

## Feature Flags

### Sound

Enable audio feedback for execution events:

```yaml
sound_enabled: true
```

Plays a sound on:

- Execution completion
- Execution failure

### Notifications

Enable desktop notifications (macOS):

```yaml
notifications_enabled: true
```

Sends notifications for:

- Execution started
- Execution completed
- Execution failed

### Watch Mode

Enable automatic refresh when `sprint-status.yaml` changes:

```yaml
watch_enabled: true
watch_debounce: 500 # milliseconds
```

### API Server

Enable the REST API server:

```yaml
api_enabled: true
api_port: 8080
```

### Parallel Execution

Enable parallel story execution:

```yaml
parallel_enabled: true
max_workers: 2 # Number of concurrent executions
```

## Environment Variables

BMAD Automate respects these environment variables:

| Variable             | Description                                |
| -------------------- | ------------------------------------------ |
| `BMAD_SPRINT_STATUS` | Override sprint status path                |
| `BMAD_STORY_DIR`     | Override story directory                   |
| `BMAD_TIMEOUT`       | Override default timeout                   |
| `BMAD_THEME`         | Override theme                             |
| `BMAD_DATA_DIR`      | Override data directory (default: `.bmad`) |

Example:

```bash
BMAD_THEME=dracula BMAD_TIMEOUT=900 bmad
```

## Command Line Options

```bash
bmad [options]

Options:
  --api              Enable REST API server
  --port PORT        API server port (default: 8080)
  --watch            Enable file watch mode
  --theme THEME      Set theme (catppuccin, dracula, nord)
  --profile PROFILE  Use specific profile
  --help             Show help message
```

Examples:

```bash
# Run with API server
bmad --api --port 8080

# Run with watch mode
bmad --watch

# Run with specific profile and theme
bmad --profile production --theme nord
```

## Sprint Status File Format

BMAD Automate reads stories from `sprint-status.yaml`:

```yaml
# sprint-status.yaml

# Stories are defined as key-value pairs
# Key format: {epic}-{number}-{slug}
# Status values: in-progress, ready-for-dev, backlog, done, blocked

3-1-user-auth: ready-for-dev
3-2-password-reset: ready-for-dev
3-3-oauth-integration: backlog
4-1-dashboard: in-progress
4-2-analytics: blocked
5-1-refactor-api: done
```

### Story Key Format

Stories use the format: `{epic}-{number}-{slug}`

- **epic**: Epic number (integer)
- **number**: Story number within epic (integer)
- **slug**: URL-friendly story identifier

Example: `3-1-user-auth` = Epic 3, Story 1, "User Auth"

### Story Statuses

| Status          | Description               | Can Execute |
| --------------- | ------------------------- | ----------- |
| `in-progress`   | Currently being worked on | Yes         |
| `ready-for-dev` | Ready to be picked up     | Yes         |
| `backlog`       | Not yet started           | Yes         |
| `done`          | Completed                 | No          |
| `blocked`       | Blocked by dependencies   | No          |

## Timeouts and Retries

### Step Timeouts

Configure timeout per step in workflow definitions:

```yaml
# .bmad/workflows/custom.yaml
steps:
  - name: dev-story
    timeout: 1800 # 30 minutes for development
  - name: code-review
    timeout: 600 # 10 minutes for review
```

### Retry Configuration

```yaml
retries: 2 # Global retry count
```

When a step fails:

1. BMAD waits 2 seconds
2. Retries the step
3. If all retries fail, marks execution as failed

### Per-Step Retry Override

```yaml
# .bmad/workflows/custom.yaml
steps:
  - name: dev-story
    retries: 3 # More retries for complex steps
  - name: git-commit
    retries: 1 # Fewer retries for simple steps
```

## Database Configuration

The SQLite database is stored at `.bmad/bmad.db`.

### Database Location

```yaml
database_path: /custom/path/bmad.db # Override database location
```

### Database Migration

The database schema is managed automatically. On startup, BMAD:

1. Creates the database if it doesn't exist
2. Runs any pending migrations
3. Creates required indexes

### Backup

To backup execution history:

```bash
cp .bmad/bmad.db .bmad/bmad.db.backup
```

## Troubleshooting

### Configuration Issues

**Problem**: BMAD can't find sprint-status.yaml

```
Error: sprint-status.yaml not found
```

**Solution**: Ensure you're running BMAD from the correct directory or set the path in your profile:

```yaml
sprint_status_path: /full/path/to/sprint-status.yaml
```

### Theme Issues

**Problem**: Custom theme colors don't apply

**Solution**: Ensure all required color fields are defined in your theme YAML. Missing colors will fall back to the default theme.

### Permission Issues

**Problem**: Can't create .bmad directory

**Solution**: Ensure you have write permissions in the project directory:

```bash
chmod 755 /path/to/project
```

### Database Issues

**Problem**: Database locked error

**Solution**: Ensure only one instance of BMAD is running. If the issue persists:

```bash
# Remove the lock (only if BMAD is not running)
rm .bmad/bmad.db-journal
```

## Complete Configuration Example

```yaml
# .bmad/profiles/production.yaml
name: production
description: Production environment configuration

# Paths
sprint_status_path: ./sprint-status.yaml
story_dir: ./stories
working_dir: /home/user/project

# Execution
timeout: 1200 # 20 minutes
retries: 3 # 3 retry attempts

# Appearance
theme: catppuccin

# Workflow
workflow: default

# Features
sound_enabled: true
notifications_enabled: true
watch_enabled: true
watch_debounce: 1000

# API
api_enabled: true
api_port: 9000

# Parallel
parallel_enabled: false
max_workers: 1
```
