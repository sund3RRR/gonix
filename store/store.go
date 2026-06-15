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
	"github.com/sund3RRR/gonix/internal/utils"
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
// but all other methods return an error wrapping status.ErrClosed after Close.
// Store is not documented as goroutine-safe.
type Store struct {
	ctx *nix.NixCContext
	ptr *nix.Store
}

// New opens a Nix store using an already-initialized Nix context.
//
// The uri is passed to Nix as the store URI, for example "dummy://",
// "local", or a remote store URI supported by the linked Nix libraries. Options
// are serialized as store parameters, equivalent to URL query parameters in the
// low-level Nix API.
//
// The returned Store owns the raw Nix store handle and must be closed by the
// caller. The ctx argument is borrowed and must remain valid for as long as the
// Store is used.
func New(ctx *nix.NixCContext, uri string, opts ...Option) (*Store, error) {
	var cfg Config
	for _, opt := range opts {
		opt(&cfg)
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

	ptr := nix.StoreOpen(ctx, uri, storeParams)
	if ptr == nil {
		return nil, fmt.Errorf("store: failed to open store: %w", status.FromContext(ctx))
	}

	return &Store{
		ctx: ctx,
		ptr: ptr,
	}, nil
}

// Version returns the store backend version when the backend reports one.
//
// Some store backends do not expose a version. In that case Version returns an
// empty string and a nil error. A non-nil error means Nix reported a real
// failure while asking the backend for its version.
func (s *Store) Version() (string, error) {
	if s.ptr == nil {
		return "", status.ErrClosed
	}

	ptr := nix.StoreGetVersion(s.ctx, s.ptr)
	if ptr == nil {
		if err := status.FromContext(s.ctx); err != nil {
			return "", fmt.Errorf("store: failed to get version: %w", err)
		}
		return "", nil
	}

	return utils.TakeCString(ptr), nil
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
	if s.ptr == nil {
		return nil
	}

	nix.StoreFree(s.ptr)
	s.ptr = nil

	return nil
}
