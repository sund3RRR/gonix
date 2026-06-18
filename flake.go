package gonix

import (
	"errors"
	"fmt"

	"github.com/sund3RRR/gonix/eval"
	"github.com/sund3RRR/gonix/fetchers"
	"github.com/sund3RRR/gonix/flake"
	"github.com/sund3RRR/gonix/flakesettings"
	"github.com/sund3RRR/gonix/internal/status"
	"github.com/sund3RRR/gonix/nixcontext"
	"github.com/sund3RRR/gonix/scripts"
	"github.com/sund3RRR/gonix/store"
)

const packageProjectionPath = "projections/package.nix"

// Flake is a parsed and locked flake with high-level output workflows.
//
// Flake owns its parsed reference, lock, and package projection. It borrows the
// Store, Evaluator, settings, and Context used to create it.
type Flake struct {
	parsedRef         *flake.Ref
	lock              *flake.LockedFlake
	packageProjection *eval.Value
	store             *store.Store
	evaluator         *eval.Evaluator
}

// NewFlake parses and locks a flake using an explicitly assembled resource graph.
//
// This is an advanced constructor. ctx, fetchSettings, flakeSettings, s, and e
// are borrowed and must all belong to the same Nix context and outlive the
// returned Flake. The Flake owns its parsed reference, locked flake, and package
// projection and must be closed by the caller.
func NewFlake(
	ctx *nixcontext.Context,
	fetchSettings *fetchers.Settings,
	flakeSettings *flakesettings.Settings,
	s *store.Store,
	e *eval.Evaluator,
	ref string,
	opts ...FlakeOption,
) (*Flake, error) {
	var cfg flakeConfig
	for _, opt := range opts {
		opt(&cfg)
	}

	f := &Flake{
		store:     s,
		evaluator: e,
	}

	var err error
	defer func() {
		if err != nil {
			_ = f.Close()
		}
	}()

	f.parsedRef, err = flake.NewParsedRef(ctx, fetchSettings, flakeSettings, ref, cfg.parseOpts...)
	if err != nil {
		return nil, fmt.Errorf("flake: parse ref: %w", err)
	}

	f.lock, err = flake.NewLockedFlake(ctx, fetchSettings, flakeSettings, e, f.parsedRef, cfg.lockOpts...)
	if err != nil {
		return nil, fmt.Errorf("flake: lock flake: %w", err)
	}

	var projection []byte
	projection, err = scripts.Projections.ReadFile(packageProjectionPath)
	if err != nil {
		return nil, fmt.Errorf("flake: read package projection: %w", err)
	}
	f.packageProjection, err = e.EvalString(string(projection), packageProjectionPath)
	if err != nil {
		return nil, fmt.Errorf("flake: evaluate package projection: %w", err)
	}

	return f, nil
}

// FetchPackage fetches a package by name from locked and decodes it.
func (c *Flake) FetchPackage(name string, opts ...FetchPackageOption) (Package, error) {
	if c.lock == nil {
		return Package{}, status.ErrClosed
	}

	var cfg fetchPackageConfig
	for _, opt := range opts {
		opt(&cfg)
	}

	outputs, err := c.lock.OutputAttrs()
	if err != nil {
		return Package{}, fmt.Errorf("flake: fetch package: output attrs: %w", err)
	}
	defer outputs.Close() //nolint:errcheck

	args := map[string]eval.GoValue{
		"outputs": eval.Copy(outputs),
		"name":    eval.String(name),
	}
	if cfg.system != "" {
		args["system"] = eval.String(cfg.system)
	}

	arg, err := c.evaluator.NewValue(eval.Attrs(args))
	if err != nil {
		return Package{}, fmt.Errorf("flake: fetch package: build projection argument: %w", err)
	}
	defer arg.Close() //nolint:errcheck

	value, err := c.evaluator.Call(c.packageProjection, arg)
	if err != nil {
		return Package{}, fmt.Errorf("flake: fetch package: call projection: %w", err)
	}
	defer value.Close() //nolint:errcheck

	var pkg Package
	if err := c.evaluator.Unmarshal(value, &pkg); err != nil {
		return Package{}, fmt.Errorf("flake: fetch package: unmarshal package: %w", err)
	}

	return pkg, nil
}

// DownloadPackage realizes all outputs of pkg.
//
// DownloadPackage returns pure Go DTOs and closes the Nix store path handles
// produced while realizing the package before returning. The store used to
// create the Flake must support realization.
func (c *Flake) DownloadPackage(pkg Package) ([]DownloadedPackageOutput, error) {
	if c.lock == nil {
		return nil, status.ErrClosed
	}

	if pkg.Type != "" && pkg.Type != PackageTypeDerivation {
		return nil, fmt.Errorf("flake: download package: unsupported package type %q", pkg.Type)
	}
	if pkg.DrvPath == "" {
		return nil, fmt.Errorf("flake: download package: missing drvPath")
	}

	drvPath, err := c.store.ParsePath(pkg.DrvPath)
	if err != nil {
		return nil, fmt.Errorf("flake: download package: parse drv path: %w", err)
	}
	defer drvPath.Close() //nolint:errcheck

	realizations, err := c.store.Realise(drvPath)
	if err != nil {
		return nil, fmt.Errorf("flake: download package: realise package: %w", err)
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
			return nil, fmt.Errorf("flake: download package: realization %q has nil path", realizations[i].OutputName)
		}

		storePath, err := c.store.PrintPath(realizedPath)
		if err != nil {
			return nil, fmt.Errorf("flake: download package: print output path %q: %w", realizations[i].OutputName, err)
		}

		realPath, err := c.store.RealPath(realizedPath)
		if err != nil {
			return nil, fmt.Errorf("flake: download package: real output path %q: %w", realizations[i].OutputName, err)
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

// Close releases resources owned by f.
//
// Close is idempotent and also cleans up a partially initialized Flake.
func (f *Flake) Close() error {
	if f == nil || f.lock == nil {
		return nil
	}

	errs := make([]error, 0, 3)

	if err := f.packageProjection.Close(); err != nil {
		errs = append(errs, err)
	}
	f.packageProjection = nil

	if err := f.lock.Close(); err != nil {
		errs = append(errs, err)
	}
	f.lock = nil

	if err := f.parsedRef.Close(); err != nil {
		errs = append(errs, err)
	}
	f.parsedRef = nil

	if len(errs) != 0 {
		return fmt.Errorf("flake: failed to close resources: %w", errors.Join(errs...))
	}

	return nil
}
