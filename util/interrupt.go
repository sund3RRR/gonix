package util

import (
	"fmt"

	"github.com/sund3RRR/gonix/internal/status"
	"github.com/sund3RRR/gonix/nixcontext"
	"github.com/sund3RRR/gonix/store"
	"github.com/sund3RRR/nix-go-bindings"
)

func StoreInterrupt(ctx *nixcontext.Context, s *store.Store) error {
	rawCtx, err := ctx.Borrow()
	if err != nil {
		return fmt.Errorf("util: failed to borrow context: %w", err)
	}

	rawStore, err := s.Borrow()
	if err != nil {
		return fmt.Errorf("util: failed to borrow store: %w", err)
	}

	if code := nix.StoreInterrupt(rawCtx, rawStore); status.ErrorCode(code) != status.ErrorCodeOK {
		return fmt.Errorf("util: failed to interrupt store: %w", status.FromContext(rawCtx))
	}

	return nil
}

func InterruptRequest() {
	nix.InterruptRequest()
}

func InterruptRequested() {
	nix.InterruptRequested()
}

func InterruptClear() {
	nix.InterruptClear()
}
