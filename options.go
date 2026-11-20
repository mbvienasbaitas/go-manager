package manager

import "sync"

type Options[T any] struct {
	lock         *sync.RWMutex
	factories    []Factory[T]
	errorHandler func(name string, err error)
}

type Option[T any] func(options *Options[T])

// OptionErrorHandler sets a callback for errors that cannot be returned to the
// caller, such as an expired service failing to shut down while Make replaces it.
// It is called with no manager locks held, so it may call Make or Forget. Without
// a handler these errors are logged with slog.Warn.
func OptionErrorHandler[T any](handler func(name string, err error)) Option[T] {
	return func(options *Options[T]) {
		options.lock.Lock()

		defer options.lock.Unlock()

		options.errorHandler = handler
	}
}

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
