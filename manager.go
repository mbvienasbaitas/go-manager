package manager

import (
	"context"
	"errors"
	"sync"
)

type Manager[T any] struct {
	lock     *sync.RWMutex
	opts     Options[T]
	registry map[string]Service[T]
}

func (receiver *Manager[T]) Make(ctx context.Context, name string) (T, error) {
	receiver.lock.RLock()

	existing, ok := receiver.registry[name]

	if ok {
		svc, err := existing.GetService()

		if err != nil {
			if errors.Is(err, ErrServiceExpired) {
				receiver.lock.RUnlock()

				err = receiver.Forget(name)

				if err != nil {
					return *new(T), err
				}

				return receiver.makeAndBind(ctx, name)
			}

			return svc, err
		}

		receiver.lock.RUnlock()

		return svc, nil
	}

	receiver.lock.RUnlock()

	return receiver.makeAndBind(ctx, name)
}

func (receiver *Manager[T]) Forget(name string) error {
	receiver.lock.Lock()

	defer receiver.lock.Unlock()

	svc, ok := receiver.registry[name]

	if ok {
		err := shutdown(svc)

		if err != nil {
			return err
		}
	}

	delete(receiver.registry, name)

	return nil
}

func (receiver *Manager[T]) Options(opts ...Option[T]) *Manager[T] {
	for _, o := range opts {
		o(&receiver.opts)
	}

	return receiver
}

func (receiver *Manager[T]) Shutdown() error {
	receiver.lock.Lock()

	defer receiver.lock.Unlock()

	for _, svc := range receiver.registry {
		if err := shutdown(svc); err != nil {
			return err
		}
	}

	return nil
}

func (receiver *Manager[T]) makeAndBind(ctx context.Context, name string) (T, error) {
	receiver.lock.Lock()

	receiver.opts.lock.RLock()

	defer receiver.lock.Unlock()

	defer receiver.opts.lock.RUnlock()

	for _, factory := range receiver.opts.factories {
		built, err := factory.Build(ctx, name)

		if err != nil {
			if errors.Is(err, ErrServiceUnsupported) {
				continue
			}

			return *new(T), err
		}

		err = boot(built)

		if err != nil {
			return *new(T), err
		}

		receiver.registry[name] = built

		return built.GetService()
	}

	return *new(T), ErrFactoryNotSet
}

func New[T any](opts ...Option[T]) (*Manager[T], error) {
	options := NewOptions[T]()

	for _, o := range opts {
		o(&options)
	}

	return &Manager[T]{
		lock:     &sync.RWMutex{},
		opts:     options,
		registry: make(map[string]Service[T]),
	}, nil
}
