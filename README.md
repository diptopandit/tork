# Tork

<p align="center">
  <img src="assets/logo.png" alt="tork logo" width="200">
</p>

[![Go Version](https://img.shields.io/badge/go-1.22%2B-00ADD8)](https://go.dev/)
[![Build](https://img.shields.io/badge/build-passing-brightgreen)](#development)
[![Tests](https://img.shields.io/badge/tests-passing-brightgreen)](#development)
[![Platform](https://img.shields.io/badge/platform-macOS%20%7C%20Linux%20%7C%20Windows-blue)](#installation)

tork is a terminal-first task manager written in Go.

It provides:
- A CLI for fast, scriptable task workflows
- A Bubble Tea TUI for interactive task management with split-pane layout
- SQLite persistence with FTS5 full-text search
- JSON configuration, structured logging, and configurable keybindings

## Table of Contents

- [Why tork](#why-tork)
- [Features](#features)
- [Installation](#installation)
- [Quick Start](#quick-start)
- [CLI Reference](#cli-reference)
- [TUI Keybindings](#tui-keybindings)
- [Task Lists](#task-lists)
- [Data, Config, and Logs](#data-config-and-logs)
- [Roadmap](#roadmap)
- [Security](#security)
- [License](#license)

## Why tork

tork is built for engineers who want a local-first task manager that is:
- Fast in terminal environments
- Easy to automate via command-line workflows
- Friendly to extension and contributor-driven development

## Features

- **Split-pane TUI** — left pane: task list table, right pane: task detail + scrollable updates + inline input
- **Task lifecycle** — configurable statuses (default: `todo` → `in_progress` → `done` → `cancelled`)
- **Priority levels** — configurable priorities (default: low, medium, high, urgent)
- **Task metadata** — description, due date, tags, dependencies, numeric short IDs
- **Task updates** — immutable timestamped comments on any task (press `u` to add)
- **Task lists** — create, rename, delete, and switch between multiple task lists
- **Full-text search** — powered by SQLite FTS5
- **Configurable** — keybindings, date format, named themes, display columns, statuses, priorities
- **Themes** — 5 built-in themes (default, light, dracula, solarized-dark, nord) + custom theme files
- **Offline-first** — all data stored locally in SQLite

## Installation

### Option 1: Run from source (recommended right now)

```bash
git clone https://github.com/diptopandit/tork.git
cd tork
go run ./cmd/tork-cli --help
```

### Option 2: Install CLI binary with go install

```bash
go install github.com/diptopandit/tork/cmd/tork-cli@latest
```

### Option 3: Build local binaries

```bash
go build -o bin/tork-cli ./cmd/tork-cli
go build -o bin/tork ./cmd/tork
```

## Quick Start

### CLI

```bash
go run ./cmd/tork-cli add "Ship README" --priority high --due 15-04-2026 --tags docs,release
go run ./cmd/tork-cli list
go run ./cmd/tork-cli search "README"
```

### TUI

```bash
go run ./cmd/tork
```

## CLI Reference

### Task Commands

| Command | Description |
|---------|-------------|
| `add <title>` | Create a new task |
| `list` | List tasks (with optional filters) |
| `show <id>` | Show full task details and updates |
| `edit <id>` | Edit task fields (--title, --status, --priority, --due, --tags, --description) |
| `status <id> <status>` | Change task status (todo/in_progress/done/cancelled) |
| `done <id>` | Mark a task as done |
| `delete <id>` | Delete a task |
| `search <query>` | Full-text search tasks |
| `update <id> <message>` | Add an update/comment to a task |

### List Management Commands

| Command | Description |
|---------|-------------|
| `lists` | List all task lists |
| `list-create <name>` | Create a new task list |
| `list-rename <id> <name>` | Rename a task list |
| `list-delete <id> --force` | Delete a list and all its tasks |

### Examples

```bash
go run ./cmd/tork-cli add "Pay cloud bill" --priority urgent --due 30-04-2026 --list Ops
go run ./cmd/tork-cli list --status todo --priority high
go run ./cmd/tork-cli show 3
go run ./cmd/tork-cli edit 3 --title "New title" --priority high --status in_progress
go run ./cmd/tork-cli status 3 in_progress
go run ./cmd/tork-cli update 3 "Started working on this"
go run ./cmd/tork-cli done 3
go run ./cmd/tork-cli delete 3
go run ./cmd/tork-cli search "cloud"
go run ./cmd/tork-cli lists
go run ./cmd/tork-cli list-create "Work"
go run ./cmd/tork-cli list-rename abc123 "Personal"
go run ./cmd/tork-cli list-delete abc123 --force
```

Date format: DD-MM-YYYY (default) or YYYY-MM-DD (both accepted).

## TUI Keybindings

### Navigation

| Key | Action |
|-----|--------|
| `j` / `↓` | Move down |
| `k` / `↑` | Move up |
| `h` / `←` | Focus left pane |
| `l` / `→` | Focus right pane |
| `Tab` | Next status tab |
| `Shift+Tab` | Previous status tab |
| `Enter` | Select / view detail |

### Task Actions

| Key | Action |
|-----|--------|
| `n` | New task |
| `e` | Edit task |
| `s` | Cycle status (follows configured order) |
| `x` | Mark done |
| `d` | Delete task |
| `u` | Add update (Enter to submit, Esc to cancel) |

### General

| Key | Action |
|-----|--------|
| `L` | Open task list switcher (n: new, r: rename, d: delete) |
| `/` | Search / filter |
| `?` | Toggle keybinding help |
| `q` / `Ctrl+C` | Quit |
| `Esc` | Close overlay / go back |

## Task Lists

Press `L` to open the list switcher overlay:
- **Navigate**: `j`/`k` to move, `Enter` to select
- **Create**: `n` to create a new list (type name, press Enter)
- **Rename**: `r` to rename the highlighted list
- **Delete**: `d` to delete (requires double confirmation — all tasks in the list will be permanently deleted)

## Data, Config, and Logs

tork stores runtime files in `~/.tork`:
- **Database**: `~/.tork/tork.db` (configurable via `data_dir`)
- **Config**: `~/.tork/config.json`
- **Log file**: `~/.tork/tork.log`

Config example:

```json
{
  "data_dir": "",
  "keybindings": {
    "up": "k",
    "down": "j",
    "left": "h",
    "right": "l",
    "select": "enter",
    "quit": "q",
    "help": "?",
    "new": "n",
    "edit": "e",
    "delete": "d",
    "search": "/",
    "done": "x",
    "status": "s"
  },
  "display": {
    "columns": ["title", "status", "priority", "due_date"],
    "date_format": "02-01-2006",
    "show_timestamps": true,
    "tab_order": ["todo", "in_progress", "done", "all"],
    "default_tab": "todo"
  },
  "theme": "default",
  "statuses": [
    { "name": "todo", "label": "Todo" },
    { "name": "in_progress", "label": "In Progress" },
    { "name": "done", "label": "Done" },
    { "name": "cancelled", "label": "Cancelled" }
  ],
  "priorities": [
    { "name": "low", "value": 1, "label": "Low" },
    { "name": "medium", "value": 2, "label": "Medium" },
    { "name": "high", "value": 3, "label": "High" },
    { "name": "urgent", "value": 4, "label": "Urgent" }
  ],
  "default_list": "",
  "last_list": ""
}
```

### Date Format

The default date format is DD-MM-YYYY (`02-01-2006` in Go reference time). This is used throughout the TUI for display and input. Both DD-MM-YYYY and YYYY-MM-DD are accepted as input in the CLI and edit modal.

### Tab Order

`tab_order` controls which status tabs appear in the TUI header and their order. Valid values: `all` plus any names from the `statuses` array. `default_tab` sets which tab is selected on startup.

### Custom Statuses

Define your own statuses in the `statuses` array. Each entry has a `name` (stored in DB) and a `label` (displayed in TUI). The first status is used as the default for new tasks. The `s` key cycles through statuses in configured order. Tab order references these names.

### Custom Priorities

Define your own priorities in the `priorities` array. Each entry has a `name` (used in CLI), a numeric `value` (stored in DB, used for sorting), and a `label` (displayed in TUI). In the edit modal, enter the 1-indexed position (e.g., `1` for the first priority).

### Themes

tork supports named themes. Set the `"theme"` field in config.json to one of the built-in theme names or the name of a custom theme file.

**Built-in themes:** `default`, `light`, `dracula`, `solarized-dark`, `nord`

```json
{
  "theme": "dracula"
}
```

#### Custom Themes

Create a JSON file in `~/.tork/themes/` with the theme name as the filename:

```
~/.tork/themes/my-theme.json
```

Then set `"theme": "my-theme"` in config.json.

**Theme file format:**

```json
{
  "name": "my-theme",
  "description": "My custom tork theme",
  "author": "Your Name",
  "colors": {
    "primary": "#7C3AED",
    "secondary": "#6B7280",
    "active": "#7C3AED",
    "inactive": "#374151",
    "success": "#10B981",
    "warning": "#F59E0B",
    "danger": "#EF4444",
    "text": "#E5E7EB",
    "text_muted": "#9CA3AF",
    "text_bright": "#FFFFFF",
    "accent": "#60A5FA"
  },
  "border": "rounded",
  "status_colors": ["#F59E0B", "#60A5FA", "#10B981", "#EF4444"],
  "priority_colors": ["#6B7280", "#F59E0B", "#FB923C", "#EF4444"]
}
```

| Field | Description |
|-------|-------------|
| `colors` | 11 semantic color slots (all required) |
| `border` | Border style: `rounded`, `normal`, `double`, or `hidden` |
| `status_colors` | Positional array — index 0 is the first status in your `statuses` config, index 1 is the second, etc. |
| `priority_colors` | Positional array — index 0 is the first priority in your `priorities` config, etc. |

All fields are required. Theme files are validated on load; missing fields produce a clear error message.

## Roadmap

- Recurring tasks
- Better query language for advanced filtering
- Export command wiring for Markdown/CSV
- Optional sync adapter
- Packaged release binaries

## Security

If you discover a security issue:
- Do not open a public issue with exploit details
- Privately contact project maintainers first
- Include reproduction steps and affected versions

## License

Licensed under the [Apache License, Version 2.0](LICENSE).
