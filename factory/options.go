package factory

import (
	"github.com/mbvienasbaitas/go-manager"
)

type Options[T any] struct {
	evaluator manager.Evaluator[T]
	builder   manager.Builder[T]
}

type Option[T any] func(options *Options[T])

func OptionEvaluator[T any](evaluator manager.Evaluator[T]) Option[T] {
	return func(options *Options[T]) {
		options.evaluator = evaluator
	}
}

func OptionBuilder[T any](builder manager.Builder[T]) Option[T] {
	return func(options *Options[T]) {
		options.builder = builder
	}
}

func NewOptions[T any]() Options[T] {
	return Options[T]{}
}
