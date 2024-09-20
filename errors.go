package manager

import "errors"

var (
	ErrOptionFactoriesNotSet = errors.New("factories not set")
	ErrFactoryNotSet         = errors.New("factory not set")
)
