package observable

import (
	"errors"
	"sync"
)

var (
	errObservableClosed = errors.New("observable is closed")
)

type Observable[T any] struct {
	iterable Iterable[T]
	// listener is a mapping of Subscriptions to Subscribers
	listener map[Subscription[T]]*Subscriber[T]
	mux      sync.Mutex
	done     bool
}

// NewObservable creates a new Observable[T]
func NewObservable[T any](iter Iterable[T]) *Observable[T] {
	observable := &Observable[T]{
		iterable: iter,
		listener: map[Subscription[T]]*Subscriber[T]{},
	}
	go observable.process()
	return observable
}

func (o *Observable[T]) process() {
	for item := range o.iterable {
		o.mux.Lock()
		for _, sub := range o.listener {
			sub.Emit(item)
		}
		o.mux.Unlock()
	}
	o.close()
}

func (o *Observable[T]) close() {
	o.mux.Lock()
	defer o.mux.Unlock()

	o.done = true
	for _, sub := range o.listener {
		sub.Close()
	}
}

// Subscribe creates a new Subscriber and adds it to the mapping of subscriptions
func (o *Observable[T]) Subscribe() (Subscription[T], error) {
	o.mux.Lock()
	defer o.mux.Unlock()
	if o.done {
		return nil, errObservableClosed
	}
	subscriber := newSubscriber[T]()
	o.listener[subscriber.Out()] = subscriber
	return subscriber.Out(), nil
}

// UnSubscribe removes the subscriber associated with the given Subscription
func (o *Observable[T]) UnSubscribe(sub Subscription[T]) {
	o.mux.Lock()
	defer o.mux.Unlock()
	subscriber, exist := o.listener[sub]
	if !exist {
		return
	}
	delete(o.listener, sub)
	subscriber.Close()
}
