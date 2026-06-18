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
	"github.com/sund3RRR/gonix/scripts"
	"github.com/sund3RRR/gonix/store"
	nix "github.com/sund3RRR/nix-go-bindings"
)

const packageProjectionPath = "projections/package.nix"

// Client owns high-level gonix workflow resources.
//
// A Client borrows the Runtime used to create it. Close releases resources
// created through the client, but it does not close the borrowed Runtime.
type Client struct {
	closed            bool
	ctx               *nix.NixCContext
	store             *store.Store
	evaluator         *eval.Evaluator
	packageProjection *eval.Value
	flakeSettings     *flakesettings.Settings
	fetcherSettings   *fetchers.Settings
	resources         []io.Closer
}

// NewClient creates a high-level client using r.
func NewClient(r *Runtime, opts ...ClientOption) (*Client, error) {
	var cfg clientConfig
	for _, opt := range opts {
		opt(&cfg)
	}

	c := Client{
		ctx: r.ctx,
	}

	var err error
	defer func() {
		if err != nil {
			_ = c.Close()
		}
	}()

	if cfg.store != nil {
		c.store = cfg.store
	} else {
		c.store, err = r.OpenStore(store.Auto, store.WithReadOnly(true))
		if err != nil {
			return nil, fmt.Errorf("client: open store: %w", err)
		}
		c.resources = append(c.resources, c.store)
	}

	c.flakeSettings, err = r.GetFlakeSettings()
	if err != nil {
		return nil, fmt.Errorf("client: get flake settings: %w", err)
	}

	c.fetcherSettings, err = r.GetFlakeFetcherSettings()
	if err != nil {
		return nil, fmt.Errorf("client: get fetcher settings: %w", err)
	}

	c.evaluator, err = r.NewEvaluator(c.store, eval.WithFlakeSettings(c.flakeSettings))
	if err != nil {
		return nil, fmt.Errorf("client: new evaluator: %w", err)
	}
	c.resources = append(c.resources, c.evaluator)

	var projection []byte
	projection, err = scripts.Projections.ReadFile(packageProjectionPath)
	if err != nil {
		return nil, fmt.Errorf("client: read package projection: %w", err)
	}
	c.packageProjection, err = c.evaluator.EvalString(string(projection), packageProjectionPath)
	if err != nil {
		return nil, fmt.Errorf("client: evaluate package projection: %w", err)
	}
	c.resources = append(c.resources, c.packageProjection)

	return &c, nil
}

// ParseFlakeRef parses a flake reference and tracks it for Client.Close.
func (c *Client) ParseFlakeRef(ref string, opts ...flake.ParseOption) (*flake.Ref, error) {
	if c.closed {
		return nil, status.ErrClosed
	}

	parsed, err := flake.NewParsedRef(c.ctx, c.fetcherSettings, c.flakeSettings, ref, opts...)
	if err != nil {
		return nil, fmt.Errorf("client: parse flake ref: %w", err)
	}
	c.resources = append(c.resources, parsed)

	return parsed, nil
}

// LockFlake locks a flake reference and tracks it for Client.Close.
func (c *Client) LockFlake(ref *flake.Ref, opts ...flake.LockOption) (*flake.LockedFlake, error) {
	if c.closed {
		return nil, status.ErrClosed
	}

	locked, err := flake.NewLockedFlake(c.ctx, c.fetcherSettings, c.flakeSettings, c.evaluator, ref, opts...)
	if err != nil {
		return nil, fmt.Errorf("client: lock flake: %w", err)
	}
	c.resources = append(c.resources, locked)

	return locked, nil
}

// FetchPackage fetches a package by name from locked and decodes it.
func (c *Client) FetchPackage(locked *flake.LockedFlake, name string, opts ...FetchPackageOption) (Package, error) {
	if c.closed {
		return Package{}, status.ErrClosed
	}

	var cfg fetchPackageConfig
	for _, opt := range opts {
		opt(&cfg)
	}

	outputs, err := locked.OutputAttrs()
	if err != nil {
		return Package{}, fmt.Errorf("client: fetch package: output attrs: %w", err)
	}
	defer func() {
		_ = outputs.Close()
	}()

	args := map[string]eval.GoValue{
		"outputs": eval.Copy(outputs),
		"name":    eval.String(name),
	}
	if cfg.system != "" {
		args["system"] = eval.String(cfg.system)
	}

	arg, err := c.evaluator.NewValue(eval.Attrs(args))
	if err != nil {
		return Package{}, fmt.Errorf("client: fetch package: build projection argument: %w", err)
	}
	defer func() {
		_ = arg.Close()
	}()

	value, err := c.evaluator.Call(c.packageProjection, arg)
	if err != nil {
		return Package{}, fmt.Errorf("client: fetch package: call projection: %w", err)
	}
	defer func() {
		_ = value.Close()
	}()

	var pkg Package
	if err := c.evaluator.Unmarshal(value, &pkg); err != nil {
		return Package{}, fmt.Errorf("client: fetch package: unmarshal package: %w", err)
	}

	return pkg, nil
}

// DownloadPackage fetches a package by name and realizes all of its outputs.
//
// DownloadPackage returns pure Go DTOs and closes the Nix store path handles
// produced while realizing the package before returning. The client store must
// support realization; callers that need writes should pass a writable store to
// NewClient with WithClientStore.
func (c *Client) DownloadPackage(locked *flake.LockedFlake, name string, opts ...DownloadPackageOption) ([]DownloadedPackageOutput, error) {
	if c.closed {
		return nil, status.ErrClosed
	}

	var cfg downloadPackageConfig
	for _, opt := range opts {
		opt(&cfg)
	}

	fetchOpts := make([]FetchPackageOption, 0, 1)
	if cfg.system != "" {
		fetchOpts = append(fetchOpts, WithFetchPackageSystem(cfg.system))
	}

	pkg, err := c.FetchPackage(locked, name, fetchOpts...)
	if err != nil {
		return nil, fmt.Errorf("client: download package: fetch package: %w", err)
	}
	if pkg.Type != "" && pkg.Type != PackageTypeDerivation {
		return nil, fmt.Errorf("client: download package: unsupported package type %q", pkg.Type)
	}
	if pkg.DrvPath == "" {
		return nil, fmt.Errorf("client: download package: missing drvPath")
	}

	drvPath, err := c.store.ParsePath(pkg.DrvPath)
	if err != nil {
		return nil, fmt.Errorf("client: download package: parse drv path: %w", err)
	}
	defer func() {
		_ = drvPath.Close()
	}()

	realizations, err := c.store.Realise(drvPath)
	if err != nil {
		return nil, fmt.Errorf("client: download package: realise package: %w", err)
	}
	defer func() {
		for i := range realizations {
			_ = realizations[i].Close()
		}
	}()

	outputs := make([]DownloadedPackageOutput, 0, len(realizations))
	for i := range realizations {
		realizedPath := realizations[i].Path
		if realizedPath == nil {
			return nil, fmt.Errorf("client: download package: realization %q has nil path", realizations[i].OutputName)
		}

		storePath, err := c.store.PrintPath(realizedPath)
		if err != nil {
			return nil, fmt.Errorf("client: download package: print output path %q: %w", realizations[i].OutputName, err)
		}

		realPath, err := c.store.RealPath(realizedPath)
		if err != nil {
			return nil, fmt.Errorf("client: download package: real output path %q: %w", realizations[i].OutputName, err)
		}

		outputs = append(outputs, DownloadedPackageOutput{
			OutputName: realizations[i].OutputName,
			StorePath:  storePath,
			RealPath:   realPath,
			Name:       realizedPath.Name(),
			Hash:       realizedPath.Hash(),
		})
	}

	return outputs, nil
}

// Close releases resources created through c.
//
// Close is idempotent. It does not close the Runtime passed to NewClient.
func (c *Client) Close() error {
	if c.closed {
		return nil
	}

	errs := make([]error, 0, len(c.resources)+3)
	for i := len(c.resources) - 1; i >= 0; i-- {
		if err := c.resources[i].Close(); err != nil {
			errs = append(errs, err)
		}
	}

	c.ctx = nil
	c.store = nil
	c.evaluator = nil
	c.packageProjection = nil
	c.flakeSettings = nil
	c.fetcherSettings = nil
	c.resources = nil

	c.closed = true

	if len(errs) != 0 {
		return fmt.Errorf("client: failed to close resources: %w", errors.Join(errs...))
	}

	return nil
}
