package db

import (
	"database/sql"
	"fmt"
)

const schema = `
CREATE TABLE IF NOT EXISTS task_lists (
    id          TEXT PRIMARY KEY,
    name        TEXT NOT NULL,
    schema_json TEXT NOT NULL DEFAULT '{}',
    created_at  DATETIME NOT NULL
);

CREATE TABLE IF NOT EXISTS tasks (
    id            TEXT PRIMARY KEY,
    num_id        INTEGER UNIQUE,
    list_id       TEXT NOT NULL REFERENCES task_lists(id) ON DELETE CASCADE,
    title         TEXT NOT NULL,
    description   TEXT NOT NULL DEFAULT '',
    status        TEXT NOT NULL DEFAULT 'todo',
    priority      INTEGER NOT NULL DEFAULT 2,
    due_date      DATETIME,
    tags          TEXT NOT NULL DEFAULT '[]',
    custom_fields TEXT NOT NULL DEFAULT '{}',
    parent_id     TEXT,
    depends_on    TEXT NOT NULL DEFAULT '[]',
    created_at    DATETIME NOT NULL,
    updated_at    DATETIME NOT NULL
);

CREATE TABLE IF NOT EXISTS task_updates (
    id         TEXT PRIMARY KEY,
    task_id    TEXT NOT NULL REFERENCES tasks(id) ON DELETE CASCADE,
    body       TEXT NOT NULL,
    created_at DATETIME NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_task_updates_task_id ON task_updates(task_id);
CREATE INDEX IF NOT EXISTS idx_tasks_list_id  ON tasks(list_id);
CREATE INDEX IF NOT EXISTS idx_tasks_status   ON tasks(status);
CREATE INDEX IF NOT EXISTS idx_tasks_due_date ON tasks(due_date);

CREATE VIRTUAL TABLE IF NOT EXISTS tasks_fts USING fts5(
    id UNINDEXED,
    title,
    description,
    content=tasks,
    content_rowid=rowid
);

CREATE TRIGGER IF NOT EXISTS tasks_ai AFTER INSERT ON tasks BEGIN
    INSERT INTO tasks_fts(rowid, id, title, description)
    VALUES (new.rowid, new.id, new.title, new.description);
END;

CREATE TRIGGER IF NOT EXISTS tasks_ad AFTER DELETE ON tasks BEGIN
    INSERT INTO tasks_fts(tasks_fts, rowid, id, title, description)
    VALUES ('delete', old.rowid, old.id, old.title, old.description);
END;

CREATE TRIGGER IF NOT EXISTS tasks_au AFTER UPDATE ON tasks BEGIN
    INSERT INTO tasks_fts(tasks_fts, rowid, id, title, description)
    VALUES ('delete', old.rowid, old.id, old.title, old.description);
    INSERT INTO tasks_fts(rowid, id, title, description)
    VALUES (new.rowid, new.id, new.title, new.description);
END;
`

// Migrate applies the schema DDL idempotently to db.
func Migrate(db *sql.DB) error {
	if _, err := db.Exec(schema); err != nil {
		return fmt.Errorf("migrate: %w", err)
	}

	// Add num_id column if missing (pre-existing databases).
	var hasNumID bool
	rows, err := db.Query(`PRAGMA table_info(tasks)`)
	if err == nil {
		defer rows.Close()
		for rows.Next() {
			var cid int
			var name, typ string
			var notnull int
			var dflt sql.NullString
			var pk int
			if err := rows.Scan(&cid, &name, &typ, &notnull, &dflt, &pk); err == nil {
				if name == "num_id" {
					hasNumID = true
				}
			}
		}
	}
	if !hasNumID {
		_, _ = db.Exec(`ALTER TABLE tasks ADD COLUMN num_id INTEGER UNIQUE`)
	}

	// Create task_updates table if it doesn't exist (covered in schema, but
	// older databases may lack it).
	_, _ = db.Exec(`CREATE TABLE IF NOT EXISTS task_updates (
		id         TEXT PRIMARY KEY,
		task_id    TEXT NOT NULL REFERENCES tasks(id) ON DELETE CASCADE,
		body       TEXT NOT NULL,
		created_at DATETIME NOT NULL
	)`)
	_, _ = db.Exec(`CREATE INDEX IF NOT EXISTS idx_task_updates_task_id ON task_updates(task_id)`)

	// Backfill num_id for any existing rows that lack one.
	_, _ = db.Exec(`
		UPDATE tasks SET num_id = (
			SELECT COUNT(*) FROM tasks t2
			WHERE t2.rowid <= tasks.rowid
		) WHERE num_id IS NULL
	`)

	return nil
}
