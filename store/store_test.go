// Package store wraps Nix store handles and store-backed operations.
package store

import (
	"errors"
	"strings"
	"testing"

	"github.com/sund3RRR/gonix/internal/status"
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

func requireStoreClosedError(t *testing.T, err error) {
	t.Helper()

	if err == nil {
		t.Fatal("error = nil, want status.ErrClosed")
	}
	if !errors.Is(err, status.ErrClosed) {
		t.Fatalf("error = %v, want errors.Is(..., status.ErrClosed)", err)
	}
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

				uri, err := s.URI()
				if err != nil {
					t.Fatalf("Store.URI() error = %v", err)
				}
				if strings.TrimSpace(uri) == "" {
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

func TestStore_Version(t *testing.T) {
	tests := []struct {
		name    string
		setup   func(t *testing.T) *Store
		wantErr bool
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
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := tt.setup(t).Version()
			if tt.wantErr {
				requireStoreClosedError(t, err)
				return
			}
			if err != nil {
				t.Fatalf("Store.Version() error = %v", err)
			}
		})
	}
}
