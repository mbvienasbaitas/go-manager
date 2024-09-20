package manager

import (
	"context"
	"sync"
)

type Manager[T any] struct {
	lock  *sync.RWMutex
	opts  Options[T]
	built map[string]T
}

func (receiver *Manager[T]) Make(ctx context.Context, name string) (T, error) {
	receiver.lock.RLock()

	svc, ok := receiver.built[name]

	if ok {
		receiver.lock.RUnlock()

		return svc, nil
	}

	receiver.lock.RUnlock()

	return receiver.makeAndBind(ctx, name)
}

func (receiver *Manager[T]) Options(opts ...Option[T]) *Manager[T] {
	for _, o := range opts {
		o(&receiver.opts)
	}

	return receiver
}

func (receiver *Manager[T]) makeAndBind(ctx context.Context, name string) (T, error) {
	receiver.lock.Lock()

	receiver.opts.lock.RLock()

	defer receiver.lock.Unlock()

	defer receiver.opts.lock.RUnlock()

	for _, factory := range receiver.opts.factories {
		if factory.Supports(ctx, name) {
			built, err := factory.Build(ctx, name)

			if err != nil {
				return *new(T), err
			}

			receiver.built[name] = built

			return built, nil
		}
	}

	return *new(T), ErrFactoryNotSet
}

func New[T any](opts ...Option[T]) (*Manager[T], error) {
	options := NewOptions[T]()

	for _, o := range opts {
		o(&options)
	}

	if options.factories == nil {
		return nil, ErrOptionFactoriesNotSet
	}

	return &Manager[T]{
		lock:  &sync.RWMutex{},
		opts:  options,
		built: make(map[string]T),
	}, nil
}
