package store

import (
	"strings"
	"testing"

	"github.com/sund3RRR/gonix/storepath"
)

func TestStore_URI(t *testing.T) {
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
			got, err := tt.setup(t).URI()
			if tt.wantErr {
				requireStoreClosedError(t, err)
				return
			}
			if err != nil {
				t.Fatalf("Store.URI() error = %v", err)
			}
			if strings.TrimSpace(got) == "" {
				t.Fatal("Store.URI() returned empty URI")
			}
		})
	}
}

func TestStore_StoreDir(t *testing.T) {
	tests := []struct {
		name    string
		setup   func(t *testing.T) *Store
		want    string
		wantErr bool
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
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := tt.setup(t).StoreDir()
			if tt.wantErr {
				requireStoreClosedError(t, err)
				return
			}
			if err != nil {
				t.Fatalf("Store.StoreDir() error = %v", err)
			}
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
			got, err := s.PrintPath(path)
			if tt.wantErr {
				if err == nil {
					t.Fatal("Store.PrintPath() error = nil, want error")
				}
				return
			}
			if err != nil {
				t.Fatalf("Store.PrintPath() error = %v", err)
			}
			if got != rawPath {
				t.Fatalf("Store.PrintPath() = %q, want %q", got, rawPath)
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
