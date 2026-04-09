# Terminal-First Task Management Application Architecture (Detailed)

## 1. System Overview

This system is a terminal-first task management application built in Go, designed with:
- LazyGit-like TUI interaction
- Cross-platform compatibility (Mac, Linux, Windows)
- Offline-first architecture
- Future-ready remote sync and collaboration

### Architectural Style
A hybrid of:
- MVC (Model-View-Controller)
- Hexagonal Architecture (Ports & Adapters)

This ensures:
- Clear separation of concerns
- Testability via interfaces
- Replaceable infrastructure (DB, sync, UI)

---

## 2. High-Level Architecture

```
Interface Layer (TUI / CLI)
        ↓
Application Layer (Use Cases / Controllers)
        ↓
Domain Layer (Business Logic)
        ↓
Infrastructure Layer (DB, Config, Logging, Search)
```

---

## 3. Package Structure (Go)

```
/cmd
  /taskcli
  /tasktui

/internal
  /domain
  /application
  /interface
    /tui
    /cli
  /infrastructure
    /db
    /repository
    /search
    /config
    /logger
    /sync
```

---

## 4. Domain Model

### Task Entity

```go
type Task struct {
    ID          string
    ListID      string
    Title       string
    Description string
    Status      Status
    Priority    Priority
    DueDate     *time.Time
    Tags        []string

    CustomFields map[string]interface{}

    ParentID   *string
    DependsOn  []string

    CreatedAt time.Time
    UpdatedAt time.Time
}
```

### TaskList

```go
type TaskList struct {
    ID        string
    Name      string
    Schema    map[string]FieldDefinition
    CreatedAt time.Time
}
```

### FieldDefinition

```go
type FieldDefinition struct {
    Name     string
    Type     string
    Required bool
}
```

---

## 5. Repository Interfaces

```go
type TaskRepository interface {
    Create(task *Task) error
    Update(task *Task) error
    Delete(taskID string) error
    GetByID(id string) (*Task, error)
    List(filter TaskFilter) ([]Task, error)
}
```

---

## 6. Application Layer

Handles business workflows.

### TaskService

Responsibilities:
- Validation
- Orchestration
- Delegation to repositories

```go
type TaskService struct {
    repo TaskRepository
    search SearchService
}
```

---

## 7. TUI Architecture

### Event Loop

```
Input → Update(State) → Render(View)
```

### Views
- List View
- Detail View
- Edit Modal
- Filter Panel

### State

```go
type AppState struct {
    ActiveView   ViewType
    SelectedTask *Task
    Tasks        []Task
    TaskLists    []TaskList
    Filter       TaskFilter
    Cursor       int
    Mode         ModeType
}
```

---

## 8. Persistence Layer

### Database: SQLite

### Schema

```sql
tasks (
  id TEXT PRIMARY KEY,
  list_id TEXT,
  title TEXT,
  description TEXT,
  status TEXT,
  priority INTEGER,
  due_date DATETIME,
  tags TEXT,
  custom_fields TEXT,
  parent_id TEXT,
  depends_on TEXT,
  created_at DATETIME,
  updated_at DATETIME
)
```

---

## 9. Search

Use SQLite FTS5 for full-text search.

---

## 10. Configuration

### JSON Config Example

```json
{
  "keybindings": {
    "up": "k",
    "down": "j"
  }
}
```

---

## 11. CLI Support

Optional command interface:

```
task add "Buy milk"
task list
```

---

## 12. Logging

- Structured logging
- Debug / Info / Error levels

---

## 13. Remote Sync (Future)

### Strategy
- Offline-first
- Last-write-wins

### Sync Flow

```
Local → Push → Pull → Resolve → Update
```

---

## 14. Extensibility

Exporter interface:

```go
type Exporter interface {
    Export(tasks []Task) ([]byte, error)
}
```

---

## 15. Performance

- Indexed queries
- Pagination
- Lazy rendering

---

## 16. Testing Strategy

- Domain: Unit tests
- Application: Mocked tests
- Infra: Integration tests

---

## 17. Recommended Libraries

- bubbletea
- lipgloss
- sqlite

---

## 18. Summary

- MVC + Hexagonal
- SQLite storage
- JSON config
- Offline-first design

---

## 19. Next Steps

- Implement core domain
- Build TUI
- Add persistence
- Add search
- Add sync (future)

