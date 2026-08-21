package application

import "sync"

// keyedMutex serializes work per key while leaving different keys fully
// parallel. Entries are refcounted and evicted once idle, so the map tracks
// in-flight keys only instead of growing with every key ever seen.
type keyedMutex struct {
	mu    sync.Mutex
	locks map[string]*keyedLock
}

type keyedLock struct {
	mu   sync.Mutex
	refs int
}

func newKeyedMutex() *keyedMutex {
	return &keyedMutex{locks: make(map[string]*keyedLock)}
}

// Lock blocks until key is free and returns the function that releases it.
func (k *keyedMutex) Lock(key string) func() {
	k.mu.Lock()
	l, ok := k.locks[key]
	if !ok {
		l = &keyedLock{}
		k.locks[key] = l
	}
	// Held while waiting, so the entry cannot be evicted out from under a
	// waiter and split the queue across two different mutexes.
	l.refs++
	k.mu.Unlock()

	l.mu.Lock()

	return func() {
		l.mu.Unlock()

		k.mu.Lock()
		l.refs--
		if l.refs == 0 {
			delete(k.locks, key)
		}
		k.mu.Unlock()
	}
}
