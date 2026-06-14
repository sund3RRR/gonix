package store

import (
	"fmt"

	"github.com/sund3RRR/gonix/internal/status"
	"github.com/sund3RRR/gonix/internal/utils"
	"github.com/sund3RRR/gonix/storepath"
	nix "github.com/sund3RRR/nix-go-bindings"
)

// URI returns the store's canonical URI.
func (s *Store) URI() (string, error) {
	if s.ptr == nil {
		return "", status.ErrClosed
	}

	ptr := nix.StoreGetUri(s.ctx, s.ptr)
	if ptr == nil {
		return "", fmt.Errorf("store: failed to get uri: %w", status.FromContext(s.ctx))
	}

	return utils.TakeCString(ptr), nil
}

// StoreDir returns the store's logical Nix store directory.
func (s *Store) StoreDir() (string, error) {
	if s.ptr == nil {
		return "", status.ErrClosed
	}

	ptr := nix.StoreGetStoredir(s.ctx, s.ptr)
	if ptr == nil {
		return "", fmt.Errorf("store: failed to get store dir: %w", status.FromContext(s.ctx))
	}

	return utils.TakeCString(ptr), nil
}

// Version returns the store backend version when the backend reports one.
func (s *Store) Version() (string, error) {
	if s.ptr == nil {
		return "", status.ErrClosed
	}

	ptr := nix.StoreGetVersion(s.ctx, s.ptr)
	if ptr == nil {
		return "", fmt.Errorf("store: failed to get version: %w", status.FromContext(s.ctx))
	}

	return utils.TakeCString(ptr), nil
}

// ParsePath parses a full store path for this store.
func (s *Store) ParsePath(path string) (*storepath.Path, error) {
	if s.ptr == nil {
		return nil, status.ErrClosed
	}

	ptr := nix.StoreParsePath(s.ctx, s.ptr, path)
	if ptr == nil {
		return nil, fmt.Errorf("store: failed to parse path: %w", status.FromContext(s.ctx))
	}

	return storepath.New(s.ctx, ptr), nil
}

// PathFromHash returns the store path whose hash part matches hashPart.
func (s *Store) PathFromHash(hashPart []byte) (*storepath.Path, error) {
	if s.ptr == nil {
		return nil, status.ErrClosed
	}

	ptr := nix.StoreQueryPathFromHashPart(s.ctx, s.ptr, string(hashPart))
	if ptr == nil {
		return nil, fmt.Errorf("store: failed to get path from hash: %w", status.FromContext(s.ctx))
	}

	return storepath.New(s.ctx, ptr), nil
}

// RealPath returns the concrete filesystem path for path in this store.
func (s *Store) RealPath(path *storepath.Path) (string, error) {
	if s.ptr == nil {
		return "", status.ErrClosed
	}

	pathPtr, err := path.Borrow()
	if err != nil {
		return "", fmt.Errorf("store: failed to borrow path: %w", err)
	}

	ptr := nix.StoreRealPath(s.ctx, s.ptr, pathPtr)
	if ptr == nil {
		return "", fmt.Errorf("store: failed to get real path: %w", status.FromContext(s.ctx))
	}

	return utils.TakeCString(ptr), nil
}

// IsValidPath reports whether path is valid in this store.
func (s *Store) IsValidPath(path *storepath.Path) (bool, error) {
	if s.ptr == nil {
		return false, status.ErrClosed
	}

	pathPtr, err := path.Borrow()
	if err != nil {
		return false, fmt.Errorf("store: failed to borrow path: %w", err)
	}

	valid := nix.StoreIsValidPath(s.ctx, s.ptr, pathPtr)
	if err := status.FromContext(s.ctx); err != nil {
		return false, fmt.Errorf("store: failed to check path validity: %w", err)
	}

	return valid, nil
}
