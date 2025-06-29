package workspace

import "fmt"

// ErrNotFound is returned when a workspace is not found
type ErrNotFound struct {
	Identifier string
}

func (e ErrNotFound) Error() string {
	return fmt.Sprintf("workspace not found: %s", e.Identifier)
}

// ErrAlreadyExists is returned when attempting to create a workspace that already exists
type ErrAlreadyExists struct {
	Name string
}

func (e ErrAlreadyExists) Error() string {
	return fmt.Sprintf("workspace already exists: %s", e.Name)
}
