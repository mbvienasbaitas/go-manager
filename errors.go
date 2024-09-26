package manager

import "errors"

var (
	ErrFactoryNotSet      = errors.New("factory not set")
	ErrServiceExpired     = errors.New("service expired")
	ErrServiceUnsupported = errors.New("service unsupported")
)
