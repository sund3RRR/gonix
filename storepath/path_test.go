package storepath

import (
	"errors"
	"testing"
	"unsafe"

	"github.com/sund3RRR/gonix/internal/status"
	nix "github.com/sund3RRR/nix-go-bindings"
)

const (
	zeroStorePath    = "/nix/store/00000000000000000000000000000000-demo"
	nonZeroStorePath = "/nix/store/11111111111111111111111111111111-source"
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

	return ctx
}

func newTestStore(t *testing.T, ctx *nix.NixCContext) *nix.Store {
	t.Helper()

	if got := nix.LibstoreInitNoLoadConfig(ctx); got != nix.NixOk {
		t.Fatalf("LibstoreInitNoLoadConfig = %v, want %v", got, nix.NixOk)
	}

	store := nix.StoreOpen(ctx, "dummy://", nix.StoreParams{})
	if store == nil {
		t.Fatalf("StoreOpen(dummy://) returned nil: err=%v", statusMessage(ctx))
	}
	t.Cleanup(func() {
		nix.StoreFree(store)
	})

	return store
}

func parseRawPath(t *testing.T, ctx *nix.NixCContext, store *nix.Store, rawPath string) *nix.StorePath {
	t.Helper()

	ptr := nix.StoreParsePath(ctx, store, rawPath)
	if ptr == nil {
		t.Fatalf("StoreParsePath(%q) returned nil: err=%v", rawPath, statusMessage(ctx))
	}

	return ptr
}

func newTestPath(t *testing.T, ctx *nix.NixCContext, store *nix.Store, rawPath string) *Path {
	t.Helper()

	path, err := New(ctx, parseRawPath(t, ctx, store, rawPath))
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	t.Cleanup(func() {
		if err := path.Close(); err != nil {
			t.Fatalf("Path.Close() error = %v", err)
		}
	})

	return path
}

func statusMessage(ctx *nix.NixCContext) string {
	if ctx == nil {
		return ""
	}
	ptr := nix.ErrMsg(nil, ctx)
	if ptr == nil {
		return ""
	}
	defer nix.StringFree(ptr)

	var msg []byte
	for p := ptr; *p != 0; p = (*byte)(unsafeAdd(p, 1)) {
		msg = append(msg, *p)
	}

	return string(msg)
}

func unsafeAdd(ptr *byte, offset uintptr) unsafe.Pointer {
	return unsafe.Add(unsafe.Pointer(ptr), offset)
}

func requireClosedError(t *testing.T, err error) {
	t.Helper()

	if err == nil {
		t.Fatal("error = nil, want gonix.ErrClosed")
	}
	if !errors.Is(err, status.ErrClosed) {
		t.Fatalf("error = %v, want errors.Is(..., status.ErrClosed)", err)
	}
}

func TestFromParts(t *testing.T) {
	tests := []struct {
		name          string
		setup         func(t *testing.T) (*nix.NixCContext, [20]byte, string)
		wantName      string
		wantHash      func(t *testing.T) [20]byte
		wantClosedErr bool
	}{
		{
			name: "creates path from hash and name",
			setup: func(t *testing.T) (*nix.NixCContext, [20]byte, string) {
				t.Helper()

				ctx := newTestContext(t)
				store := newTestStore(t, ctx)
				source := newTestPath(t, ctx, store, nonZeroStorePath)
				return ctx, source.Hash(), "created-from-parts"
			},
			wantName: "created-from-parts",
			wantHash: func(t *testing.T) [20]byte {
				t.Helper()

				ctx := newTestContext(t)
				store := newTestStore(t, ctx)
				source := newTestPath(t, ctx, store, nonZeroStorePath)
				return source.Hash()
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx, hash, name := tt.setup(t)

			got, err := FromParts(ctx, hash, name)
			if tt.wantClosedErr {
				requireClosedError(t, err)
				if got != nil {
					t.Fatalf("FromParts() = %v, want nil", got)
				}
				return
			}
			if err != nil {
				t.Fatalf("FromParts() error = %v", err)
			}
			t.Cleanup(func() {
				if err := got.Close(); err != nil {
					t.Fatalf("created.Close() error = %v", err)
				}
			})

			if gotName := got.Name(); gotName != tt.wantName {
				t.Fatalf("created.Name() = %q, want %q", gotName, tt.wantName)
			}

			if gotHash, wantHash := got.Hash(), tt.wantHash(t); gotHash != wantHash {
				t.Fatalf("created.Hash() = %v, want %v", gotHash, wantHash)
			}
		})
	}
}

func TestPathName(t *testing.T) {
	tests := []struct {
		name  string
		setup func(t *testing.T) *Path
		want  string
	}{
		{
			name: "open path",
			setup: func(t *testing.T) *Path {
				t.Helper()

				ctx := newTestContext(t)
				store := newTestStore(t, ctx)
				return newTestPath(t, ctx, store, zeroStorePath)
			},
			want: "demo",
		},
		{
			name: "closed path keeps cached name",
			setup: func(t *testing.T) *Path {
				t.Helper()

				ctx := newTestContext(t)
				store := newTestStore(t, ctx)
				path := newTestPath(t, ctx, store, zeroStorePath)
				if err := path.Close(); err != nil {
					t.Fatalf("Close() error = %v", err)
				}

				return path
			},
			want: "demo",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.setup(t).Name()
			if got != tt.want {
				t.Fatalf("Name() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestPathHash(t *testing.T) {
	tests := []struct {
		name        string
		setup       func(t *testing.T) *Path
		want        [20]byte
		wantNonZero bool
	}{
		{
			name: "zero hash path",
			setup: func(t *testing.T) *Path {
				t.Helper()

				ctx := newTestContext(t)
				store := newTestStore(t, ctx)
				return newTestPath(t, ctx, store, zeroStorePath)
			},
			want: [20]byte{},
		},
		{
			name: "non-zero hash path",
			setup: func(t *testing.T) *Path {
				t.Helper()

				ctx := newTestContext(t)
				store := newTestStore(t, ctx)
				return newTestPath(t, ctx, store, nonZeroStorePath)
			},
			wantNonZero: true,
		},
		{
			name: "closed path keeps cached hash",
			setup: func(t *testing.T) *Path {
				t.Helper()

				ctx := newTestContext(t)
				store := newTestStore(t, ctx)
				path := newTestPath(t, ctx, store, zeroStorePath)
				if err := path.Close(); err != nil {
					t.Fatalf("Close() error = %v", err)
				}

				return path
			},
			want: [20]byte{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.setup(t).Hash()
			if tt.wantNonZero {
				if got == ([20]byte{}) {
					t.Fatal("Hash() returned zero hash, want non-zero hash")
				}
				return
			}
			if got != tt.want {
				t.Fatalf("Hash() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestPathClone(t *testing.T) {
	tests := []struct {
		name          string
		setup         func(t *testing.T) *Path
		wantClosedErr bool
	}{
		{
			name: "open path",
			setup: func(t *testing.T) *Path {
				t.Helper()

				ctx := newTestContext(t)
				store := newTestStore(t, ctx)
				return newTestPath(t, ctx, store, zeroStorePath)
			},
		},
		{
			name: "closed path",
			setup: func(t *testing.T) *Path {
				t.Helper()

				ctx := newTestContext(t)
				store := newTestStore(t, ctx)
				path := newTestPath(t, ctx, store, zeroStorePath)
				if err := path.Close(); err != nil {
					t.Fatalf("Close() error = %v", err)
				}

				return path
			},
			wantClosedErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			source := tt.setup(t)
			got, err := source.Clone()
			if tt.wantClosedErr {
				requireClosedError(t, err)
				if got != nil {
					t.Fatalf("Clone() = %v, want nil", got)
				}
				return
			}
			if err != nil {
				t.Fatalf("Clone() error = %v", err)
			}
			t.Cleanup(func() {
				if err := got.Close(); err != nil {
					t.Fatalf("clone.Close() error = %v", err)
				}
			})

			sourceName := source.Name()
			cloneName := got.Name()
			if cloneName != sourceName {
				t.Fatalf("clone.Name() = %q, want %q", cloneName, sourceName)
			}

			sourceHash := source.Hash()
			cloneHash := got.Hash()
			if cloneHash != sourceHash {
				t.Fatalf("clone.Hash() = %v, want %v", cloneHash, sourceHash)
			}

			if err := got.Close(); err != nil {
				t.Fatalf("clone.Close() error = %v", err)
			}
			if gotName := source.Name(); gotName != sourceName {
				t.Fatalf("source.Name() after clone close = %q, want %q", gotName, sourceName)
			}
		})
	}
}

func TestPathBorrow(t *testing.T) {
	tests := []struct {
		name          string
		setup         func(t *testing.T) *Path
		wantClosedErr bool
	}{
		{
			name: "open path",
			setup: func(t *testing.T) *Path {
				t.Helper()

				ctx := newTestContext(t)
				store := newTestStore(t, ctx)
				return newTestPath(t, ctx, store, zeroStorePath)
			},
		},
		{
			name: "closed path",
			setup: func(t *testing.T) *Path {
				t.Helper()

				ctx := newTestContext(t)
				store := newTestStore(t, ctx)
				path := newTestPath(t, ctx, store, zeroStorePath)
				if err := path.Close(); err != nil {
					t.Fatalf("Close() error = %v", err)
				}

				return path
			},
			wantClosedErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			path := tt.setup(t)
			got, err := path.Borrow()
			if tt.wantClosedErr {
				requireClosedError(t, err)
				if got != nil {
					t.Fatalf("Borrow() = %v, want nil", got)
				}
				return
			}
			if err != nil {
				t.Fatalf("Borrow() error = %v", err)
			}
			if got == nil {
				t.Fatal("Borrow() = nil, want raw pointer")
			}
			if got != path.ptr {
				t.Fatalf("Borrow() = %p, want %p", got, path.ptr)
			}
		})
	}
}

func TestPathClose(t *testing.T) {
	tests := []struct {
		name  string
		setup func(t *testing.T) *Path
	}{
		{
			name: "open path",
			setup: func(t *testing.T) *Path {
				t.Helper()

				ctx := newTestContext(t)
				store := newTestStore(t, ctx)
				path, err := New(ctx, parseRawPath(t, ctx, store, zeroStorePath))
				if err != nil {
					t.Fatalf("New() error = %v", err)
				}
				return path
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			path := tt.setup(t)
			if err := path.Close(); err != nil {
				t.Fatalf("Close() error = %v", err)
			}
			if err := path.Close(); err != nil {
				t.Fatalf("second Close() error = %v", err)
			}
			if path != nil && path.ptr != nil {
				t.Fatal("Close() left raw pointer non-nil")
			}
		})
	}
}
