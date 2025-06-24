// Package state provides session state management functionality
package state

import "fmt"

// ErrInvalidTransition is returned when an invalid state transition is attempted
type ErrInvalidTransition struct {
	From Status
	To   Status
}

// Error implements the error interface
func (e *ErrInvalidTransition) Error() string {
	return fmt.Sprintf("invalid state transition from %s to %s", e.From, e.To)
}

// ErrAlreadyInState is returned when trying to transition to the current state
type ErrAlreadyInState struct {
	State Status
}

// Error implements the error interface
func (e *ErrAlreadyInState) Error() string {
	return fmt.Sprintf("already in state %s", e.State)
}
