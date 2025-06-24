package semaphore

import "fmt"

// ErrSemaphoreFull is returned when trying to acquire a semaphore that is at capacity.
type ErrSemaphoreFull struct {
	Capacity int
	Current  int
}

func (e ErrSemaphoreFull) Error() string {
	return fmt.Sprintf("semaphore is full (capacity: %d, current: %d)", e.Capacity, e.Current)
}

// ErrAlreadyHolder is returned when trying to acquire a semaphore that the holder already has.
type ErrAlreadyHolder struct {
	ID string
}

func (e ErrAlreadyHolder) Error() string {
	return fmt.Sprintf("holder %s already has the semaphore", e.ID)
}

// ErrNotHolder is returned when trying to release a semaphore that the holder doesn't have.
type ErrNotHolder struct {
	ID string
}

func (e ErrNotHolder) Error() string {
	return fmt.Sprintf("holder %s does not have the semaphore", e.ID)
}
