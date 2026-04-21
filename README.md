# Tork

<p align="center">
  <img src="assets/logo.png" alt="tork logo" width="200">
</p>

[![Go Version](https://img.shields.io/badge/go-1.22%2B-00ADD8)](https://go.dev/)
[![Build](https://img.shields.io/badge/build-passing-brightgreen)](#development)
[![Tests](https://img.shields.io/badge/tests-passing-brightgreen)](#development)
[![Platform](https://img.shields.io/badge/platform-macOS%20%7C%20Linux%20%7C%20Windows-blue)](#installation)
[![Release](https://img.shields.io/github/v/release/diptopandit/tork?include_prereleases)](https://github.com/diptopandit/tork/releases)
[![Coverage](https://img.shields.io/badge/coverage-43%25-yellow)](#development)

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
- [Logging and Debugging](#logging-and-debugging)
- [Remote Database (MySQL)](#remote-database-mysql)
- [Development](#development)
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
- **Remote MySQL backend** — optional multi-user mode with shared/private task lists

## Installation

### Option 1: Install script (macOS / Linux)

```bash
curl -sSfL https://raw.githubusercontent.com/diptopandit/tork/main/install.sh | sh
```

This downloads the latest release and installs `tork` and `tork-cli` to `/usr/local/bin`. To install elsewhere:

```bash
INSTALL_DIR=~/.local/bin curl -sSfL https://raw.githubusercontent.com/diptopandit/tork/main/install.sh | sh
```

### Option 2: Download release binary

Download the archive for your platform from [Releases](https://github.com/diptopandit/tork/releases), extract it, and move the binaries to a directory on your `PATH`:

| Platform | Archive |
|----------|---------|
| macOS (Apple Silicon) | `tork_vX.Y.Z_darwin_arm64.tar.gz` |
| macOS (Intel) | `tork_vX.Y.Z_darwin_amd64.tar.gz` |
| Linux (x86_64) | `tork_vX.Y.Z_linux_amd64.tar.gz` |
| Linux (ARM64) | `tork_vX.Y.Z_linux_arm64.tar.gz` |
| Windows (x86_64) | `tork_vX.Y.Z_windows_amd64.zip` |
| Windows (ARM64) | `tork_vX.Y.Z_windows_arm64.zip` |

### Option 3: Install with `go install`

Requires Go 1.22+:

```bash
go install github.com/diptopandit/tork/cmd/tork@latest
go install github.com/diptopandit/tork/cmd/tork-cli@latest
```

### Option 4: Build from source

```bash
git clone https://github.com/diptopandit/tork.git
cd tork
make build
# Binaries: build/tork and build/tork-cli
```

### Verify installation

```bash
tork --version
tork-cli --version
```

## Quick Start

### CLI

```bash
tork-cli add "Ship README" --priority high --due 15-04-2026 --tags docs,release
tork-cli list
tork-cli search "README"
```

### TUI

```bash
tork
```

## CLI Reference

### Task Commands

| Command | Description |
|---------|-------------|
| `tork-cli add <title>` | Create a new task |
| `tork-cli list` | List tasks (with optional filters) |
| `tork-cli show <id>` | Show full task details and updates |
| `tork-cli edit <id>` | Edit task fields (--title, --status, --priority, --due, --tags, --description) |
| `tork-cli status <id> <status>` | Change task status (todo/in_progress/done/cancelled) |
| `tork-cli done <id>` | Mark a task as done |
| `tork-cli delete <id>` | Delete a task |
| `tork-cli search <query>` | Full-text search tasks |
| `tork-cli update <id> <message>` | Add an update/comment to a task |

### List Management Commands

| Command | Description |
|---------|-------------|
| `tork-cli lists` | List all task lists |
| `tork-cli list-create <name>` | Create a new task list |
| `tork-cli list-rename <id> <name>` | Rename a task list |
| `tork-cli list-delete <id> --force` | Delete a list and all its tasks |

### Examples

```bash
tork-cli add "Pay cloud bill" --priority urgent --due 30-04-2026 --list Ops
tork-cli list --status todo --priority high
tork-cli show 3
tork-cli edit 3 --title "New title" --priority high --status in_progress
tork-cli status 3 in_progress
tork-cli update 3 "Started working on this"
tork-cli done 3
tork-cli delete 3
tork-cli search "cloud"
tork-cli lists
tork-cli list-create "Work"
tork-cli list-rename abc123 "Personal"
tork-cli list-delete abc123 --force
```

Date format: DD-MM-YYYY (default) or YYYY-MM-DD (both accepted).

## TUI Keybindings

### Navigation

| Key | Action |
|-----|--------|
| `j` / `↓` | Move down (scroll focused section in detail pane) |
| `k` / `↑` | Move up (scroll focused section in detail pane) |
| `h` / `←` | Focus left pane |
| `l` / `→` | Focus right pane |
| `Tab` | Next status tab (left pane) / cycle detail↔updates focus (right pane) |
| `Shift+Tab` | Previous status tab (left pane) / cycle updates↔detail focus (right pane) |
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
| `T` | Open theme picker (live preview, Enter to apply, Esc to revert) |
| `S` | Sort tasks (by priority, due date, or ID) |
| `R` | Switch remote database (if remotes configured) |
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
  "log_level": "info",
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
    "status": "s",
    "theme_picker": "T",
    "list_switch": "L",
    "sort": "S"
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
  "last_list": "",
  "remotes": {
    "work": {
      "driver": "mysql",
      "host": "work-server.example.com",
      "port": 3306,
      "database": "tork",
      "username": "alice"
    },
    "personal": {
      "driver": "mysql",
      "host": "personal-server.example.com",
      "username": "alice"
    }
  },
  "last_remote": "work"
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

tork supports named themes. Set the `"theme"` field in config.json to one of the built-in theme names or the name of a custom theme file. You can also press `T` in the TUI to open the **theme picker** — browse all themes with live preview, press Enter to apply, or Esc to revert.

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

## Logging and Debugging

tork writes structured JSON logs to `~/.tork/tork.log`. Both the TUI (`tork`) and the CLI (`tork-cli`) write to the same log file.

### Log Levels

Set `"log_level"` in `~/.tork/config.json`:

```json
{
  "log_level": "debug"
}
```

| Level | Description |
|-------|-------------|
| `debug` | Verbose output — includes every task update, DB query details |
| `info` | Default — task create/delete, DB connections, migrations |
| `warn` | Warnings only |
| `error` | Errors only |

### Viewing Logs

```bash
# Follow logs in real time
tail -f ~/.tork/tork.log

# Pretty-print with jq
tail -f ~/.tork/tork.log | jq .

# Filter for errors
grep '"level":"error"' ~/.tork/tork.log | jq .
```

### What Gets Logged

- Database open/close and migration events
- Remote MySQL connection attempts (host, port, database, user — never passwords)
- Task creation, updates, and deletion (with IDs)
- List creation, rename, and deletion
- Errors with full context

### Disabling Logs

Set `"log_level": "error"` to minimize output. The log file is always created but will remain small at higher log levels.

## Remote Database (MySQL)

By default, tork stores everything locally in SQLite. For multi-user or team use, you can configure one or more remote MySQL backends. When a remote is selected, tork connects to MySQL instead of SQLite — there is no sync; the remote server **is** the data source.

### Setup

1. Create a MySQL database:

```sql
CREATE DATABASE tork CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;
CREATE USER 'torkuser'@'%' IDENTIFIED BY 'yourpassword';
GRANT ALL PRIVILEGES ON tork.* TO 'torkuser'@'%';
FLUSH PRIVILEGES;
```

2. Add to `~/.tork/config.json`:

```json
{
  "remotes": {
    "work": {
      "driver": "mysql",
      "host": "db.example.com",
      "port": 3306,
      "database": "tork",
      "username": "torkuser"
    }
  }
}
```

Password is prompted on each connection — it is never stored in config. The `username` field is both the MySQL login user and the unique identity for multi-user scoping. Default port is 3306, default database is `tork`.

### Selecting a Remote

| Method | Example |
|---|---|
| CLI flag | `tork-cli --remote work list` or `tork-cli --local list` |
| TUI flag | `tork --remote work` or `tork --local` |
| Last used | tork remembers your last selection (`last_remote` in config) |
| Auto-connect | If `last_remote` is a remote, the TUI prompts for a password on startup and connects |
| On-the-fly | Press `R` in the TUI to switch between local and remote databases |

The TUI always starts with local SQLite so the interface is immediately available. If a remote was last used, a password overlay appears; on failure it falls back to local.

The priority is: **CLI flag > last_remote > local (default)**.

### Multi-User Access

Multiple tork instances can connect to the same MySQL database simultaneously. Each user sees:

- **Their own private lists** (default)
- **Shared lists** they've been added to as a member

### List Visibility

| Visibility | Who can see | Who can edit |
|---|---|---|
| `private` | Owner only | Owner only |
| `shared` | Owner + members | Owner + members with `editor` or `admin` role |

### Member Roles

| Role | Permissions |
|---|---|
| `viewer` | Read-only access to list and its tasks |
| `editor` | Create, edit, delete tasks and updates |
| `admin` | Editor + rename/delete list, manage members |

### Schema Differences

The MySQL schema includes additional tables (`users`, `list_members`) and columns (`owner_id`, `visibility` on `task_lists`) for multi-user support. Full-text search uses MySQL `FULLTEXT` indexes instead of SQLite FTS5. The SQLite schema is unchanged and fully backward compatible.

## Development

### Running Tests

```bash
# Unit tests
make test

# Tests with coverage report
make test-coverage

# HTML coverage report (opens in browser)
make test-coverage-html

# Integration tests (requires Docker — spins up MySQL)
make test-integration
```

### Coverage

Coverage runs on every push and pull request via GitHub Actions. The workflow uploads a coverage report as a build artifact.

| Command | Description |
|---------|-------------|
| `make test` | Run all unit tests |
| `make test-coverage` | Run tests and print per-function coverage |
| `make test-coverage-html` | Generate `coverage.html` report |
| `make test-integration` | Run full suite including MySQL integration tests |

### Building

```bash
make build              # build/tork and build/tork-cli
make release-all        # cross-compile for all platforms
```

## Roadmap

- Recurring tasks
- Better query language for advanced filtering
- Export command wiring for Markdown/CSV
- Packaged release binaries

## Security

If you discover a security issue:
- Do not open a public issue with exploit details
- Privately contact project maintainers first
- Include reproduction steps and affected versions

## License

Licensed under the [Apache License, Version 2.0](LICENSE).
