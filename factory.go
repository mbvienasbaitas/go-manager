package manager

import (
	"context"
)

type Evaluator[T any] interface {
	Supports(ctx context.Context, name string) bool
}

type Builder[T any] interface {
	Build(ctx context.Context, name string) (Service[T], error)
}

type Factory[T any] interface {
	Evaluator[T]
	Builder[T]
}
