package observable

import (
	"sync"
)

// Subscription is a channel of type T
type Subscription[T any] <-chan T

// Subscriber represents a channel buffer that events may be emitted to
type Subscriber[T any] struct {
	buffer chan T
	once   sync.Once
}

// newSubscriber creates a new instance of Subscriber
func newSubscriber[T any]() *Subscriber[T] {
	sub := &Subscriber[T]{
		buffer: make(chan T, 200),
	}
	return sub
}

func (s *Subscriber[T]) Emit(item T) {
	s.buffer <- item
}

func (s *Subscriber[T]) Out() Subscription[T] {
	return s.buffer
}

func (s *Subscriber[T]) Close() {
	s.once.Do(func() {
		close(s.buffer)
	})
}
