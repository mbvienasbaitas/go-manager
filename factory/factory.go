package factory

import (
	"context"
)

type Factory[T any] struct {
	opts Options[T]
}

func New[T any](opts ...Option[T]) (*Factory[T], error) {
	options := NewOptions[T]()

	for _, o := range opts {
		o(&options)
	}

	if options.evaluator == nil {
		return nil, ErrEvaluatorNotSet
	}

	if options.builder == nil {
		return nil, ErrBuilderNotSet
	}

	return &Factory[T]{
		opts: options,
	}, nil
}

func (f *Factory[T]) Supports(ctx context.Context, name string) bool {
	return f.opts.evaluator.Supports(ctx, name)
}

func (f *Factory[T]) Build(ctx context.Context, name string) (T, error) {
	return f.opts.builder.Build(ctx, name)
}
