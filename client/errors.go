package client

import (
	"errors"
	"fmt"
)

// ErrConnectionFailed is an error raised when the connection between the client and the server failed.
var ErrConnectionFailed = errors.New("Cannot connect to the Providence server.")

// ErrorConnectionFailed returns an error with host in the error message when connection to server failed.
func ErrorConnectionFailed(host string) error {
	return fmt.Errorf("Cannot connect to the server at %s.", host)
}

type notFound interface {
	error
	NotFound() bool // Is the error a NotFound error
}

// IsErrNotFound returns true if the error is caused with an
// object (engine, network, …) is not found in the Providence host.
func IsErrNotFound(err error) bool {
	te, ok := err.(notFound)
	return ok && te.NotFound()
}

// engineNotFoundError implements an error returned when an engine is not in the Providence host.
type engineNotFoundError struct {
	engineID string
}

// NotFound indicates that this error type is of NotFound
func (e engineNotFoundError) NotFound() bool {
	return true
}

// Error returns a string representation of an engineNotFoundError
func (e engineNotFoundError) Error() string {
	return fmt.Sprintf("Error: No such engine: %s", e.engineID)
}

// IsErrEngineNotFound returns true if the error is caused
// when an engine is not found in the Providence host.
func IsErrEngineNotFound(err error) bool {
	return IsErrNotFound(err)
}
