// Package flake provides owned locked flakes and access to their output values.
package flake

import (
	"fmt"

	"github.com/sund3RRR/gonix/eval"
	"github.com/sund3RRR/gonix/fetchers"
	"github.com/sund3RRR/gonix/flakesettings"
	"github.com/sund3RRR/gonix/internal/status"
	"github.com/sund3RRR/gonix/nixcontext"
	"github.com/sund3RRR/gonix/pkg/utils"
	nix "github.com/sund3RRR/nix-go-bindings"
)

// LockMode selects how Nix should handle a flake lock file.
type LockMode int

const (
	// LockModeVirtual resolves the lock in memory without writing flake.lock.
	LockModeVirtual LockMode = iota
	// LockModeCheck fails unless the existing lock file is already usable.
	LockModeCheck
	// LockModeWriteAsNeeded creates or updates flake.lock when needed.
	LockModeWriteAsNeeded
)

// Flake is an owned Nix locked flake.
//
// A Flake owns the underlying Nix locked flake and borrows the evaluator,
// Nix context, and flake settings used to create it. Close is idempotent, but
// methods that need the raw locked flake return status.ErrClosed after Close.
type Flake struct {
	fragment      string
	ctx           *nixcontext.Context
	flakeSettings *flakesettings.Settings
	evaluator     *eval.Evaluator
	ptr           *nix.NixLockedFlake
}

// New locks ref using already-created fetcher and flake settings.
//
// The returned Flake owns the raw locked flake. The ctx, fetchSettings,
// flakeSettings, and evaluator arguments are borrowed and must outlive the
// returned Flake.
func New(
	ctx *nixcontext.Context,
	fetchSettings *fetchers.Settings,
	flakeSettings *flakesettings.Settings,
	evaluator *eval.Evaluator,
	ref string,
	opts ...Option,
) (*Flake, error) {
	var cfg flakeConfig
	for _, opt := range opts {
		opt(&cfg)
	}

	// Borrow context
	rawCtx, err := ctx.Borrow()
	if err != nil {
		return nil, fmt.Errorf("flake: borrow context: %w", err)
	}

	// Borrow evaluator state
	state, err := evaluator.Borrow()
	if err != nil {
		return nil, fmt.Errorf("flake: failed to borrow evaluator: %w", err)
	}

	// Borrow fetcher settings
	fetchSettingsPtr, err := fetchSettings.Borrow()
	if err != nil {
		return nil, fmt.Errorf("flake: failed to borrow fetch settings: %w", err)
	}

	// Borrow flake settings
	flakeSettingsPtr, err := flakeSettings.Borrow()
	if err != nil {
		return nil, fmt.Errorf("flake: failed to borrow flake settings: %w", err)
	}

	// Allocate and defer free parse flags
	parseflags := nix.FlakeReferenceParseFlagsNew(rawCtx, flakeSettingsPtr)
	if parseflags == nil {
		return nil, fmt.Errorf("flake: failed to create parse flags: %w", status.FromContext(rawCtx))
	}
	defer nix.FlakeReferenceParseFlagsFree(parseflags)

	// Set flake's base directory
	if cfg.baseDirectory != "" {
		if code := nix.FlakeReferenceParseFlagsSetBaseDirectory(rawCtx, parseflags, cfg.baseDirectory, uint64(len(cfg.baseDirectory))); status.ErrorCode(code) != status.ErrorCodeOK {
			return nil, fmt.Errorf("flake: failed to set base directory: %w", status.FromContext(rawCtx))
		}
	}

	// Allocate and defer free flake reference result
	result := nix.FlakeReferenceAndFragmentFromString(rawCtx, fetchSettingsPtr, flakeSettingsPtr, parseflags, ref, uint64(len(ref)))
	if result == nil {
		return nil, fmt.Errorf("flake: failed to parse reference %q: %w", ref, status.FromContext(rawCtx))
	}
	defer nix.FlakeReferenceResultFree(result)

	// Allocate and defer free flake reference
	refPtr := nix.FlakeReferenceResultTakeReference(result)
	defer nix.FlakeReferenceFree(refPtr)

	// Take flake reference fragment
	fragment := utils.TakeCString(nix.FlakeReferenceResultTakeFragment(result))

	// Allocate and defer free flake lock flags
	lockFlags := nix.FlakeLockFlagsNew(rawCtx, flakeSettingsPtr)
	if lockFlags == nil {
		return nil, fmt.Errorf("flake: failed to create lock flags: %w", status.FromContext(rawCtx))
	}
	defer nix.FlakeLockFlagsFree(lockFlags)

	// Set lock mode
	if err := applyLockMode(rawCtx, lockFlags, cfg.mode); err != nil {
		return nil, fmt.Errorf("flake: failed to set lock mode: %w", err)
	}

	// Apply flake's input overrides
	for _, override := range cfg.inputOverrides {
		// Allocate and defer free input override flake reference result
		overrideResult := nix.FlakeReferenceAndFragmentFromString(rawCtx, fetchSettingsPtr, flakeSettingsPtr, parseflags, override.ref, uint64(len(override.ref)))
		if overrideResult == nil {
			return nil, fmt.Errorf("flake: failed to parse override reference %q: %w", override.ref, status.FromContext(rawCtx))
		}
		defer nix.FlakeReferenceResultFree(overrideResult)

		// Allocate and defer free override flake reference
		overrideRefPtr := nix.FlakeReferenceResultTakeReference(overrideResult)
		defer nix.FlakeReferenceFree(overrideRefPtr)

		// Add input override
		if code := nix.FlakeLockFlagsAddInputOverride(rawCtx, lockFlags, override.path, overrideRefPtr); status.ErrorCode(code) != status.ErrorCodeOK {
			return nil, fmt.Errorf("flake: failed to add input override '%q': %w", override.path, status.FromContext(rawCtx))
		}
	}

	// Finally allocate the locked flake pointer
	locked := nix.FlakeLock(rawCtx, fetchSettingsPtr, flakeSettingsPtr, state, lockFlags, refPtr)
	if locked == nil {
		return nil, fmt.Errorf("flake: failed to lock reference: %w", status.FromContext(rawCtx))
	}

	return &Flake{
		fragment:      fragment,
		ctx:           ctx,
		flakeSettings: flakeSettings,
		evaluator:     evaluator,
		ptr:           locked,
	}, nil
}

// Fragment returns the fragment parsed from the original flake reference.
func (f *Flake) Fragment() string {
	return f.fragment
}

// Output decodes a locked flake output selected by path into out.
//
// Each path element names one exact Nix attribute; dots have no special
// meaning. An empty path decodes the complete flake output attribute set. The
// out argument must be a non-nil pointer to a type supported by
// eval.Evaluator.Unmarshal.
func (f *Flake) Output(path []string, out any) error {
	if f.ptr == nil {
		return status.ErrClosed
	}

	value, err := f.OutputValue(path)
	if err != nil {
		return fmt.Errorf("flake: failed to get output attribute: %w", err)
	}
	defer value.Close() //nolint:errcheck

	if err := f.evaluator.Unmarshal(value, out); err != nil {
		return fmt.Errorf("flake: failed to decode output: %w", err)
	}

	return nil
}

// OutputValue returns the caller-owned locked flake output selected by path.
//
// Each path element names one exact Nix attribute; dots have no special
// meaning. An empty path returns the complete output attribute set. Every
// intermediate value is closed before this method returns; the final value has
// its own Nix reference and remains valid until the caller closes it.
func (f *Flake) OutputValue(path []string) (*eval.Value, error) {
	if f.ptr == nil {
		return nil, status.ErrClosed
	}

	value, err := f.OutputAttrs()
	if err != nil {
		return nil, fmt.Errorf("flake: failed to get output attributes: %w", err)
	}

	for _, attr := range path {
		typ, typeErr := value.Type()
		if typeErr != nil {
			_ = value.Close()
			return nil, fmt.Errorf("flake: failed to inspect output before attribute %q: %w", attr, typeErr)
		}
		if typ != eval.ValueTypeAttrs {
			_ = value.Close()
			return nil, fmt.Errorf(
				"flake: failed to get output attribute %q: %w",
				attr,
				&eval.ValueTypeError{Actual: typ, Expected: eval.ValueTypeAttrs},
			)
		}

		child, attrErr := f.evaluator.Attr(value, attr)
		_ = value.Close()
		if attrErr != nil {
			return nil, fmt.Errorf("flake: failed to get output attribute %q: %w", attr, attrErr)
		}
		value = child
	}

	return value, nil
}

// OutputAttrs returns the locked flake output attributes as a caller-owned
// evaluator value.
func (f *Flake) OutputAttrs() (*eval.Value, error) {
	if f.ptr == nil {
		return nil, status.ErrClosed
	}

	rawCtx, err := f.ctx.Borrow()
	if err != nil {
		return nil, fmt.Errorf("flake: failed to borrow context: %w", err)
	}

	state, err := f.evaluator.Borrow()
	if err != nil {
		return nil, fmt.Errorf("flake: failed to borrow evaluator: %w", err)
	}

	flakeSettingsPtr, err := f.flakeSettings.Borrow()
	if err != nil {
		return nil, fmt.Errorf("flake: failed to borrow flake settings: %w", err)
	}

	ptr := nix.LockedFlakeGetOutputAttrs(rawCtx, flakeSettingsPtr, state, f.ptr)
	if ptr == nil {
		return nil, fmt.Errorf("flake: failed to get output attrs: %w", status.FromContext(rawCtx))
	}

	value, err := f.evaluator.WrapValue(ptr)
	if err != nil {
		if code := nix.ValueDecref(rawCtx, ptr); status.ErrorCode(code) != status.ErrorCodeOK {
			return nil, fmt.Errorf("flake: failed to wrap output attrs and decref value: %w", status.FromContext(rawCtx))
		}
		return nil, fmt.Errorf("flake: failed to wrap output attrs: %w", err)
	}

	return value, nil
}

// Borrow returns the borrowed raw Nix locked flake.
//
// Callers must not free the returned pointer and must not retain it beyond the
// immediate raw Nix call that needs it. This is an escape hatch for integration
// with lower-level bindings.
func (f *Flake) Borrow() (*nix.NixLockedFlake, error) {
	if f.ptr == nil {
		return nil, status.ErrClosed
	}

	return f.ptr, nil
}

// Close releases the owned Nix locked flake.
//
// Close is safe to call more than once. Once Close returns, methods that need
// the raw locked flake report status.ErrClosed.
func (f *Flake) Close() error {
	if f.ptr == nil {
		return nil
	}

	nix.LockedFlakeFree(f.ptr)
	f.ptr = nil
	f.evaluator = nil
	f.flakeSettings = nil

	return nil
}

func applyLockMode(ctx *nix.NixCContext, flags *nix.NixFlakeLockFlags, mode LockMode) error {
	var code nix.NixErr
	switch mode {
	case LockModeVirtual:
		code = nix.FlakeLockFlagsSetModeVirtual(ctx, flags)
	case LockModeCheck:
		code = nix.FlakeLockFlagsSetModeCheck(ctx, flags)
	case LockModeWriteAsNeeded:
		code = nix.FlakeLockFlagsSetModeWriteAsNeeded(ctx, flags)
	default:
		return fmt.Errorf("flake: invalid lock mode %d", mode)
	}

	if status.ErrorCode(code) != status.ErrorCodeOK {
		return fmt.Errorf("unknown lock mode: %w", status.FromContext(ctx))
	}

	return nil
}
