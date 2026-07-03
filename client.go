package gonix

import (
	"context"
	"errors"
	"fmt"
	"io"
	"math"
	"sort"
	"sync/atomic"
	"syscall"

	"github.com/sund3RRR/gonix/eval"
	"github.com/sund3RRR/gonix/fetchers"
	"github.com/sund3RRR/gonix/flake"
	"github.com/sund3RRR/gonix/flakesettings"
	"github.com/sund3RRR/gonix/internal/status"
	"github.com/sund3RRR/gonix/nixcontext"
	"github.com/sund3RRR/gonix/pkg/raw"
	"github.com/sund3RRR/gonix/store"
	"github.com/sund3RRR/gonix/util"
)

// Client owns the resources for high-level Nix workflows.
//
// Client owns a hidden Nix context, a default store and evaluator, fetcher and
// flake settings. Flakes created through OpenFlake may be closed early by the
// caller and are otherwise closed by Client.Close. Client is not goroutine-safe
// and rejects overlapping operations with ErrConcurrentUse. Close releases
// owned resources in reverse dependency order.
type Client struct {
	busy atomic.Bool

	ctx             *nixcontext.Context
	store           *store.Store
	evaluator       *eval.Evaluator
	fetcherSettings *fetchers.Settings
	flakeSettings   *flakesettings.Settings
	resources       []io.Closer
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
	if err := ctx.SetLogSinkPath(cfg.LogSinkPath); err != nil {
		return nil, fmt.Errorf("client: failed to set log sink path: %w", err)
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

// WithStore runs handler with the Client's owned Store.
//
// The operation participates in Client cancellation and exclusive-use
// handling. The Store is borrowed and must not be closed or retained by
// handler.
func (c *Client) WithStore(ctx context.Context, handler func(*store.Store) error) error {
	return c.middleware(ctx, func() error {
		return handler(c.store)
	})
}

// WithEvaluator runs handler with the Client's owned Evaluator.
//
// The operation participates in Client cancellation and exclusive-use
// handling. The Evaluator is borrowed and must not be closed or retained by
// handler.
func (c *Client) WithEvaluator(ctx context.Context, handler func(*eval.Evaluator) error) error {
	return c.middleware(ctx, func() error {
		return handler(c.evaluator)
	})
}

// WithStoreAndEvaluator runs handler with the Client's owned Store and
// Evaluator.
//
// The operation participates in Client cancellation and exclusive-use
// handling. Both resources are borrowed and must not be closed or retained by
// handler.
func (c *Client) WithStoreAndEvaluator(ctx context.Context, handler func(*store.Store, *eval.Evaluator) error) error {
	return c.middleware(ctx, func() error {
		return handler(c.store, c.evaluator)
	})
}

// OpenFlake parses and locks ref.
//
// The returned Flake may be closed early by the caller. If it remains open,
// Client.Close closes it before releasing the resources it borrows.
func (c *Client) OpenFlake(ref string, opts ...flake.Option) (*flake.Flake, error) {
	var f *flake.Flake
	err := c.middleware(context.Background(), func() error {
		var err error
		f, err = flake.New(c.ctx, c.store, c.fetcherSettings, c.flakeSettings, c.evaluator, ref, opts...)
		if err != nil {
			return fmt.Errorf("client: failed to create flake: %w", err)
		}
		c.resources = append(c.resources, f)

		return nil
	})
	return f, err
}

// EvalFlakeOutput decodes a locked flake output selected by path into out.
//
// Each path element names one exact Nix attribute; dots have no special
// meaning. An empty path decodes the complete flake output attribute set. The
// out argument must be a non-nil pointer to a type supported by
// eval.Evaluator.Unmarshal.
func (c *Client) EvalFlakeOutput(ctx context.Context, f *flake.Flake, path []string, out any) error {
	return c.middleware(ctx, func() error {
		value, err := c.getOutputValue(f, nil, path)
		if err != nil {
			return fmt.Errorf("client: failed to get output value: %w", err)
		}
		defer value.Close() //nolint:errcheck

		if err := c.evaluator.Unmarshal(value, out); err != nil {
			return fmt.Errorf("client: failed to decode flake output: %w", err)
		}

		return nil
	})
}

// GetFlakeOutputValue returns the flake output selected by path.
//
// Each path element names one exact Nix attribute; dots have no special
// meaning. An empty path returns the complete flake output attribute set.
//
// The returned Value is caller-owned and must be closed.
func (c *Client) GetFlakeOutputValue(ctx context.Context, f *flake.Flake, path []string) (*eval.Value, error) {
	var value *eval.Value
	err := c.middleware(ctx, func() error {
		var err error
		value, err = c.getOutputValue(f, nil, path)
		if err != nil {
			return fmt.Errorf("client: failed to get output value: %w", err)
		}

		return nil
	})
	return value, err
}

// Eval evaluates expr and decodes its result into out.
//
// The out argument must be a non-nil pointer to a type supported by
// eval.Evaluator.Unmarshal. Use the eval package directly when the raw
// caller-owned Value or a custom diagnostic path is needed.
func (c *Client) Eval(ctx context.Context, expr string, out any) (err error) {
	return c.middleware(ctx, func() error {
		value, err := c.evaluator.EvalString(expr, "<gonix>")
		if err != nil {
			return fmt.Errorf("client: failed to evaluate expression: %w", err)
		}
		defer value.Close() //nolint:errcheck

		if err := c.evaluator.Unmarshal(value, out); err != nil {
			return fmt.Errorf("client: failed to decode expression result: %w", err)
		}
		return nil
	})
}

// Unmarshal decodes value into out using the Client's evaluator.
//
// Value must belong to this Client's evaluator. The out argument must be a
// non-nil pointer to a type supported by eval.Evaluator.Unmarshal.
func (c *Client) Unmarshal(ctx context.Context, value *eval.Value, out any) error {
	return c.middleware(ctx, func() error {
		if err := c.evaluator.Unmarshal(value, out); err != nil {
			return fmt.Errorf("client: failed to unmarshal eval.Value: %w", err)
		}
		return nil
	})
}

// Realize realizes every output of the derivation at drvPath.
//
// Realize returns pure Go DTOs and closes the Nix store path handles produced
// by Nix before returning. The Client's store must support realization.
func (c *Client) Realize(ctx context.Context, drvPath string) ([]RealizedOutput, error) {
	var result []RealizedOutput
	err := c.middleware(ctx, func() error {
		drvStorePath, err := c.store.ParsePath(drvPath)
		if err != nil {
			return fmt.Errorf("client: failed to realize derivation: parse drv path: %w", err)
		}
		defer drvStorePath.Close() //nolint:errcheck

		realizations, err := c.store.Realise(drvStorePath)
		if err != nil {
			return fmt.Errorf("client: failed to realize derivation: realize store path: %w", err)
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
				return fmt.Errorf("client: failed to realize derivation: realization %q has nil path", realizations[i].OutputName)
			}

			storePath := c.store.PrintPath(realizedPath.Hash(), realizedPath.Name())

			realPath, err := c.store.RealPath(realizedPath)
			if err != nil {
				return fmt.Errorf("client: failed to realize derivation: real output path %q: %w", realizations[i].OutputName, err)
			}

			outputs = append(outputs, RealizedOutput{
				OutputName: realizations[i].OutputName,
				StorePath:  storePath,
				RealPath:   realPath,
				Name:       realizedPath.Name(),
				Hash:       realizedPath.Hash(),
			})
		}

		result = outputs

		return nil
	})
	return result, err
}

// ProcessDaemonConnection processes separate input and output file descriptors.
//
// ProcessDaemonConnection duplicates both descriptors, sets the duplicates to
// blocking mode, and closes only the duplicates after Nix returns.
func (c *Client) ProcessDaemonConnection(ctx context.Context, fromFD, toFD int, trusted, recursive bool) error {
	return c.middleware(ctx, func() error {
		rawCtx, err := c.ctx.Borrow()
		if err != nil {
			return fmt.Errorf("client: failed to borrow context: %w", err)
		}

		rawStore, err := c.store.Borrow()
		if err != nil {
			return fmt.Errorf("client: failed to borrow store: %w", err)
		}

		fromDup, err := duplicateFD(fromFD)
		if err != nil {
			return fmt.Errorf("client: failed to duplicate input fd: %w", err)
		}

		toDup, err := duplicateFD(toFD)
		if err != nil {
			closeErr := closeFD(fromDup)
			return errors.Join(
				fmt.Errorf("client: failed to duplicate output fd: %w", err),
				closeErr,
			)
		}

		code := raw.DaemonProcessConnectionStore(rawCtx, rawStore, int32(fromDup), int32(toDup), trusted, recursive)

		var errs []error
		if ErrorCode(code) != ErrorCodeOK {
			errs = append(errs, fmt.Errorf("client: failed to process connection: %w", status.FromContext(rawCtx)))
		}

		if err := closeFD(fromDup); err != nil {
			errs = append(errs, err)
		}

		if err := closeFD(toDup); err != nil {
			errs = append(errs, err)
		}

		return errors.Join(errs...)
	})
}

// Close releases flakes created by the Client, followed by its evaluator,
// store, settings, and context.
//
// Close is idempotent. It attempts every cleanup and joins multiple errors.
func (c *Client) Close() error {
	if !c.busy.CompareAndSwap(false, true) {
		return ErrConcurrentUse
	}
	defer c.busy.Store(false)

	return c.closeLocked()
}

func (c *Client) closeLocked() error {
	if c.ctx == nil {
		return nil
	}

	errs := make([]error, 0)

	for _, r := range c.resources {
		if err := r.Close(); err != nil {
			errs = append(errs, err)
		}
	}
	c.resources = nil

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

// getOutputValue recursively resolves path starting at root.
//
// When root is nil, resolution starts from the flake output attribute set.
//
// Ownership:
//
// The returned Value is transferred to the caller. Every intermediate Value
// created while descending the attribute path is closed before the function
// returns. Only the final resolved Value remains open.
//
// Example:
//
//	path = ["packages", "x86_64-linux", "hello"]
//
//	OutputAttrs()                 // open
//	  -> Attr("packages")         // close OutputAttrs()
//	  -> Attr("x86_64-linux")     // close packages
//	  -> Attr("hello")            // close x86_64-linux
//	  -> return hello             // caller closes
func (c *Client) getOutputValue(f *flake.Flake, root *eval.Value, path []string) (*eval.Value, error) {
	if root == nil {
		val, err := f.OutputAttrs()
		if err != nil {
			return nil, fmt.Errorf("client: failed to get flake output attrs: %w", err)
		}
		root = val
	}

	if len(path) == 0 {
		// Base case: transfer ownership of the resolved leaf value
		// to the caller. No defer is installed for this value.
		return root, nil
	}
	defer root.Close() //nolint:errcheck

	attr := path[0]

	typ, err := root.Type()
	if err != nil {
		return nil, fmt.Errorf("client: failed to inspect flake output before attribute %q: %w", attr, err)
	}

	if typ != eval.ValueTypeAttrs {
		err := &eval.ValueTypeError{Actual: typ, Expected: eval.ValueTypeAttrs}
		return nil, fmt.Errorf("client: failed to get flake output attribute %q: %w", attr, err)
	}

	child, err := c.evaluator.Attr(root, attr)
	if err != nil {
		return nil, fmt.Errorf("client: failed to get flake output attribute %q: %w", attr, err)
	}

	return c.getOutputValue(f, child, path[1:])
}

func (c *Client) middleware(ctx context.Context, handler func() error) error {
	if !c.busy.CompareAndSwap(false, true) {
		return ErrConcurrentUse
	}
	defer c.busy.Store(false)

	if c.ctx == nil {
		return ErrClosed
	}

	doneChan := make(chan error, 1)
	go func() {
		doneChan <- handler()
	}()

	select {
	case err := <-doneChan:
		if err != nil {
			return fmt.Errorf("middleware: failed to call handler: %w", err)
		}
		return nil
	case <-ctx.Done():
		// start the interrupt process
		util.InterruptRequest()

		interruptCtx, err := nixcontext.New(nixcontext.Config{})
		if err != nil {
			// await for operation to stop
			opErr := <-doneChan
			// clear the interrupt state
			util.InterruptClear()
			// close the client
			closeErr := c.closeLocked()
			return errors.Join(ctx.Err(), err, opErr, closeErr)
		}
		defer interruptCtx.Close() //nolint:errcheck

		// interrupt the remote store (if any)
		interruptErr := util.StoreInterrupt(interruptCtx, c.store)
		// await for operation to stop
		opErr := <-doneChan
		// clear the interrupt state
		util.InterruptClear()
		// close the client
		closeErr := c.closeLocked()

		return errors.Join(ctx.Err(), interruptErr, opErr, closeErr)
	}
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

func duplicateFD(fd int) (int, error) {
	if fd < 0 {
		return -1, fmt.Errorf("fd must be non-negative")
	}
	if fd > math.MaxInt32 {
		return -1, fmt.Errorf("fd %d overflows int32", fd)
	}

	dup, err := syscall.Dup(fd)
	if err != nil {
		return -1, err
	}
	if dup > math.MaxInt32 {
		_ = syscall.Close(dup)
		return -1, fmt.Errorf("duplicated fd %d overflows int32", dup)
	}
	if err := syscall.SetNonblock(dup, false); err != nil {
		_ = syscall.Close(dup)
		return -1, err
	}

	return dup, nil
}

func closeFD(fd int) error {
	if err := syscall.Close(fd); err != nil {
		return fmt.Errorf("client: failed to close duplicate fd %d: %w", fd, err)
	}

	return nil
}
