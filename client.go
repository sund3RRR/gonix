package gonix

import (
	"errors"
	"fmt"
	"sort"

	"github.com/sund3RRR/gonix/eval"
	"github.com/sund3RRR/gonix/fetchers"
	"github.com/sund3RRR/gonix/flakesettings"
	"github.com/sund3RRR/gonix/internal/status"
	"github.com/sund3RRR/gonix/nixcontext"
	"github.com/sund3RRR/gonix/store"
)

// Client owns the resources for high-level flake workflows.
//
// Client owns a hidden Nix context, a default store and evaluator, fetcher and
// flake settings. Flakes created through NewFlake are caller-owned and must be
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

// NewFlake parses and locks ref.
//
// The returned Flake is owned by the caller and must be closed before c.
func (c *Client) NewFlake(ref string, opts ...FlakeOption) (*Flake, error) {
	if c.ctx == nil {
		return nil, status.ErrClosed
	}

	f, err := NewFlake(c.ctx, c.fetcherSettings, c.flakeSettings, c.store, c.evaluator, ref, opts...)
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
	if c == nil || c.ctx == nil {
		return status.ErrClosed
	}

	value, err := c.evaluator.EvalString(expr, "<gonix>")
	if err != nil {
		return fmt.Errorf("client: failed to evaluate expression: %w", err)
	}
	defer func() {
		if closeErr := value.Close(); closeErr != nil {
			closeErr = fmt.Errorf("client: failed to close evaluated value: %w", closeErr)
			if err != nil {
				err = errors.Join(err, closeErr)
				return
			}
			err = closeErr
		}
	}()

	if err := c.evaluator.Unmarshal(value, out); err != nil {
		return fmt.Errorf("client: failed to decode expression result: %w", err)
	}

	return nil
}

// Close releases the evaluator, store, settings, and context.
//
// Close is idempotent. It attempts every cleanup and joins multiple errors.
func (c *Client) Close() error {
	if c == nil || c.ctx == nil {
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
