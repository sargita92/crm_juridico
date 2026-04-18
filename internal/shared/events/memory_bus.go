package events

import (
	"sync"
)

const eventBufferSize = 100

type MemoryEventBus struct {
	mu                sync.RWMutex
	subscribers       map[string][]chan Event
	globalSubscribers []chan Event
}

func NewMemoryEventBus() *MemoryEventBus {
	return &MemoryEventBus{
		subscribers: make(map[string][]chan Event),
	}
}

func (b *MemoryEventBus) Publish(event Event) {
	b.mu.RLock()
	defer b.mu.RUnlock()

	if channels, ok := b.subscribers[event.TenantID]; ok {
		for _, ch := range channels {
			select {
			case ch <- event:
			default:
				// Channel full, discard event to avoid blocking
			}
		}
	}

	for _, ch := range b.globalSubscribers {
		select {
		case ch <- event:
		default:
			// Channel full, discard event to avoid blocking
		}
	}
}

func (b *MemoryEventBus) Subscribe(tenantID string) (<-chan Event, func()) {
	ch := make(chan Event, eventBufferSize)

	b.mu.Lock()
	b.subscribers[tenantID] = append(b.subscribers[tenantID], ch)
	b.mu.Unlock()

	cleanup := func() {
		b.mu.Lock()
		defer b.mu.Unlock()

		channels := b.subscribers[tenantID]
		for i, c := range channels {
			if c == ch {
				b.subscribers[tenantID] = append(channels[:i], channels[i+1:]...)
				break
			}
		}
		if len(b.subscribers[tenantID]) == 0 {
			delete(b.subscribers, tenantID)
		}
		close(ch)
	}

	return ch, cleanup
}

// SubscribeAll registers a subscriber that receives events from all tenants.
// The returned cleanup function unregisters the subscriber and closes the channel.
func (b *MemoryEventBus) SubscribeAll() (<-chan Event, func()) {
	ch := make(chan Event, eventBufferSize)

	b.mu.Lock()
	b.globalSubscribers = append(b.globalSubscribers, ch)
	b.mu.Unlock()

	cleanup := func() {
		b.mu.Lock()
		defer b.mu.Unlock()

		for i, c := range b.globalSubscribers {
			if c == ch {
				b.globalSubscribers = append(b.globalSubscribers[:i], b.globalSubscribers[i+1:]...)
				break
			}
		}
		close(ch)
	}

	return ch, cleanup
}

// Compile-time check that MemoryEventBus satisfies GlobalEventBus.
var _ GlobalEventBus = (*MemoryEventBus)(nil)
