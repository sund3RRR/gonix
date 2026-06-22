package util

import (
	"fmt"

	"github.com/sund3RRR/gonix/internal/status"
	"github.com/sund3RRR/gonix/nixcontext"
	"github.com/sund3RRR/gonix/pkg/raw"
	"github.com/sund3RRR/gonix/store"
)

// StoreInterrupt requests interruption of active operations on s.
//
// ctx must be distinct from the context used by the operation being
// interrupted. A remotely interrupted Store must be discarded after the
// operation returns.
func StoreInterrupt(ctx *nixcontext.Context, s *store.Store) error {
	rawCtx, err := ctx.Borrow()
	if err != nil {
		return fmt.Errorf("util: failed to borrow context: %w", err)
	}

	rawStore, err := s.Borrow()
	if err != nil {
		return fmt.Errorf("util: failed to borrow store: %w", err)
	}

	if code := raw.StoreInterrupt(rawCtx, rawStore); status.ErrorCode(code) != status.ErrorCodeOK {
		return fmt.Errorf("util: failed to interrupt store: %w", status.FromContext(rawCtx))
	}

	return nil
}

// InterruptRequest sets Nix's process-global cooperative interruption flag.
func InterruptRequest() {
	raw.InterruptRequest()
}

// InterruptRequested reports whether process-global interruption is pending.
func InterruptRequested() bool {
	return raw.InterruptRequested()
}

// InterruptClear clears Nix's process-global cooperative interruption flag.
func InterruptClear() {
	raw.InterruptClear()
}
