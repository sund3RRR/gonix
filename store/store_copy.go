package store

import (
	"fmt"

	"github.com/sund3RRR/gonix/internal/status"
	"github.com/sund3RRR/gonix/storepath"
	nix "github.com/sund3RRR/nix-go-bindings"
)

// CopyConfig configures copying a store path between stores.
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
func (s *Store) CopyClosure(dst *Store, path *storepath.Path) error {
	if s.ptr == nil {
		return status.ErrClosed
	}

	pathPtr, err := path.Borrow()
	if err != nil {
		return fmt.Errorf("store: failed to borrow path: %w", err)
	}

	if code := nix.StoreCopyClosure(s.ctx, s.ptr, s.ptr, pathPtr); status.ErrorCode(code) != status.ErrorCodeOK {
		return fmt.Errorf("store: failed to copy closure: %w", status.FromContext(s.ctx))
	}

	return nil
}

// CopyPathTo copies path from this store to dst.
func (s *Store) CopyPathTo(dst *Store, path *storepath.Path, opts ...CopyOption) error {
	if s.ptr == nil {
		return status.ErrClosed
	}

	dstPtr, err := dst.Borrow()
	if err != nil {
		return fmt.Errorf("store: failed to borrow dst store: %w", err)
	}

	var cfg CopyConfig
	for _, opt := range opts {
		opt(&cfg)
	}

	pathPtr, err := path.Borrow()
	if err != nil {
		return fmt.Errorf("store: failed to borrow path: %w", err)
	}

	if code := nix.StoreCopyPath(s.ctx, s.ptr, dstPtr, pathPtr, cfg.Repair, cfg.CheckSignatures); status.ErrorCode(code) != status.ErrorCodeOK {
		return fmt.Errorf("store: copy path: %w", status.FromContext(s.ctx))
	}

	return nil
}
