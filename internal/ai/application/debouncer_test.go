package application

import (
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestDebouncer_GroupsMessages(t *testing.T) {
	var mu sync.Mutex
	var gotID string
	var gotMessages []string

	d := NewDebouncer(50*time.Millisecond, func(id string, msgs []string) {
		mu.Lock()
		gotID = id
		gotMessages = msgs
		mu.Unlock()
	})

	d.Add("conv-1", "hello")
	d.Add("conv-1", "world")
	d.Add("conv-1", "!")

	time.Sleep(100 * time.Millisecond)

	mu.Lock()
	defer mu.Unlock()
	assert.Equal(t, "conv-1", gotID)
	assert.Equal(t, []string{"hello", "world", "!"}, gotMessages)
}

func TestDebouncer_SeparateConversations(t *testing.T) {
	var mu sync.Mutex
	results := make(map[string][]string)

	d := NewDebouncer(50*time.Millisecond, func(id string, msgs []string) {
		mu.Lock()
		results[id] = msgs
		mu.Unlock()
	})

	d.Add("conv-A", "msg-A1")
	d.Add("conv-B", "msg-B1")
	d.Add("conv-A", "msg-A2")

	time.Sleep(100 * time.Millisecond)

	mu.Lock()
	defer mu.Unlock()
	assert.Equal(t, []string{"msg-A1", "msg-A2"}, results["conv-A"])
	assert.Equal(t, []string{"msg-B1"}, results["conv-B"])
}

func TestDebouncer_ResetTimerOnNewMessage(t *testing.T) {
	var mu sync.Mutex
	callCount := 0

	d := NewDebouncer(60*time.Millisecond, func(_ string, _ []string) {
		mu.Lock()
		callCount++
		mu.Unlock()
	})

	// Add messages with small gaps — the timer should keep resetting.
	d.Add("conv-1", "a")
	time.Sleep(30 * time.Millisecond)
	d.Add("conv-1", "b")
	time.Sleep(30 * time.Millisecond)
	d.Add("conv-1", "c")

	// At this point 60ms should not have elapsed since last Add; callback should not have fired yet.
	mu.Lock()
	before := callCount
	mu.Unlock()
	assert.Equal(t, 0, before)

	// Wait long enough for the final timer to fire.
	time.Sleep(100 * time.Millisecond)

	mu.Lock()
	after := callCount
	mu.Unlock()
	assert.Equal(t, 1, after)
}

func TestDebouncer_Stop(t *testing.T) {
	called := false

	d := NewDebouncer(100*time.Millisecond, func(_ string, _ []string) {
		called = true
	})

	d.Add("conv-1", "msg")
	d.Stop()

	time.Sleep(150 * time.Millisecond)

	assert.False(t, called)
}
