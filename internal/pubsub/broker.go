package pubsub

import (
	"context"
	"sync"
)

const bufferSize = 64

// Broker is a generic publish-subscribe event broker
type Broker[T any] struct {
	subs map[chan Event[T]]struct{}
	mu   sync.RWMutex
	done chan struct{}
}

// NewBroker creates a new event broker
func NewBroker[T any]() *Broker[T] {
	return &Broker[T]{
		subs: make(map[chan Event[T]]struct{}),
		done: make(chan struct{}),
	}
}

// Subscribe returns a channel that receives events.
// The channel is closed when the context is cancelled or the broker shuts down.
func (b *Broker[T]) Subscribe(ctx context.Context) <-chan Event[T] {
	ch := make(chan Event[T], bufferSize)

	b.mu.Lock()
	b.subs[ch] = struct{}{}
	b.mu.Unlock()

	go func() {
		select {
		case <-ctx.Done():
		case <-b.done:
		}
		b.mu.Lock()
		delete(b.subs, ch)
		b.mu.Unlock()
		close(ch)
	}()

	return ch
}

// Publish sends an event to all subscribers. Non-blocking: drops if buffer full.
func (b *Broker[T]) Publish(t EventType, payload T) {
	evt := Event[T]{Type: t, Payload: payload}

	b.mu.RLock()
	defer b.mu.RUnlock()

	for ch := range b.subs {
		select {
		case ch <- evt:
		default:
			// Drop if subscriber can't keep up
		}
	}
}

// Shutdown closes the broker and all subscriber channels
func (b *Broker[T]) Shutdown() {
	close(b.done)
}
