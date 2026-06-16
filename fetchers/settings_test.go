package fetchers

import (
	"errors"
	"testing"

	"github.com/sund3RRR/gonix/internal/status"
	nix "github.com/sund3RRR/nix-go-bindings"
)

func newTestContext(t *testing.T) *nix.NixCContext {
	t.Helper()

	ctx := nix.CContextCreate()
	if ctx == nil {
		t.Fatal("CContextCreate returned nil")
	}
	t.Cleanup(func() {
		nix.CContextFree(ctx)
	})

	if code := nix.LibutilInit(ctx); status.ErrorCode(code) != status.ErrorCodeOK {
		t.Fatalf("LibutilInit = %v, want %v: %v", code, nix.NixOk, status.FromContext(ctx))
	}

	return ctx
}

func requireClosedError(t *testing.T, err error) {
	t.Helper()

	if err == nil {
		t.Fatal("error = nil, want status.ErrClosed")
	}
	if !errors.Is(err, status.ErrClosed) {
		t.Fatalf("error = %v, want errors.Is(..., status.ErrClosed)", err)
	}
}

func TestNew(t *testing.T) {
	ctx := newTestContext(t)

	settings, err := NewSettings(ctx)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	t.Cleanup(func() {
		if err := settings.Close(); err != nil {
			t.Fatalf("Settings.Close() error = %v", err)
		}
	})

	ptr, err := settings.Borrow()
	if err != nil {
		t.Fatalf("Settings.Borrow() error = %v", err)
	}
	if ptr == nil {
		t.Fatal("Settings.Borrow() = nil, want non-nil")
	}
}

func TestSettingsClose(t *testing.T) {
	ctx := newTestContext(t)

	settings, err := NewSettings(ctx)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	if err := settings.Close(); err != nil {
		t.Fatalf("Settings.Close() error = %v", err)
	}
	if err := settings.Close(); err != nil {
		t.Fatalf("second Settings.Close() error = %v", err)
	}

	_, err = settings.Borrow()
	requireClosedError(t, err)
}
