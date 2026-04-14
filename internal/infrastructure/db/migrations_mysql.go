package db

import (
	"database/sql"
	"fmt"
)

const mysqlSchema = `
CREATE TABLE IF NOT EXISTS users (
    id         VARCHAR(36) PRIMARY KEY,
    username   VARCHAR(100) UNIQUE NOT NULL,
    created_at DATETIME NOT NULL
);

CREATE TABLE IF NOT EXISTS task_lists (
    id          VARCHAR(36) PRIMARY KEY,
    name        VARCHAR(255) NOT NULL,
    schema_json TEXT NOT NULL,
    owner_id    VARCHAR(36) NOT NULL,
    visibility  VARCHAR(20) NOT NULL DEFAULT 'private',
    created_at  DATETIME NOT NULL,
    INDEX idx_task_lists_owner (owner_id),
    CONSTRAINT fk_task_lists_owner FOREIGN KEY (owner_id) REFERENCES users(id)
);

CREATE TABLE IF NOT EXISTS list_members (
    list_id VARCHAR(36) NOT NULL,
    user_id VARCHAR(36) NOT NULL,
    role    VARCHAR(20) NOT NULL DEFAULT 'editor',
    PRIMARY KEY (list_id, user_id),
    CONSTRAINT fk_list_members_list FOREIGN KEY (list_id) REFERENCES task_lists(id) ON DELETE CASCADE,
    CONSTRAINT fk_list_members_user FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
);

CREATE TABLE IF NOT EXISTS tasks (
    id            VARCHAR(36) PRIMARY KEY,
    num_id        INT UNIQUE,
    list_id       VARCHAR(36) NOT NULL,
    title         VARCHAR(500) NOT NULL,
    description   TEXT NOT NULL,
    status        VARCHAR(50) NOT NULL DEFAULT 'todo',
    priority      INT NOT NULL DEFAULT 2,
    due_date      DATETIME,
    tags          TEXT NOT NULL,
    custom_fields TEXT NOT NULL,
    parent_id     VARCHAR(36),
    depends_on    TEXT NOT NULL,
    created_at    DATETIME NOT NULL,
    updated_at    DATETIME NOT NULL,
    INDEX idx_tasks_list_id (list_id),
    INDEX idx_tasks_status (status),
    INDEX idx_tasks_due_date (due_date),
    FULLTEXT INDEX idx_tasks_fts (title, description),
    CONSTRAINT fk_tasks_list FOREIGN KEY (list_id) REFERENCES task_lists(id) ON DELETE CASCADE
);

CREATE TABLE IF NOT EXISTS task_updates (
    id         VARCHAR(36) PRIMARY KEY,
    task_id    VARCHAR(36) NOT NULL,
    body       TEXT NOT NULL,
    created_at DATETIME NOT NULL,
    INDEX idx_task_updates_task_id (task_id),
    CONSTRAINT fk_task_updates_task FOREIGN KEY (task_id) REFERENCES tasks(id) ON DELETE CASCADE
);
`

// MigrateMySQL applies the MySQL schema idempotently.
// MySQL CREATE TABLE IF NOT EXISTS and CREATE INDEX IF NOT EXISTS ensure
// this is safe to run on every startup.
func MigrateMySQL(db *sql.DB) error {
	// MySQL doesn't support multi-statement exec by default, so we split.
	stmts := splitStatements(mysqlSchema)
	for _, stmt := range stmts {
		if stmt == "" {
			continue
		}
		if _, err := db.Exec(stmt); err != nil {
			return fmt.Errorf("mysql migrate: %w\nstatement: %s", err, stmt)
		}
	}
	return nil
}

// splitStatements splits a multi-statement SQL string on semicolons.
func splitStatements(schema string) []string {
	var stmts []string
	current := ""
	for _, ch := range schema {
		current += string(ch)
		if ch == ';' {
			trimmed := trimSpace(current)
			if trimmed != "" && trimmed != ";" {
				stmts = append(stmts, trimmed)
			}
			current = ""
		}
	}
	if trimmed := trimSpace(current); trimmed != "" {
		stmts = append(stmts, trimmed)
	}
	return stmts
}

func trimSpace(s string) string {
	// Simple trim without importing strings to avoid circular dep concerns.
	start, end := 0, len(s)
	for start < end && (s[start] == ' ' || s[start] == '\t' || s[start] == '\n' || s[start] == '\r') {
		start++
	}
	for end > start && (s[end-1] == ' ' || s[end-1] == '\t' || s[end-1] == '\n' || s[end-1] == '\r') {
		end--
	}
	return s[start:end]
}
