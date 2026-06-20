package gonix

import (
	"errors"
	"fmt"
	"sort"

	"github.com/sund3RRR/gonix/eval"
	"github.com/sund3RRR/gonix/fetchers"
	"github.com/sund3RRR/gonix/flake"
	"github.com/sund3RRR/gonix/flakesettings"
	"github.com/sund3RRR/gonix/internal/status"
	"github.com/sund3RRR/gonix/nixcontext"
	"github.com/sund3RRR/gonix/store"
)

// Client owns the resources for high-level Nix workflows.
//
// Client owns a hidden Nix context, a default store and evaluator, fetcher and
// flake settings. Flakes created through OpenFlake are caller-owned and must be
// closed before the Client. Client is not goroutine-safe. Close releases owned
// resources in reverse dependency order.
type Client struct {
	ctx             *nixcontext.Context
	store           *store.Store
	evaluator       *eval.Evaluator
	fetcherSettings *fetchers.Settings
	flakeSettings   *flakesettings.Settings
}

// NewClient creates a flake-ready Client.
//
// ClientConfig{} is a valid quick-start configuration. The returned Client owns
// all resources it creates and must be closed.
func NewClient(cfg ClientConfig) (*Client, error) {
	ctx, err := nixcontext.New(nixcontext.Config{LoadConfig: cfg.LoadConfig})
	if err != nil {
		return nil, fmt.Errorf("client: failed to create context: %w", err)
	}

	c := &Client{ctx: ctx}
	defer func() {
		if err != nil {
			_ = c.Close()
		}
	}()

	if err = applyClientSettings(ctx, cfg.Serialize()); err != nil {
		return nil, err
	}
	if err = ctx.SetVerbosity(cfg.Verbosity); err != nil {
		return nil, fmt.Errorf("client: failed to set verbosity: %w", err)
	}
	if err = ctx.SetLogFormat(cfg.LogFormat); err != nil {
		return nil, fmt.Errorf("client: failed to set log format: %w", err)
	}

	c.fetcherSettings, err = fetchers.NewSettings(ctx)
	if err != nil {
		return nil, fmt.Errorf("client: failed to create fetcher settings: %w", err)
	}

	c.flakeSettings, err = flakesettings.New(ctx)
	if err != nil {
		return nil, fmt.Errorf("client: failed to create flake settings: %w", err)
	}

	storeURI := cfg.Store.URI
	if storeURI == "" {
		storeURI = store.Auto
	}
	c.store, err = store.New(ctx, storeURI, cfg.Store.Opts...)
	if err != nil {
		return nil, fmt.Errorf("client: failed to open store: %w", err)
	}

	evalOpts := append([]eval.Option(nil), cfg.Eval.Opts...)
	evalOpts = append(evalOpts, eval.WithFlakeSettings(c.flakeSettings))
	c.evaluator, err = eval.New(ctx, c.store, evalOpts...)
	if err != nil {
		return nil, fmt.Errorf("client: failed to create evaluator: %w", err)
	}

	return c, nil
}

// OpenFlake parses and locks ref.
//
// The returned Flake is owned by the caller and must be closed before c.
func (c *Client) OpenFlake(ref string, opts ...flake.Option) (*flake.Flake, error) {
	if c.ctx == nil {
		return nil, status.ErrClosed
	}

	f, err := flake.New(c.ctx, c.store, c.fetcherSettings, c.flakeSettings, c.evaluator, ref, opts...)
	if err != nil {
		return nil, fmt.Errorf("client: failed to create flake: %w", err)
	}

	return f, nil
}

// Eval evaluates expr and decodes its result into out.
//
// The out argument must be a non-nil pointer to a type supported by
// eval.Evaluator.Unmarshal. Use the eval package directly when the raw
// caller-owned Value or a custom diagnostic path is needed.
func (c *Client) Eval(expr string, out any) (err error) {
	if c.ctx == nil {
		return status.ErrClosed
	}

	value, err := c.evaluator.EvalString(expr, "<gonix>")
	if err != nil {
		return fmt.Errorf("client: failed to evaluate expression: %w", err)
	}
	defer value.Close() //nolint:errcheck

	if err := c.evaluator.Unmarshal(value, out); err != nil {
		return fmt.Errorf("client: failed to decode expression result: %w", err)
	}

	return nil
}

// Unmarshal decodes value into out using the Client's evaluator.
//
// Value must belong to this Client's evaluator. The out argument must be a
// non-nil pointer to a type supported by eval.Evaluator.Unmarshal.
func (c *Client) Unmarshal(value *eval.Value, out any) error {
	if c.ctx == nil {
		return status.ErrClosed
	}

	if err := c.evaluator.Unmarshal(value, out); err != nil {
		return fmt.Errorf("client: failed to unmarshal eval.Value: %w", err)
	}

	return nil
}

// Realize realizes every output of the derivation at drvPath.
//
// Realize returns pure Go DTOs and closes the Nix store path handles produced
// by Nix before returning. The Client's store must support realization.
func (c *Client) Realize(drvPath string) ([]RealizedOutput, error) {
	if c.ctx == nil {
		return nil, status.ErrClosed
	}

	drvStorePath, err := c.store.ParsePath(drvPath)
	if err != nil {
		return nil, fmt.Errorf("client: failed to realize derivation: parse drv path: %w", err)
	}
	defer drvStorePath.Close() //nolint:errcheck

	realizations, err := c.store.Realise(drvStorePath)
	if err != nil {
		return nil, fmt.Errorf("client: failed to realize derivation: realize store path: %w", err)
	}
	defer func() {
		for i := range realizations {
			_ = realizations[i].Close()
		}
	}()

	outputs := make([]RealizedOutput, 0, len(realizations))
	for i := range realizations {
		realizedPath := realizations[i].Path
		if realizedPath == nil {
			return nil, fmt.Errorf("client: failed to realize derivation: realization %q has nil path", realizations[i].OutputName)
		}

		storePath := c.store.PrintPath(realizedPath.Hash(), realizedPath.Name())

		realPath, err := c.store.RealPath(realizedPath)
		if err != nil {
			return nil, fmt.Errorf("client: failed to realize derivation: real output path %q: %w", realizations[i].OutputName, err)
		}

		outputs = append(outputs, RealizedOutput{
			OutputName: realizations[i].OutputName,
			StorePath:  storePath,
			RealPath:   realPath,
			Name:       realizedPath.Name(),
			Hash:       realizedPath.Hash(),
		})
	}

	return outputs, nil
}

// Close releases the evaluator, store, settings, and context.
//
// Close is idempotent. It attempts every cleanup and joins multiple errors.
func (c *Client) Close() error {
	if c.ctx == nil {
		return nil
	}

	errs := make([]error, 0)

	if err := c.evaluator.Close(); err != nil {
		errs = append(errs, err)
	}
	c.evaluator = nil

	if err := c.store.Close(); err != nil {
		errs = append(errs, err)
	}
	c.store = nil

	if err := c.flakeSettings.Close(); err != nil {
		errs = append(errs, err)
	}
	c.flakeSettings = nil

	if err := c.fetcherSettings.Close(); err != nil {
		errs = append(errs, err)
	}
	c.fetcherSettings = nil

	if err := c.ctx.Close(); err != nil {
		errs = append(errs, err)
	}
	c.ctx = nil

	if len(errs) != 0 {
		return fmt.Errorf("client: failed to close resources: %w", errors.Join(errs...))
	}

	return nil
}

func applyClientSettings(ctx *nixcontext.Context, settings map[string]string) error {
	if value, ok := settings[settingExperimentalFeatures]; ok {
		if err := ctx.SetSetting(settingExperimentalFeatures, value); err != nil {
			return fmt.Errorf("client: failed to set %q: %w", settingExperimentalFeatures, err)
		}
	}

	keys := make([]string, 0, len(settings))
	for key := range settings {
		if key != settingExperimentalFeatures {
			keys = append(keys, key)
		}
	}
	sort.Strings(keys)

	for _, key := range keys {
		if err := ctx.SetSetting(key, settings[key]); err != nil {
			return fmt.Errorf("client: failed to set %q: %w", key, err)
		}
	}

	return nil
}
