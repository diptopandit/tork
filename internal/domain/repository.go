package domain

// TaskRepository is the port for persisting and querying tasks.
type TaskRepository interface {
	Create(task *Task) error
	Update(task *Task) error
	Delete(taskID string) error
	GetByID(id string) (*Task, error)
	GetByNumID(numID int) (*Task, error)
	List(filter TaskFilter) ([]Task, error)
	NextNumID() (int, error)
}

// UpdateRepository is the port for persisting task updates/comments.
type UpdateRepository interface {
	AddUpdate(u *Update) error
	ListByTaskID(taskID string) ([]Update, error)
}

// TaskListRepository is the port for persisting and querying task lists.
type TaskListRepository interface {
	Create(list *TaskList) error
	Update(list *TaskList) error
	GetByID(id string) (*TaskList, error)
	GetAll() ([]TaskList, error)
	Delete(listID string) error
}

// UserRepository is the port for persisting users (remote DB only).
type UserRepository interface {
	EnsureUser(user *User) error // create if not exists
	GetByID(id string) (*User, error)
	GetByUsername(name string) (*User, error)
}
