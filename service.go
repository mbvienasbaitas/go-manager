package manager

import "context"

type Service[T any] interface {
	Valid(ctx context.Context, name string) bool
	Get() T
}
