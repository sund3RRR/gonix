package flake

import (
	"fmt"

	"github.com/sund3RRR/gonix/eval"
	"github.com/sund3RRR/gonix/fetchers"
	"github.com/sund3RRR/gonix/internal/status"
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

// LockedFlake is an owned Nix locked flake.
//
// A LockedFlake owns the underlying Nix locked flake and borrows the evaluator,
// Nix context, and flake settings used to create it. Close is idempotent, but
// methods that need the raw locked flake return status.ErrClosed after Close.
type LockedFlake struct {
	ctx           *nix.NixCContext
	flakeSettings *Settings
	evaluator     *eval.Evaluator
	ptr           *nix.NixLockedFlake
}

// NewLockedFlake locks ref using already-created fetcher and flake settings.
//
// The returned LockedFlake owns the raw locked flake. The ctx, fetchSettings,
// flakeSettings, and evaluator arguments are borrowed and must outlive the
// returned LockedFlake.
func NewLockedFlake(
	ctx *nix.NixCContext,
	fetchSettings *fetchers.Settings,
	flakeSettings *Settings,
	evaluator *eval.Evaluator,
	ref *Ref,
	opts ...LockOption,
) (*LockedFlake, error) {
	state, err := evaluator.Borrow()
	if err != nil {
		return nil, fmt.Errorf("flake: failed to borrow evaluator: %w", err)
	}

	refPtr, err := ref.Borrow()
	if err != nil {
		return nil, fmt.Errorf("flake: failed to borrow reference: %w", err)
	}

	var cfg lockConfig
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

	flags := nix.FlakeLockFlagsNew(ctx, flakeSettingsPtr)
	if flags == nil {
		return nil, fmt.Errorf("flake: failed to create lock flags: %w", status.FromContext(ctx))
	}
	defer nix.FlakeLockFlagsFree(flags)

	if err := applyLockMode(ctx, flags, cfg.mode); err != nil {
		return nil, err
	}

	for _, override := range cfg.inputOverrides {
		overridePtr, err := override.ref.Borrow()
		if err != nil {
			return nil, fmt.Errorf("flake: failed to borrow input override %q: %w", override.path, err)
		}
		if code := nix.FlakeLockFlagsAddInputOverride(ctx, flags, override.path, overridePtr); status.ErrorCode(code) != status.ErrorCodeOK {
			return nil, fmt.Errorf("flake: failed to add input override %q: %w", override.path, status.FromContext(ctx))
		}
	}

	locked := nix.FlakeLock(ctx, fetchSettingsPtr, flakeSettingsPtr, state, flags, refPtr)
	if locked == nil {
		return nil, fmt.Errorf("flake: failed to lock reference: %w", status.FromContext(ctx))
	}

	return &LockedFlake{
		ctx:           ctx,
		flakeSettings: flakeSettings,
		evaluator:     evaluator,
		ptr:           locked,
	}, nil
}

// OutputAttrs returns the locked flake output attributes as an evaluator value.
func (l *LockedFlake) OutputAttrs() (*eval.Value, error) {
	if l.ptr == nil {
		return nil, status.ErrClosed
	}

	state, err := l.evaluator.Borrow()
	if err != nil {
		return nil, fmt.Errorf("flake: failed to borrow evaluator: %w", err)
	}

	flakeSettingsPtr, err := l.flakeSettings.Borrow()
	if err != nil {
		return nil, fmt.Errorf("flake: failed to borrow flake settings: %w", err)
	}

	ptr := nix.LockedFlakeGetOutputAttrs(l.ctx, flakeSettingsPtr, state, l.ptr)
	if ptr == nil {
		return nil, fmt.Errorf("flake: failed to get output attrs: %w", status.FromContext(l.ctx))
	}

	value, err := l.evaluator.WrapValue(ptr)
	if err != nil {
		if code := nix.ValueDecref(l.ctx, ptr); status.ErrorCode(code) != status.ErrorCodeOK {
			return nil, fmt.Errorf("flake: failed to wrap output attrs and decref value: %w", status.FromContext(l.ctx))
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
func (l *LockedFlake) Borrow() (*nix.NixLockedFlake, error) {
	if l.ptr == nil {
		return nil, status.ErrClosed
	}

	return l.ptr, nil
}

// Close releases the owned Nix locked flake.
//
// Close is safe to call more than once. Once Close returns, methods that need
// the raw locked flake report status.ErrClosed.
func (l *LockedFlake) Close() error {
	if l.ptr == nil {
		return nil
	}

	nix.LockedFlakeFree(l.ptr)
	l.ptr = nil
	l.evaluator = nil
	l.flakeSettings = nil

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
		return fmt.Errorf("flake: failed to set lock mode: %w", status.FromContext(ctx))
	}

	return nil
}
