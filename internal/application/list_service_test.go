package application

import (
	"fmt"
	"testing"

	"github.com/diptopandit/tork/internal/domain"
)

// ---- mock list repo ----------------------------------------------------------

type mockListRepo struct {
	lists map[string]*domain.TaskList
}

func newMockListRepo() *mockListRepo {
	return &mockListRepo{lists: map[string]*domain.TaskList{}}
}

func (m *mockListRepo) Create(l *domain.TaskList) error {
	m.lists[l.ID] = l
	return nil
}

func (m *mockListRepo) Update(l *domain.TaskList) error {
	if _, ok := m.lists[l.ID]; !ok {
		return domain.ErrNotFound(l.ID)
	}
	m.lists[l.ID] = l
	return nil
}

func (m *mockListRepo) GetByID(id string) (*domain.TaskList, error) {
	if l, ok := m.lists[id]; ok {
		copy := *l
		return &copy, nil
	}
	return nil, domain.ErrNotFound(id)
}

func (m *mockListRepo) GetAll() ([]domain.TaskList, error) {
	out := make([]domain.TaskList, 0, len(m.lists))
	for _, l := range m.lists {
		out = append(out, *l)
	}
	return out, nil
}

func (m *mockListRepo) Delete(id string) error {
	delete(m.lists, id)
	return nil
}

// ---- error-returning list repo -----------------------------------------------

type errListRepo struct {
	mockListRepo
}

func (m *errListRepo) Create(*domain.TaskList) error { return fmt.Errorf("db error") }
func (m *errListRepo) Update(*domain.TaskList) error { return fmt.Errorf("db error") }
func (m *errListRepo) Delete(string) error           { return fmt.Errorf("db error") }

// ---- helpers ----------------------------------------------------------------

func newTestListSvc() *ListService {
	return NewListService(newMockListRepo(), nil)
}

// ---- tests ------------------------------------------------------------------

func TestCreateList_Valid(t *testing.T) {
	svc := newTestListSvc()
	l, err := svc.CreateList("Work", nil)
	if err != nil {
		t.Fatal(err)
	}
	if l.ID == "" {
		t.Error("expected non-empty ID")
	}
	if l.Name != "Work" {
		t.Errorf("name = %q, want Work", l.Name)
	}
}

func TestCreateList_EmptyName(t *testing.T) {
	svc := newTestListSvc()
	_, err := svc.CreateList("", nil)
	if err == nil {
		t.Error("expected error for empty name")
	}
}

func TestCreateList_WithSchema(t *testing.T) {
	svc := newTestListSvc()
	schema := map[string]domain.FieldDefinition{
		"custom": {Type: "text"},
	}
	l, err := svc.CreateList("Custom", schema)
	if err != nil {
		t.Fatal(err)
	}
	if len(l.Schema) != 1 {
		t.Errorf("schema len = %d, want 1", len(l.Schema))
	}
}

func TestCreateList_NilSchema(t *testing.T) {
	svc := newTestListSvc()
	l, _ := svc.CreateList("NoSchema", nil)
	if l.Schema == nil {
		t.Error("schema should be {} not nil")
	}
}

func TestCreateList_RepoError(t *testing.T) {
	svc := NewListService(&errListRepo{mockListRepo: *newMockListRepo()}, nil)
	_, err := svc.CreateList("Fail", nil)
	if err == nil {
		t.Error("expected error from repo")
	}
}

func TestGetAllLists(t *testing.T) {
	svc := newTestListSvc()
	svc.CreateList("A", nil)
	svc.CreateList("B", nil)
	lists, err := svc.GetAllLists()
	if err != nil {
		t.Fatal(err)
	}
	if len(lists) != 2 {
		t.Errorf("got %d lists, want 2", len(lists))
	}
}

func TestGetAllLists_Empty(t *testing.T) {
	svc := newTestListSvc()
	lists, err := svc.GetAllLists()
	if err != nil {
		t.Fatal(err)
	}
	if len(lists) != 0 {
		t.Errorf("got %d lists, want 0", len(lists))
	}
}

func TestGetList(t *testing.T) {
	svc := newTestListSvc()
	created, _ := svc.CreateList("Test", nil)
	got, err := svc.GetList(created.ID)
	if err != nil {
		t.Fatal(err)
	}
	if got.Name != "Test" {
		t.Errorf("name = %q, want Test", got.Name)
	}
}

func TestGetList_NotFound(t *testing.T) {
	svc := newTestListSvc()
	_, err := svc.GetList("nonexistent")
	if err == nil {
		t.Error("expected error for nonexistent list")
	}
}

func TestUpdateList_Valid(t *testing.T) {
	svc := newTestListSvc()
	l, _ := svc.CreateList("Old", nil)
	updated, err := svc.UpdateList(l.ID, "New")
	if err != nil {
		t.Fatal(err)
	}
	if updated.Name != "New" {
		t.Errorf("name = %q, want New", updated.Name)
	}
}

func TestUpdateList_EmptyName(t *testing.T) {
	svc := newTestListSvc()
	l, _ := svc.CreateList("Valid", nil)
	_, err := svc.UpdateList(l.ID, "")
	if err == nil {
		t.Error("expected error for empty name")
	}
}

func TestUpdateList_NotFound(t *testing.T) {
	svc := newTestListSvc()
	_, err := svc.UpdateList("nonexistent", "Name")
	if err == nil {
		t.Error("expected error for nonexistent list")
	}
}

func TestDeleteList_Valid(t *testing.T) {
	svc := newTestListSvc()
	l, _ := svc.CreateList("ToDelete", nil)
	err := svc.DeleteList(l.ID)
	if err != nil {
		t.Fatal(err)
	}
	_, err = svc.GetList(l.ID)
	if err == nil {
		t.Error("expected not found after delete")
	}
}

func TestDeleteList_RepoError(t *testing.T) {
	svc := NewListService(&errListRepo{mockListRepo: *newMockListRepo()}, nil)
	err := svc.DeleteList("any")
	if err == nil {
		t.Error("expected error from repo")
	}
}

func TestNewListService_NilLogger(t *testing.T) {
	svc := NewListService(newMockListRepo(), nil)
	if svc.log == nil {
		t.Error("logger should be nop, not nil")
	}
}
