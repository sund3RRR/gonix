// Package fetchers wraps Nix fetcher settings.
package fetchers

import (
	"fmt"

	"github.com/sund3RRR/gonix/internal/status"
	"github.com/sund3RRR/gonix/nixcontext"
	"github.com/sund3RRR/gonix/pkg/raw"
)

// Settings owns a Nix fetcher settings handle.
//
// Settings borrows the Nix context used to create it. The context must outlive
// the Settings. Close is idempotent, but other methods return an error wrapping
// gonix.ErrClosed after Close.
type Settings struct {
	ctx *nixcontext.Context
	ptr *raw.NixFetchersSettings
}

// NewSettings creates fetcher settings using an initialized Nix context.
//
// The returned Settings owns the raw Nix fetcher settings handle and must be
// closed by the caller. The ctx argument is borrowed and must remain valid for
// as long as the Settings is used.
func NewSettings(ctx *nixcontext.Context) (*Settings, error) {
	rawCtx, err := ctx.Borrow()
	if err != nil {
		return nil, fmt.Errorf("fetchers: failed to borrow context: %w", err)
	}

	ptr := raw.FetchersSettingsNew(rawCtx)
	if ptr == nil {
		return nil, fmt.Errorf("fetchers: failed to create settings: %w", status.FromContext(rawCtx))
	}

	return &Settings{
		ctx: ctx,
		ptr: ptr,
	}, nil
}

// Borrow returns the borrowed raw Nix fetcher settings handle.
//
// Callers must not free the returned pointer and must not retain it beyond the
// immediate raw Nix call that needs it. This is an escape hatch for integration
// with lower-level bindings.
func (s *Settings) Borrow() (*raw.NixFetchersSettings, error) {
	if s.ptr == nil {
		return nil, status.ErrClosed
	}

	return s.ptr, nil
}

// Close releases the owned Nix fetcher settings handle.
//
// Close is safe to call more than once. Once Close returns, methods that need
// the raw settings handle report status.ErrClosed.
func (s *Settings) Close() error {
	if s == nil || s.ptr == nil {
		return nil
	}

	raw.FetchersSettingsFree(s.ptr)
	s.ptr = nil

	return nil
}
