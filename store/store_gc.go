package store

import (
	"fmt"
	"math"

	"github.com/sund3RRR/gonix/internal/status"
	"github.com/sund3RRR/gonix/pkg/utils"
	"github.com/sund3RRR/gonix/storepath"
	nix "github.com/sund3RRR/nix-go-bindings"
)

// GCAction selects the operation performed by Store.CollectGarbage.
type GCAction int

const (
	// GCReturnLive returns paths reachable from the current GC roots without
	// deleting anything.
	GCReturnLive GCAction = iota

	// GCReturnDead returns paths not reachable from the current GC roots without
	// deleting anything.
	GCReturnDead

	// GCDeleteDead deletes paths not reachable from the current GC roots.
	GCDeleteDead

	// GCDeleteSpecific deletes the paths supplied with WithGCPathsToDelete when
	// Nix determines that they may be deleted.
	GCDeleteSpecific
)

// GCConfig configures Store.CollectGarbage.
//
// CollectGarbage initializes MaxFreed to math.MaxUint64, which means no byte
// limit. Use WithGCMaxFreed to set an explicit limit, including zero.
type GCConfig struct {
	// IgnoreLiveness ignores reachability from GC roots when deleting specific
	// paths. This is dangerous: rooted paths may be deleted.
	IgnoreLiveness bool

	// PathsToDelete contains paths considered by GCDeleteSpecific.
	PathsToDelete []string

	// MaxFreed stops collection after at least this many bytes have been freed.
	MaxFreed uint64
}

// GCOption configures Store.CollectGarbage.
type GCOption func(*GCConfig)

// WithGCIgnoreLiveness controls whether GCDeleteSpecific ignores reachability
// from GC roots.
//
// Enabling this option is dangerous: rooted paths may be deleted. Nix still
// refuses to delete paths referenced by other store paths.
func WithGCIgnoreLiveness(ignore bool) GCOption {
	return func(c *GCConfig) {
		c.IgnoreLiveness = ignore
	}
}

// WithGCPathsToDelete sets the paths considered by GCDeleteSpecific.
//
// The slice is copied. Store.CollectGarbage parses each string as a store path
// and releases the temporary path handles before returning.
func WithGCPathsToDelete(paths ...string) GCOption {
	return func(c *GCConfig) {
		c.PathsToDelete = append([]string(nil), paths...)
	}
}

// WithGCMaxFreed sets the approximate byte limit for garbage collection.
//
// Nix stops after at least maxFreed bytes have been freed. A value of zero is
// an explicit zero-byte limit. When this option is omitted, collection is
// unlimited.
func WithGCMaxFreed(maxFreed uint64) GCOption {
	return func(c *GCConfig) {
		c.MaxFreed = maxFreed
	}
}

// GCResult is the Go-owned result of Store.CollectGarbage.
type GCResult struct {
	// Paths contains the live, dead, or deleted paths reported by Nix, depending
	// on the requested action.
	Paths []string

	// BytesFreed is the number of bytes freed by a deleting action.
	BytesFreed uint64
}

// GCRoot describes one link that keeps a store path alive.
type GCRoot struct {
	// Path is the logical rooted store path.
	Path string

	// Link identifies the filesystem or runtime root reported by Nix.
	Link string
}

// AddTempRoot keeps path alive for the lifetime of the current Nix process or
// store connection.
//
// path must be a full store path for this store.
func (s *Store) AddTempRoot(path string) error {
	if s.ptr == nil {
		return status.ErrClosed
	}

	storePath, err := s.ParsePath(path)
	if err != nil {
		return fmt.Errorf("store: failed to parse temporary GC root path: %w", err)
	}
	defer storePath.Close() //nolint:errcheck

	rawCtx, err := s.ctx.Borrow()
	if err != nil {
		return fmt.Errorf("store: failed to borrow context: %w", err)
	}

	pathPtr, err := storePath.Borrow()
	if err != nil {
		return fmt.Errorf("store: failed to borrow path: %w", err)
	}

	if code := nix.StoreAddTempRoot(rawCtx, s.ptr, pathPtr); status.ErrorCode(code) != status.ErrorCodeOK {
		return fmt.Errorf("store: failed to add temporary GC root: %w", status.FromContext(rawCtx))
	}

	return nil
}

// AddPermanentRoot creates a persistent GC root for path at root.
//
// path must be a full store path for this store. The returned string is the
// canonical root path chosen by Nix.
func (s *Store) AddPermanentRoot(path, root string) (string, error) {
	if s.ptr == nil {
		return "", status.ErrClosed
	}

	storePath, err := s.ParsePath(path)
	if err != nil {
		return "", fmt.Errorf("store: failed to parse permanent GC root path: %w", err)
	}
	defer storePath.Close() //nolint:errcheck

	rawCtx, err := s.ctx.Borrow()
	if err != nil {
		return "", fmt.Errorf("store: failed to borrow context: %w", err)
	}

	pathPtr, err := storePath.Borrow()
	if err != nil {
		return "", fmt.Errorf("store: failed to borrow path: %w", err)
	}

	rootPtr := nix.StoreAddPermanentRoot(rawCtx, s.ptr, pathPtr, root)
	if rootPtr == nil {
		if err := status.FromContext(rawCtx); err != nil {
			return "", fmt.Errorf("store: failed to add permanent GC root: %w", err)
		}
		return "", fmt.Errorf("store: failed to add permanent GC root")
	}

	return utils.TakeCString(rootPtr), nil
}

// FindRoots returns the GC roots known to this store.
//
// When censor is true, Nix may hide details of runtime roots from untrusted
// callers. The returned roots contain only Go-owned strings and require no
// cleanup.
func (s *Store) FindRoots(censor bool) ([]GCRoot, error) {
	if s.ptr == nil {
		return nil, status.ErrClosed
	}

	rawCtx, err := s.ctx.Borrow()
	if err != nil {
		return nil, fmt.Errorf("store: failed to borrow context: %w", err)
	}

	rawRoots := nix.StoreFindRoots(rawCtx, s.ptr, censor)
	if rawRoots == nil {
		if err := status.FromContext(rawCtx); err != nil {
			return nil, fmt.Errorf("store: failed to find GC roots: %w", err)
		}
		return nil, fmt.Errorf("store: failed to find GC roots")
	}
	defer nix.StoreRootsFree(rawRoots)

	roots, err := s.convertRoots(rawCtx, rawRoots)
	if err != nil {
		return nil, fmt.Errorf("store: failed to convert GC roots: %w", err)
	}

	return roots, nil
}

// CollectGarbage queries or deletes paths according to action.
//
// The default collection has no byte limit. Paths supplied through
// WithGCPathsToDelete are parsed and released during the call. The returned
// result contains only Go-owned strings and requires no cleanup.
func (s *Store) CollectGarbage(action GCAction, opts ...GCOption) (GCResult, error) {
	if s.ptr == nil {
		return GCResult{}, status.ErrClosed
	}

	rawAction, err := rawGCAction(action)
	if err != nil {
		return GCResult{}, err
	}

	cfg := GCConfig{MaxFreed: math.MaxUint64}
	for _, opt := range opts {
		opt(&cfg)
	}

	rawCtx, err := s.ctx.Borrow()
	if err != nil {
		return GCResult{}, fmt.Errorf("store: failed to borrow context: %w", err)
	}

	parsedPaths := make([]*storepath.Path, 0, len(cfg.PathsToDelete))
	defer func() {
		for _, path := range parsedPaths {
			_ = path.Close()
		}
	}()

	pathItems := make([]nix.StorePathItem, len(cfg.PathsToDelete))
	for i, path := range cfg.PathsToDelete {
		parsedPath, err := s.ParsePath(path)
		if err != nil {
			return GCResult{}, fmt.Errorf("store: failed to parse GC path %d: %w", i, err)
		}
		parsedPaths = append(parsedPaths, parsedPath)

		pathPtr, err := parsedPath.Borrow()
		if err != nil {
			return GCResult{}, fmt.Errorf("store: failed to borrow GC path %d: %w", i, err)
		}
		pathItems[i].Path = pathPtr
	}

	rawOptions := nix.StoreGCOptions{
		Action:         rawAction,
		IgnoreLiveness: cfg.IgnoreLiveness,
		PathsToDelete: nix.StorePathList{
			Items: pathItems,
			Len:   uint64(len(pathItems)),
		},
		MaxFreed: cfg.MaxFreed,
	}

	rawResults := nix.StoreCollectGarbage(rawCtx, s.ptr, rawOptions)
	if rawResults == nil {
		if err := status.FromContext(rawCtx); err != nil {
			return GCResult{}, fmt.Errorf("store: failed to collect garbage: %w", err)
		}
		return GCResult{}, fmt.Errorf("store: failed to collect garbage")
	}
	defer nix.StoreGCResultsFree(rawResults)

	result, err := convertGCResult(rawCtx, rawResults)
	if err != nil {
		return GCResult{}, fmt.Errorf("store: failed to convert garbage collection result: %w", err)
	}

	return result, nil
}

func rawGCAction(action GCAction) (nix.StoreGCAction, error) {
	switch action {
	case GCReturnLive:
		return nix.StoreGCReturnLive, nil
	case GCReturnDead:
		return nix.StoreGCReturnDead, nil
	case GCDeleteDead:
		return nix.StoreGCDeleteDead, nil
	case GCDeleteSpecific:
		return nix.StoreGCDeleteSpecific, nil
	default:
		return 0, fmt.Errorf("store: invalid garbage collection action %d", action)
	}
}

func (s *Store) convertRoots(rawCtx *nix.NixCContext, rawRoots *nix.StoreRoots) ([]GCRoot, error) {
	roots := make([]GCRoot, 0, nix.StoreRootsCount(rawRoots))

	for i := range nix.StoreRootsCount(rawRoots) {
		pathPtr := nix.StoreRootsPathClone(rawRoots, i)
		if pathPtr == nil {
			if err := status.FromContext(rawCtx); err != nil {
				return nil, fmt.Errorf("clone GC root path %d: %w", i, err)
			}
			return nil, fmt.Errorf("clone GC root path %d", i)
		}

		linkPtr := nix.StoreRootsLink(rawRoots, i)
		if linkPtr == nil {
			nix.StorePathFree(pathPtr)
			if err := status.FromContext(rawCtx); err != nil {
				return nil, fmt.Errorf("read GC root link %d: %w", i, err)
			}
			return nil, fmt.Errorf("read GC root link %d", i)
		}
		link := utils.TakeCString(linkPtr)

		path, err := storepath.New(s.ctx, pathPtr)
		if err != nil {
			nix.StorePathFree(pathPtr)
			return nil, fmt.Errorf("create GC root path %d: %w", i, err)
		}
		pathString := s.PrintPath(path.Hash(), path.Name())
		if err := path.Close(); err != nil {
			return nil, fmt.Errorf("close GC root path %d: %w", i, err)
		}

		roots = append(roots, GCRoot{
			Path: pathString,
			Link: link,
		})
	}

	return roots, nil
}

func convertGCResult(rawCtx *nix.NixCContext, rawResults *nix.StoreGCResults) (GCResult, error) {
	count := nix.StoreGCResultsCount(rawResults)
	result := GCResult{
		Paths:      make([]string, 0, count),
		BytesFreed: nix.StoreGCResultsBytesFreed(rawResults),
	}

	for i := range count {
		pathPtr := nix.StoreGCResultsPath(rawResults, i)
		if pathPtr == nil {
			if err := status.FromContext(rawCtx); err != nil {
				return GCResult{}, fmt.Errorf("read garbage collection path %d: %w", i, err)
			}
			return GCResult{}, fmt.Errorf("read garbage collection path %d", i)
		}
		result.Paths = append(result.Paths, utils.TakeCString(pathPtr))
	}

	return result, nil
}
