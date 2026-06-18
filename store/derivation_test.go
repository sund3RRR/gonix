package store

import (
	"encoding/json"
	"errors"
	"reflect"
	"testing"
	"unsafe"

	"github.com/sund3RRR/gonix/internal/status"
	"github.com/sund3RRR/gonix/nixcontext"
	nix "github.com/sund3RRR/nix-go-bindings"
)

const testDerivationJSON = `{
  "name": "gonix-test",
  "version": 4,
  "outputs": {
    "out": {
      "path": "awjawq2kj29m8cg6cmdpyksrjnmlk7jp-gonix-test"
    }
  },
  "inputs": {
    "srcs": [],
    "drvs": {}
  },
  "system": "x86_64-linux",
  "builder": "/bin/sh",
  "args": [],
  "env": {
    "builder": "/bin/sh",
    "name": "gonix-test",
    "out": "/nix/store/awjawq2kj29m8cg6cmdpyksrjnmlk7jp-gonix-test",
    "system": "x86_64-linux"
  }
}`

func newDerivationTestContext(t *testing.T) *nixcontext.Context {
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

func newDerivationTestStore(t *testing.T, ctx *nixcontext.Context) *nix.Store {
	t.Helper()

	rawCtx, err := ctx.Borrow()
	if err != nil {
		t.Fatalf("Context.Borrow() error = %v", err)
	}

	store := nix.StoreOpen(rawCtx, "dummy://", nix.StoreParams{})
	if store == nil {
		t.Fatalf("StoreOpen(dummy://) returned nil: err=%v msg=%q", nix.ErrCode(rawCtx), derivationStatusMessage(rawCtx))
	}
	t.Cleanup(func() {
		nix.StoreFree(store)
	})

	return store
}

func newTestDerivation(t *testing.T) *Derivation {
	t.Helper()

	ctx := newDerivationTestContext(t)
	store := newDerivationTestStore(t, ctx)
	rawCtx, err := ctx.Borrow()
	if err != nil {
		t.Fatalf("Context.Borrow() error = %v", err)
	}
	ptr := nix.DerivationFromJson(rawCtx, store, testDerivationJSON)
	if ptr == nil {
		t.Fatalf("DerivationFromJson returned nil: err=%v msg=%q", nix.ErrCode(rawCtx), derivationStatusMessage(rawCtx))
	}

	d := NewDerivation(ctx, ptr)
	t.Cleanup(func() {
		if err := d.Close(); err != nil {
			t.Fatalf("Derivation.Close() error = %v", err)
		}
	})

	return d
}

func mustParseDerivationJSON(t *testing.T, raw []byte) map[string]any {
	t.Helper()

	var parsed map[string]any
	if err := json.Unmarshal(raw, &parsed); err != nil {
		t.Fatalf("json.Unmarshal derivation JSON: %v\n%s", err, raw)
	}

	return parsed
}

func derivationStatusMessage(ctx *nix.NixCContext) string {
	if ctx == nil {
		return ""
	}
	ptr := nix.ErrMsg(nil, ctx)
	if ptr == nil {
		return ""
	}
	defer nix.StringFree(ptr)

	var msg []byte
	for p := ptr; *p != 0; p = (*byte)(unsafe.Add(unsafe.Pointer(p), 1)) {
		msg = append(msg, *p)
	}

	return string(msg)
}

func requireDerivationClosedError(t *testing.T, err error) {
	t.Helper()

	if err == nil {
		t.Fatal("error = nil, want status.ErrClosed")
	}
	if !errors.Is(err, status.ErrClosed) {
		t.Fatalf("error = %v, want errors.Is(..., status.ErrClosed)", err)
	}
}

func TestDerivation_JSON(t *testing.T) {
	tests := []struct {
		name          string
		setup         func(t *testing.T) *Derivation
		wantClosedErr bool
	}{
		{
			name: "open_derivation",
			setup: func(t *testing.T) *Derivation {
				t.Helper()
				return newTestDerivation(t)
			},
		},
		{
			name: "closed_derivation",
			setup: func(t *testing.T) *Derivation {
				t.Helper()
				return NewDerivation(newDerivationTestContext(t), nil)
			},
			wantClosedErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			d := tt.setup(t)

			got, err := d.JSON()
			if tt.wantClosedErr {
				requireDerivationClosedError(t, err)
				return
			}
			if err != nil {
				t.Fatalf("Derivation.JSON() error = %v", err)
			}

			gotJSON := mustParseDerivationJSON(t, got)
			if gotJSON["name"] != "gonix-test" {
				t.Fatalf("name = %v, want gonix-test", gotJSON["name"])
			}
			if gotJSON["version"] != float64(4) {
				t.Fatalf("version = %v, want 4", gotJSON["version"])
			}
			outputs, ok := gotJSON["outputs"].(map[string]any)
			if !ok {
				t.Fatalf("outputs = %T, want JSON object", gotJSON["outputs"])
			}
			if _, ok := outputs["out"]; !ok {
				t.Fatal("outputs missing out")
			}
		})
	}
}

func TestDerivation_Clone(t *testing.T) {
	tests := []struct {
		name          string
		setup         func(t *testing.T) *Derivation
		wantClosedErr bool
	}{
		{
			name: "open_derivation",
			setup: func(t *testing.T) *Derivation {
				t.Helper()
				return newTestDerivation(t)
			},
		},
		{
			name: "closed_derivation",
			setup: func(t *testing.T) *Derivation {
				t.Helper()
				return NewDerivation(newDerivationTestContext(t), nil)
			},
			wantClosedErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			d := tt.setup(t)

			clone, err := d.Clone()
			if tt.wantClosedErr {
				requireDerivationClosedError(t, err)
				if clone != nil {
					t.Fatalf("Derivation.Clone() = %v, want nil", clone)
				}
				return
			}
			if err != nil {
				t.Fatalf("Derivation.Clone() error = %v", err)
			}
			t.Cleanup(func() {
				if err := clone.Close(); err != nil {
					t.Fatalf("clone.Close() error = %v", err)
				}
			})

			originalJSON := mustJSONMap(t, d)
			cloneJSON := mustJSONMap(t, clone)
			if !reflect.DeepEqual(cloneJSON, originalJSON) {
				t.Fatalf("clone JSON = %#v, want %#v", cloneJSON, originalJSON)
			}

			if err := clone.Close(); err != nil {
				t.Fatalf("clone.Close() error = %v", err)
			}
			if _, err := d.JSON(); err != nil {
				t.Fatalf("original.JSON() after clone.Close() error = %v", err)
			}
		})
	}
}

func TestDerivation_Borrow(t *testing.T) {
	tests := []struct {
		name          string
		setup         func(t *testing.T) *Derivation
		wantClosedErr bool
	}{
		{
			name: "open_derivation",
			setup: func(t *testing.T) *Derivation {
				t.Helper()
				return newTestDerivation(t)
			},
		},
		{
			name: "closed_derivation",
			setup: func(t *testing.T) *Derivation {
				t.Helper()
				return NewDerivation(newDerivationTestContext(t), nil)
			},
			wantClosedErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			d := tt.setup(t)

			got, err := d.Borrow()
			if tt.wantClosedErr {
				requireDerivationClosedError(t, err)
				if got != nil {
					t.Fatalf("Derivation.Borrow() = %v, want nil", got)
				}
				return
			}
			if err != nil {
				t.Fatalf("Derivation.Borrow() error = %v", err)
			}
			if got != d.ptr {
				t.Fatalf("Derivation.Borrow() = %p, want %p", got, d.ptr)
			}
		})
	}
}

func TestDerivation_Close(t *testing.T) {
	tests := []struct {
		name  string
		setup func(t *testing.T) *Derivation
	}{
		{
			name: "open_derivation",
			setup: func(t *testing.T) *Derivation {
				t.Helper()
				return newTestDerivation(t)
			},
		},
		{
			name: "closed_derivation",
			setup: func(t *testing.T) *Derivation {
				t.Helper()
				return NewDerivation(newDerivationTestContext(t), nil)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			d := tt.setup(t)

			if err := d.Close(); err != nil {
				t.Fatalf("Derivation.Close() error = %v", err)
			}
			if err := d.Close(); err != nil {
				t.Fatalf("second Derivation.Close() error = %v", err)
			}
			if d.ptr != nil {
				t.Fatalf("d.ptr = %p, want nil", d.ptr)
			}

			_, err := d.JSON()
			requireDerivationClosedError(t, err)
			_, err = d.Clone()
			requireDerivationClosedError(t, err)
			_, err = d.Borrow()
			requireDerivationClosedError(t, err)
		})
	}
}

func mustJSONMap(t *testing.T, d *Derivation) map[string]any {
	t.Helper()

	raw, err := d.JSON()
	if err != nil {
		t.Fatalf("Derivation.JSON() error = %v", err)
	}

	return mustParseDerivationJSON(t, raw)
}
