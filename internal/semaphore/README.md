# File-Based Semaphore Library

A generic, reusable file-based semaphore library for coordinating access to shared resources across multiple processes.

## Overview

This package provides a file-based semaphore implementation that can be used to coordinate access between multiple processes. The primary use case in amux is managing workspace access, but the design is generic enough for any resource coordination needs.

## Features

- **Process-safe**: Works across different processes using file locking
- **Crash-resilient**: Persists state to disk, handles ungraceful shutdowns
- **Configurable capacity**: Support for both exclusive (capacity=1) and shared access
- **Atomic operations**: All file operations are atomic to prevent corruption
- **Simple API**: Easy to use correctly, hard to misuse

## Installation

This package is part of the amux project. To use it in your own project:

```go
import "github.com/aki/amux/internal/semaphore"
```

## Usage

### Basic Example

```go
// Create a semaphore with capacity 1 (exclusive access)
sem, err := semaphore.New("/path/to/semaphore.json", 1)
if err != nil {
    return err
}
defer sem.Close()

// Define a holder
type MyHolder struct {
    id string
}

func (h *MyHolder) ID() string {
    return h.id
}

// Acquire the semaphore
holder := &MyHolder{id: "process-123"}
err = sem.Acquire(holder)
if err == semaphore.ErrNoCapacity {
    // Semaphore is full
    return err
}
if err != nil {
    return err
}

// Do work with exclusive access...

// Release when done
err = sem.Release(holder.ID())
if err != nil {
    return err
}
```

### Shared Access Example

```go
// Create a semaphore with capacity 3
sem, err := semaphore.New("/path/to/shared.json", 3)
if err != nil {
    return err
}
defer sem.Close()

// Multiple holders can acquire up to capacity
for i := 0; i < 3; i++ {
    holder := &MyHolder{id: fmt.Sprintf("worker-%d", i)}
    if err := sem.Acquire(holder); err != nil {
        log.Printf("Worker %d failed to acquire: %v", i, err)
    }
}
```

### Query Operations

```go
// Get current holders
holders := sem.Holders() // ["holder-1", "holder-2"]

// Get count of current holders
count := sem.Count() // 2

// Get available capacity
available := sem.Available() // 1 (if capacity is 3)
```

### Cleanup Stale Holders

```go
// Remove specific holders (useful for cleanup after crashes)
err = sem.Remove("stale-holder-1", "stale-holder-2")
if err != nil {
    return err
}
```

## API Reference

### Types

```go
// Holder represents an entity that can hold a semaphore
type Holder interface {
    ID() string
}

// FileSemaphore manages access to a resource using file-based locking
type FileSemaphore struct {
    // unexported fields
}
```

### Functions

#### New

```go
func New(path string, capacity int) (*FileSemaphore, error)

```

Creates a new file-based semaphore. If capacity < 1, it defaults to 1.

#### Acquire

```go
func (s *FileSemaphore) Acquire(holder Holder) error
```

Attempts to acquire the semaphore for a holder. Returns `ErrNoCapacity` if the semaphore is full, or `ErrAlreadyHeld` if this holder already has the semaphore.

#### Release

```go
func (s *FileSemaphore) Release(holderID string) error
```

Releases the semaphore for a specific holder ID. Returns `ErrNotHeld` if the holder doesn't have the semaphore.

#### Remove

```go
func (s *FileSemaphore) Remove(holderIDs ...string) error

```

Removes one or more holders from the semaphore. Useful for cleaning up after crashed processes. This operation is idempotent - removing non-existent holders doesn't cause an error.

#### Holders

```go
func (s *FileSemaphore) Holders() []string
```

Returns the IDs of all current holders.

#### Count

```go

func (s *FileSemaphore) Count() int
```

Returns the number of current holders.

#### Available

```go
func (s *FileSemaphore) Available() int
```

Returns the number of available slots.

#### Close

```go
func (s *FileSemaphore) Close() error
```

Closes the semaphore and releases resources. Should be called when done with the semaphore.

### Error Types

```go
var (
    ErrNoCapacity   = errors.New("semaphore has no available capacity")
    ErrAlreadyHeld  = errors.New("semaphore already held by this holder")
    ErrNotHeld      = errors.New("semaphore not held by this holder")
)
```

## File Format

The semaphore state is persisted as JSON:

```json

{
  "capacity": 1,
  "holders": [
    {
      "id": "session-123",
      "acquired_at": "2024-01-01T10:00:00Z"
    }
  ]
}

```

## Implementation Details

### File Locking

The library uses OS-level file locking (`flock` on Unix) to ensure process safety. Each operation:

1. Acquires an exclusive lock on the lock file
2. Reads the current state
3. Performs the operation

4. Writes the new state atomically
5. Releases the lock

### Atomic Writes

All writes are performed atomically:

1. Write to a temporary file (`.tmp` suffix)
2. Atomically rename to the target file
3. Clean up temporary file on failure

### Process Safety

The library is safe for concurrent use across multiple processes. File locking ensures that only one process can modify the semaphore state at a time.

## Testing

The package includes comprehensive tests:

- Unit tests for all operations
- Concurrent access tests (multiple goroutines)
- Process safety tests (multiple processes)
- Persistence tests
- Atomic operation tests

Run tests:

```bash
go test ./internal/semaphore/...
```

## Future Enhancements

Potential future improvements:

- TTL support for automatic holder expiry
- Wait/notify mechanism for capacity availability
- Distributed locking support (etcd, Redis)
- Metrics and monitoring hooks
- Priority queue for waiting holders

## License

This package is part of the amux project and follows the same license.
