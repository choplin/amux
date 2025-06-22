package session

import (
	"bytes"
	"sync"
)

// circularBuffer implements a fixed-size circular buffer for output
type circularBuffer struct {
	data     []byte
	size     int64
	writePos int64
	full     bool
	mu       sync.RWMutex
}

// newCircularBuffer creates a new circular buffer with the given size
func newCircularBuffer(size int64) *circularBuffer {
	return &circularBuffer{
		data: make([]byte, size),
		size: size,
	}
}

// Write writes data to the circular buffer
func (cb *circularBuffer) Write(p []byte) (n int, err error) {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	n = len(p)
	if n == 0 {
		return 0, nil
	}

	// If data is larger than buffer, only keep the last part
	if int64(n) >= cb.size {
		copy(cb.data, p[n-int(cb.size):])
		cb.writePos = 0
		cb.full = true
		return n, nil
	}

	// Calculate how much space is left before wrapping
	spaceToEnd := cb.size - cb.writePos

	if int64(n) <= spaceToEnd {
		// Data fits without wrapping
		copy(cb.data[cb.writePos:], p)
		cb.writePos += int64(n)
		if cb.writePos == cb.size {
			cb.writePos = 0
			cb.full = true
		}
	} else {
		// Data needs to wrap around
		copy(cb.data[cb.writePos:], p[:spaceToEnd])
		copy(cb.data[0:], p[spaceToEnd:])
		cb.writePos = int64(n) - spaceToEnd
		cb.full = true
	}

	return n, nil
}

// Bytes returns the current contents of the buffer in the correct order
func (cb *circularBuffer) Bytes() []byte {
	cb.mu.RLock()
	defer cb.mu.RUnlock()

	if !cb.full && cb.writePos == 0 {
		// Buffer is empty
		return []byte{}
	}

	if !cb.full {
		// Buffer is not full, return from start to writePos
		result := make([]byte, cb.writePos)
		copy(result, cb.data[:cb.writePos])
		return result
	}

	// Buffer is full, need to return in correct order
	result := make([]byte, cb.size)

	// Copy from writePos to end
	copy(result, cb.data[cb.writePos:])

	// Copy from start to writePos
	if cb.writePos > 0 {
		copy(result[cb.size-cb.writePos:], cb.data[:cb.writePos])
	}

	// Trim any trailing null bytes (from initialization)
	return bytes.TrimRight(result, "\x00")
}

// Size returns the capacity of the buffer
func (cb *circularBuffer) Size() int64 {
	return cb.size
}

// IsFull returns whether the buffer has wrapped around
func (cb *circularBuffer) IsFull() bool {
	cb.mu.RLock()
	defer cb.mu.RUnlock()
	return cb.full
}

// Clear resets the buffer
func (cb *circularBuffer) Clear() {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	cb.writePos = 0
	cb.full = false
	// Optionally clear the data
	for i := range cb.data {
		cb.data[i] = 0
	}
}
