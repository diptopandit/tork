package application

import (
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/diptopandit/tork/internal/domain"
)

// ListService orchestrates task-list use-cases.
type ListService struct {
	repo domain.TaskListRepository
}

// NewListService constructs a ListService.
func NewListService(repo domain.TaskListRepository) *ListService {
	return &ListService{repo: repo}
}

// CreateList creates and persists a new TaskList.
func (s *ListService) CreateList(name string, schema map[string]domain.FieldDefinition) (*domain.TaskList, error) {
	if name == "" {
		return nil, fmt.Errorf("list name is required")
	}
	if schema == nil {
		schema = map[string]domain.FieldDefinition{}
	}
	l := &domain.TaskList{
		ID:        uuid.NewString(),
		Name:      name,
		Schema:    schema,
		CreatedAt: time.Now().UTC(),
	}
	if err := s.repo.Create(l); err != nil {
		return nil, fmt.Errorf("create list: %w", err)
	}
	return l, nil
}

// GetAllLists returns every persisted TaskList.
func (s *ListService) GetAllLists() ([]domain.TaskList, error) {
	return s.repo.GetAll()
}

// GetList retrieves a single TaskList by ID.
func (s *ListService) GetList(id string) (*domain.TaskList, error) {
	return s.repo.GetByID(id)
}

// UpdateList renames a TaskList.
func (s *ListService) UpdateList(id, name string) (*domain.TaskList, error) {
	if name == "" {
		return nil, fmt.Errorf("list name is required")
	}
	l, err := s.repo.GetByID(id)
	if err != nil {
		return nil, fmt.Errorf("update list: %w", err)
	}
	l.Name = name
	if err := s.repo.Update(l); err != nil {
		return nil, fmt.Errorf("update list: %w", err)
	}
	return l, nil
}

// DeleteList removes a TaskList and all its tasks (via DB cascade).
func (s *ListService) DeleteList(id string) error {
	if err := s.repo.Delete(id); err != nil {
		return fmt.Errorf("delete list: %w", err)
	}
	return nil
}
