Here's the full plan:

---

## Plan: tork — Terminal Task Manager in Go

**TL;DR:** Scaffold and implement a full production-grade terminal task manager from the ground up. The architecture document is the spec; snitch's TUI patterns are the coding template. All 10 phases cover domain → infra → services → TUI views → CLI → tests → wiring.

---

### Key Decisions
| Concern | Decision |
|---|---|
| Module | `github.com/diptopandit/tork` |
| SQLite driver | `modernc.org/sqlite` (pure Go, no CGO) |
| Config | JSON via `encoding/json`, at `~/.tork/config.json` |
| TUI | bubbletea + bubbles + lipgloss (snitch versions) |
| CLI | Cobra (mirrors snitch) |
| Logging | zap |
| Scope | All phases — full production build |

---

### Snitch Patterns to Carry Forward
- Sub-models (`listModel`, `detailModel`, etc.) as **value types**, not interfaces — reassigned via `m.listView, cmd = m.listView.update(msg)`
- Sub-model signature: `update(msg tea.Msg) (selfType, tea.Cmd)`
- `recalcLayout()` called on every `tea.WindowSizeMsg`; track inner vs outer sizes (border adds 2px per side)
- All **render helpers are methods on root Model** (not on sub-models) to keep layout logic centralized
- Global intercepts (Ctrl+C, Esc, overlay) handled **first** in root `Update()` before routing to pane handlers
- **Async DB ops** via `tea.Cmd` goroutine returning typed `tea.Msg` (e.g., `taskCreatedMsg`, `taskLoadedMsg`)
- Border color switches active/inactive pane: dim color vs highlight color via lipgloss

---

### Phase 1 — Foundation: Domain + Project Scaffold
*No dependencies — start here*

1. `go.mod` with module `github.com/diptopandit/tork`, all deps: bubbletea, bubbles, lipgloss, cobra, modernc.org/sqlite, zap
2. `internal/domain/task.go` — `Task`, `TaskList`, `FieldDefinition`, `Status` (todo/in_progress/done/cancelled), `Priority` (low/medium/high/urgent) enums
3. `internal/domain/repository.go` — `TaskRepository`, `TaskListRepository` interfaces (ports)
4. `internal/domain/search.go` — `SearchService` interface
5. `internal/domain/exporter.go` — `Exporter` interface
6. `internal/application/dto.go` — `CreateTaskInput`, `UpdateTaskInput`, `TaskFilter` (ListIDs, Status[], Tags[], Search, DueBefore)

### Phase 2 — Infrastructure: SQLite *depends on Phase 1*

7. `internal/infrastructure/db/sqlite.go` — open SQLite connection, DB file at `~/.tork/tork.db` (create dir if missing)
8. `internal/infrastructure/db/migrations.go` — `CREATE TABLE tasks`, `task_lists`, `tasks_fts` (FTS5 virtual table on title + description), indexes on `list_id`, `status`, `due_date`
9. `internal/infrastructure/repository/task_repo_sqlite.go` — implements `TaskRepository`; tags/depends_on/custom_fields stored as JSON columns; `List()` builds dynamic WHERE clause from `TaskFilter`
10. `internal/infrastructure/repository/list_repo_sqlite.go` — implements `TaskListRepository`
11. `internal/infrastructure/search/fts_search.go` — implements `SearchService` via FTS5 `tasks_fts` virtual table; `Index()` inserts/updates, `Search()` returns `[]string` of IDs

### Phase 3 — Application Services *depends on Phase 1 + 2*

12. `internal/application/task_service.go` — `TaskService{repo, search}`: `CreateTask`, `UpdateTask`, `DeleteTask`, `ListTasks`, `GetTask`, `AddDependency`; validation + delegation
13. `internal/application/list_service.go` — `ListService{listRepo}`: `CreateList`, `GetAllLists`
14. `internal/application/filter_service.go` — helpers for building `TaskFilter` from user inputs

### Phase 4 — Config + Logger *parallel with Phase 2/3*

15. `internal/infrastructure/config/config.go` — `Config{Keybindings KeyMap, Display DisplayConfig, Theme ThemeConfig}` structs
16. `internal/infrastructure/config/loader.go` — `Load() (*Config, error)`: reads `~/.tork/config.json`, merges defaults (vim-style j/k/l/h, columns: title/status/priority/due_date)
17. `internal/infrastructure/logger/logger.go` — `Logger` interface + zap implementation; log to `~/.tork/tork.log`

### Phase 5 — TUI Core *depends on Phase 3 + 4*

18. `internal/interface/tui/state.go` — `ViewType` enum (list/detail/edit/filter), `ModeType` enum (normal/edit/filter), `AppState` struct
19. `internal/interface/tui/keymap.go` — `KeyMap` using `bubbles/key`; populated from `config.Keybindings`; implements `help.KeyMap` interface
20. `internal/interface/tui/views/list_view.go` — `listModel` wrapping `bubbles/table`; columns from `config.Display.Columns`; `update()` + `setSize()`; `refreshTasks()` method
21. `internal/interface/tui/app.go` — root `Model{services, config, state, keymap, listView, detailView, editView, filterView, width, height}`; `Init()`, `Update()` routing, `View()` layout composition, `recalcLayout()`

### Phase 6 — TUI Views *depends on Phase 5*

22. `internal/interface/tui/views/detail_view.go` — `detailModel` wrapping `bubbles/viewport`; renders task fields + custom fields; `setTask()` + `setSize()`
23. `internal/interface/tui/views/edit_modal.go` — `editModel` with `bubbles/textarea` for description + `textinput` for other fields; schema-driven rendering of custom fields; returns `taskSavedMsg` on submit
24. `internal/interface/tui/views/filter_panel.go` — `filterModel` with dynamic filter builder; checkboxes for status/priority, tag input, date input

### Phase 7 — CLI *depends on Phase 3, parallel with Phase 5/6*

25. `internal/interface/cli/commands.go` — Cobra root + subcommands: `add`, `list`, `done`, `delete`, `search`; each command calls `TaskService` directly
26. `internal/interface/cli/parser.go` — flag parsing helpers (--list, --status, --priority, --tag, --due)
27. `cmd/taskcli/main.go` — wires DB → repos → services → cobra; same `~/.tork/tork.db`

### Phase 8 — Exporters *depends on Phase 3, parallel with Phase 7*

28. `internal/application/export_service.go` — implements `Exporter`: `MarkdownExporter`, `CSVExporter`

### Phase 9 — Testing *depends on all phases above*

29. `internal/domain/*_test.go` — pure unit tests for Task validation logic
30. `internal/application/*_test.go` — mocked `TaskRepository` (interface mock), test `CreateTask`, `ListTasks`, filters
31. `internal/infrastructure/repository/*_test.go` — integration tests with an in-memory SQLite instance
32. `internal/interface/tui/*_test.go` — state transition tests: send key msgs, assert `AppState` changes

### Phase 10 — TUI Entry Point + Full Wiring *depends on all phases*

33. `cmd/tasktui/main.go` — constructor-injection wiring: open DB → run migrations → init repos → init services → load config → build keymap → start `tea.NewProgram()`; handle terminal raw mode

---

### Relevant Files (Reference from snitch)
- [internal/view/tui/app.go](../juniper/snitch/internal/view/tui/app.go) — root Model pattern, `recalcLayout`, layout math
- [internal/view/tui/list.go](../juniper/snitch/internal/view/tui/list.go) — `listModel` with `bubbles/table`
- [internal/view/tui/detail.go](../juniper/snitch/internal/view/tui/detail.go) — `detailModel` + `bubbles/viewport`
- [cmd/snitch/main.go](../juniper/snitch/cmd/snitch/main.go) — entry point wiring pattern

---

### Verification

1. `go build ./...` — zero compile errors with `modernc.org/sqlite` (no CGO)
2. `go test ./...` — all domain + application mock tests pass
3. `go run ./cmd/tasktui` — TUI launches, list view renders, j/k navigation works
4. Add a task in TUI (edit modal), verify it persists across restarts (SQLite)
5. `go run ./cmd/taskcli add "Buy milk" --list personal` — task appears in next TUI launch
6. Search `task search "milk"` returns correct task IDs via FTS5
7. Resize terminal → layout reflows correctly via `recalcLayout()`
8. Edit `~/.tork/config.json` to change keybinding, verify TUI respects it on restart

---

### Scope Boundaries
- **Included:** All phases 1–10 as listed above
- **Excluded (future):** Remote sync (`infrastructure/sync/` is stubbed with interface only), plugin system
- **Deferred:** Bleve external search (using SQLite FTS5 exclusively)

---

**Further Considerations**

1. **FTS5 sync strategy**: FTS5 virtual table can be populated via triggers on the `tasks` table or explicitly from `SearchService.Index()`. The explicit approach (calling `Index()` from `TaskService`) keeps the infrastructure boundary clean and matches the hexagonal design — recommend this over DB triggers.

2. **Edit modal input**: `bubbletea` doesn't have a first-class form component. Two options: (A) `bubbles/textinput` per field, tab-navigated; (B) a single `bubbles/textarea` in JSON/key:value mode. Option A is more ergonomic for users — recommend it.

3. **Data directory creation**: `~/.tork/` must exist before DB + log file creation. Recommend creating it in `cmd/tasktui/main.go` and `cmd/taskcli/main.go` at startup with `os.MkdirAll`.

---

Ready to hand off to implementation? Or would you like to adjust any phase or scope?