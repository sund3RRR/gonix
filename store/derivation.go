package store

import (
	"fmt"

	"github.com/sund3RRR/gonix/internal/status"
	"github.com/sund3RRR/gonix/internal/utils"
	"github.com/sund3RRR/gonix/nixcontext"
	nix "github.com/sund3RRR/nix-go-bindings"
)

// Derivation owns a Nix derivation handle.
//
// A derivation describes how Nix can build one or more outputs: builder,
// arguments, environment, input sources, input derivations, and output paths.
// This wrapper owns the underlying Nix derivation object, not the store path of
// a .drv file. Use Store.AddDerivation to write a derivation to a store and get
// its store path.
//
// A Derivation must be closed when the caller is done with it. The Nix context
// used to create it must remain valid for as long as it is used.
type Derivation struct {
	ctx *nixcontext.Context
	ptr *nix.NixDerivation
}

// NewDerivation wraps an owned raw Nix derivation handle.
//
// The returned Derivation takes ownership of ptr and releases it from Close.
// The ctx argument is borrowed and must remain valid for as long as the
// derivation is used. Passing a nil ptr creates a closed Derivation.
func NewDerivation(ctx *nixcontext.Context, ptr *nix.NixDerivation) *Derivation {
	return &Derivation{
		ctx: ctx,
		ptr: ptr,
	}
}

// JSON exports d as Nix derivation JSON.
//
// The returned bytes are a Go-owned copy. The Nix-allocated string is released
// before JSON returns.
func (d *Derivation) JSON() ([]byte, error) {
	if d.ptr == nil {
		return nil, status.ErrClosed
	}

	rawCtx, err := d.ctx.Borrow()
	if err != nil {
		return nil, fmt.Errorf("derivation: borrow context: %w", err)
	}

	rawJSON := nix.DerivationToJson(rawCtx, d.ptr)
	if rawJSON == nil {
		return nil, fmt.Errorf("derivation: failed to export to json: %w", status.FromContext(rawCtx))
	}

	return []byte(utils.TakeCString(rawJSON)), nil
}

// Clone returns an independently owned copy of d.
//
// The caller must close the returned derivation independently from d.
func (d *Derivation) Clone() (*Derivation, error) {
	if d.ptr == nil {
		return nil, status.ErrClosed
	}

	rawCtx, err := d.ctx.Borrow()
	if err != nil {
		return nil, fmt.Errorf("derivation: borrow context: %w", err)
	}

	clone := nix.DerivationClone(d.ptr)
	if clone == nil {
		return nil, fmt.Errorf("derivation: failed to clone derivation: %w", status.FromContext(rawCtx))
	}

	return NewDerivation(d.ctx, clone), nil
}

// Borrow returns d's borrowed raw Nix derivation handle.
//
// Callers must not free the returned pointer and must not retain it beyond the
// immediate raw Nix call that needs it. This is an escape hatch for integration
// with lower-level bindings.
func (d *Derivation) Borrow() (*nix.NixDerivation, error) {
	if d.ptr == nil {
		return nil, status.ErrClosed
	}
	if _, err := d.ctx.Borrow(); err != nil {
		return nil, err
	}

	return d.ptr, nil
}

// Close releases the owned Nix derivation handle.
//
// Close is safe to call more than once. Once Close returns, methods that need
// the raw derivation handle report status.ErrClosed.
func (d *Derivation) Close() error {
	if d == nil || d.ptr == nil {
		return nil
	}

	nix.DerivationFree(d.ptr)
	d.ptr = nil

	return nil
}
