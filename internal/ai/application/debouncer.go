package application

import (
	"sync"
	"time"
)

// DebouncerCallback is called when a debounce window closes for a conversation.
type DebouncerCallback func(conversationID string, messages []string)

// Debouncer groups rapid successive messages within a time window before
// invoking the callback.
type Debouncer struct {
	duration time.Duration
	callback DebouncerCallback
	timers   map[string]*time.Timer
	messages map[string][]string
	mu       sync.Mutex
}

// NewDebouncer creates a Debouncer with the given window duration and callback.
func NewDebouncer(duration time.Duration, callback DebouncerCallback) *Debouncer {
	return &Debouncer{
		duration: duration,
		callback: callback,
		timers:   make(map[string]*time.Timer),
		messages: make(map[string][]string),
	}
}

// Add appends a message for the given conversationID and resets the debounce timer.
func (d *Debouncer) Add(conversationID, message string) {
	d.mu.Lock()
	d.messages[conversationID] = append(d.messages[conversationID], message)

	if t, ok := d.timers[conversationID]; ok {
		t.Stop()
	}

	t := time.AfterFunc(d.duration, func() {
		d.fire(conversationID)
	})
	d.timers[conversationID] = t
	d.mu.Unlock()
}

// fire extracts and clears accumulated messages for the conversation, then
// calls the callback outside the lock.
func (d *Debouncer) fire(conversationID string) {
	d.mu.Lock()
	msgs := d.messages[conversationID]
	delete(d.messages, conversationID)
	delete(d.timers, conversationID)
	d.mu.Unlock()

	if len(msgs) > 0 {
		d.callback(conversationID, msgs)
	}
}

// Stop cancels all pending timers.
func (d *Debouncer) Stop() {
	d.mu.Lock()
	defer d.mu.Unlock()

	for id, t := range d.timers {
		t.Stop()
		delete(d.timers, id)
	}
}
