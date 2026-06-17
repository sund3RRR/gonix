package gonix

import (
	"errors"
	"fmt"
	"io"

	"github.com/sund3RRR/gonix/eval"
	"github.com/sund3RRR/gonix/fetchers"
	"github.com/sund3RRR/gonix/flake"
	"github.com/sund3RRR/gonix/flakesettings"
	"github.com/sund3RRR/gonix/internal/status"
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
	ctx                  *nix.NixCContext
	flakeFetcherSettings *fetchers.Settings
	flakeSettings        *flakesettings.Settings
	resources            []io.Closer
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

	// Init nix libutil
	if code := nix.LibutilInit(ctx); status.ErrorCode(code) != status.ErrorCodeOK {
		err = status.FromContext(ctx)
		return nil, fmt.Errorf("runtime: failed to initialize util library: %w", err)
	}

	// Init nix libstore
	libStoreInitFn := nix.LibstoreInitNoLoadConfig
	if cfg.loadConfig {
		libStoreInitFn = nix.LibstoreInit
	}
	if code := libStoreInitFn(ctx); status.ErrorCode(code) != status.ErrorCodeOK {
		err = status.FromContext(ctx)
		return nil, fmt.Errorf("runtime: failed to initialize store library: %w", err)
	}

	// Init nix libexpr
	if code := nix.LibexprInit(r.ctx); status.ErrorCode(code) != status.ErrorCodeOK {
		err = status.FromContext(r.ctx)
		return nil, fmt.Errorf("runtime: failed to initialize expression library: %w", err)
	}

	// Init nix libfetchers
	var fetcherSettings *fetchers.Settings
	fetcherSettings, err = fetchers.NewSettings(r.ctx)
	if err != nil {
		return nil, fmt.Errorf("runtime: failed to create fetcher settings: %w", err)
	}
	r.flakeFetcherSettings = fetcherSettings
	r.resources = append(r.resources, fetcherSettings)

	// Init nix libflake
	var flakeSettings *flakesettings.Settings
	flakeSettings, err = flakesettings.New(r.ctx)
	if err != nil {
		return nil, fmt.Errorf("runtime: failed to create flake settings: %w", err)
	}
	r.flakeSettings = flakeSettings
	r.resources = append(r.resources, flakeSettings)

	// Apply config
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

// GetFlakeSettings returns the flake settings.
func (r *Runtime) GetFlakeSettings() (*flakesettings.Settings, error) {
	if r.ctx == nil {
		return nil, status.ErrClosed
	}

	return r.flakeSettings, nil
}

// GetFlakeFetcherSettings returns the fetcher settings.
func (r *Runtime) GetFlakeFetcherSettings() (*fetchers.Settings, error) {
	if r.ctx == nil {
		return nil, status.ErrClosed
	}

	return r.flakeFetcherSettings, nil
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

// NewEvaluator creates a Nix evaluator and tracks it for Runtime.Close.
func (r *Runtime) NewEvaluator(s *store.Store, opts ...eval.Option) (*eval.Evaluator, error) {
	if r.ctx == nil {
		return nil, status.ErrClosed
	}

	e, err := eval.New(r.ctx, s, opts...)
	if err != nil {
		return nil, fmt.Errorf("runtime: new evaluator: %w", err)
	}
	r.resources = append(r.resources, e)

	return e, nil
}

// ParseFlakeRef parses a flake reference and tracks it for Runtime.Close.
func (r *Runtime) ParseFlakeRef(ref string, opts ...flake.ParseOption) (*flake.Ref, error) {
	if r.ctx == nil {
		return nil, status.ErrClosed
	}

	parsed, err := flake.NewParsedRef(r.ctx, r.flakeFetcherSettings, r.flakeSettings, ref, opts...)
	if err != nil {
		return nil, fmt.Errorf("runtime: parse flake ref: %w", err)
	}
	r.resources = append(r.resources, parsed)

	return parsed, nil
}

// LockFlake locks a flake reference and tracks it for Runtime.Close.
func (r *Runtime) LockFlake(e *eval.Evaluator, ref *flake.Ref, opts ...flake.LockOption) (*flake.LockedFlake, error) {
	if r.ctx == nil {
		return nil, status.ErrClosed
	}

	locked, err := flake.NewLockedFlake(r.ctx, r.flakeFetcherSettings, r.flakeSettings, e, ref, opts...)
	if err != nil {
		return nil, fmt.Errorf("runtime: lock flake: %w", err)
	}
	r.resources = append(r.resources, locked)

	return locked, nil
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
	r.flakeFetcherSettings = nil
	r.flakeSettings = nil

	if len(errs) != 0 {
		return fmt.Errorf("runtime: failed to close resources: %w", errors.Join(errs...))
	}

	return nil
}
