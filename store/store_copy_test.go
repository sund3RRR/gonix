package store

import (
	"errors"
	"testing"

	"github.com/sund3RRR/gonix/internal/status"
	"github.com/sund3RRR/gonix/storepath"
)

func TestStore_CopyClosure(t *testing.T) {
	tests := []struct {
		name         string
		setup        func(t *testing.T) (*Store, *Store, *storepath.Path)
		wantErr      bool
		wantClosed   bool
		backendError bool
	}{
		{
			name: "dummy_to_dummy",
			setup: func(t *testing.T) (*Store, *Store, *storepath.Path) {
				t.Helper()
				src := newStoreTestStore(t)
				dst := newStoreTestStore(t)
				return src, dst, newStoreTestPath(t, src, testZeroStorePath)
			},
			wantErr:      true,
			backendError: true,
		},
		{
			name: "closed_source_store",
			setup: func(t *testing.T) (*Store, *Store, *storepath.Path) {
				t.Helper()
				src := newStoreTestStore(t)
				dst := newStoreTestStore(t)
				path := newStoreTestPath(t, src, testZeroStorePath)
				if err := src.Close(); err != nil {
					t.Fatalf("Store.Close() error = %v", err)
				}
				return src, dst, path
			},
			wantErr:    true,
			wantClosed: true,
		},
		{
			name: "closed_destination_store",
			setup: func(t *testing.T) (*Store, *Store, *storepath.Path) {
				t.Helper()
				src := newStoreTestStore(t)
				dst := newStoreTestStore(t)
				path := newStoreTestPath(t, src, testZeroStorePath)
				if err := dst.Close(); err != nil {
					t.Fatalf("Store.Close() error = %v", err)
				}
				return src, dst, path
			},
			wantErr:    true,
			wantClosed: true,
		},
		{
			name: "closed_path",
			setup: func(t *testing.T) (*Store, *Store, *storepath.Path) {
				t.Helper()
				src := newStoreTestStore(t)
				dst := newStoreTestStore(t)
				path := newStoreTestPath(t, src, testZeroStorePath)
				if err := path.Close(); err != nil {
					t.Fatalf("Path.Close() error = %v", err)
				}
				return src, dst, path
			},
			wantErr:    true,
			wantClosed: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			src, dst, path := tt.setup(t)
			err := src.CopyClosure(dst, path)
			if tt.wantErr {
				if err == nil {
					t.Fatal("Store.CopyClosure() error = nil, want error")
				}
				if tt.wantClosed && !errors.Is(err, status.ErrClosed) {
					t.Fatalf("Store.CopyClosure() error = %v, want status.ErrClosed", err)
				}
				if tt.backendError && errors.Is(err, status.ErrClosed) {
					t.Fatalf("Store.CopyClosure() error = %v, want backend error", err)
				}
				return
			}
			if err != nil {
				t.Fatalf("Store.CopyClosure() error = %v", err)
			}
		})
	}
}

func TestStore_CopyPathTo(t *testing.T) {
	tests := []struct {
		name         string
		setup        func(t *testing.T) (*Store, *Store, *storepath.Path)
		opts         []CopyOption
		wantErr      bool
		wantClosed   bool
		backendError bool
	}{
		{
			name: "dummy_to_dummy_default_options",
			setup: func(t *testing.T) (*Store, *Store, *storepath.Path) {
				t.Helper()
				src := newStoreTestStore(t)
				dst := newStoreTestStore(t)
				return src, dst, newStoreTestPath(t, src, testZeroStorePath)
			},
			wantErr:      true,
			backendError: true,
		},
		{
			name: "dummy_to_dummy_explicit_options",
			setup: func(t *testing.T) (*Store, *Store, *storepath.Path) {
				t.Helper()
				src := newStoreTestStore(t)
				dst := newStoreTestStore(t)
				return src, dst, newStoreTestPath(t, src, testZeroStorePath)
			},
			opts:         []CopyOption{WithCopyRepair(false), WithCopyCheckSignatures(false)},
			wantErr:      true,
			backendError: true,
		},
		{
			name: "closed_source_store",
			setup: func(t *testing.T) (*Store, *Store, *storepath.Path) {
				t.Helper()
				src := newStoreTestStore(t)
				dst := newStoreTestStore(t)
				path := newStoreTestPath(t, src, testZeroStorePath)
				if err := src.Close(); err != nil {
					t.Fatalf("Store.Close() error = %v", err)
				}
				return src, dst, path
			},
			wantErr:    true,
			wantClosed: true,
		},
		{
			name: "closed_destination_store",
			setup: func(t *testing.T) (*Store, *Store, *storepath.Path) {
				t.Helper()
				src := newStoreTestStore(t)
				dst := newStoreTestStore(t)
				path := newStoreTestPath(t, src, testZeroStorePath)
				if err := dst.Close(); err != nil {
					t.Fatalf("Store.Close() error = %v", err)
				}
				return src, dst, path
			},
			wantErr:    true,
			wantClosed: true,
		},
		{
			name: "closed_path",
			setup: func(t *testing.T) (*Store, *Store, *storepath.Path) {
				t.Helper()
				src := newStoreTestStore(t)
				dst := newStoreTestStore(t)
				path := newStoreTestPath(t, src, testZeroStorePath)
				if err := path.Close(); err != nil {
					t.Fatalf("Path.Close() error = %v", err)
				}
				return src, dst, path
			},
			wantErr:    true,
			wantClosed: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			src, dst, path := tt.setup(t)
			err := src.CopyPathTo(dst, path, tt.opts...)
			if tt.wantErr {
				if err == nil {
					t.Fatal("Store.CopyPathTo() error = nil, want error")
				}
				if tt.wantClosed && !errors.Is(err, status.ErrClosed) {
					t.Fatalf("Store.CopyPathTo() error = %v, want status.ErrClosed", err)
				}
				if tt.backendError && errors.Is(err, status.ErrClosed) {
					t.Fatalf("Store.CopyPathTo() error = %v, want backend error", err)
				}
				return
			}
			if err != nil {
				t.Fatalf("Store.CopyPathTo() error = %v", err)
			}
		})
	}
}
