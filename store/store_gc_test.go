package store

import (
	"errors"
	"math"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/sund3RRR/gonix/internal/status"
	"github.com/sund3RRR/gonix/nixcontext"
)

const testGCContentAddressedDerivationJSON = `{
  "name": "gonix-gc-test",
  "version": 4,
  "outputs": {
    "out": {
      "method": "nar",
      "hashAlgo": "sha256"
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
    "name": "gonix-gc-test",
    "out": "/unused",
    "system": "x86_64-linux"
  }
}`

func newStoreGCTestStore(t *testing.T, names ...string) (*Store, string, []string) {
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

	if err := ctx.SetSetting("experimental-features", "ca-derivations"); err != nil {
		t.Fatalf("Context.SetSetting(ca-derivations) error = %v", err)
	}

	root, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatalf("filepath.EvalSymlinks() error = %v", err)
	}
	storeOptions := []Option{
		WithStoreDir(filepath.Join(root, "store")),
		WithStateDir(filepath.Join(root, "state")),
		WithLogDir(filepath.Join(root, "log")),
	}

	seedStore, err := New(ctx, "local", storeOptions...)
	if err != nil {
		t.Fatalf("New(local seed store) error = %v", err)
	}

	paths := make([]string, 0, len(names))
	for _, name := range names {
		raw := strings.ReplaceAll(testGCContentAddressedDerivationJSON, "gonix-gc-test", name)
		derivation, err := seedStore.DerivationFromJSON([]byte(raw))
		if err != nil {
			_ = seedStore.Close()
			t.Fatalf("Store.DerivationFromJSON(%q) error = %v", name, err)
		}

		path, addErr := seedStore.AddDerivation(derivation)
		closeErr := derivation.Close()
		if addErr != nil {
			_ = seedStore.Close()
			t.Fatalf("Store.AddDerivation(%q) error = %v", name, addErr)
		}
		if closeErr != nil {
			_ = path.Close()
			_ = seedStore.Close()
			t.Fatalf("Derivation.Close(%q) error = %v", name, closeErr)
		}

		paths = append(paths, seedStore.PrintPath(path.Hash(), path.Name()))
		if err := path.Close(); err != nil {
			_ = seedStore.Close()
			t.Fatalf("Path.Close(%q) error = %v", name, err)
		}
	}

	if err := seedStore.Close(); err != nil {
		t.Fatalf("seed Store.Close() error = %v", err)
	}

	gcStore, err := New(ctx, "local", storeOptions...)
	if err != nil {
		t.Fatalf("New(local GC store) error = %v", err)
	}
	t.Cleanup(func() {
		if err := gcStore.Close(); err != nil {
			t.Fatalf("Store.Close() error = %v", err)
		}
	})

	return gcStore, root, paths
}

func resultPathSet(paths []string) map[string]struct{} {
	result := make(map[string]struct{}, len(paths))
	for _, path := range paths {
		result[path] = struct{}{}
	}
	return result
}

func TestStore_RootsAndCollection(t *testing.T) {
	s, root, paths := newStoreGCTestStore(
		t,
		"gonix-gc-temp-root",
		"gonix-gc-permanent-root",
		"gonix-gc-dead-specific",
		"gonix-gc-dead-all",
	)
	tempPath, permanentPath, deadPath, deadAllPath := paths[0], paths[1], paths[2], paths[3]

	if err := s.AddTempRoot(tempPath); err != nil {
		t.Fatalf("Store.AddTempRoot() error = %v", err)
	}

	requestedRoot := filepath.Join(root, "roots", "..", "roots", "permanent")
	wantRoot := filepath.Clean(requestedRoot)
	gotRoot, err := s.AddPermanentRoot(permanentPath, requestedRoot)
	if err != nil {
		t.Fatalf("Store.AddPermanentRoot() error = %v", err)
	}
	if gotRoot != wantRoot {
		t.Fatalf("Store.AddPermanentRoot() = %q, want %q", gotRoot, wantRoot)
	}
	target, err := os.Readlink(wantRoot)
	if err != nil {
		t.Fatalf("os.Readlink(%q) error = %v", wantRoot, err)
	}
	if target != permanentPath {
		t.Fatalf("permanent root target = %q, want %q", target, permanentPath)
	}

	for _, censor := range []bool{false, true} {
		roots, err := s.FindRoots(censor)
		if err != nil {
			t.Fatalf("Store.FindRoots(%t) error = %v", censor, err)
		}

		found := false
		for _, root := range roots {
			if root.Path == permanentPath && root.Link == wantRoot {
				found = true
			}
		}
		if !found {
			t.Fatalf("Store.FindRoots(%t) did not include %q -> %q", censor, wantRoot, permanentPath)
		}
	}

	live, err := s.CollectGarbage(GCReturnLive)
	if err != nil {
		t.Fatalf("Store.CollectGarbage(GCReturnLive) error = %v", err)
	}
	livePaths := resultPathSet(live.Paths)
	for _, path := range []string{tempPath, permanentPath} {
		if _, ok := livePaths[path]; !ok {
			t.Fatalf("live paths missing rooted path %q", path)
		}
	}

	dead, err := s.CollectGarbage(GCReturnDead)
	if err != nil {
		t.Fatalf("Store.CollectGarbage(GCReturnDead) error = %v", err)
	}
	deadPaths := resultPathSet(dead.Paths)
	if _, ok := deadPaths[deadPath]; !ok {
		t.Fatalf("dead paths missing %q", deadPath)
	}
	for _, path := range []string{tempPath, permanentPath} {
		if _, ok := deadPaths[path]; ok {
			t.Fatalf("dead paths unexpectedly include rooted path %q", path)
		}
	}

	limited, err := s.CollectGarbage(GCDeleteDead, WithGCMaxFreed(0))
	if err != nil {
		t.Fatalf("Store.CollectGarbage(GCDeleteDead, max=0) error = %v", err)
	}
	if limited.BytesFreed != 0 {
		t.Fatalf("zero-limit GC freed %d bytes, want 0", limited.BytesFreed)
	}
	if len(limited.Paths) != 0 {
		t.Fatalf("zero-limit GC returned %d paths, want 0", len(limited.Paths))
	}
	if _, err := os.Stat(deadPath); err != nil {
		t.Fatalf("os.Stat(%q) after zero-limit GC error = %v", deadPath, err)
	}

	specific, err := s.CollectGarbage(
		GCDeleteSpecific,
		WithGCPathsToDelete(deadPath),
	)
	if err != nil {
		t.Fatalf("Store.CollectGarbage(GCDeleteSpecific) error = %v", err)
	}
	if specific.BytesFreed == 0 {
		t.Fatal("specific GC reported zero bytes freed")
	}
	if !slices.Contains(specific.Paths, deadPath) {
		t.Fatalf("specific GC paths = %v, want %q", specific.Paths, deadPath)
	}
	if _, err := os.Stat(deadPath); !os.IsNotExist(err) {
		t.Fatalf("os.Stat(%q) after specific GC error = %v, want not exist", deadPath, err)
	}

	all, err := s.CollectGarbage(GCDeleteDead)
	if err != nil {
		t.Fatalf("Store.CollectGarbage(GCDeleteDead) error = %v", err)
	}
	if all.BytesFreed == 0 {
		t.Fatal("unlimited GC reported zero bytes freed")
	}
	if !slices.Contains(all.Paths, deadAllPath) {
		t.Fatalf("unlimited GC paths = %v, want %q", all.Paths, deadAllPath)
	}
	if _, err := os.Stat(deadAllPath); !os.IsNotExist(err) {
		t.Fatalf("os.Stat(%q) after unlimited GC error = %v, want not exist", deadAllPath, err)
	}
}

func TestStore_GCOptions(t *testing.T) {
	paths := []string{testZeroStorePath}
	cfg := GCConfig{MaxFreed: math.MaxUint64}

	WithGCIgnoreLiveness(true)(&cfg)
	WithGCPathsToDelete(paths...)(&cfg)
	WithGCMaxFreed(0)(&cfg)
	paths[0] = testNonZeroStorePath

	if !cfg.IgnoreLiveness {
		t.Fatal("WithGCIgnoreLiveness(true) did not enable IgnoreLiveness")
	}
	if len(cfg.PathsToDelete) != 1 || cfg.PathsToDelete[0] != testZeroStorePath {
		t.Fatalf("WithGCPathsToDelete did not copy input slice: %v", cfg.PathsToDelete)
	}
	if cfg.MaxFreed != 0 {
		t.Fatalf("WithGCMaxFreed(0) = %d, want 0", cfg.MaxFreed)
	}
}

func TestStore_GCErrors(t *testing.T) {
	t.Run("dummy_backend", func(t *testing.T) {
		s := newStoreTestStore(t)

		if err := s.AddTempRoot(testZeroStorePath); err != nil {
			t.Fatalf("Store.AddTempRoot(dummy) error = %v", err)
		}
		if root, err := s.AddPermanentRoot(testZeroStorePath, filepath.Join(t.TempDir(), "root")); err == nil {
			t.Fatalf("Store.AddPermanentRoot(dummy) = %q, nil; want error", root)
		}
		if _, err := s.FindRoots(false); err == nil {
			t.Fatal("Store.FindRoots(dummy) error = nil, want error")
		}
		if result, err := s.CollectGarbage(GCReturnDead); err == nil {
			t.Fatalf("Store.CollectGarbage(dummy) = %+v, nil; want error", result)
		}
	})

	t.Run("invalid_action", func(t *testing.T) {
		s := newStoreTestStore(t)
		if result, err := s.CollectGarbage(GCAction(99)); err == nil {
			t.Fatalf("Store.CollectGarbage(invalid) = %+v, nil; want error", result)
		}
	})

	t.Run("closed_store", func(t *testing.T) {
		s := newStoreTestStore(t)
		if err := s.Close(); err != nil {
			t.Fatalf("Store.Close() error = %v", err)
		}

		requireGCClosed(t, s.AddTempRoot(testZeroStorePath))
		_, err := s.AddPermanentRoot(testZeroStorePath, filepath.Join(t.TempDir(), "root"))
		requireGCClosed(t, err)
		_, err = s.FindRoots(false)
		requireGCClosed(t, err)
		_, err = s.CollectGarbage(GCReturnDead)
		requireGCClosed(t, err)
	})

	t.Run("invalid_path", func(t *testing.T) {
		s := newStoreTestStore(t)

		if err := s.AddTempRoot("/not-a-store-path"); err == nil {
			t.Fatal("Store.AddTempRoot(invalid path) error = nil")
		}
		if _, err := s.AddPermanentRoot("/not-a-store-path", filepath.Join(t.TempDir(), "root")); err == nil {
			t.Fatal("Store.AddPermanentRoot(invalid path) error = nil")
		}
		if _, err := s.CollectGarbage(GCDeleteSpecific, WithGCPathsToDelete("/not-a-store-path")); err == nil {
			t.Fatal("Store.CollectGarbage(invalid path) error = nil")
		}
	})

	t.Run("closed_context", func(t *testing.T) {
		ctx, err := nixcontext.New(nixcontext.Config{})
		if err != nil {
			t.Fatalf("nixcontext.New() error = %v", err)
		}
		s, err := New(ctx, "dummy://")
		if err != nil {
			t.Fatalf("New(dummy://) error = %v", err)
		}
		t.Cleanup(func() {
			_ = s.Close()
		})
		if err := ctx.Close(); err != nil {
			t.Fatalf("Context.Close() error = %v", err)
		}

		requireGCClosed(t, s.AddTempRoot(testZeroStorePath))
		_, err = s.AddPermanentRoot(testZeroStorePath, filepath.Join(t.TempDir(), "root"))
		requireGCClosed(t, err)
		_, err = s.FindRoots(false)
		requireGCClosed(t, err)
		_, err = s.CollectGarbage(GCReturnDead)
		requireGCClosed(t, err)
	})
}

func requireGCClosed(t *testing.T, err error) {
	t.Helper()

	if !errors.Is(err, status.ErrClosed) {
		t.Fatalf("error = %v, want status.ErrClosed", err)
	}
}
