package manager

import "sync"

type Options[T any] struct {
	lock      *sync.RWMutex
	factories []Factory[T]
}

type Option[T any] func(options *Options[T])

func OptionFactory[T any](factory Factory[T]) Option[T] {
	return func(options *Options[T]) {
		options.lock.Lock()

		defer options.lock.Unlock()

		options.factories = append(options.factories, factory)
	}
}

func OptionFactories[T any](factories []Factory[T]) Option[T] {
	return func(options *Options[T]) {
		options.lock.Lock()

		defer options.lock.Unlock()

		options.factories = factories
	}
}

func NewOptions[T any]() Options[T] {
	return Options[T]{
		lock:      &sync.RWMutex{},
		factories: make([]Factory[T], 0),
	}
}
