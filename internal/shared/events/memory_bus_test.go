package events

import (
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMemoryEventBus_PublishAndSubscribe(t *testing.T) {
	bus := NewMemoryEventBus()
	ch, cleanup := bus.Subscribe("tenant-1")
	defer cleanup()

	event := Event{
		Type:     EventNewMessage,
		TenantID: "tenant-1",
		Payload:  "hello",
	}

	bus.Publish(event)

	select {
	case received := <-ch:
		assert.Equal(t, EventNewMessage, received.Type)
		assert.Equal(t, "tenant-1", received.TenantID)
		assert.Equal(t, "hello", received.Payload)
	case <-time.After(time.Second):
		t.Fatal("timeout waiting for event")
	}
}

func TestMemoryEventBus_IsolatedByTenant(t *testing.T) {
	bus := NewMemoryEventBus()
	chA, cleanupA := bus.Subscribe("tenant-a")
	defer cleanupA()
	chB, cleanupB := bus.Subscribe("tenant-b")
	defer cleanupB()

	bus.Publish(Event{
		Type:     EventNewMessage,
		TenantID: "tenant-a",
		Payload:  "for A",
	})

	select {
	case received := <-chA:
		assert.Equal(t, "for A", received.Payload)
	case <-time.After(time.Second):
		t.Fatal("timeout waiting for event on tenant-a")
	}

	select {
	case <-chB:
		t.Fatal("tenant-b should not receive tenant-a events")
	case <-time.After(50 * time.Millisecond):
		// Expected: no event for tenant-b
	}
}

func TestMemoryEventBus_MultipleSubscribers(t *testing.T) {
	bus := NewMemoryEventBus()
	ch1, cleanup1 := bus.Subscribe("tenant-1")
	defer cleanup1()
	ch2, cleanup2 := bus.Subscribe("tenant-1")
	defer cleanup2()

	bus.Publish(Event{
		Type:     EventNewMessage,
		TenantID: "tenant-1",
		Payload:  "broadcast",
	})

	for _, ch := range []<-chan Event{ch1, ch2} {
		select {
		case received := <-ch:
			assert.Equal(t, "broadcast", received.Payload)
		case <-time.After(time.Second):
			t.Fatal("timeout waiting for event")
		}
	}
}

func TestMemoryEventBus_Cleanup_RemovesSubscriber(t *testing.T) {
	bus := NewMemoryEventBus()
	ch, cleanup := bus.Subscribe("tenant-1")
	cleanup()

	bus.Publish(Event{
		Type:     EventNewMessage,
		TenantID: "tenant-1",
		Payload:  "after cleanup",
	})

	// Channel is closed, should not block
	_, ok := <-ch
	assert.False(t, ok, "channel should be closed after cleanup")
}

func TestMemoryEventBus_Cleanup_RemovesTenantKeyWhenEmpty(t *testing.T) {
	bus := NewMemoryEventBus()
	_, cleanup := bus.Subscribe("tenant-1")
	cleanup()

	bus.mu.RLock()
	_, exists := bus.subscribers["tenant-1"]
	bus.mu.RUnlock()

	assert.False(t, exists, "tenant key should be removed when no subscribers")
}

func TestMemoryEventBus_PublishNoSubscribers_NoPanic(t *testing.T) {
	bus := NewMemoryEventBus()

	assert.NotPanics(t, func() {
		bus.Publish(Event{
			Type:     EventNewMessage,
			TenantID: "no-one",
			Payload:  "void",
		})
	})
}

func TestMemoryEventBus_FullBuffer_DropsEvent(t *testing.T) {
	bus := NewMemoryEventBus()
	ch, cleanup := bus.Subscribe("tenant-1")
	defer cleanup()

	// Fill the buffer
	for i := 0; i < eventBufferSize; i++ {
		bus.Publish(Event{
			Type:     EventNewMessage,
			TenantID: "tenant-1",
			Payload:  i,
		})
	}

	// This should not block (event dropped)
	done := make(chan struct{})
	go func() {
		bus.Publish(Event{
			Type:     EventNewMessage,
			TenantID: "tenant-1",
			Payload:  "overflow",
		})
		close(done)
	}()

	select {
	case <-done:
		// Expected: publish did not block
	case <-time.After(time.Second):
		t.Fatal("publish blocked on full buffer")
	}

	// Drain and verify we got exactly eventBufferSize events
	count := 0
	for range ch {
		count++
		if count == eventBufferSize {
			break
		}
	}
	assert.Equal(t, eventBufferSize, count)
}

func TestMemoryEventBus_ConcurrentPublishAndSubscribe(t *testing.T) {
	bus := NewMemoryEventBus()
	var wg sync.WaitGroup

	// Concurrent subscribers
	channels := make([]<-chan Event, 5)
	cleanups := make([]func(), 5)
	for i := 0; i < 5; i++ {
		channels[i], cleanups[i] = bus.Subscribe("tenant-1")
		defer cleanups[i]()
	}

	// Concurrent publishers
	eventCount := 10
	wg.Add(eventCount)
	for i := 0; i < eventCount; i++ {
		go func(n int) {
			defer wg.Done()
			bus.Publish(Event{
				Type:     EventNewMessage,
				TenantID: "tenant-1",
				Payload:  n,
			})
		}(i)
	}

	wg.Wait()

	// Each subscriber should have received all events
	for _, ch := range channels {
		received := 0
		timeout := time.After(time.Second)
	drain:
		for {
			select {
			case <-ch:
				received++
				if received == eventCount {
					break drain
				}
			case <-timeout:
				break drain
			}
		}
		assert.Equal(t, eventCount, received)
	}
}

func TestMemoryEventBus_EventTypes(t *testing.T) {
	bus := NewMemoryEventBus()
	ch, cleanup := bus.Subscribe("tenant-1")
	defer cleanup()

	bus.Publish(Event{Type: EventNewMessage, TenantID: "tenant-1"})
	bus.Publish(Event{Type: EventConversationUpdate, TenantID: "tenant-1"})

	ev1 := <-ch
	assert.Equal(t, EventNewMessage, ev1.Type)

	ev2 := <-ch
	assert.Equal(t, EventConversationUpdate, ev2.Type)
}

func TestMemoryEventBus_CleanupOneOfMany(t *testing.T) {
	bus := NewMemoryEventBus()
	ch1, cleanup1 := bus.Subscribe("tenant-1")
	_, cleanup2 := bus.Subscribe("tenant-1")
	defer cleanup2()

	// Remove first subscriber
	cleanup1()

	// Second subscriber should still work
	bus.Publish(Event{
		Type:     EventNewMessage,
		TenantID: "tenant-1",
		Payload:  "still here",
	})

	// ch1 closed
	_, ok := <-ch1
	assert.False(t, ok)

	bus.mu.RLock()
	require.Len(t, bus.subscribers["tenant-1"], 1)
	bus.mu.RUnlock()
}
