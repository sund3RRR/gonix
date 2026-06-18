// Package flake wraps Nix flake references and locked flakes.
package flake

import (
	"fmt"

	"github.com/sund3RRR/gonix/fetchers"
	"github.com/sund3RRR/gonix/flakesettings"
	"github.com/sund3RRR/gonix/internal/status"
	"github.com/sund3RRR/gonix/nixcontext"
	"github.com/sund3RRR/gonix/pkg/utils"
	nix "github.com/sund3RRR/nix-go-bindings"
)

// Ref is an owned Nix flake reference.
//
// A Ref owns the underlying Nix flake reference and borrows the Nix context,
// fetcher settings, and flake settings used to parse it. Close is idempotent,
// but methods that need the raw reference return status.ErrClosed after Close.
type Ref struct {
	ctx      *nixcontext.Context
	ptr      *nix.NixFlakeReference
	fragment string
}

// NewParsedRef parses ref using already-created fetcher and flake settings.
//
// The returned Ref owns the raw flake reference. The ctx, fetchSettings, and
// flakeSettings arguments are borrowed and must outlive the returned Ref.
func NewParsedRef(
	ctx *nixcontext.Context,
	fetchSettings *fetchers.Settings,
	flakeSettings *flakesettings.Settings,
	ref string,
	opts ...ParseOption,
) (*Ref, error) {
	var cfg parseConfig
	for _, opt := range opts {
		opt(&cfg)
	}

	rawCtx, err := ctx.Borrow()
	if err != nil {
		return nil, fmt.Errorf("flake: failed to borrow context: %w", err)
	}

	fetchSettingsPtr, err := fetchSettings.Borrow()
	if err != nil {
		return nil, fmt.Errorf("flake: failed to borrow fetch settings: %w", err)
	}

	flakeSettingsPtr, err := flakeSettings.Borrow()
	if err != nil {
		return nil, fmt.Errorf("flake: failed to borrow flake settings: %w", err)
	}

	flags := nix.FlakeReferenceParseFlagsNew(rawCtx, flakeSettingsPtr)
	if flags == nil {
		return nil, fmt.Errorf("flake: failed to create parse flags: %w", status.FromContext(rawCtx))
	}
	defer nix.FlakeReferenceParseFlagsFree(flags)

	if cfg.baseDirectory != "" {
		if code := nix.FlakeReferenceParseFlagsSetBaseDirectory(rawCtx, flags, cfg.baseDirectory, uint64(len(cfg.baseDirectory))); status.ErrorCode(code) != status.ErrorCodeOK {
			return nil, fmt.Errorf("flake: failed to set base directory: %w", status.FromContext(rawCtx))
		}
	}

	result := nix.FlakeReferenceAndFragmentFromString(rawCtx, fetchSettingsPtr, flakeSettingsPtr, flags, ref, uint64(len(ref)))
	if result == nil {
		return nil, fmt.Errorf("flake: failed to parse reference %q: %w", ref, status.FromContext(rawCtx))
	}
	defer nix.FlakeReferenceResultFree(result)

	ptr := nix.FlakeReferenceResultTakeReference(result)
	if ptr == nil {
		return nil, fmt.Errorf("flake: failed to take parsed reference: %w", status.FromContext(rawCtx))
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
	if r == nil || r.ptr == nil {
		return nil
	}

	nix.FlakeReferenceFree(r.ptr)
	r.ptr = nil

	return nil
}
