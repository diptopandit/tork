package domain

import "fmt"

// ErrNotFound returns a descriptive error for a missing entity.
func ErrNotFound(id string) error {
	return fmt.Errorf("not found: %s", id)
}
