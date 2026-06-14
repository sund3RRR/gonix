package store

import (
	"fmt"

	"github.com/sund3RRR/gonix/internal/status"
	"github.com/sund3RRR/gonix/internal/utils"
	nix "github.com/sund3RRR/nix-go-bindings"
)

// Derivation owns a Nix derivation handle.
//
// A Derivation must be closed when the caller is done with it. The Nix context
// used to create it must remain valid for as long as it is used.
type Derivation struct {
	ctx *nix.NixCContext
	ptr *nix.NixDerivation
}

// NewDerivation wraps an owned raw Nix derivation handle.
//
// The returned Derivation takes ownership of ptr and releases it from Close.
// The ctx argument must remain valid for as long as the derivation is used.
func NewDerivation(ctx *nix.NixCContext, ptr *nix.NixDerivation) *Derivation {
	return &Derivation{
		ctx: ctx,
		ptr: ptr,
	}
}

// JSON exports d as Nix derivation JSON.
func (d *Derivation) JSON() ([]byte, error) {
	if d.ptr == nil {
		return nil, status.ErrClosed
	}

	rawJSON := nix.DerivationToJson(d.ctx, d.ptr)
	if rawJSON == nil {
		return nil, fmt.Errorf("derivation: failed to export to json: %w", status.FromContext(d.ctx))
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

	clone := nix.DerivationClone(d.ptr)
	if clone == nil {
		return nil, fmt.Errorf("derivation: failed to clone derivation: %w", status.FromContext(d.ctx))
	}

	return NewDerivation(d.ctx, clone), nil
}

// Borrow returns d's borrowed raw Nix derivation handle.
//
// Callers must not free the returned pointer and must not retain it beyond the
// immediate raw Nix call that needs it.
func (d *Derivation) Borrow() (*nix.NixDerivation, error) {
	if d.ptr == nil {
		return nil, status.ErrClosed
	}

	return d.ptr, nil
}

// Close releases the owned Nix derivation handle and is safe to call more than once.
func (d *Derivation) Close() error {
	if d.ptr == nil {
		return nil
	}

	nix.DerivationFree(d.ptr)
	d.ptr = nil
	if err := status.FromContext(d.ctx); err != nil {
		return fmt.Errorf("derivation: failed to free resource: %w", err)
	}

	return nil
}
