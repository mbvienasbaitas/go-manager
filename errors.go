package manager

import "errors"

var (
	ErrFactoryNotSet      = errors.New("factory not set")
	ErrServiceInvalidated = errors.New("service invalided")
	ErrServiceUnsupported = errors.New("service unsupported")
)
