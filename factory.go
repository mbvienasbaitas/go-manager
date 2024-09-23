package manager

import "context"

type Factory[T any] interface {
	Build(ctx context.Context, name string) (Service[T], error)
}
