package repository

import (
	"database/sql"
	"encoding/json"
	"fmt"

	"github.com/diptopandit/tork/internal/domain"
)

// ListRepoMySQL implements domain.TaskListRepository backed by MySQL.
// All operations are scoped to the current user (owner or member).
type ListRepoMySQL struct {
	db     *sql.DB
	userID string
}

// NewListRepoMySQL creates a new ListRepoMySQL.
func NewListRepoMySQL(db *sql.DB, userID string) *ListRepoMySQL {
	return &ListRepoMySQL{db: db, userID: userID}
}

// Create inserts a TaskList owned by the current user.
func (r *ListRepoMySQL) Create(list *domain.TaskList) error {
	schemaJSON, err := json.Marshal(list.Schema)
	if err != nil {
		return fmt.Errorf("list create: marshal schema: %w", err)
	}
	ownerID := list.OwnerID
	if ownerID == "" {
		ownerID = r.userID
	}
	visibility := list.Visibility
	if visibility == "" {
		visibility = "private"
	}
	_, err = r.db.Exec(
		`INSERT INTO task_lists (id, name, schema_json, owner_id, visibility, created_at)
		 VALUES (?,?,?,?,?,?)`,
		list.ID, list.Name, string(schemaJSON), ownerID, visibility, list.CreatedAt.UTC(),
	)
	if err != nil {
		return fmt.Errorf("list create: %w", err)
	}
	return nil
}

// Update updates a TaskList's name and schema.
// Only the owner or an admin member can update.
func (r *ListRepoMySQL) Update(list *domain.TaskList) error {
	schemaJSON, err := json.Marshal(list.Schema)
	if err != nil {
		return fmt.Errorf("list update: marshal schema: %w", err)
	}
	res, err := r.db.Exec(
		`UPDATE task_lists SET name=?, schema_json=?
		 WHERE id=? AND (owner_id=? OR id IN (
			SELECT list_id FROM list_members WHERE user_id=? AND role='admin'
		 ))`,
		list.Name, string(schemaJSON), list.ID, r.userID, r.userID,
	)
	if err != nil {
		return fmt.Errorf("list update: %w", err)
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return fmt.Errorf("list update: not found or access denied: %s", list.ID)
	}
	return nil
}

// GetByID fetches a TaskList accessible to the current user.
func (r *ListRepoMySQL) GetByID(id string) (*domain.TaskList, error) {
	row := r.db.QueryRow(
		`SELECT id, name, schema_json, owner_id, visibility, created_at
		 FROM task_lists
		 WHERE id=? AND (owner_id=? OR id IN (
			SELECT list_id FROM list_members WHERE user_id=?
		 ))`, id, r.userID, r.userID)
	l, err := scanListMySQL(row)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("list not found: %s", id)
	}
	if err != nil {
		return nil, fmt.Errorf("list get: %w", err)
	}
	return l, nil
}

// GetAll returns all TaskLists the current user owns or is a member of.
func (r *ListRepoMySQL) GetAll() ([]domain.TaskList, error) {
	rows, err := r.db.Query(
		`SELECT id, name, schema_json, owner_id, visibility, created_at
		 FROM task_lists
		 WHERE owner_id=? OR id IN (
			SELECT list_id FROM list_members WHERE user_id=?
		 )
		 ORDER BY created_at ASC`, r.userID, r.userID)
	if err != nil {
		return nil, fmt.Errorf("list get-all: %w", err)
	}
	defer rows.Close()

	var lists []domain.TaskList
	for rows.Next() {
		l, err := scanListMySQL(rows)
		if err != nil {
			return nil, fmt.Errorf("list get-all scan: %w", err)
		}
		lists = append(lists, *l)
	}
	return lists, rows.Err()
}

// Delete removes a TaskList (only allowed for owner).
func (r *ListRepoMySQL) Delete(listID string) error {
	res, err := r.db.Exec(
		"DELETE FROM task_lists WHERE id=? AND owner_id=?", listID, r.userID)
	if err != nil {
		return fmt.Errorf("list delete: %w", err)
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return fmt.Errorf("list delete: not found or not owner: %s", listID)
	}
	return nil
}

// ---- list membership --------------------------------------------------------

// AddMember adds a user as a member of a list. Only owner can do this.
func (r *ListRepoMySQL) AddMember(listID, memberUserID, role string) error {
	if role == "" {
		role = "editor"
	}
	// Verify caller is owner.
	var ownerID string
	err := r.db.QueryRow("SELECT owner_id FROM task_lists WHERE id=?", listID).Scan(&ownerID)
	if err != nil {
		return fmt.Errorf("add member: list not found: %w", err)
	}
	if ownerID != r.userID {
		return fmt.Errorf("add member: only the list owner can add members")
	}
	_, err = r.db.Exec(
		`INSERT INTO list_members (list_id, user_id, role)
		 VALUES (?,?,?)
		 ON DUPLICATE KEY UPDATE role=VALUES(role)`,
		listID, memberUserID, role,
	)
	if err != nil {
		return fmt.Errorf("add member: %w", err)
	}
	return nil
}

// RemoveMember removes a user from a list. Only owner can do this.
func (r *ListRepoMySQL) RemoveMember(listID, memberUserID string) error {
	var ownerID string
	err := r.db.QueryRow("SELECT owner_id FROM task_lists WHERE id=?", listID).Scan(&ownerID)
	if err != nil {
		return fmt.Errorf("remove member: list not found: %w", err)
	}
	if ownerID != r.userID {
		return fmt.Errorf("remove member: only the list owner can remove members")
	}
	_, err = r.db.Exec(
		"DELETE FROM list_members WHERE list_id=? AND user_id=?", listID, memberUserID)
	if err != nil {
		return fmt.Errorf("remove member: %w", err)
	}
	return nil
}

// ListMembers returns all members of a list.
func (r *ListRepoMySQL) ListMembers(listID string) ([]domain.ListMember, error) {
	rows, err := r.db.Query(
		`SELECT list_id, user_id, role FROM list_members WHERE list_id=?`, listID)
	if err != nil {
		return nil, fmt.Errorf("list members: %w", err)
	}
	defer rows.Close()

	var members []domain.ListMember
	for rows.Next() {
		var m domain.ListMember
		if err := rows.Scan(&m.ListID, &m.UserID, &m.Role); err != nil {
			return nil, fmt.Errorf("list members scan: %w", err)
		}
		members = append(members, m)
	}
	return members, rows.Err()
}

// ---- helpers ----------------------------------------------------------------

type mysqlScanner interface {
	Scan(dest ...interface{}) error
}

func scanListMySQL(s mysqlScanner) (*domain.TaskList, error) {
	var l domain.TaskList
	var schemaJSON string
	var ownerID, visibility string
	if err := s.Scan(&l.ID, &l.Name, &schemaJSON, &ownerID, &visibility, &l.CreatedAt); err != nil {
		return nil, err
	}
	l.OwnerID = ownerID
	l.Visibility = visibility
	if err := json.Unmarshal([]byte(schemaJSON), &l.Schema); err != nil {
		l.Schema = nil
	}
	return &l, nil
}
