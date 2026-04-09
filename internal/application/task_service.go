package application

import (
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/diptopandit/tork/internal/domain"
)

// TaskService orchestrates task use-cases.
type TaskService struct {
	repo       domain.TaskRepository
	updateRepo domain.UpdateRepository
	search     domain.SearchService
}

// NewTaskService constructs a TaskService.
func NewTaskService(repo domain.TaskRepository, updateRepo domain.UpdateRepository, search domain.SearchService) *TaskService {
	return &TaskService{repo: repo, updateRepo: updateRepo, search: search}
}

// CreateTask validates and persists a new task, then indexes it.
func (s *TaskService) CreateTask(in CreateTaskInput) (*domain.Task, error) {
	if in.Title == "" {
		return nil, fmt.Errorf("task title is required")
	}
	if in.ListID == "" {
		return nil, fmt.Errorf("task list ID is required")
	}

	numID, err := s.repo.NextNumID()
	if err != nil {
		return nil, fmt.Errorf("create task: next num_id: %w", err)
	}

	now := time.Now().UTC()
	t := &domain.Task{
		ID:           uuid.NewString(),
		NumID:        numID,
		ListID:       in.ListID,
		Title:        in.Title,
		Description:  in.Description,
		Status:       domain.StatusTodo,
		Priority:     in.Priority,
		DueDate:      in.DueDate,
		Tags:         in.Tags,
		CustomFields: in.CustomFields,
		ParentID:     in.ParentID,
		DependsOn:    in.DependsOn,
		CreatedAt:    now,
		UpdatedAt:    now,
	}
	if t.Priority == 0 {
		t.Priority = domain.PriorityMedium
	}
	if t.Tags == nil {
		t.Tags = []string{}
	}
	if t.DependsOn == nil {
		t.DependsOn = []string{}
	}

	if err := s.repo.Create(t); err != nil {
		return nil, fmt.Errorf("create task: %w", err)
	}
	// FTS index is maintained by DB triggers; explicit call here is a no-op safety net.
	_ = s.search.Index(t)
	return t, nil
}

// UpdateTask applies the non-nil fields of UpdateTaskInput to the existing task.
func (s *TaskService) UpdateTask(in UpdateTaskInput) (*domain.Task, error) {
	t, err := s.repo.GetByID(in.ID)
	if err != nil {
		return nil, fmt.Errorf("update task: %w", err)
	}
	if in.Title != nil {
		if *in.Title == "" {
			return nil, fmt.Errorf("task title cannot be empty")
		}
		t.Title = *in.Title
	}
	if in.Description != nil {
		t.Description = *in.Description
	}
	if in.Status != nil {
		t.Status = *in.Status
	}
	if in.Priority != nil {
		t.Priority = *in.Priority
	}
	if in.DueDate != nil {
		t.DueDate = in.DueDate
	}
	if in.Tags != nil {
		t.Tags = in.Tags
	}
	if in.CustomFields != nil {
		t.CustomFields = in.CustomFields
	}
	if in.DependsOn != nil {
		t.DependsOn = in.DependsOn
	}
	t.UpdatedAt = time.Now().UTC()

	if err := s.repo.Update(t); err != nil {
		return nil, fmt.Errorf("update task: %w", err)
	}
	_ = s.search.Index(t)
	return t, nil
}

// DeleteTask removes a task and its search index entry.
func (s *TaskService) DeleteTask(id string) error {
	if err := s.repo.Delete(id); err != nil {
		return fmt.Errorf("delete task: %w", err)
	}
	_ = s.search.Delete(id)
	return nil
}

// GetTask retrieves a single task by ID.
func (s *TaskService) GetTask(id string) (*domain.Task, error) {
	return s.repo.GetByID(id)
}

// GetTaskByNum retrieves a single task by its numeric short ID.
func (s *TaskService) GetTaskByNum(numID int) (*domain.Task, error) {
	return s.repo.GetByNumID(numID)
}

// AddUpdate appends an immutable, timestamped comment to a task.
func (s *TaskService) AddUpdate(taskID, body string) (*domain.Update, error) {
	if body == "" {
		return nil, fmt.Errorf("update body is required")
	}
	// Verify task exists.
	if _, err := s.repo.GetByID(taskID); err != nil {
		return nil, fmt.Errorf("add update: %w", err)
	}
	u := &domain.Update{
		ID:        uuid.NewString(),
		TaskID:    taskID,
		Body:      body,
		CreatedAt: time.Now().UTC(),
	}
	if err := s.updateRepo.AddUpdate(u); err != nil {
		return nil, fmt.Errorf("add update: %w", err)
	}
	return u, nil
}

// GetUpdates returns all updates for a task, newest first.
func (s *TaskService) GetUpdates(taskID string) ([]domain.Update, error) {
	return s.updateRepo.ListByTaskID(taskID)
}

// ListTasks returns tasks matching the given filter.
func (s *TaskService) ListTasks(filter TaskFilter) ([]domain.Task, error) {
	if filter.Search != "" {
		ids, err := s.search.Search(filter.Search)
		if err != nil {
			return nil, fmt.Errorf("list tasks search: %w", err)
		}
		if len(ids) == 0 {
			return nil, nil
		}
		// Resolve each ID individually (preserves FTS ranking order).
		tasks := make([]domain.Task, 0, len(ids))
		for _, id := range ids {
			t, err := s.repo.GetByID(id)
			if err != nil {
				continue // task may have been deleted
			}
			tasks = append(tasks, *t)
		}
		return tasks, nil
	}
	return s.repo.List(filter)
}

// AddDependency records that taskID depends on depID.
func (s *TaskService) AddDependency(taskID, depID string) error {
	t, err := s.repo.GetByID(taskID)
	if err != nil {
		return fmt.Errorf("add dependency: %w", err)
	}
	for _, d := range t.DependsOn {
		if d == depID {
			return nil // already recorded
		}
	}
	t.DependsOn = append(t.DependsOn, depID)
	t.UpdatedAt = time.Now().UTC()
	return s.repo.Update(t)
}
