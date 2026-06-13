package status

import (
	"errors"
	"fmt"

	"github.com/sund3RRR/gonix/internal/utils"
	raw "github.com/sund3RRR/nix-go-bindings"
)

var (
	// ErrClosed is returned when an operation is attempted after a wrapper has been closed.
	//
	// Close methods in gonix are expected to be idempotent, but using a wrapper
	// after Close is a caller error and should surface as ErrClosed instead of
	// reaching the raw Nix resource again.
	ErrClosed = errors.New("resource is closed")
)

// NixError is a Go-native snapshot of an error stored in a Nix C context.
//
// Nix contexts are mutable error carriers. NewNixError copies the interesting
// fields into this struct before clearing the context, so a NixError remains
// valid after the underlying raw context is reused or freed.
type NixError struct {
	// Code is the Nix C API status code captured from the context.
	Code ErrorCode

	// Message is the primary human-readable error message from the context.
	Message string

	// Name is the upstream Nix exception name, when the context contains one.
	//
	// This is distinct from Code.String. For non-ErrorCodeNix errors it is empty.
	Name string

	// Info is the detailed upstream Nix error report, when one is available.
	//
	// Nix only exposes detailed info for structured Nix exceptions. It is empty
	// for ordinary C API errors and may also be empty for ErrorCodeNix when the
	// code was set without a backing Nix exception object.
	Info string
}

// Error formats e as a Go error string.
//
// Structured Nix details are included only for ErrorCodeNix so ordinary C API
// failures do not grow empty name/info fields in user-facing messages.
func (e NixError) Error() string {
	if e.Code == ErrorCodeNix {
		return fmt.Sprintf("%s (%d): message=%s name=%s info=%s", e.Code, e.Code, e.Message, e.Name, e.Info)
	}
	return fmt.Sprintf("%s (%d): message=%s", e.Code, e.Code, e.Message)
}

// NewNixError copies the pending error from ctx into a NixError and clears ctx.
//
// The returned error does not own or borrow ctx. Passing nil, or a context with
// ErrorCodeOK, returns nil. A non-OK context is cleared after its error data has
// been copied.
//
// The Nix C API functions that read structured error details accept two
// contexts: one context to report failures that happen while reading, and one
// context to inspect. NewNixError uses a temporary background context for the
// former so it does not overwrite the original error before it has been copied.
//
// For ErrorCodeNix, Name and Info are read only when the context appears to
// contain a structured Nix exception. Some low-level code can set the
// NIX_ERR_NIX_ERROR code without attaching detailed exception data, and asking
// Nix for missing detailed info may be unsafe.
func NewNixError(ctx *raw.NixCContext) *NixError {
	if ctx == nil {
		return nil
	}

	code := ErrorCode(raw.ErrCode(ctx))
	if code == ErrorCodeOK {
		return nil
	}

	backgroundCtx := raw.CContextCreate()
	defer raw.CContextFree(backgroundCtx)

	message := utils.TakeCString(raw.ErrMsg(backgroundCtx, ctx))

	var name, info string
	if code == ErrorCodeNix {
		name = utils.TakeCString(raw.ErrName(backgroundCtx, ctx))
		raw.ClearErr(backgroundCtx)
		if name != "" {
			info = utils.TakeCString(raw.ErrInfoMsg(backgroundCtx, ctx))
			raw.ClearErr(backgroundCtx)
		}
	}

	raw.ClearErr(ctx)

	return &NixError{
		Code:    code,
		Name:    name,
		Message: message,
		Info:    info,
	}
}
