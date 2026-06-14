package store

import (
	"errors"
	"fmt"

	"github.com/sund3RRR/gonix/internal/status"
	"github.com/sund3RRR/gonix/internal/utils"
	"github.com/sund3RRR/gonix/storepath"
	nix "github.com/sund3RRR/nix-go-bindings"
)

// Closure contains the store paths in a filesystem closure.
//
// A filesystem closure is the set of paths reachable from a root path by Nix's
// reference graph, optionally including outputs, derivers, or reverse
// referrers. Closure owns every path in Paths; call Close when done.
type Closure struct {
	// Paths are the owned store paths in the closure.
	Paths []*storepath.Path
}

// Close releases every owned path in c.
//
// Close attempts to close every path and returns a joined error if one or more
// closes fail. It is safe to call more than once.
func (c *Closure) Close() error {
	if len(c.Paths) == 0 {
		return nil
	}

	errs := make([]error, 0, len(c.Paths))
	for _, path := range c.Paths {
		if err := path.Close(); err != nil {
			errs = append(errs, err)
		}
	}
	c.Paths = nil

	if len(errs) != 0 {
		return fmt.Errorf("closure: failed to close paths: %w", errors.Join(errs...))
	}

	return nil
}

// ClosureConfig configures filesystem closure traversal.
//
// The zero value asks Nix for the forward reference closure of the root path
// without adding derivation outputs or derivers.
type ClosureConfig struct {
	// Reverse traverses referrers instead of references.
	Reverse bool

	// IncludeOutputs includes derivation outputs in the closure.
	IncludeOutputs bool

	// IncludeDerivers includes derivations that produced paths in the closure.
	IncludeDerivers bool
}

// ClosureOption configures Store.Closure.
type ClosureOption func(*ClosureConfig)

// WithClosureReverse traverses referrers instead of references.
//
// The default is false, which follows references from the root path to its
// dependencies.
func WithClosureReverse(reverse bool) ClosureOption {
	return func(c *ClosureConfig) {
		c.Reverse = reverse
	}
}

// WithClosureOutputs includes derivation outputs in the closure.
func WithClosureOutputs(include bool) ClosureOption {
	return func(c *ClosureConfig) {
		c.IncludeOutputs = include
	}
}

// WithClosureDerivers includes derivations that produced paths in the closure.
func WithClosureDerivers(include bool) ClosureOption {
	return func(c *ClosureConfig) {
		c.IncludeDerivers = include
	}
}

// Realization describes one realized derivation output.
//
// Realise returns one Realization per output produced by Nix. The Path is an
// owned StorePath clone and must be closed, either directly through Path.Close
// or by calling Realization.Close.
type Realization struct {
	// OutputName is the derivation output name.
	OutputName string

	// Path is the realized store path for the output.
	Path *storepath.Path
}

// Close releases the realized output path.
//
// Close is safe to call more than once.
func (r *Realization) Close() error {
	if r.Path == nil {
		return nil
	}

	err := r.Path.Close()
	r.Path = nil
	if err != nil {
		return fmt.Errorf("store: failed to close realization: %w", err)
	}

	return nil
}

// Realise builds or substitutes path and returns its realized outputs.
//
// The path is borrowed for the duration of the call. Returned realization paths
// are owned by the caller. If Nix returns several outputs, each output appears
// as a separate Realization with its output name.
func (s *Store) Realise(path *storepath.Path) ([]Realization, error) {
	if s.ptr == nil {
		return nil, status.ErrClosed
	}

	pathPtr, err := path.Borrow()
	if err != nil {
		return nil, fmt.Errorf("store: failed to borrow path: %w", err)
	}

	results := nix.StoreRealiseToArray(s.ctx, s.ptr, pathPtr)
	if results == nil {
		return nil, fmt.Errorf("store: failed to realise to array: %w", status.FromContext(s.ctx))
	}
	defer nix.StoreRealiseResultsFree(results)

	realizations, err := convertRealiseResults(s.ctx, results)
	if err != nil {
		return nil, fmt.Errorf("store: failed to convert realise results: %w", err)
	}

	return realizations, nil
}

// Closure returns the filesystem closure for path.
//
// The path is borrowed for the duration of the call. The returned Closure owns
// its Paths and must be closed by the caller. Use ClosureOption values to ask
// Nix for reverse closures, outputs, or derivers.
func (s *Store) Closure(path *storepath.Path, opts ...ClosureOption) (*Closure, error) {
	if s.ptr == nil {
		return nil, status.ErrClosed
	}

	var cfg ClosureConfig
	for _, opt := range opts {
		opt(&cfg)
	}

	pathPtr, err := path.Borrow()
	if err != nil {
		return nil, fmt.Errorf("store: failed to borrow path: %w", err)
	}

	paths := nix.StoreGetFsClosureArray(s.ctx, s.ptr, pathPtr, cfg.Reverse, cfg.IncludeOutputs, cfg.IncludeDerivers)
	if paths == nil {
		return nil, fmt.Errorf("store: failed to get fs closure array: %w", status.FromContext(s.ctx))
	}
	defer nix.StorePathArrayFree(paths)

	closure, err := convertClosurePaths(s.ctx, paths)
	if err != nil {
		return nil, fmt.Errorf("store: failed to convert closure paths: %w", err)
	}

	return closure, nil
}

func convertClosurePaths(ctx *nix.NixCContext, paths *nix.StorePathArray) (*Closure, error) {
	count := nix.StorePathArrayCount(paths)
	closure := &Closure{
		Paths: make([]*storepath.Path, 0, count),
	}

	for i := range count {
		pathPtr := nix.StorePathArrayPathClone(paths, i)
		if pathPtr == nil {
			return nil, fmt.Errorf("store: failed to clone closure path from array %d: %w", i, status.FromContext(ctx))
		}

		closure.Paths = append(closure.Paths, storepath.New(ctx, pathPtr))
	}

	return closure, nil
}

func convertRealiseResults(ctx *nix.NixCContext, results *nix.StoreRealiseResults) ([]Realization, error) {
	count := nix.StoreRealiseResultsCount(results)
	realizations := make([]Realization, 0, count)

	for i := range count {
		outputNamePtr := nix.StoreRealiseResultsOutname(results, i)
		if outputNamePtr == nil {
			return nil, fmt.Errorf("store: failed to realize result %d output name: %w", i, status.FromContext(ctx))
		}
		outputName := utils.TakeCString(outputNamePtr)

		pathPtr := nix.StoreRealiseResultsPathClone(results, i)
		if pathPtr == nil {
			return nil, fmt.Errorf("store: failed to realize result %d path: %w", i, status.FromContext(ctx))
		}

		realizations = append(realizations, Realization{
			OutputName: outputName,
			Path:       storepath.New(ctx, pathPtr),
		})
	}

	return realizations, nil
}
