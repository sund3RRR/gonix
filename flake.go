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
	"github.com/sund3RRR/gonix/scripts"
	"github.com/sund3RRR/gonix/store"
)

const (
	packageProjectionPath    = "projections/package.nix"
	maintainerProjectionPath = "projections/maintainer.nix"
)

// Flake is a parsed and locked flake with high-level output workflows.
//
// Flake owns its parsed reference, lock, and package projections. It borrows
// the Store, Evaluator, settings, and Context used to create it, all of which
// must remain open until the Flake is closed.
type Flake struct {
	parsedRef            *flake.Ref
	lock                 *flake.LockedFlake
	packageProjection    *eval.Value
	maintainerProjection *eval.Value
	defaultSystem        string
	store                *store.Store
	evaluator            *eval.Evaluator
}

// NewFlake parses and locks a flake using an explicitly assembled resource graph.
//
// This is an advanced constructor. ctx, fetchSettings, flakeSettings, s, and e
// are borrowed and must all belong to the same Nix context and outlive the
// returned Flake. The Flake owns its parsed reference, locked flake, and
// package projections and must be closed by the caller.
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
		return nil, fmt.Errorf("flake: failed to parse ref: %w", err)
	}

	f.lock, err = flake.NewLockedFlake(ctx, fetchSettings, flakeSettings, e, f.parsedRef, cfg.lockOpts...)
	if err != nil {
		return nil, fmt.Errorf("flake: failed to lock flake: %w", err)
	}

	var projection []byte
	projection, err = scripts.Projections.ReadFile(packageProjectionPath)
	if err != nil {
		return nil, fmt.Errorf("flake: failed to read package projection: %w", err)
	}
	f.packageProjection, err = e.EvalString(string(projection), packageProjectionPath)
	if err != nil {
		return nil, fmt.Errorf("flake: failed to evaluate package projection: %w", err)
	}

	projection, err = scripts.Projections.ReadFile(maintainerProjectionPath)
	if err != nil {
		return nil, fmt.Errorf("flake: failed to read maintainer projection: %w", err)
	}
	f.maintainerProjection, err = e.EvalString(string(projection), maintainerProjectionPath)
	if err != nil {
		return nil, fmt.Errorf("flake: failed to evaluate maintainer projection: %w", err)
	}

	f.defaultSystem, err = currentSystem(e)
	if err != nil {
		return nil, fmt.Errorf("flake: failed to get current system: %w", err)
	}

	return f, nil
}

// Output decodes a locked flake output selected by path into out.
//
// Each path element names one exact Nix attribute; dots have no special
// meaning. An empty path decodes the complete flake output attribute set. The
// out argument must be a non-nil pointer to a type supported by
// eval.Evaluator.Unmarshal.
func (f *Flake) Output(path []string, out any) (err error) {
	if f.lock == nil {
		return status.ErrClosed
	}

	root, err := f.lock.OutputAttrs()
	if err != nil {
		return fmt.Errorf("flake: failed to get output attributes: %w", err)
	}

	values := []*eval.Value{root}
	defer func() {
		closeErrs := make([]error, 0, len(values))
		for i := len(values) - 1; i >= 0; i-- {
			if closeErr := values[i].Close(); closeErr != nil {
				closeErrs = append(closeErrs, closeErr)
			}
		}
		if len(closeErrs) == 0 {
			return
		}

		closeErr := fmt.Errorf("flake: failed to close output values: %w", errors.Join(closeErrs...))
		if err != nil {
			err = errors.Join(err, closeErr)
			return
		}
		err = closeErr
	}()

	value := root
	for _, attr := range path {
		typ, typeErr := value.Type()
		if typeErr != nil {
			return fmt.Errorf("flake: failed to inspect output before attribute %q: %w", attr, typeErr)
		}
		if typ != eval.ValueTypeAttrs {
			return fmt.Errorf(
				"flake: failed to get output attribute %q: %w",
				attr,
				&eval.ValueTypeError{Actual: typ, Expected: eval.ValueTypeAttrs},
			)
		}

		value, err = f.evaluator.Attr(value, attr)
		if err != nil {
			return fmt.Errorf("flake: failed to get output attribute %q: %w", attr, err)
		}
		values = append(values, value)
	}

	if err := f.evaluator.Unmarshal(value, out); err != nil {
		return fmt.Errorf("flake: failed to decode output: %w", err)
	}

	return nil
}

// ListPackages returns sorted top-level package references for one flake system.
//
// ListPackages enumerates names from packages.<system> without forcing or
// decoding individual package values. Missing packages or system attributes
// produce an empty result.
func (f *Flake) ListPackages(opts ...ListPackagesOption) (refs []PackageRef, err error) {
	if f == nil || f.lock == nil {
		return nil, status.ErrClosed
	}

	var cfg listPackagesConfig
	for _, opt := range opts {
		opt(&cfg)
	}

	system := f.packageSystem(cfg.system)

	root, err := f.lock.OutputAttrs()
	if err != nil {
		return nil, fmt.Errorf("flake: failed to list packages: get output attributes: %w", err)
	}

	values := []*eval.Value{root}
	defer func() {
		closeErrs := make([]error, 0, len(values))
		for i := len(values) - 1; i >= 0; i-- {
			if closeErr := values[i].Close(); closeErr != nil {
				closeErrs = append(closeErrs, closeErr)
			}
		}
		if len(closeErrs) == 0 {
			return
		}

		closeErr := fmt.Errorf("flake: failed to list packages: close values: %w", errors.Join(closeErrs...))
		if err != nil {
			err = errors.Join(err, closeErr)
			return
		}
		err = closeErr
	}()

	if err := f.requireAttrs(root, "outputs"); err != nil {
		return nil, err
	}
	hasPackages, err := f.evaluator.HasAttr(root, "packages")
	if err != nil {
		return nil, fmt.Errorf("flake: failed to list packages: check packages output: %w", err)
	}
	if !hasPackages {
		return []PackageRef{}, nil
	}

	packages, err := f.evaluator.AttrLazy(root, "packages")
	if err != nil {
		return nil, fmt.Errorf("flake: failed to list packages: get packages output: %w", err)
	}
	values = append(values, packages)
	if err := f.requireAttrs(packages, "packages"); err != nil {
		return nil, err
	}

	hasSystem, err := f.evaluator.HasAttr(packages, system)
	if err != nil {
		return nil, fmt.Errorf("flake: failed to list packages: check system %q: %w", system, err)
	}
	if !hasSystem {
		return []PackageRef{}, nil
	}

	systemPackages, err := f.evaluator.AttrLazy(packages, system)
	if err != nil {
		return nil, fmt.Errorf("flake: failed to list packages: get system %q: %w", system, err)
	}
	values = append(values, systemPackages)
	if err := f.requireAttrs(systemPackages, fmt.Sprintf("packages.%s", system)); err != nil {
		return nil, err
	}

	count, err := systemPackages.AttrLen()
	if err != nil {
		return nil, fmt.Errorf("flake: failed to list packages: get package count: %w", err)
	}

	names := make([]string, count)
	for i := range count {
		names[i], err = f.evaluator.AttrName(systemPackages, i)
		if err != nil {
			return nil, fmt.Errorf("flake: failed to list packages: get package name %d: %w", i, err)
		}
	}
	sort.Strings(names)

	refs = make([]PackageRef, len(names))
	for i, name := range names {
		refs[i] = PackageRef{Name: name, System: system}
	}
	return refs, nil
}

func (f *Flake) packageSystem(explicit string) string {
	if explicit != "" {
		return explicit
	}

	return f.defaultSystem
}

func currentSystem(e *eval.Evaluator) (system string, err error) {
	value, err := e.EvalString("builtins.currentSystem", "<gonix/current-system>")
	if err != nil {
		return "", fmt.Errorf("evaluate builtins.currentSystem: %w", err)
	}
	defer func() {
		if closeErr := value.Close(); closeErr != nil {
			closeErr = fmt.Errorf("close current-system value: %w", closeErr)
			if err != nil {
				err = errors.Join(err, closeErr)
				return
			}
			err = closeErr
		}
	}()

	if err = e.Force(value); err != nil {
		return "", fmt.Errorf("force builtins.currentSystem: %w", err)
	}
	system, err = value.String()
	if err != nil {
		return "", fmt.Errorf("decode builtins.currentSystem: %w", err)
	}
	return system, nil
}

func (f *Flake) requireAttrs(value *eval.Value, name string) error {
	if err := f.evaluator.Force(value); err != nil {
		return fmt.Errorf("flake: failed to list packages: force %s: %w", name, err)
	}
	typ, err := value.Type()
	if err != nil {
		return fmt.Errorf("flake: failed to list packages: get %s type: %w", name, err)
	}
	if typ != eval.ValueTypeAttrs {
		return fmt.Errorf(
			"flake: failed to list packages: %s: %w",
			name,
			&eval.ValueTypeError{Actual: typ, Expected: eval.ValueTypeAttrs},
		)
	}
	return nil
}

// FetchPackage fetches a package by name from locked and decodes it.
//
// Maintainer metadata is best-effort. If its independent projection cannot be
// evaluated or decoded, FetchPackage succeeds with an empty maintainer list.
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
		return Package{}, fmt.Errorf("flake: failed to fetch package: output attrs: %w", err)
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
		return Package{}, fmt.Errorf("flake: failed to fetch package: build projection argument: %w", err)
	}
	defer arg.Close() //nolint:errcheck

	value, err := c.evaluator.Call(c.packageProjection, arg)
	if err != nil {
		return Package{}, fmt.Errorf("flake: failed to fetch package: call projection: %w", err)
	}
	defer value.Close() //nolint:errcheck

	var pkg Package
	if err := c.evaluator.Unmarshal(value, &pkg); err != nil {
		return Package{}, fmt.Errorf("flake: failed to fetch package: unmarshal package: %w", err)
	}

	pkg.Meta.Maintainers = []Maintainer{}
	maintainerValue, err := c.evaluator.Call(c.maintainerProjection, arg)
	if err != nil {
		return pkg, nil
	}
	defer maintainerValue.Close() //nolint:errcheck

	var maintainers []Maintainer
	if err := c.evaluator.Unmarshal(maintainerValue, &maintainers); err != nil {
		return pkg, nil
	}
	if maintainers != nil {
		pkg.Meta.Maintainers = maintainers
	}

	return pkg, nil
}

// RealizePackage realizes all outputs of pkg.
//
// RealizePackage returns pure Go DTOs and closes the Nix store path handles
// produced while realizing the package before returning. The store used to
// create the Flake must support realization.
func (c *Flake) RealizePackage(pkg Package) ([]RealizedPackageOutput, error) {
	if c.lock == nil {
		return nil, status.ErrClosed
	}

	if pkg.Type != "" && pkg.Type != PackageTypeDerivation {
		return nil, fmt.Errorf("flake: failed to realize package: unsupported package type %q", pkg.Type)
	}
	if pkg.DrvPath == "" {
		return nil, fmt.Errorf("flake: failed to realize package: missing drvPath")
	}

	drvPath, err := c.store.ParsePath(pkg.DrvPath)
	if err != nil {
		return nil, fmt.Errorf("flake: failed to realize package: parse drv path: %w", err)
	}
	defer drvPath.Close() //nolint:errcheck

	realizations, err := c.store.Realise(drvPath)
	if err != nil {
		return nil, fmt.Errorf("flake: failed to realize package: realize store path: %w", err)
	}
	defer func() {
		for i := range realizations {
			_ = realizations[i].Close()
		}
	}()

	outputs := make([]RealizedPackageOutput, 0, len(realizations))
	for i := range realizations {
		realizedPath := realizations[i].Path
		if realizedPath == nil {
			return nil, fmt.Errorf("flake: failed to realize package: realization %q has nil path", realizations[i].OutputName)
		}

		storePath := c.store.PrintPath(realizedPath.Hash(), realizedPath.Name())

		realPath, err := c.store.RealPath(realizedPath)
		if err != nil {
			return nil, fmt.Errorf("flake: failed to realize package: real output path %q: %w", realizations[i].OutputName, err)
		}

		outputs = append(outputs, RealizedPackageOutput{
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
	if f == nil || (f.parsedRef == nil && f.lock == nil) {
		return nil
	}

	errs := make([]error, 0, 4)

	if err := f.maintainerProjection.Close(); err != nil {
		errs = append(errs, err)
	}
	f.maintainerProjection = nil

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
