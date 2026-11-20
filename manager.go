package manager

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"sync"
)

type Manager[T any] struct {
	lock     *sync.RWMutex
	opts     Options[T]
	registry map[string]Service[T]
}

// Make returns the service registered under name, building it with the
// configured factories if it is missing or expired.
//
// Builds run under the manager's write lock, so while one is in progress every
// other Make, Forget and Shutdown waits for it.
//
// A value Make has returned is not protected from a later Forget or Shutdown:
// those may shut the underlying service down while a caller still holds the
// value. Coordinate Forget and Shutdown with your own users.
func (receiver *Manager[T]) Make(ctx context.Context, name string) (T, error) {
	receiver.lock.RLock()

	existing, ok := receiver.registry[name]

	if ok {
		svc, err := existing.GetService()

		receiver.lock.RUnlock()

		if !errors.Is(err, ErrServiceExpired) {
			return svc, err
		}

		return receiver.makeAndBind(ctx, name)
	}

	receiver.lock.RUnlock()

	return receiver.makeAndBind(ctx, name)
}

// Forget removes the service from the registry and shuts it down. The service
// is removed even when its shutdown fails, so the next Make builds a new one.
func (receiver *Manager[T]) Forget(name string) error {
	receiver.lock.Lock()

	defer receiver.lock.Unlock()

	svc, ok := receiver.registry[name]

	if !ok {
		return nil
	}

	delete(receiver.registry, name)

	return shutdown(svc)
}

func (receiver *Manager[T]) Options(opts ...Option[T]) *Manager[T] {
	for _, o := range opts {
		o(&receiver.opts)
	}

	return receiver
}

// Shutdown shuts down every registered service and empties the registry. Every
// service is shut down even if some fail; the failures are joined together.
// The manager stays usable: a later Make builds a new service.
func (receiver *Manager[T]) Shutdown() error {
	receiver.lock.Lock()

	defer receiver.lock.Unlock()

	var errs []error

	for name, svc := range receiver.registry {
		if err := shutdown(svc); err != nil {
			errs = append(errs, fmt.Errorf("%s: %w", name, err))
		}
	}

	receiver.registry = make(map[string]Service[T])

	return errors.Join(errs...)
}

func (receiver *Manager[T]) makeAndBind(ctx context.Context, name string) (T, error) {
	var staleErr error

	svc, err := func() (T, error) {
		receiver.lock.Lock()

		defer receiver.lock.Unlock()

		return receiver.bind(ctx, name, &staleErr)
	}()

	// Reported only after the lock is released, so the handler may call Make or
	// Forget.
	if staleErr != nil {
		receiver.reportError(name, staleErr)
	}

	return svc, err
}

// bind builds and registers the service for name. The caller must hold the
// write lock. A failure to shut down the expired service being replaced does
// not fail the call; it is stored in staleErr for the caller to report.
func (receiver *Manager[T]) bind(ctx context.Context, name string, staleErr *error) (T, error) {
	// Another caller may have (re)built the service while this one waited for the
	// write lock; reuse it instead of building again.
	if existing, ok := receiver.registry[name]; ok {
		svc, err := existing.GetService()

		if !errors.Is(err, ErrServiceExpired) {
			return svc, err
		}

		delete(receiver.registry, name)

		*staleErr = shutdown(existing)
	}

	receiver.opts.lock.RLock()

	factories := receiver.opts.factories

	receiver.opts.lock.RUnlock()

	for _, factory := range factories {
		built, err := factory.Build(ctx, name)

		if err != nil {
			if errors.Is(err, ErrServiceUnsupported) {
				continue
			}

			return *new(T), err
		}

		// Shutdown is called even when Boot fails, so it can release what Build
		// opened; Shutdown implementations must cope with a failed Boot.
		if err = boot(built); err != nil {
			return *new(T), errors.Join(err, shutdown(built))
		}

		// A service that is unusable straight after boot (for example already
		// expired) is a failed build: registering it would only make every later
		// Make fail or rebuild the same way.
		svc, err := built.GetService()

		if err != nil {
			return *new(T), errors.Join(err, shutdown(built))
		}

		receiver.registry[name] = built

		return svc, nil
	}

	return *new(T), ErrFactoryNotSet
}

func (receiver *Manager[T]) reportError(name string, err error) {
	receiver.opts.lock.RLock()

	handler := receiver.opts.errorHandler

	receiver.opts.lock.RUnlock()

	if handler != nil {
		handler(name, err)

		return
	}

	slog.Warn("manager: expired service failed to shut down", "name", name, "error", err)
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
