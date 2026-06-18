// Package store wraps Nix store handles and store-backed operations.
//
// The package is the high-level entry point for operations that need an open
// Nix store: path parsing, metadata lookup, derivation import/export,
// realization, closure traversal, and copying paths between stores. Returned
// resource wrappers own their underlying Nix handles unless their documentation
// says otherwise, and must be closed by the caller.
package store

import (
	"fmt"

	"github.com/sund3RRR/gonix/internal/status"
	"github.com/sund3RRR/gonix/nixcontext"
	"github.com/sund3RRR/gonix/pkg/utils"
	"github.com/sund3RRR/gonix/storepath"
	nix "github.com/sund3RRR/nix-go-bindings"
)

// Store is an open Nix store backend.
//
// A Store owns the underlying Nix store handle and borrows the Nix context used
// to create it. Store methods are the place for operations whose meaning
// depends on a particular backend, such as formatting real paths, checking
// validity, realizing paths, or copying paths to another store.
//
// A Store must be closed when the caller is done with it. Close is idempotent,
// and methods that need the raw Nix handle return an error wrapping
// status.ErrClosed after Close. URI, StoreDir, Version, and PrintPath use
// metadata cached during construction and remain available after Close.
// Store is not documented as goroutine-safe.
type Store struct {
	uri      string
	storeDir string
	version  string
	ctx      *nixcontext.Context
	ptr      *nix.Store
}

// New opens a Nix store using an already-initialized Nix context.
//
// The uri is passed to Nix as the store URI, for example "dummy://",
// "local", or a remote store URI supported by the linked Nix libraries. Options
// are serialized as store parameters, equivalent to URL query parameters in the
// low-level Nix API.
//
// New caches the store's canonical URI, logical store directory, and backend
// version before returning. The returned Store owns the raw Nix store handle
// and must be closed by the caller. The ctx argument is borrowed and must
// remain valid for as long as operations requiring the raw handle are used.
func New(ctx *nixcontext.Context, uri string, opts ...Option) (*Store, error) {
	var cfg Config
	for _, opt := range opts {
		opt(&cfg)
	}

	rawCtx, err := ctx.Borrow()
	if err != nil {
		return nil, fmt.Errorf("store: failed to borrow context: %w", err)
	}

	params, err := cfg.Params()
	if err != nil {
		return nil, fmt.Errorf("store: failed to convert config to params: %w", err)
	}

	items := make([]nix.StoreParam, 0, len(params))
	for k, v := range params {
		items = append(items, nix.StoreParam{
			Key:      []byte(k),
			KeyLen:   uint64(len(k)),
			Value:    []byte(v),
			ValueLen: uint64(len(v)),
		})
	}

	storeParams := nix.StoreParams{
		Items: items,
		Len:   uint64(len(items)),
	}

	if storeParams.Len > 0 {
		_, _ = storeParams.PassRef()
		defer storeParams.Free()
	}

	ptr := nix.StoreOpen(rawCtx, uri, storeParams)
	if ptr == nil {
		return nil, fmt.Errorf("store: failed to open store: %w", status.FromContext(rawCtx))
	}

	uriPtr := nix.StoreGetUri(rawCtx, ptr)
	if uriPtr == nil {
		nix.StoreFree(ptr)
		if err := status.FromContext(rawCtx); err != nil {
			return nil, fmt.Errorf("store: failed to get uri: %w", err)
		}
		return nil, fmt.Errorf("store: failed to get uri")
	}
	canonicalURI := utils.TakeCString(uriPtr)

	storeDirPtr := nix.StoreGetStoredir(rawCtx, ptr)
	if storeDirPtr == nil {
		nix.StoreFree(ptr)
		if err := status.FromContext(rawCtx); err != nil {
			return nil, fmt.Errorf("store: failed to get store dir: %w", err)
		}
		return nil, fmt.Errorf("store: failed to get store dir")
	}
	storeDir := utils.TakeCString(storeDirPtr)

	versionPtr := nix.StoreGetVersion(rawCtx, ptr)
	if versionPtr == nil {
		nix.StoreFree(ptr)
		if err := status.FromContext(rawCtx); err != nil {
			return nil, fmt.Errorf("store: failed to get version: %w", err)
		}
		return nil, fmt.Errorf("store: failed to get version")
	}
	version := utils.TakeCString(versionPtr)

	return &Store{
		uri:      canonicalURI,
		storeDir: storeDir,
		version:  version,
		ctx:      ctx,
		ptr:      ptr,
	}, nil
}

// Version returns the cached store backend version.
//
// Some store backends do not expose a version. In that case Version returns an
// empty string. The cached value remains available after Store or Context
// closure.
func (s *Store) Version() string {
	return s.version
}

// PrintPath formats a hash and name as a logical store path for this store.
//
// PrintPath uses the cached logical store directory and does not validate name.
// It remains available after Store or Context closure. For rooted or redirected
// stores, use RealPath when the concrete filesystem path is needed.
func (s *Store) PrintPath(hash [20]byte, name string) string {
	return s.storeDir + "/" + utils.EncodeNix32(hash) + "-" + name
}

// URI returns the store's cached canonical URI.
//
// The returned value is produced by Nix and may be normalized from the URI that
// was passed to New. The cached value remains available after Store or Context
// closure.
func (s *Store) URI() string {
	return s.uri
}

// StoreDir returns the store's cached logical Nix store directory.
//
// This is the directory embedded in store path strings, usually DefaultDir.
// The cached value remains available after Store or Context closure.
func (s *Store) StoreDir() string {
	return s.storeDir
}

// Borrow returns the borrowed raw Nix store handle.
//
// Callers must not free the returned pointer and must not retain it beyond the
// immediate raw Nix call that needs it. This is an escape hatch for integration
// with lower-level bindings and should not be needed for ordinary store
// workflows.
func (p *Store) Borrow() (*nix.Store, error) {
	if p.ptr == nil {
		return nil, status.ErrClosed
	}

	return p.ptr, nil
}

// Close releases the owned Nix store handle.
//
// Close is safe to call more than once. Once Close returns, methods that need
// the raw store handle report status.ErrClosed.
func (s *Store) Close() error {
	if s == nil || s.ptr == nil {
		return nil
	}

	nix.StoreFree(s.ptr)
	s.ptr = nil

	return nil
}

// ParsePath parses a full store path for this store.
//
// The returned path owns a Nix StorePath handle and must be closed by the
// caller. ParsePath validates the syntax of the path and leaves semantic
// validity in this store to IsValidPath.
func (s *Store) ParsePath(pathStr string) (*storepath.Path, error) {
	if s.ptr == nil {
		return nil, status.ErrClosed
	}

	rawCtx, err := s.ctx.Borrow()
	if err != nil {
		return nil, fmt.Errorf("store: failed to borrow context: %w", err)
	}

	ptr := nix.StoreParsePath(rawCtx, s.ptr, pathStr)
	if ptr == nil {
		return nil, fmt.Errorf("store: failed to parse path: %w", status.FromContext(rawCtx))
	}

	path, err := storepath.New(s.ctx, ptr)
	if err != nil {
		nix.StorePathFree(ptr)
		return nil, fmt.Errorf("store: failed to create store path: %w", err)
	}

	return path, nil
}

// PathFromHash returns the store path whose hash part matches hashPart.
//
// hashPart is the encoded hash portion of a Nix store path, without the store
// directory or name. If the store has no matching path, PathFromHash returns
// ErrPathNotFound. The returned path is owned by the caller.
func (s *Store) PathFromHash(hashPart []byte) (*storepath.Path, error) {
	if s.ptr == nil {
		return nil, status.ErrClosed
	}

	rawCtx, err := s.ctx.Borrow()
	if err != nil {
		return nil, fmt.Errorf("store: failed to borrow context: %w", err)
	}

	ptr := nix.StoreQueryPathFromHashPart(rawCtx, s.ptr, string(hashPart))
	if ptr == nil {
		if err := status.FromContext(rawCtx); err != nil {
			return nil, fmt.Errorf("store: failed to get path from hash: %w", err)
		}
		return nil, fmt.Errorf("store: hash %q: %w", hashPart, ErrPathNotFound)
	}

	path, err := storepath.New(s.ctx, ptr)
	if err != nil {
		nix.StorePathFree(ptr)
		return nil, fmt.Errorf("store: failed to create store path: %w", err)
	}

	return path, nil
}

// RealPath returns the concrete filesystem path for path in this store.
//
// For ordinary local stores this is normally the same string as the logical
// store path. For rooted or redirected stores it may point somewhere else on
// the host filesystem.
func (s *Store) RealPath(path *storepath.Path) (string, error) {
	if s.ptr == nil {
		return "", status.ErrClosed
	}

	rawCtx, err := s.ctx.Borrow()
	if err != nil {
		return "", fmt.Errorf("store: failed to borrow context: %w", err)
	}

	pathPtr, err := path.Borrow()
	if err != nil {
		return "", fmt.Errorf("store: failed to borrow path: %w", err)
	}

	ptr := nix.StoreRealPath(rawCtx, s.ptr, pathPtr)
	if ptr == nil {
		return "", fmt.Errorf("store: failed to get real path: %w", status.FromContext(rawCtx))
	}

	return utils.TakeCString(ptr), nil
}

// IsValidPath reports whether path is valid in this store.
//
// A syntactically valid StorePath may still be invalid if it is not registered
// or available in this store. A false result with a nil error means Nix
// successfully answered that the path is not valid.
func (s *Store) IsValidPath(path *storepath.Path) (bool, error) {
	if s.ptr == nil {
		return false, status.ErrClosed
	}

	rawCtx, err := s.ctx.Borrow()
	if err != nil {
		return false, fmt.Errorf("store: failed to borrow context: %w", err)
	}

	pathPtr, err := path.Borrow()
	if err != nil {
		return false, fmt.Errorf("store: failed to borrow path: %w", err)
	}

	valid := nix.StoreIsValidPath(rawCtx, s.ptr, pathPtr)
	if err := status.FromContext(rawCtx); err != nil {
		return false, fmt.Errorf("store: failed to check path validity: %w", err)
	}

	return valid, nil
}
