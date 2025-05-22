package log

import (
	"sync"

	"github.com/sirupsen/logrus"
)

// RingBuffer is a thread-safe fixed-size ring buffer for storing logrus.Entry objects.
type RingBuffer struct {
	mutex  sync.RWMutex
	buffer []*logrus.Entry
	size   int
	start  int
	count  int
}

// NewRingBuffer creates a new ring buffer with the specified capacity.
func NewRingBuffer(capacity int) *RingBuffer {
	return &RingBuffer{
		buffer: make([]*logrus.Entry, capacity),
		size:   capacity,
	}
}

// Add adds a logrus.Entry to the ring buffer.
func (rb *RingBuffer) Add(entry *logrus.Entry) {
	rb.mutex.Lock()
	defer rb.mutex.Unlock()

	// Create a copy of the entry to ensure it's not modified by callers
	entryCopy := *entry
	
	// Calculate position for the new entry
	position := (rb.start + rb.count) % rb.size
	
	// Store the entry
	rb.buffer[position] = &entryCopy
	
	// Increment count if we haven't filled the buffer yet
	if rb.count < rb.size {
		rb.count++
	} else {
		// Otherwise, move the start position
		rb.start = (rb.start + 1) % rb.size
	}
}

// GetAll returns all entries in the buffer in chronological order (oldest to newest).
func (rb *RingBuffer) GetAll() []*logrus.Entry {
	rb.mutex.RLock()
	defer rb.mutex.RUnlock()
	
	result := make([]*logrus.Entry, rb.count)
	
	for i := 0; i < rb.count; i++ {
		result[i] = rb.buffer[(rb.start+i)%rb.size]
	}
	
	return result
}

// GetCount returns the current number of entries in the buffer.
func (rb *RingBuffer) GetCount() int {
	rb.mutex.RLock()
	defer rb.mutex.RUnlock()
	return rb.count
}

// Clear removes all entries from the buffer.
func (rb *RingBuffer) Clear() {
	rb.mutex.Lock()
	defer rb.mutex.Unlock()
	rb.start = 0
	rb.count = 0
}