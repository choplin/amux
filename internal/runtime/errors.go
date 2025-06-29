package runtime

import "errors"

var (
	// ErrRuntimeNotAvailable indicates that the runtime is not available or properly configured
	ErrRuntimeNotAvailable = errors.New("runtime is not available")

	// ErrProcessNotFound indicates that the requested process was not found
	ErrProcessNotFound = errors.New("process not found")

	// ErrProcessAlreadyDone indicates that the process has already completed
	ErrProcessAlreadyDone = errors.New("process already completed")

	// ErrInvalidCommand indicates that the command specification is invalid
	ErrInvalidCommand = errors.New("invalid command")

	// ErrNotSupported indicates that the operation is not supported by this runtime
	ErrNotSupported = errors.New("operation not supported by this runtime")
)
