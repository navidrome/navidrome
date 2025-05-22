package log

import (
	"sync"

	"github.com/sirupsen/logrus"
)

const defaultLogBufferCapacity = 1000

var (
	logBuffer     *RingBuffer
	logBufferOnce sync.Once
	logListeners  = make(map[chan *logrus.Entry]struct{})
	listenerMutex sync.RWMutex
)

// bufferHook is a logrus hook that stores log entries in a ring buffer
// and broadcasts them to any active listeners
type bufferHook struct {
}

// Levels returns all log levels, to capture everything
func (h *bufferHook) Levels() []logrus.Level {
	return logrus.AllLevels
}

// Fire is called when a log event occurs
func (h *bufferHook) Fire(entry *logrus.Entry) error {
	// Initialize buffer if not already done
	logBufferOnce.Do(func() {
		logBuffer = NewRingBuffer(defaultLogBufferCapacity)
	})

	// Add to buffer
	logBuffer.Add(entry)

	// Broadcast to listeners
	listenerMutex.RLock()
	defer listenerMutex.RUnlock()
	
	if len(logListeners) > 0 {
		for ch := range logListeners {
			select {
			case ch <- entry:
				// Log sent successfully
			default:
				// Channel is full, skip this entry for this listener
			}
		}
	}
	
	return nil
}

// GetLogBuffer returns the global log buffer instance
func GetLogBuffer() *RingBuffer {
	logBufferOnce.Do(func() {
		logBuffer = NewRingBuffer(defaultLogBufferCapacity)
	})
	return logBuffer
}

// RegisterLogListener adds a new channel to receive log entries
func RegisterLogListener(ch chan *logrus.Entry) {
	listenerMutex.Lock()
	defer listenerMutex.Unlock()
	logListeners[ch] = struct{}{}
}

// UnregisterLogListener removes a channel from receiving log entries
func UnregisterLogListener(ch chan *logrus.Entry) {
	listenerMutex.Lock()
	defer listenerMutex.Unlock()
	delete(logListeners, ch)
}

// EnableLogBuffer adds the buffer hook to the default logger
func EnableLogBuffer() {
	logBufferOnce.Do(func() {
		logBuffer = NewRingBuffer(defaultLogBufferCapacity)
	})
	
	// Add the hook to the default logger
	defaultLogger.AddHook(&bufferHook{})
}