package factory

import (
	"context"
	"time"
)

type GenericService[T any] struct {
	service T
}

func (receiver *GenericService[T]) Valid(_ context.Context, _ string) bool {
	return true
}

func (receiver *GenericService[T]) Get() T {
	return receiver.service
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

func (receiver *TimedService[T]) Get() T {
	return receiver.service
}

func NewTimedGenericService[T any](service T, validUntil time.Time) *TimedService[T] {
	return &TimedService[T]{
		validUntil: validUntil,
		service:    service,
	}
}
