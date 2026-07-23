package core

import "errors"

var (
	ErrComponentNotRegistered     = errors.New("component not registered")
	ErrComponentAlreadyRegistered = errors.New("component already registered")
)
