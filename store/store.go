// Package store wraps Nix store handles and store-backed operations.
package store

import (
	"fmt"

	"github.com/sund3RRR/gonix/internal/status"
	nix "github.com/sund3RRR/nix-go-bindings"
)

// Store owns a Nix store handle.
type Store struct {
	ctx *nix.NixCContext
	ptr *nix.Store
}

// New opens a Nix store using an already-initialized Nix context.
func New(ctx *nix.NixCContext, uri string, opts ...Option) (*Store, error) {
	var cfg Config
	for _, opt := range opts {
		opt(&cfg)
	}

	storeParams, err := newStoreParams(cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to get store params: %w", err)
	}

	if storeParams.Len > 0 {
		_, _ = storeParams.PassRef()
		defer storeParams.Free()
	}

	ptr := nix.StoreOpen(ctx, uri, storeParams)
	if ptr == nil {
		return nil, fmt.Errorf("failed to open store: %w", status.FromContext(ctx))
	}

	return &Store{
		ctx: ctx,
		ptr: ptr,
	}, nil
}

func newStoreParams(cfg Config) (nix.StoreParams, error) {
	params, err := cfg.Params()
	if err != nil {
		return nix.StoreParams{}, fmt.Errorf("failed to convert config to params: %w", err)
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

	return nix.StoreParams{
		Items: items,
		Len:   uint64(len(items)),
	}, nil
}

// Close releases the owned Nix store handle and is safe to call more than once.
func (s *Store) Close() error {
	if s.ptr == nil {
		return nil
	}

	nix.StoreFree(s.ptr)
	s.ptr = nil
	if err := status.FromContext(s.ctx); err != nil {
		return fmt.Errorf("store: failed to free resource: %w", err)
	}

	return nil
}
