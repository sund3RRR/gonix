// Package flake wraps Nix flake references and locked flakes.
package flake

import (
	"fmt"

	"github.com/sund3RRR/gonix/fetchers"
	"github.com/sund3RRR/gonix/internal/status"
	"github.com/sund3RRR/gonix/internal/utils"
	nix "github.com/sund3RRR/nix-go-bindings"
)

// Ref is an owned Nix flake reference.
//
// A Ref owns the underlying Nix flake reference and borrows the Nix context,
// fetcher settings, and flake settings used to parse it. Close is idempotent,
// but methods that need the raw reference return status.ErrClosed after Close.
type Ref struct {
	ctx      *nix.NixCContext
	ptr      *nix.NixFlakeReference
	fragment string
}

// NewParsedRef parses ref using already-created fetcher and flake settings.
//
// The returned Ref owns the raw flake reference. The ctx, fetchSettings, and
// flakeSettings arguments are borrowed and must outlive the returned Ref.
func NewParsedRef(
	ctx *nix.NixCContext,
	fetchSettings *fetchers.Settings,
	flakeSettings *Settings,
	ref string,
	opts ...ParseOption,
) (*Ref, error) {
	var cfg parseConfig
	for _, opt := range opts {
		opt(&cfg)
	}

	fetchSettingsPtr, err := fetchSettings.Borrow()
	if err != nil {
		return nil, fmt.Errorf("flake: failed to borrow fetch settings: %w", err)
	}

	flakeSettingsPtr, err := flakeSettings.Borrow()
	if err != nil {
		return nil, fmt.Errorf("flake: failed to borrow flake settings: %w", err)
	}

	flags := nix.FlakeReferenceParseFlagsNew(ctx, flakeSettingsPtr)
	if flags == nil {
		return nil, fmt.Errorf("flake: failed to create parse flags: %w", status.FromContext(ctx))
	}
	defer nix.FlakeReferenceParseFlagsFree(flags)

	if cfg.baseDirectory != "" {
		if code := nix.FlakeReferenceParseFlagsSetBaseDirectory(ctx, flags, cfg.baseDirectory, uint64(len(cfg.baseDirectory))); status.ErrorCode(code) != status.ErrorCodeOK {
			return nil, fmt.Errorf("flake: failed to set base directory: %w", status.FromContext(ctx))
		}
	}

	result := nix.FlakeReferenceAndFragmentFromString(ctx, fetchSettingsPtr, flakeSettingsPtr, flags, ref, uint64(len(ref)))
	if result == nil {
		return nil, fmt.Errorf("flake: failed to parse reference %q: %w", ref, status.FromContext(ctx))
	}
	defer nix.FlakeReferenceResultFree(result)

	ptr := nix.FlakeReferenceResultTakeReference(result)
	if ptr == nil {
		return nil, fmt.Errorf("flake: failed to take parsed reference: %w", status.FromContext(ctx))
	}

	fragment := utils.TakeCString(nix.FlakeReferenceResultTakeFragment(result))
	return &Ref{
		ctx:      ctx,
		ptr:      ptr,
		fragment: fragment,
	}, nil
}

// Fragment returns the fragment parsed from the original flake reference.
func (r *Ref) Fragment() string {
	return r.fragment
}

// Borrow returns the borrowed raw Nix flake reference.
//
// Callers must not free the returned pointer and must not retain it beyond the
// immediate raw Nix call that needs it. This is an escape hatch for integration
// with lower-level bindings.
func (r *Ref) Borrow() (*nix.NixFlakeReference, error) {
	if r.ptr == nil {
		return nil, status.ErrClosed
	}

	return r.ptr, nil
}

// Close releases the owned Nix flake reference.
//
// Close is safe to call more than once. Once Close returns, methods that need
// the raw reference report status.ErrClosed.
func (r *Ref) Close() error {
	if r.ptr == nil {
		return nil
	}

	nix.FlakeReferenceFree(r.ptr)
	r.ptr = nil

	return nil
}
