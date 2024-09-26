package manager

import (
	"context"
	"time"
)

type Shutdownable interface {
	Shutdown() error
}

type Service[T any] interface {
	GetService() (T, error)
}

type GenericService[T any] struct {
	service T
}

func (receiver *GenericService[T]) GetService() (T, error) {
	return receiver.service, nil
}

func (receiver *GenericService[T]) Shutdown() error {
	return shutdown(receiver.service)
}

func NewGenericService[T any](service T) *GenericService[T] {
	return &GenericService[T]{
		service: service,
	}
}

type TimedService[T any] struct {
	validUntil time.Time
	service    T
}

func (receiver *TimedService[T]) Valid(_ context.Context, _ string) bool {
	return time.Now().Before(receiver.validUntil)
}

func (receiver *TimedService[T]) GetService() (T, error) {
	if time.Now().Before(receiver.validUntil) {
		return receiver.service, nil
	}

	return receiver.service, ErrServiceExpired
}

func (receiver *TimedService[T]) Shutdown() error {
	return shutdown(receiver.service)
}

func NewTimedService[T any](service T, validUntil time.Time) *TimedService[T] {
	return &TimedService[T]{
		validUntil: validUntil,
		service:    service,
	}
}
