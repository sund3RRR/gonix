package fetchers

import (
	"errors"
	"testing"

	"github.com/sund3RRR/gonix/internal/status"
	"github.com/sund3RRR/gonix/nixcontext"
)

func newTestContext(t *testing.T) *nixcontext.Context {
	t.Helper()

	ctx, err := nixcontext.New(nixcontext.Config{})
	if err != nil {
		t.Fatalf("nixcontext.New() error = %v", err)
	}
	t.Cleanup(func() {
		if err := ctx.Close(); err != nil {
			t.Fatalf("Context.Close() error = %v", err)
		}
	})

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
