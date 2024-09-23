package manager

import (
	"context"
)

type FuncFactory[T any] func(ctx context.Context, name string) (Service[T], error)

func (receiver FuncFactory[T]) Build(ctx context.Context, name string) (Service[T], error) {
	return receiver(ctx, name)
}
