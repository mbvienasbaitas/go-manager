package manager

import "sync"

type Options[T any] struct {
	lock      *sync.RWMutex
	factories map[string]Factory[T]
}

type Option[T any] func(options *Options[T])

func OptionFactory[T any](name string, factory Factory[T]) Option[T] {
	return func(options *Options[T]) {
		options.lock.Lock()

		defer options.lock.Unlock()

		options.factories[name] = factory
	}
}

func OptionFactories[T any](factories map[string]Factory[T]) Option[T] {
	return func(options *Options[T]) {
		options.lock.Lock()

		defer options.lock.Unlock()

		options.factories = factories
	}
}

func NewOptions[T any]() Options[T] {
	return Options[T]{
		lock:      &sync.RWMutex{},
		factories: make(map[string]Factory[T]),
	}
}
