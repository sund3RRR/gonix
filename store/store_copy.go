package store

import (
	"fmt"

	"github.com/sund3RRR/gonix/internal/status"
	"github.com/sund3RRR/gonix/pkg/raw"
	"github.com/sund3RRR/gonix/storepath"
)

// CopyConfig configures copying a store path between stores.
//
// The zero value uses Nix's default copy behavior: no repair pass and no
// explicit signature check flag.
type CopyConfig struct {
	// Repair asks Nix to repair the destination path while copying.
	Repair bool

	// CheckSignatures asks Nix to verify signatures while copying.
	CheckSignatures bool
}

// CopyOption configures Store.CopyPathTo.
type CopyOption func(*CopyConfig)

// WithCopyRepair asks Nix to repair the destination path while copying.
func WithCopyRepair(repair bool) CopyOption {
	return func(c *CopyConfig) {
		c.Repair = repair
	}
}

// WithCopyCheckSignatures asks Nix to verify signatures while copying.
func WithCopyCheckSignatures(check bool) CopyOption {
	return func(c *CopyConfig) {
		c.CheckSignatures = check
	}
}

// CopyClosure copies path and its closure from this store to dst.
//
// The source store, destination store, and path are borrowed for the duration
// of the call. Both stores must use the same logical store directory.
func (s *Store) CopyClosure(dst *Store, path *storepath.Path) error {
	if s.ptr == nil {
		return status.ErrClosed
	}

	rawCtx, err := s.ctx.Borrow()
	if err != nil {
		return fmt.Errorf("store: failed to borrow context: %w", err)
	}

	dstPtr, err := dst.Borrow()
	if err != nil {
		return fmt.Errorf("store: failed to borrow dst store: %w", err)
	}

	pathPtr, err := path.Borrow()
	if err != nil {
		return fmt.Errorf("store: failed to borrow path: %w", err)
	}

	if code := raw.StoreCopyClosure(rawCtx, s.ptr, dstPtr, pathPtr); status.ErrorCode(code) != status.ErrorCodeOK {
		return fmt.Errorf("store: failed to copy closure: %w", status.FromContext(rawCtx))
	}

	return nil
}

// CopyPathTo copies path from this store to dst.
//
// CopyPathTo copies only the requested path. Use CopyClosure when dependencies
// should be copied as well. The source store, destination store, and path are
// borrowed for the duration of the call.
func (s *Store) CopyPathTo(dst *Store, path *storepath.Path, opts ...CopyOption) error {
	if s.ptr == nil {
		return status.ErrClosed
	}

	var cfg CopyConfig
	for _, opt := range opts {
		opt(&cfg)
	}

	rawCtx, err := s.ctx.Borrow()
	if err != nil {
		return fmt.Errorf("store: failed to borrow context: %w", err)
	}

	dstPtr, err := dst.Borrow()
	if err != nil {
		return fmt.Errorf("store: failed to borrow dst store: %w", err)
	}

	pathPtr, err := path.Borrow()
	if err != nil {
		return fmt.Errorf("store: failed to borrow path: %w", err)
	}

	if code := raw.StoreCopyPath(rawCtx, s.ptr, dstPtr, pathPtr, cfg.Repair, cfg.CheckSignatures); status.ErrorCode(code) != status.ErrorCodeOK {
		return fmt.Errorf("store: copy path: %w", status.FromContext(rawCtx))
	}

	return nil
}
