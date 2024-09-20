package factory

import "context"

type FuncEvaluator[T any] func(ctx context.Context, name string) bool

func (receiver FuncEvaluator[T]) Supports(ctx context.Context, name string) bool {
	return receiver(ctx, name)
}

type FuncBuilder[T any] func(ctx context.Context, name string) (T, error)

func (receiver FuncBuilder[T]) Build(ctx context.Context, name string) (T, error) {
	return receiver(ctx, name)
}
