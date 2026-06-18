package store

import (
	"fmt"

	"github.com/sund3RRR/gonix/internal/status"
	"github.com/sund3RRR/gonix/internal/utils"
	"github.com/sund3RRR/gonix/storepath"
	nix "github.com/sund3RRR/nix-go-bindings"
)

// URI returns the store's canonical URI.
//
// The returned value is produced by Nix and may be normalized from the URI that
// was passed to New.
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
//
// This is the directory embedded in store path strings, usually DefaultDir.
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

// ParsePath parses a full store path for this store.
//
// The returned path owns a Nix StorePath handle and must be closed by the
// caller. ParsePath validates the syntax of the path and leaves semantic
// validity in this store to IsValidPath.
func (s *Store) ParsePath(pathStr string) (*storepath.Path, error) {
	if s.ptr == nil {
		return nil, status.ErrClosed
	}

	ptr := nix.StoreParsePath(s.ctx, s.ptr, pathStr)
	if ptr == nil {
		return nil, fmt.Errorf("store: failed to parse path: %w", status.FromContext(s.ctx))
	}

	path, err := storepath.New(s.ctx, ptr)
	if err != nil {
		nix.StorePathFree(ptr)
		return nil, fmt.Errorf("store: failed to create store path: %w", err)
	}

	return path, nil
}

// PrintPath formats path as a logical store path for this store.
//
// PrintPath uses the store's logical store directory. For rooted or redirected
// stores, use RealPath when the concrete filesystem path is needed.
func (s *Store) PrintPath(path *storepath.Path) (string, error) {
	if s.ptr == nil {
		return "", status.ErrClosed
	}

	pathPtr, err := path.Borrow()
	if err != nil {
		return "", fmt.Errorf("store: failed to borrow path: %w", err)
	}

	ptr := nix.StorePrintPath(s.ctx, s.ptr, pathPtr)
	if ptr == nil {
		return "", fmt.Errorf("store: failed to print path: %w", status.FromContext(s.ctx))
	}

	return utils.TakeCString(ptr), nil
}

// PathFromHash returns the store path whose hash part matches hashPart.
//
// hashPart is the encoded hash portion of a Nix store path, without the store
// directory or name. If the store has no matching path, Nix reports an error.
// The returned path is owned by the caller.
func (s *Store) PathFromHash(hashPart []byte) (*storepath.Path, error) {
	if s.ptr == nil {
		return nil, status.ErrClosed
	}

	ptr := nix.StoreQueryPathFromHashPart(s.ctx, s.ptr, string(hashPart))
	if ptr == nil {
		return nil, fmt.Errorf("store: failed to get path from hash: %w", status.FromContext(s.ctx))
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
//
// A syntactically valid StorePath may still be invalid if it is not registered
// or available in this store. A false result with a nil error means Nix
// successfully answered that the path is not valid.
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
