package gonix

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/sund3RRR/gonix/eval"
	"github.com/sund3RRR/gonix/flake"
	"github.com/sund3RRR/gonix/store"
)

func TestNewRuntime(t *testing.T) {
	tests := []struct {
		name          string
		opts          []RuntimeOption
		wantErr       bool
		wantClosedErr bool
	}{
		{
			name: "default_runtime",
		},
		{
			name: "load_config",
			opts: []RuntimeOption{
				WithLoadConfig(),
			},
		},
		{
			name: "invalid_setting",
			opts: []RuntimeOption{
				WithSetting("go-bindings-test-setting-that-does-not-exist", "x"),
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r, err := NewRuntime(tt.opts...)
			if tt.wantErr {
				if err == nil {
					t.Fatal("NewRuntime() error = nil, want error")
				}
				if r != nil {
					t.Fatalf("NewRuntime() runtime = %v, want nil", r)
				}
				return
			}
			if err != nil {
				t.Fatalf("NewRuntime() error = %v", err)
			}

			if err := r.Close(); err != nil {
				t.Fatalf("Runtime.Close() error = %v", err)
			}
			if err := r.Close(); err != nil {
				t.Fatalf("second Runtime.Close() error = %v", err)
			}
		})
	}
}

func TestRuntimeOpenStore(t *testing.T) {
	tests := []struct {
		name    string
		run     func(t *testing.T, r *Runtime)
		wantErr bool
	}{
		{
			name: "open_store",
			run: func(t *testing.T, r *Runtime) {
				t.Helper()

				s, err := r.OpenStore("dummy://")
				if err != nil {
					t.Fatalf("Runtime.OpenStore() error = %v", err)
				}
				if s == nil {
					t.Fatal("Runtime.OpenStore() store = nil")
				}
			},
		},
		{
			name: "manual_store_close_before_runtime_close",
			run: func(t *testing.T, r *Runtime) {
				t.Helper()

				s, err := r.OpenStore("dummy://")
				if err != nil {
					t.Fatalf("Runtime.OpenStore() error = %v", err)
				}
				if err := s.Close(); err != nil {
					t.Fatalf("Store.Close() error = %v", err)
				}
				if err := r.Close(); err != nil {
					t.Fatalf("Runtime.Close() after Store.Close() error = %v", err)
				}
			},
		},
		{
			name: "runtime_close_closes_store",
			run: func(t *testing.T, r *Runtime) {
				t.Helper()

				s, err := r.OpenStore("dummy://")
				if err != nil {
					t.Fatalf("Runtime.OpenStore() error = %v", err)
				}
				if err := r.Close(); err != nil {
					t.Fatalf("Runtime.Close() error = %v", err)
				}
				if err := s.Close(); err != nil {
					t.Fatalf("Store.Close() after Runtime.Close() error = %v", err)
				}
			},
		},
		{
			name: "invalid_uri",
			run: func(t *testing.T, r *Runtime) {
				t.Helper()

				if _, err := r.OpenStore("go-bindings-test-invalid-store://"); err == nil {
					t.Fatal("Runtime.OpenStore(invalid) error = nil, want error")
				}
			},
		},
		{
			name: "open_store_after_close",
			run: func(t *testing.T, r *Runtime) {
				t.Helper()

				if err := r.Close(); err != nil {
					t.Fatalf("Runtime.Close() error = %v", err)
				}
				_, err := r.OpenStore("dummy://")
				requireRuntimeClosedError(t, err)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r, err := NewRuntime()
			if err != nil {
				t.Fatalf("NewRuntime() error = %v", err)
			}
			t.Cleanup(func() {
				if err := r.Close(); err != nil {
					t.Fatalf("Runtime.Close() error = %v", err)
				}
			})

			tt.run(t, r)
		})
	}
}

func TestContextFreeChildrenCloseAfterRuntimeClose(t *testing.T) {
	t.Run("store_path", func(t *testing.T) {
		r, err := NewRuntime()
		if err != nil {
			t.Fatalf("NewRuntime() error = %v", err)
		}
		s, err := r.OpenStore("dummy://")
		if err != nil {
			t.Fatalf("Runtime.OpenStore() error = %v", err)
		}
		path, err := s.ParsePath("/nix/store/00000000000000000000000000000000-demo")
		if err != nil {
			t.Fatalf("Store.ParsePath() error = %v", err)
		}

		if err := r.Close(); err != nil {
			t.Fatalf("Runtime.Close() error = %v", err)
		}
		if err := path.Close(); err != nil {
			t.Fatalf("Path.Close() after Runtime.Close() error = %v", err)
		}
	})

	t.Run("derivation", func(t *testing.T) {
		r, err := NewRuntime()
		if err != nil {
			t.Fatalf("NewRuntime() error = %v", err)
		}
		s, err := r.OpenStore("dummy://")
		if err != nil {
			t.Fatalf("Runtime.OpenStore() error = %v", err)
		}
		d, err := s.DerivationFromJSON([]byte(`{
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
}`))
		if err != nil {
			t.Fatalf("Store.DerivationFromJSON() error = %v", err)
		}

		if err := r.Close(); err != nil {
			t.Fatalf("Runtime.Close() error = %v", err)
		}
		if err := d.Close(); err != nil {
			t.Fatalf("Derivation.Close() after Runtime.Close() error = %v", err)
		}
	})

	t.Run("evaluator_without_live_values", func(t *testing.T) {
		r, err := NewRuntime()
		if err != nil {
			t.Fatalf("NewRuntime() error = %v", err)
		}
		s, err := r.OpenStore("dummy://")
		if err != nil {
			t.Fatalf("Runtime.OpenStore() error = %v", err)
		}
		e, err := r.NewEvaluator(s)
		if err != nil {
			t.Fatalf("Runtime.NewEvaluator() error = %v", err)
		}

		if err := r.Close(); err != nil {
			t.Fatalf("Runtime.Close() error = %v", err)
		}
		if err := e.Close(); err != nil {
			t.Fatalf("Evaluator.Close() after Runtime.Close() error = %v", err)
		}
	})

	t.Run("runtime_settings", func(t *testing.T) {
		r, err := NewRuntime()
		if err != nil {
			t.Fatalf("NewRuntime() error = %v", err)
		}
		fetchSettings := r.flakeFetcherSettings
		flakeSettings := r.flakeSettings

		if err := r.Close(); err != nil {
			t.Fatalf("Runtime.Close() error = %v", err)
		}
		if err := fetchSettings.Close(); err != nil {
			t.Fatalf("fetcher Settings.Close() after Runtime.Close() error = %v", err)
		}
		if err := flakeSettings.Close(); err != nil {
			t.Fatalf("flake Settings.Close() after Runtime.Close() error = %v", err)
		}
	})
}

func TestRuntimeMethodsAfterClose(t *testing.T) {
	r, err := NewRuntime()
	if err != nil {
		t.Fatalf("NewRuntime() error = %v", err)
	}
	if err := r.Close(); err != nil {
		t.Fatalf("Runtime.Close() error = %v", err)
	}

	_, err = r.OpenStore("dummy://")
	requireRuntimeClosedError(t, err)
}

func requireRuntimeClosedError(t *testing.T, err error) {
	t.Helper()

	if err == nil {
		t.Fatal("error = nil, want ErrClosed")
	}
	if !errors.Is(err, ErrClosed) {
		t.Fatalf("error = %v, want errors.Is(..., ErrClosed)", err)
	}
}

func newFlakeTestRuntime(t *testing.T) *Runtime {
	t.Helper()

	r, err := NewRuntime(WithExperimentalFeatures(ExperimentalFeatureNixCommand, ExperimentalFeatureFlakes))
	if err != nil {
		t.Fatalf("NewRuntime() error = %v", err)
	}
	t.Cleanup(func() {
		if err := r.Close(); err != nil {
			t.Fatalf("Runtime.Close() error = %v", err)
		}
	})

	return r
}

func newFlakeTestClient(t *testing.T, r *Runtime) *Client {
	t.Helper()

	resolvedRoot := realTempDir(t)
	t.Cleanup(func() {
		chmodTreeWritable(t, resolvedRoot)
	})

	s, err := r.OpenStore("local",
		store.WithStoreDir(filepath.Join(resolvedRoot, "store")),
		store.WithStateDir(filepath.Join(resolvedRoot, "state")),
		store.WithLogDir(filepath.Join(resolvedRoot, "log")),
	)
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}

	c, err := NewClient(r, WithClientStore(s))
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}
	t.Cleanup(func() {
		if err := c.Close(); err != nil {
			t.Fatalf("Client.Close() error = %v", err)
		}
	})

	return c
}

func realTempDir(t *testing.T) string {
	t.Helper()

	root := t.TempDir()
	resolvedRoot, err := filepath.EvalSymlinks(root)
	if err != nil {
		t.Fatalf("EvalSymlinks(%q): %v", root, err)
	}

	return resolvedRoot
}

func chmodTreeWritable(t *testing.T, root string) {
	t.Helper()

	if err := filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return os.Chmod(path, 0o700)
		}
		return os.Chmod(path, 0o600)
	}); err != nil && !os.IsNotExist(err) {
		t.Fatalf("chmod temp store tree: %v", err)
	}
}

func writeTestFlake(t *testing.T, dir string, contents string) {
	t.Helper()

	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("MkdirAll(%q): %v", dir, err)
	}
	if err := os.WriteFile(filepath.Join(dir, "flake.nix"), []byte(contents), 0o644); err != nil {
		t.Fatalf("WriteFile(%q/flake.nix): %v", dir, err)
	}
}

func lockedFlakeHello(t *testing.T, e *eval.Evaluator, locked *flake.LockedFlake) string {
	t.Helper()

	outputs, err := locked.OutputAttrs()
	if err != nil {
		t.Fatalf("LockedFlake.OutputAttrs() error = %v", err)
	}
	hello, err := e.Attr(outputs, "hello")
	if err != nil {
		t.Fatalf("Evaluator.Attr(hello) error = %v", err)
	}
	got, err := hello.String()
	if err != nil {
		t.Fatalf("Value.String() error = %v", err)
	}

	return got
}

func TestRuntimeParseFlakeRef(t *testing.T) {
	r := newFlakeTestRuntime(t)
	c := newFlakeTestClient(t, r)

	root := realTempDir(t)
	writeTestFlake(t, root, `
{
  outputs = { ... }: {
    hello = "BOB";
  };
}
`)

	if _, err := c.ParseFlakeRef(".#hello"); err == nil {
		t.Fatal("Client.ParseFlakeRef(relative without base) error = nil, want error")
	}

	ref, err := c.ParseFlakeRef(".#legacyPackages.aarch127-unknown...orion", flake.WithBaseDirectory(root))
	if err != nil {
		t.Fatalf("Client.ParseFlakeRef() error = %v", err)
	}
	if got := ref.Fragment(); got != "legacyPackages.aarch127-unknown...orion" {
		t.Fatalf("Ref.Fragment() = %q, want legacyPackages.aarch127-unknown...orion", got)
	}
	ptr, err := ref.Borrow()
	if err != nil {
		t.Fatalf("Ref.Borrow() error = %v", err)
	}
	if ptr == nil {
		t.Fatal("Ref.Borrow() = nil, want non-nil")
	}

	if err := ref.Close(); err != nil {
		t.Fatalf("Ref.Close() error = %v", err)
	}
	if err := ref.Close(); err != nil {
		t.Fatalf("second Ref.Close() error = %v", err)
	}
	_, err = ref.Borrow()
	requireRuntimeClosedError(t, err)
}

func TestRuntimeLockFlakeOutputAttrsAndModes(t *testing.T) {
	r := newFlakeTestRuntime(t)
	c := newFlakeTestClient(t, r)

	root := realTempDir(t)
	writeTestFlake(t, filepath.Join(root, "b"), `
{
  outputs = { ... }: {
    hello = "BOB";
  };
}
`)
	writeTestFlake(t, filepath.Join(root, "a"), `
{
  inputs.b.url = "`+filepath.ToSlash(filepath.Join(root, "b"))+`";
  outputs = { b, ... }: {
    hello = b.hello;
  };
}
`)
	writeTestFlake(t, filepath.Join(root, "c"), `
{
  outputs = { ... }: {
    hello = "Claire";
  };
}
`)

	ref, err := c.ParseFlakeRef("./a", flake.WithBaseDirectory(root))
	if err != nil {
		t.Fatalf("Client.ParseFlakeRef(./a) error = %v", err)
	}
	if got := ref.Fragment(); got != "" {
		t.Fatalf("Ref.Fragment() = %q, want empty", got)
	}

	locked, err := c.LockFlake(ref)
	if err != nil {
		t.Fatalf("Client.LockFlake(default) error = %v", err)
	}
	if got := lockedFlakeHello(t, c.evaluator, locked); got != "BOB" {
		t.Fatalf("default virtual lock hello = %q, want BOB", got)
	}
	if _, err := os.Stat(filepath.Join(root, "a", "flake.lock")); !os.IsNotExist(err) {
		t.Fatalf("default virtual lock flake.lock stat err = %v, want not exist", err)
	}
	if err := locked.Close(); err != nil {
		t.Fatalf("LockedFlake.Close() error = %v", err)
	}
	if err := locked.Close(); err != nil {
		t.Fatalf("second LockedFlake.Close() error = %v", err)
	}
	if _, err := locked.OutputAttrs(); !errors.Is(err, ErrClosed) {
		t.Fatalf("LockedFlake.OutputAttrs() after close error = %v, want ErrClosed", err)
	}

	if _, err := c.LockFlake(ref, flake.WithLockMode(flake.LockModeCheck)); err == nil {
		t.Fatal("Client.LockFlake(check before lock exists) error = nil, want error")
	}

	locked, err = c.LockFlake(ref, flake.WithLockMode(flake.LockModeWriteAsNeeded))
	if err != nil {
		t.Fatalf("Client.LockFlake(write as needed) error = %v", err)
	}
	if got := lockedFlakeHello(t, c.evaluator, locked); got != "BOB" {
		t.Fatalf("write-as-needed lock hello = %q, want BOB", got)
	}
	if _, err := os.Stat(filepath.Join(root, "a", "flake.lock")); err != nil {
		t.Fatalf("written flake.lock stat: %v", err)
	}

	locked, err = c.LockFlake(ref, flake.WithLockMode(flake.LockModeCheck))
	if err != nil {
		t.Fatalf("Client.LockFlake(check after write) error = %v", err)
	}
	if got := lockedFlakeHello(t, c.evaluator, locked); got != "BOB" {
		t.Fatalf("check lock hello = %q, want BOB", got)
	}

	overrideRef, err := c.ParseFlakeRef("./c", flake.WithBaseDirectory(root))
	if err != nil {
		t.Fatalf("Client.ParseFlakeRef(./c) error = %v", err)
	}
	locked, err = c.LockFlake(ref,
		flake.WithLockMode(flake.LockModeWriteAsNeeded),
		flake.WithInputOverride("b", overrideRef),
	)
	if err != nil {
		t.Fatalf("Client.LockFlake(with override) error = %v", err)
	}
	if got := lockedFlakeHello(t, c.evaluator, locked); got != "Claire" {
		t.Fatalf("override lock hello = %q, want Claire", got)
	}
}

func TestRuntimeFlakeErrorsAfterClose(t *testing.T) {
	r := newFlakeTestRuntime(t)
	c := newFlakeTestClient(t, r)

	root := realTempDir(t)
	writeTestFlake(t, root, `
{
  outputs = { ... }: {
    hello = "BOB";
  };
}
`)

	ref, err := c.ParseFlakeRef(".", flake.WithBaseDirectory(root))
	if err != nil {
		t.Fatalf("Client.ParseFlakeRef() error = %v", err)
	}
	if err := ref.Close(); err != nil {
		t.Fatalf("Ref.Close() error = %v", err)
	}
	if _, err := c.LockFlake(ref); !errors.Is(err, ErrClosed) {
		t.Fatalf("Client.LockFlake(closed ref) error = %v, want ErrClosed", err)
	}

	ref, err = c.ParseFlakeRef(".", flake.WithBaseDirectory(root))
	if err != nil {
		t.Fatalf("Client.ParseFlakeRef() error = %v", err)
	}
	if _, err := c.LockFlake(ref, flake.WithLockMode(flake.LockMode(99))); err == nil {
		t.Fatal("Client.LockFlake(invalid mode) error = nil, want error")
	}

	if err := c.Close(); err != nil {
		t.Fatalf("Client.Close() error = %v", err)
	}
	_, err = c.ParseFlakeRef(".")
	requireRuntimeClosedError(t, err)
	_, err = c.LockFlake(ref)
	requireRuntimeClosedError(t, err)
}

func TestClientFetchPackage(t *testing.T) {
	r := newFlakeTestRuntime(t)
	c := newFlakeTestClient(t, r)

	root := realTempDir(t)
	writeTestFlake(t, root, `
{
  outputs = { ... }: {
    legacyPackages.x86_64-linux.demo = {
      type = "derivation";
      name = "demo-1.0";
      pname = "demo";
      version = "1.0";
      system = "x86_64-linux";
      builder = "/bin/sh";
      args = [ "-c" "exit 0" ];
      drvPath = "/nix/store/00000000000000000000000000000000-demo.drv";
      outPath = "/nix/store/00000000000000000000000000000000-demo";
      outputName = "out";
      outputs = [ "out" "dev" ];
      out = "/nix/store/00000000000000000000000000000000-demo";
      dev = "/nix/store/00000000000000000000000000000000-demo-dev";
      src = {
        type = "url";
        url = "https://example.invalid/demo.tar.gz";
        sha256 = "sha256-demo";
      };
      meta = {
        description = "demo package";
        homepage = "https://example.invalid/demo";
        license = {
          shortName = "mit";
          fullName = "MIT License";
          spdxId = "MIT";
          free = true;
          redistributable = true;
        };
        maintainers = [
          {
            name = "Ada";
            email = "ada@example.invalid";
          }
        ];
        platforms = [ "x86_64-linux" "aarch64-darwin" ];
        badPlatforms = [ "i686-linux" ];
        sourceProvenance = [
          {
            shortName = "fromSource";
            isSource = true;
          }
        ];
      };
    };
  };
}
`)

	ref, err := c.ParseFlakeRef(".", flake.WithBaseDirectory(root))
	if err != nil {
		t.Fatalf("Client.ParseFlakeRef() error = %v", err)
	}
	locked, err := c.LockFlake(ref)
	if err != nil {
		t.Fatalf("Client.LockFlake() error = %v", err)
	}

	pkg, err := c.FetchPackage(locked, "demo", WithFetchPackageSystem(MakeSystem(OSLinux, ArchX86_64)))
	if err != nil {
		t.Fatalf("Client.FetchPackage() error = %v", err)
	}
	if pkg.Name != "demo-1.0" || pkg.PName != "demo" || pkg.Version != "1.0" {
		t.Fatalf("package identity = %#v", pkg)
	}
	if pkg.Outputs["dev"].OutPath != "/nix/store/00000000000000000000000000000000-demo-dev" {
		t.Fatalf("dev output = %#v", pkg.Outputs["dev"])
	}
	if pkg.Meta.License[0].SpdxID != "MIT" || pkg.Meta.Maintainers[0].Name != "Ada" {
		t.Fatalf("meta = %#v", pkg.Meta)
	}
	if pkg.Meta.Platforms[0].Arch != ArchX86_64 || pkg.Meta.Platforms[0].OS != OSLinux {
		t.Fatalf("platforms = %#v", pkg.Meta.Platforms)
	}
	if pkg.Src.URL != "https://example.invalid/demo.tar.gz" || pkg.Src.Sha256 != "sha256-demo" {
		t.Fatalf("src = %#v", pkg.Src)
	}
}
