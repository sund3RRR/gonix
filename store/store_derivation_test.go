package store

import (
	"bytes"
	"errors"
	"testing"

	"github.com/sund3RRR/gonix/internal/status"
	"github.com/sund3RRR/gonix/storepath"
)

func TestStore_DerivationFromJSON(t *testing.T) {
	tests := []struct {
		name    string
		setup   func(t *testing.T) *Store
		data    []byte
		wantErr bool
	}{
		{
			name: "valid_derivation_json",
			setup: func(t *testing.T) *Store {
				t.Helper()
				return newStoreTestStore(t)
			},
			data: []byte(testStoreDerivationJSON),
		},
		{
			name: "invalid_derivation_json",
			setup: func(t *testing.T) *Store {
				t.Helper()
				return newStoreTestStore(t)
			},
			data:    []byte(`{"version": 4}`),
			wantErr: true,
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
			data:    []byte(testStoreDerivationJSON),
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := tt.setup(t).DerivationFromJSON(tt.data)
			if tt.wantErr {
				if err == nil {
					t.Fatal("Store.DerivationFromJSON() error = nil, want error")
				}
				if got != nil {
					t.Fatalf("Store.DerivationFromJSON() = %v, want nil", got)
				}
				return
			}
			if err != nil {
				t.Fatalf("Store.DerivationFromJSON() error = %v", err)
			}
			t.Cleanup(func() {
				if err := got.Close(); err != nil {
					t.Fatalf("Derivation.Close() error = %v", err)
				}
			})

			raw := got.SerializeJSON()
			if bytes.Equal(raw, tt.data) {
				t.Fatal("SerializeJSON() returned caller formatting instead of Nix-normalized JSON")
			}

			data, err := got.Deserialize()
			if err != nil {
				t.Fatalf("Derivation.Deserialize() error = %v", err)
			}
			if data.Name != "gonix-test" {
				t.Fatalf("derivation name = %v, want gonix-test", data.Name)
			}
		})
	}
}

func TestStore_DerivationFromPath(t *testing.T) {
	tests := []struct {
		name         string
		setup        func(t *testing.T) (*Store, *storepath.Path)
		wantErr      bool
		wantClosed   bool
		backendError bool
	}{
		{
			name: "unsupported_path_in_dummy_store",
			setup: func(t *testing.T) (*Store, *storepath.Path) {
				t.Helper()
				s := newStoreTestStore(t)
				return s, newStoreTestPath(t, s, testZeroStorePath)
			},
			wantErr:      true,
			backendError: true,
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
			wantErr:    true,
			wantClosed: true,
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
			wantErr:    true,
			wantClosed: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s, path := tt.setup(t)
			got, err := s.DerivationFromPath(path)
			if tt.wantErr {
				if err == nil {
					t.Fatal("Store.DerivationFromPath() error = nil, want error")
				}
				if tt.wantClosed && !errors.Is(err, status.ErrClosed) {
					t.Fatalf("Store.DerivationFromPath() error = %v, want status.ErrClosed", err)
				}
				if tt.backendError && errors.Is(err, status.ErrClosed) {
					t.Fatalf("Store.DerivationFromPath() error = %v, want backend error", err)
				}
				if got != nil {
					t.Fatalf("Store.DerivationFromPath() = %v, want nil", got)
				}
				return
			}
			if err != nil {
				t.Fatalf("Store.DerivationFromPath() error = %v", err)
			}
			t.Cleanup(func() {
				if err := got.Close(); err != nil {
					t.Fatalf("Derivation.Close() error = %v", err)
				}
			})
		})
	}
}

func TestStore_AddDerivation(t *testing.T) {
	tests := []struct {
		name         string
		setup        func(t *testing.T) (*Store, *Derivation)
		wantErr      bool
		wantClosed   bool
		backendError bool
	}{
		{
			name: "unsupported_write_derivation_in_dummy_store",
			setup: func(t *testing.T) (*Store, *Derivation) {
				t.Helper()
				s := newStoreTestStore(t)
				return s, newStoreTestDerivation(t, s)
			},
			wantErr:      true,
			backendError: true,
		},
		{
			name: "closed_store",
			setup: func(t *testing.T) (*Store, *Derivation) {
				t.Helper()
				s := newStoreTestStore(t)
				d := newStoreTestDerivation(t, s)
				if err := s.Close(); err != nil {
					t.Fatalf("Store.Close() error = %v", err)
				}
				return s, d
			},
			wantErr:    true,
			wantClosed: true,
		},
		{
			name: "closed_derivation",
			setup: func(t *testing.T) (*Store, *Derivation) {
				t.Helper()
				s := newStoreTestStore(t)
				d := newStoreTestDerivation(t, s)
				if err := d.Close(); err != nil {
					t.Fatalf("Derivation.Close() error = %v", err)
				}
				return s, d
			},
			wantErr:    true,
			wantClosed: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s, d := tt.setup(t)
			got, err := s.AddDerivation(d)
			if tt.wantErr {
				if err == nil {
					t.Fatal("Store.AddDerivation() error = nil, want error")
				}
				if tt.wantClosed && !errors.Is(err, status.ErrClosed) {
					t.Fatalf("Store.AddDerivation() error = %v, want status.ErrClosed", err)
				}
				if tt.backendError && errors.Is(err, status.ErrClosed) {
					t.Fatalf("Store.AddDerivation() error = %v, want backend error", err)
				}
				if got != nil {
					t.Fatalf("Store.AddDerivation() = %v, want nil", got)
				}
				return
			}
			if err != nil {
				t.Fatalf("Store.AddDerivation() error = %v", err)
			}
			t.Cleanup(func() {
				if err := got.Close(); err != nil {
					t.Fatalf("Path.Close() error = %v", err)
				}
			})
		})
	}
}
