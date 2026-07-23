package ws

import "errors"

var (
	ErrSessionNotFound      = errors.New("session not found")
	ErrComponentRequired    = errors.New("component query parameter is required")
	ErrSessionNotConnected  = errors.New("session is not connected")
	ErrSessionAlreadyActive = errors.New("session already has an active connection")
	ErrServerNotConfigured  = errors.New("server is not configured")
)
