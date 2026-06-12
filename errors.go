package gonix

import (
	"errors"
	"fmt"
)

var (
	// ErrClosed is returned when an operation is attempted after a wrapper was closed.
	ErrClosed = errors.New("resource is closed")
)

// Error is a Go-native copy of the current Nix context error.
type Error struct {
	Code    ErrorCode
	Name    string
	Message string
	Info    string
}

func (e Error) Error() string {
	return fmt.Sprintf("%s (%d): name=%s message=%s info=%s", e.Code, e.Code, e.Name, e.Message, e.Info)
}
