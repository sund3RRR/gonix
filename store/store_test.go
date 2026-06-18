// Package store wraps Nix store handles and store-backed operations.
package store

import (
	"strings"
	"testing"

	"github.com/sund3RRR/gonix/nixcontext"
	"github.com/sund3RRR/gonix/storepath"
)

const (
	testZeroStorePath    = "/nix/store/00000000000000000000000000000000-demo"
	testNonZeroStorePath = "/nix/store/11111111111111111111111111111111-source"
	testZeroHashPart     = "00000000000000000000000000000000"
)

const testStoreDerivationJSON = `{
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

func newStoreTestContext(t *testing.T) *nixcontext.Context {
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

func newStoreTestStore(t *testing.T) *Store {
	t.Helper()

	s, err := New(newStoreTestContext(t), "dummy://")
	if err != nil {
		t.Fatalf("New(dummy://) error = %v", err)
	}
	t.Cleanup(func() {
		if err := s.Close(); err != nil {
			t.Fatalf("Store.Close() error = %v", err)
		}
	})

	return s
}

func newStoreTestPath(t *testing.T, s *Store, rawPath string) *storepath.Path {
	t.Helper()

	path, err := s.ParsePath(rawPath)
	if err != nil {
		t.Fatalf("Store.ParsePath(%q) error = %v", rawPath, err)
	}
	t.Cleanup(func() {
		if err := path.Close(); err != nil {
			t.Fatalf("Path.Close() error = %v", err)
		}
	})

	return path
}

func newStoreTestDerivation(t *testing.T, s *Store) *Derivation {
	t.Helper()

	d, err := s.DerivationFromJSON([]byte(testStoreDerivationJSON))
	if err != nil {
		t.Fatalf("Store.DerivationFromJSON() error = %v", err)
	}
	t.Cleanup(func() {
		if err := d.Close(); err != nil {
			t.Fatalf("Derivation.Close() error = %v", err)
		}
	})

	return d
}

func TestNew(t *testing.T) {
	tests := []struct {
		name      string
		uri       string
		opts      []Option
		wantErr   bool
		assertion func(t *testing.T, s *Store)
	}{
		{
			name: "opens_dummy_store",
			uri:  "dummy://",
			assertion: func(t *testing.T, s *Store) {
				t.Helper()

				if strings.TrimSpace(s.URI()) == "" {
					t.Fatal("Store.URI() returned empty URI")
				}
			},
		},
		{
			name: "opens_with_options",
			uri:  "dummy://",
			opts: []Option{WithStoreDir(DefaultDir), WithReadOnly(true)},
		},
		{
			name:    "invalid_uri",
			uri:     "gonix-test-invalid-store://",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := New(newStoreTestContext(t), tt.uri, tt.opts...)
			if (err != nil) != tt.wantErr {
				t.Fatalf("New() error = %v, wantErr %v", err, tt.wantErr)
			}
			if tt.wantErr {
				if got != nil {
					t.Fatalf("New() = %v, want nil", got)
				}
				return
			}
			t.Cleanup(func() {
				if err := got.Close(); err != nil {
					t.Fatalf("Store.Close() error = %v", err)
				}
			})
			if got == nil {
				t.Fatal("New() = nil, want store")
			}
			if tt.assertion != nil {
				tt.assertion(t, got)
			}
		})
	}
}

func TestStore_CachedMetadataAfterClose(t *testing.T) {
	ctx := newStoreTestContext(t)
	s, err := New(ctx, "dummy://", WithStoreDir("/custom/store"))
	if err != nil {
		t.Fatalf("New(dummy://) error = %v", err)
	}

	wantURI := s.URI()
	wantStoreDir := s.StoreDir()
	wantVersion := s.Version()
	if strings.TrimSpace(wantURI) == "" {
		t.Fatal("Store.URI() returned empty URI")
	}
	if wantStoreDir != "/custom/store" {
		t.Fatalf("Store.StoreDir() = %q, want %q", wantStoreDir, "/custom/store")
	}

	if err := s.Close(); err != nil {
		t.Fatalf("Store.Close() error = %v", err)
	}
	if err := ctx.Close(); err != nil {
		t.Fatalf("Context.Close() error = %v", err)
	}

	if got := s.URI(); got != wantURI {
		t.Fatalf("Store.URI() after close = %q, want %q", got, wantURI)
	}
	if got := s.StoreDir(); got != wantStoreDir {
		t.Fatalf("Store.StoreDir() after close = %q, want %q", got, wantStoreDir)
	}
	if got := s.Version(); got != wantVersion {
		t.Fatalf("Store.Version() after close = %q, want %q", got, wantVersion)
	}
}

func TestStore_URI(t *testing.T) {
	tests := []struct {
		name  string
		setup func(t *testing.T) *Store
	}{
		{
			name: "open_store",
			setup: func(t *testing.T) *Store {
				t.Helper()
				return newStoreTestStore(t)
			},
		},
		{
			name: "closed_store",
			setup: func(t *testing.T) *Store {
				t.Helper()
				s := newStoreTestStore(t)
				if err := s.Close(); err != nil {
					t.Fatalf("Store.Close() error = %v", err)
				}
				return s
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.setup(t).URI()
			if strings.TrimSpace(got) == "" {
				t.Fatal("Store.URI() returned empty URI")
			}
		})
	}
}

func TestStore_StoreDir(t *testing.T) {
	tests := []struct {
		name  string
		setup func(t *testing.T) *Store
		want  string
	}{
		{
			name: "open_store",
			setup: func(t *testing.T) *Store {
				t.Helper()
				return newStoreTestStore(t)
			},
			want: DefaultDir,
		},
		{
			name: "closed_store",
			setup: func(t *testing.T) *Store {
				t.Helper()
				s := newStoreTestStore(t)
				if err := s.Close(); err != nil {
					t.Fatalf("Store.Close() error = %v", err)
				}
				return s
			},
			want: DefaultDir,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.setup(t).StoreDir()
			if got != tt.want {
				t.Fatalf("Store.StoreDir() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestStore_ParsePath(t *testing.T) {
	tests := []struct {
		name    string
		path    string
		setup   func(t *testing.T) *Store
		wantErr bool
	}{
		{
			name: "valid_path",
			path: testZeroStorePath,
			setup: func(t *testing.T) *Store {
				t.Helper()
				return newStoreTestStore(t)
			},
		},
		{
			name: "invalid_path",
			path: "/not-a-store-path",
			setup: func(t *testing.T) *Store {
				t.Helper()
				return newStoreTestStore(t)
			},
			wantErr: true,
		},
		{
			name: "closed_store",
			path: testZeroStorePath,
			setup: func(t *testing.T) *Store {
				t.Helper()
				s := newStoreTestStore(t)
				if err := s.Close(); err != nil {
					t.Fatalf("Store.Close() error = %v", err)
				}
				return s
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := tt.setup(t).ParsePath(tt.path)
			if tt.wantErr {
				if err == nil {
					t.Fatal("Store.ParsePath() error = nil, want error")
				}
				return
			}
			if err != nil {
				t.Fatalf("Store.ParsePath() error = %v", err)
			}
			t.Cleanup(func() {
				if err := got.Close(); err != nil {
					t.Fatalf("Path.Close() error = %v", err)
				}
			})
			if got == nil {
				t.Fatal("Store.ParsePath() = nil, want path")
			}
		})
	}
}

func TestStore_PrintPath(t *testing.T) {
	tests := []struct {
		name     string
		setup    func(t *testing.T) *Store
		hash     [20]byte
		pathName string
		want     string
	}{
		{
			name: "zero_hash_path",
			setup: func(t *testing.T) *Store {
				t.Helper()
				return newStoreTestStore(t)
			},
			pathName: "demo",
			want:     testZeroStorePath,
		},
		{
			name: "nonzero_hash_path",
			setup: func(t *testing.T) *Store {
				t.Helper()
				return newStoreTestStore(t)
			},
			hash: [20]byte{
				0x21, 0x84, 0x10, 0x42, 0x08,
				0x21, 0x84, 0x10, 0x42, 0x08,
				0x21, 0x84, 0x10, 0x42, 0x08,
				0x21, 0x84, 0x10, 0x42, 0x08,
			},
			pathName: "source",
			want:     testNonZeroStorePath,
		},
		{
			name: "custom_store_dir",
			setup: func(t *testing.T) *Store {
				t.Helper()

				s, err := New(newStoreTestContext(t), "dummy://", WithStoreDir("/custom/store"))
				if err != nil {
					t.Fatalf("New(dummy://) error = %v", err)
				}
				t.Cleanup(func() {
					if err := s.Close(); err != nil {
						t.Fatalf("Store.Close() error = %v", err)
					}
				})
				return s
			},
			pathName: "demo",
			want:     "/custom/store/00000000000000000000000000000000-demo",
		},
		{
			name: "closed_store",
			setup: func(t *testing.T) *Store {
				t.Helper()
				s := newStoreTestStore(t)
				if err := s.Close(); err != nil {
					t.Fatalf("Store.Close() error = %v", err)
				}
				return s
			},
			pathName: "demo",
			want:     testZeroStorePath,
		},
		{
			name: "closed_context",
			setup: func(t *testing.T) *Store {
				t.Helper()

				ctx := newStoreTestContext(t)
				s, err := New(ctx, "dummy://")
				if err != nil {
					t.Fatalf("New(dummy://) error = %v", err)
				}
				t.Cleanup(func() {
					if err := s.Close(); err != nil {
						t.Fatalf("Store.Close() error = %v", err)
					}
				})

				if err := ctx.Close(); err != nil {
					t.Fatalf("Context.Close() error = %v", err)
				}
				return s
			},
			pathName: "demo",
			want:     testZeroStorePath,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.setup(t).PrintPath(tt.hash, tt.pathName)
			if got != tt.want {
				t.Fatalf("Store.PrintPath() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestStore_PathFromHash(t *testing.T) {
	tests := []struct {
		name    string
		hash    []byte
		setup   func(t *testing.T) *Store
		wantErr bool
	}{
		{
			name: "missing_hash_in_dummy_store",
			hash: []byte(testZeroHashPart),
			setup: func(t *testing.T) *Store {
				t.Helper()
				return newStoreTestStore(t)
			},
			wantErr: true,
		},
		{
			name: "closed_store",
			hash: []byte(testZeroHashPart),
			setup: func(t *testing.T) *Store {
				t.Helper()
				s := newStoreTestStore(t)
				if err := s.Close(); err != nil {
					t.Fatalf("Store.Close() error = %v", err)
				}
				return s
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := tt.setup(t).PathFromHash(tt.hash)
			if tt.wantErr {
				if err == nil {
					t.Fatal("Store.PathFromHash() error = nil, want error")
				}
				if got != nil {
					t.Fatalf("Store.PathFromHash() = %v, want nil", got)
				}
				return
			}
			if err != nil {
				t.Fatalf("Store.PathFromHash() error = %v", err)
			}
			t.Cleanup(func() {
				if err := got.Close(); err != nil {
					t.Fatalf("Path.Close() error = %v", err)
				}
			})
		})
	}
}

func TestStore_RealPath(t *testing.T) {
	tests := []struct {
		name    string
		setup   func(t *testing.T) (*Store, string)
		wantErr bool
	}{
		{
			name: "valid_path",
			setup: func(t *testing.T) (*Store, string) {
				t.Helper()
				return newStoreTestStore(t), testZeroStorePath
			},
		},
		{
			name: "closed_store",
			setup: func(t *testing.T) (*Store, string) {
				t.Helper()
				s := newStoreTestStore(t)
				_ = newStoreTestPath(t, s, testZeroStorePath)
				if err := s.Close(); err != nil {
					t.Fatalf("Store.Close() error = %v", err)
				}
				return s, testZeroStorePath
			},
			wantErr: true,
		},
		{
			name: "closed_path",
			setup: func(t *testing.T) (*Store, string) {
				t.Helper()
				return newStoreTestStore(t), ""
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s, rawPath := tt.setup(t)
			var path *storepath.Path
			if rawPath == testZeroStorePath && s.ptr == nil {
				openStore := newStoreTestStore(t)
				path = newStoreTestPath(t, openStore, testZeroStorePath)
			} else {
				path = newStoreTestPath(t, s, testZeroStorePath)
			}
			if rawPath == "" {
				if err := path.Close(); err != nil {
					t.Fatalf("Path.Close() error = %v", err)
				}
			}
			got, err := s.RealPath(path)
			if tt.wantErr {
				if err == nil {
					t.Fatal("Store.RealPath() error = nil, want error")
				}
				return
			}
			if err != nil {
				t.Fatalf("Store.RealPath() error = %v", err)
			}
			if got != rawPath {
				t.Fatalf("Store.RealPath() = %q, want %q", got, rawPath)
			}
		})
	}
}

func TestStore_IsValidPath(t *testing.T) {
	tests := []struct {
		name    string
		setup   func(t *testing.T) (*Store, *storepath.Path)
		want    bool
		wantErr bool
	}{
		{
			name: "dummy_store_arbitrary_path_is_not_valid",
			setup: func(t *testing.T) (*Store, *storepath.Path) {
				t.Helper()
				s := newStoreTestStore(t)
				return s, newStoreTestPath(t, s, testZeroStorePath)
			},
		},
		{
			name: "closed_store",
			setup: func(t *testing.T) (*Store, *storepath.Path) {
				t.Helper()
				s := newStoreTestStore(t)
				path := newStoreTestPath(t, s, testZeroStorePath)
				if err := s.Close(); err != nil {
					t.Fatalf("Store.Close() error = %v", err)
				}
				return s, path
			},
			wantErr: true,
		},
		{
			name: "closed_path",
			setup: func(t *testing.T) (*Store, *storepath.Path) {
				t.Helper()
				s := newStoreTestStore(t)
				path := newStoreTestPath(t, s, testZeroStorePath)
				if err := path.Close(); err != nil {
					t.Fatalf("Path.Close() error = %v", err)
				}
				return s, path
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s, path := tt.setup(t)
			got, err := s.IsValidPath(path)
			if tt.wantErr {
				if err == nil {
					t.Fatal("Store.IsValidPath() error = nil, want error")
				}
				return
			}
			if err != nil {
				t.Fatalf("Store.IsValidPath() error = %v", err)
			}
			if got != tt.want {
				t.Fatalf("Store.IsValidPath() = %v, want %v", got, tt.want)
			}
		})
	}
}
