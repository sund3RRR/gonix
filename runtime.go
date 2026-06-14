package gonix

import (
	"errors"
	"fmt"
	"io"
	"sort"
	"strconv"
	"strings"

	"github.com/sund3RRR/gonix/internal/status"
	"github.com/sund3RRR/gonix/internal/utils"
	"github.com/sund3RRR/gonix/store"
	nix "github.com/sund3RRR/nix-go-bindings"
)

// Runtime owns a Nix context and creates high-level gonix resources.
//
// A Runtime is not goroutine-safe. Use one Runtime as one execution stream; for
// parallel Nix work, create separate Runtime instances. Close releases every
// resource created through the Runtime and then releases the underlying Nix
// context.
type Runtime struct {
	ctx       *nix.NixCContext
	resources []io.Closer
}

// NewRuntime creates and initializes a Nix runtime.
func NewRuntime(opts ...Option) (*Runtime, error) {
	cfg := newRuntimeConfig()
	for _, opt := range opts {
		opt(&cfg)
	}

	ctx := nix.CContextCreate()
	if ctx == nil {
		return nil, fmt.Errorf("runtime: failed to create context")
	}

	r := &Runtime{ctx: ctx}

	// Free the context if an error occurs.
	var err error
	defer func() {
		if err != nil {
			_ = r.Close()
		}
	}()

	if code := nix.LibutilInit(ctx); status.ErrorCode(code) != status.ErrorCodeOK {
		err = status.FromContext(ctx)
		return nil, fmt.Errorf("runtime: failed to initialize util library: %w", err)
	}

	libStoreInitFn := nix.LibstoreInitNoLoadConfig
	if cfg.loadConfig {
		libStoreInitFn = nix.LibstoreInit
	}

	if code := libStoreInitFn(ctx); status.ErrorCode(code) != status.ErrorCodeOK {
		err = status.FromContext(ctx)
		return nil, fmt.Errorf("runtime: failed to initialize store library: %w", err)
	}

	err = r.applySettings(cfg.serialize())
	if err != nil {
		return nil, fmt.Errorf("runtime: failed to apply settings: %w", err)
	}

	if cfg.verbosity != nil {
		err = r.SetVerbosity(*cfg.verbosity)
		if err != nil {
			return nil, fmt.Errorf("runtime: failed to set verbosity: %w", err)
		}
	}

	if cfg.logFormat != nil {
		err = r.SetLogFormat(*cfg.logFormat)
		if err != nil {
			return nil, fmt.Errorf("runtime: failed to set log format: %w", err)
		}
	}

	return r, nil
}

// OpenStore opens a Nix store and tracks it for Runtime.Close.
func (r *Runtime) OpenStore(uri string, opts ...store.Option) (*store.Store, error) {
	if r.ctx == nil {
		return nil, status.ErrClosed
	}

	s, err := store.New(r.ctx, uri, opts...)
	if err != nil {
		return nil, fmt.Errorf("runtime: open store: %w", err)
	}

	r.resources = append(r.resources, s)
	return s, nil
}

// Setting returns the current value of a Nix setting.
func (r *Runtime) Setting(key string) (string, error) {
	if r.ctx == nil {
		return "", status.ErrClosed
	}

	ptr := nix.SettingGet(r.ctx, key)
	if ptr == nil {
		return "", fmt.Errorf("runtime: get setting %q: %w", key, status.FromContext(r.ctx))
	}

	return utils.TakeCString(ptr), nil
}

// SetSetting sets a Nix setting.
func (r *Runtime) SetSetting(key, value string) error {
	if r.ctx == nil {
		return status.ErrClosed
	}

	if code := nix.SettingSet(r.ctx, key, value); status.ErrorCode(code) != status.ErrorCodeOK {
		return fmt.Errorf("runtime: set setting %q: %w", key, status.FromContext(r.ctx))
	}

	return nil
}

// SetVerbosity sets the Nix verbosity level.
func (r *Runtime) SetVerbosity(level Verbosity) error {
	if r.ctx == nil {
		return status.ErrClosed
	}

	if code := nix.SetVerbosity(r.ctx, nix.NixVerbosity(level)); status.ErrorCode(code) != status.ErrorCodeOK {
		return fmt.Errorf("runtime: set verbosity: %w", status.FromContext(r.ctx))
	}

	return nil
}

// SetLogFormat sets the Nix log format.
func (r *Runtime) SetLogFormat(format LogFormat) error {
	if r.ctx == nil {
		return status.ErrClosed
	}

	if code := nix.SetLogFormat(r.ctx, string(format)); status.ErrorCode(code) != status.ErrorCodeOK {
		return fmt.Errorf("runtime: set log format: %w", status.FromContext(r.ctx))
	}

	return nil
}

// SetExperimentalFeatures sets Nix experimental features.
func (r *Runtime) SetExperimentalFeatures(features ...ExperimentalFeature) error {
	return r.SetSetting(settingExperimentalFeatures, formatExperimentalFeatures(features))
}

// SetCores sets the number of cores exposed to builders through NIX_BUILD_CORES.
func (r *Runtime) SetCores(cores int) error {
	return r.SetSetting(settingCores, strconv.Itoa(cores))
}

// SetMaxJobs sets the maximum number of local build jobs.
func (r *Runtime) SetMaxJobs(maxJobs int) error {
	return r.SetSetting(settingMaxJobs, strconv.Itoa(maxJobs))
}

// SetMaxJobsAuto lets Nix choose the maximum number of local build jobs.
func (r *Runtime) SetMaxJobsAuto() error {
	return r.SetSetting(settingMaxJobs, "auto")
}

// SetSystem sets the Nix build system.
func (r *Runtime) SetSystem(system System) error {
	return r.SetSetting(settingSystem, string(system))
}

// SetEvalSystem sets the Nix evaluation system.
func (r *Runtime) SetEvalSystem(system System) error {
	return r.SetSetting(settingEvalSystem, string(system))
}

// SetSubstituters sets the substituter store URLs.
func (r *Runtime) SetSubstituters(urls ...string) error {
	return r.SetSetting(settingSubstituters, strings.Join(urls, " "))
}

// SetTrustedPublicKeys sets trusted binary cache public keys.
func (r *Runtime) SetTrustedPublicKeys(keys ...string) error {
	return r.SetSetting(settingTrustedPublicKeys, strings.Join(keys, " "))
}

// SetPureEval controls pure evaluation mode.
func (r *Runtime) SetPureEval(pure bool) error {
	return r.SetSetting(settingPureEval, strconv.FormatBool(pure))
}

// SetImportFromDerivation controls whether evaluation may import from derivations.
func (r *Runtime) SetImportFromDerivation(allow bool) error {
	return r.SetSetting(settingAllowImportFromDerivation, strconv.FormatBool(allow))
}

// SetAcceptFlakeConfig controls whether flake-provided Nix settings are accepted.
func (r *Runtime) SetAcceptFlakeConfig(accept bool) error {
	return r.SetSetting(settingAcceptFlakeConfig, strconv.FormatBool(accept))
}

// Close releases resources created through r and then releases the Nix context.
//
// Close is idempotent. If multiple tracked resources fail to close, Close
// returns the first error and still attempts to close the rest.
func (r *Runtime) Close() error {
	if r.ctx == nil {
		return nil
	}

	errs := make([]error, 0, len(r.resources))
	for i := len(r.resources) - 1; i >= 0; i-- {
		if err := r.resources[i].Close(); err != nil {
			errs = append(errs, err)
		}
	}

	nix.CContextFree(r.ctx)
	r.ctx = nil
	r.resources = nil

	if len(errs) != 0 {
		return fmt.Errorf("runtime: failed to close resources: %w", errors.Join(errs...))
	}

	return nil
}

func (r *Runtime) applySettings(settings map[string]string) error {
	if len(settings) == 0 {
		return nil
	}

	if value, ok := settings[settingExperimentalFeatures]; ok {
		if err := r.SetSetting(settingExperimentalFeatures, value); err != nil {
			return err
		}
	}

	keys := make([]string, 0, len(settings))
	for key := range settings {
		if key == settingExperimentalFeatures {
			continue
		}
		keys = append(keys, key)
	}
	sort.Strings(keys)

	for _, key := range keys {
		if err := r.SetSetting(key, settings[key]); err != nil {
			return err
		}
	}

	return nil
}
