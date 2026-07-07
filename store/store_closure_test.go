package store

import (
	"errors"
	"testing"

	"github.com/sund3RRR/gonix/internal/status"
	"github.com/sund3RRR/gonix/storepath"
)

func TestStore_RealiseOutputValidation(t *testing.T) {
	tests := []struct {
		name       string
		setup      func(t *testing.T) (*Store, *storepath.Path)
		outputName string
		wantClosed bool
	}{
		{
			name: "empty_output_name",
			setup: func(t *testing.T) (*Store, *storepath.Path) {
				t.Helper()
				s := newStoreTestStore(t)
				return s, newStoreTestPath(t, s, testZeroStorePath+".drv")
			},
		},
		{
			name: "closed_store",
			setup: func(t *testing.T) (*Store, *storepath.Path) {
				t.Helper()
				s := newStoreTestStore(t)
				path := newStoreTestPath(t, s, testZeroStorePath+".drv")
				if err := s.Close(); err != nil {
					t.Fatalf("Store.Close() error = %v", err)
				}
				return s, path
			},
			outputName: "out",
			wantClosed: true,
		},
		{
			name: "closed_path",
			setup: func(t *testing.T) (*Store, *storepath.Path) {
				t.Helper()
				s := newStoreTestStore(t)
				path := newStoreTestPath(t, s, testZeroStorePath+".drv")
				if err := path.Close(); err != nil {
					t.Fatalf("Path.Close() error = %v", err)
				}
				return s, path
			},
			outputName: "out",
			wantClosed: true,
		},
		{
			name: "backend_error",
			setup: func(t *testing.T) (*Store, *storepath.Path) {
				t.Helper()
				s := newStoreTestStore(t)
				return s, newStoreTestPath(t, s, testZeroStorePath)
			},
			outputName: "out",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s, path := tt.setup(t)
			got, err := s.RealiseOutput(path, tt.outputName)
			if err == nil {
				_ = got.Close()
				t.Fatal("Store.RealiseOutput() error = nil, want error")
			}
			if tt.wantClosed && !errors.Is(err, status.ErrClosed) {
				t.Fatalf("Store.RealiseOutput() error = %v, want status.ErrClosed", err)
			}
			if got.Path != nil {
				t.Fatalf("Store.RealiseOutput() = %#v, want empty realization", got)
			}
		})
	}
}
