package factory

import "errors"

var (
	ErrEvaluatorNotSet = errors.New("evaluator not set")
	ErrBuilderNotSet   = errors.New("builder not set")
)
