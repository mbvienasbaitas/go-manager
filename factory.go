package manager

import (
	"context"
)

type Factory[T any] interface {
	Make(ctx context.Context) (T, error)
}

type FuncFactory[T any] func(ctx context.Context) (T, error)

func (receiver FuncFactory[T]) Make(ctx context.Context) (T, error) {
	return receiver(ctx)
}
