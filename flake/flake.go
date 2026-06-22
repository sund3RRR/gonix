// Package flake provides owned locked flakes and access to their output values.
package flake

import (
	"encoding/json"
	"fmt"

	"github.com/sund3RRR/gonix/eval"
	"github.com/sund3RRR/gonix/fetchers"
	"github.com/sund3RRR/gonix/flakesettings"
	"github.com/sund3RRR/gonix/internal/status"
	"github.com/sund3RRR/gonix/nixcontext"
	"github.com/sund3RRR/gonix/pkg/raw"
	"github.com/sund3RRR/gonix/pkg/utils"
	"github.com/sund3RRR/gonix/store"
)

// Flake is an owned Nix locked flake.
//
// A Flake owns the underlying Nix locked flake and borrows the Evaluator, Nix
// context, and flake settings used to create it. The Store and fetcher settings
// are borrowed only during construction. Close is idempotent, but methods that
// need the raw locked flake return status.ErrClosed after Close. Fragment,
// LockInfo, and Fingerprint use cached metadata and remain available after Close
// or closure of borrowed resources.
type Flake struct {
	fragment      string
	fingerprint   string
	lockInfoJSON  []byte
	ctx           *nixcontext.Context
	flakeSettings *flakesettings.Settings
	evaluator     *eval.Evaluator
	ptr           *raw.NixLockedFlake
}

// New locks ref using already-created fetcher and flake settings.
//
// New caches the resolved lock graph JSON and fingerprint before returning.
// The returned Flake owns the raw locked flake. New borrows nixStore and
// fetchSettings only during construction. The ctx, flakeSettings, and evaluator
// arguments must outlive raw operations on the returned Flake.
func New(
	ctx *nixcontext.Context,
	nixStore *store.Store,
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
	parseflags := raw.FlakeReferenceParseFlagsNew(rawCtx, flakeSettingsPtr)
	if parseflags == nil {
		return nil, fmt.Errorf("flake: failed to create parse flags: %w", status.FromContext(rawCtx))
	}
	defer raw.FlakeReferenceParseFlagsFree(parseflags)

	// Set flake's base directory
	if cfg.baseDirectory != "" {
		if code := raw.FlakeReferenceParseFlagsSetBaseDirectory(rawCtx, parseflags, cfg.baseDirectory, uint64(len(cfg.baseDirectory))); status.ErrorCode(code) != status.ErrorCodeOK {
			return nil, fmt.Errorf("flake: failed to set base directory: %w", status.FromContext(rawCtx))
		}
	}

	// Allocate and defer free flake reference result
	result := raw.FlakeReferenceAndFragmentFromString(rawCtx, fetchSettingsPtr, flakeSettingsPtr, parseflags, ref, uint64(len(ref)))
	if result == nil {
		return nil, fmt.Errorf("flake: failed to parse reference %q: %w", ref, status.FromContext(rawCtx))
	}
	defer raw.FlakeReferenceResultFree(result)

	// Allocate and defer free flake reference
	refPtr := raw.FlakeReferenceResultTakeReference(result)
	defer raw.FlakeReferenceFree(refPtr)

	// Take flake reference fragment
	fragment := utils.TakeCString(raw.FlakeReferenceResultTakeFragment(result))

	// Allocate and defer free flake lock flags
	lockFlags := raw.FlakeLockFlagsNew(rawCtx, flakeSettingsPtr)
	if lockFlags == nil {
		return nil, fmt.Errorf("flake: failed to create lock flags: %w", status.FromContext(rawCtx))
	}
	defer raw.FlakeLockFlagsFree(lockFlags)

	// Set lock mode
	if err := applyLockMode(rawCtx, lockFlags, cfg.mode); err != nil {
		return nil, fmt.Errorf("flake: failed to set lock mode: %w", err)
	}

	// Apply flake's input overrides
	for _, override := range cfg.inputOverrides {
		// Allocate and defer free input override flake reference result
		overrideResult := raw.FlakeReferenceAndFragmentFromString(rawCtx, fetchSettingsPtr, flakeSettingsPtr, parseflags, override.ref, uint64(len(override.ref)))
		if overrideResult == nil {
			return nil, fmt.Errorf("flake: failed to parse override reference %q: %w", override.ref, status.FromContext(rawCtx))
		}
		defer raw.FlakeReferenceResultFree(overrideResult)

		// Allocate and defer free override flake reference
		overrideRefPtr := raw.FlakeReferenceResultTakeReference(overrideResult)
		defer raw.FlakeReferenceFree(overrideRefPtr)

		// Add input override
		if code := raw.FlakeLockFlagsAddInputOverride(rawCtx, lockFlags, override.path, overrideRefPtr); status.ErrorCode(code) != status.ErrorCodeOK {
			return nil, fmt.Errorf("flake: failed to add input override '%q': %w", override.path, status.FromContext(rawCtx))
		}
	}

	// Finally allocate the locked flake pointer
	locked := raw.FlakeLock(rawCtx, fetchSettingsPtr, flakeSettingsPtr, state, lockFlags, refPtr)
	if locked == nil {
		return nil, fmt.Errorf("flake: failed to lock reference: %w", status.FromContext(rawCtx))
	}

	defer func() {
		if err != nil {
			raw.LockedFlakeFree(locked)
		}
	}()

	lockJSONPtr := raw.LockedFlakeGetLockJson(rawCtx, locked)
	if lockJSONPtr == nil {
		if err = status.FromContext(rawCtx); err != nil {
			return nil, fmt.Errorf("flake: failed to get lock json: %w", err)
		}
		err = fmt.Errorf("flake: failed to get lock json")
		return nil, err
	}

	lockJSON := []byte(utils.TakeCString(lockJSONPtr))

	storePtr, err := nixStore.Borrow()
	if err != nil {
		return nil, fmt.Errorf("flake: failed to borrow store: %w", err)
	}

	var fingerprint string
	fingerprintPtr := raw.LockedFlakeGetFingerprint(rawCtx, storePtr, fetchSettingsPtr, locked)
	if fingerprintPtr == nil {
		if err = status.FromContext(rawCtx); err != nil {
			return nil, fmt.Errorf("flake: failed to get fingerprint: %w", err)
		}
	} else {
		fingerprint = utils.TakeCString(fingerprintPtr)
	}

	f := &Flake{
		fragment:      fragment,
		lockInfoJSON:  lockJSON,
		fingerprint:   fingerprint,
		ctx:           ctx,
		flakeSettings: flakeSettings,
		evaluator:     evaluator,
		ptr:           locked,
	}

	return f, nil
}

// Fragment returns the fragment parsed from the original flake reference.
func (f *Flake) Fragment() string {
	return f.fragment
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

	ptr := raw.LockedFlakeGetOutputAttrs(rawCtx, flakeSettingsPtr, state, f.ptr)
	if ptr == nil {
		return nil, fmt.Errorf("flake: failed to get output attrs: %w", status.FromContext(rawCtx))
	}

	value, err := f.evaluator.WrapValue(ptr)
	if err != nil {
		if code := raw.ValueDecref(rawCtx, ptr); status.ErrorCode(code) != status.ErrorCodeOK {
			return nil, fmt.Errorf("flake: failed to wrap output attrs and decref value: %w", status.FromContext(rawCtx))
		}
		return nil, fmt.Errorf("flake: failed to wrap output attrs: %w", err)
	}

	return value, nil
}

// LockInfo returns f's cached Nix lock graph.
//
// Each call decodes a fresh graph, so callers may mutate the returned maps,
// slices, and raw attribute bytes. The metadata remains available after Flake,
// Store, or Context closure.
func (f *Flake) LockInfo() (LockInfo, error) {
	var lockInfo LockInfo
	if err := json.Unmarshal(f.lockInfoJSON, &lockInfo); err != nil {
		return LockInfo{}, fmt.Errorf("flake: failed to decode lock info: %w", err)
	}

	return lockInfo, nil
}

// Fingerprint returns the cached lowercase base16 Nix fingerprint.
//
// An empty string means Nix could not fingerprint the locked flake, for example
// because its lock graph is unlocked or its root input is not fingerprintable.
// The cached value remains available after Flake, Store, or Context closure.
func (f *Flake) Fingerprint() string {
	return f.fingerprint
}

// Borrow returns the borrowed raw Nix locked flake.
//
// Callers must not free the returned pointer and must not retain it beyond the
// immediate raw Nix call that needs it. This is an escape hatch for integration
// with lower-level bindings.
func (f *Flake) Borrow() (*raw.NixLockedFlake, error) {
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
	if f == nil || f.ptr == nil {
		return nil
	}

	raw.LockedFlakeFree(f.ptr)
	f.ptr = nil
	f.evaluator = nil
	f.flakeSettings = nil

	return nil
}

func applyLockMode(ctx *raw.NixCContext, flags *raw.NixFlakeLockFlags, mode LockMode) error {
	var code raw.NixErr
	switch mode {
	case LockModeVirtual:
		code = raw.FlakeLockFlagsSetModeVirtual(ctx, flags)
	case LockModeCheck:
		code = raw.FlakeLockFlagsSetModeCheck(ctx, flags)
	case LockModeWriteAsNeeded:
		code = raw.FlakeLockFlagsSetModeWriteAsNeeded(ctx, flags)
	default:
		return fmt.Errorf("flake: invalid lock mode %d", mode)
	}

	if status.ErrorCode(code) != status.ErrorCodeOK {
		return fmt.Errorf("unknown lock mode: %w", status.FromContext(ctx))
	}

	return nil
}
