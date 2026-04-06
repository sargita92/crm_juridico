package infrastructure

import (
	"sync"

	"github.com/sasrgita/crm-juridico/internal/whatsapp/domain"
)

const eventBufferSize = 100

type MemoryEventBus struct {
	mu          sync.RWMutex
	subscribers map[string][]chan domain.Event
}

func NewMemoryEventBus() *MemoryEventBus {
	return &MemoryEventBus{
		subscribers: make(map[string][]chan domain.Event),
	}
}

func (b *MemoryEventBus) Publish(event domain.Event) {
	b.mu.RLock()
	defer b.mu.RUnlock()

	channels, ok := b.subscribers[event.TenantID]
	if !ok {
		return
	}

	for _, ch := range channels {
		select {
		case ch <- event:
		default:
			// Channel full, discard event to avoid blocking
		}
	}
}

func (b *MemoryEventBus) Subscribe(tenantID string) (<-chan domain.Event, func()) {
	ch := make(chan domain.Event, eventBufferSize)

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
