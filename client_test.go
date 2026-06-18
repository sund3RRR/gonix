package gonix

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/sund3RRR/gonix/eval"
	flakeapi "github.com/sund3RRR/gonix/flake"
)

func TestNewClientZeroConfig(t *testing.T) {
	client, err := NewClient(ClientConfig{})
	if err != nil {
		t.Fatalf("NewClient(ClientConfig{}) error = %v", err)
	}

	if err := client.Close(); err != nil {
		t.Fatalf("Client.Close() error = %v", err)
	}
	if err := client.Close(); err != nil {
		t.Fatalf("second Client.Close() error = %v", err)
	}

	_, err = client.NewFlake("path:/does-not-matter")
	if !errors.Is(err, ErrClosed) {
		t.Fatalf("Client.NewFlake() after Close error = %v, want ErrClosed", err)
	}
}

func TestNewClientInvalidSetting(t *testing.T) {
	client, err := NewClient(ClientConfig{
		RawSettings: map[string]string{
			"gonix-test-setting-that-does-not-exist": "x",
		},
	})
	if err == nil {
		if client != nil {
			_ = client.Close()
		}
		t.Fatal("NewClient(invalid setting) error = nil")
	}
	if client != nil {
		t.Fatalf("NewClient(invalid setting) client = %v, want nil", client)
	}
}

func TestClientFlakeWorkflowAndLifecycle(t *testing.T) {
	ref := writePackageFlake(t)

	t.Run("manual flake close", func(t *testing.T) {
		client, err := NewClient(ClientConfig{})
		if err != nil {
			t.Fatalf("NewClient() error = %v", err)
		}
		t.Cleanup(func() { _ = client.Close() })

		f, err := client.NewFlake(ref)
		if err != nil {
			t.Fatalf("Client.NewFlake() error = %v", err)
		}

		pkg, err := f.FetchPackage("demo", WithFetchPackageSystem(DefaultSystem()))
		if err != nil {
			t.Fatalf("Flake.FetchPackage() error = %v", err)
		}
		if pkg.PName != "demo" || pkg.Version != "1.0" || pkg.System != DefaultSystem() {
			t.Fatalf("package = %+v", pkg)
		}
		if got := pkg.Outputs["out"].OutputName; got != "out" {
			t.Fatalf("package output name = %q, want out", got)
		}

		if err := f.Close(); err != nil {
			t.Fatalf("Flake.Close() error = %v", err)
		}
		if _, err := f.FetchPackage("demo"); !errors.Is(err, ErrClosed) {
			t.Fatalf("Flake.FetchPackage() after Close error = %v, want ErrClosed", err)
		}
		if err := client.Close(); err != nil {
			t.Fatalf("Client.Close() after Flake.Close() error = %v", err)
		}
	})

	t.Run("caller closes multiple flakes", func(t *testing.T) {
		client, err := NewClient(ClientConfig{})
		if err != nil {
			t.Fatalf("NewClient() error = %v", err)
		}
		first, err := client.NewFlake(ref)
		if err != nil {
			_ = client.Close()
			t.Fatalf("Client.NewFlake() error = %v", err)
		}
		second, err := client.NewFlake(ref)
		if err != nil {
			_ = first.Close()
			_ = client.Close()
			t.Fatalf("Client.NewFlake(second) error = %v", err)
		}

		if err := second.Close(); err != nil {
			t.Fatalf("second Flake.Close() error = %v", err)
		}
		if err := first.Close(); err != nil {
			t.Fatalf("first Flake.Close() error = %v", err)
		}
		if err := client.Close(); err != nil {
			t.Fatalf("Client.Close() error = %v", err)
		}
	})
}

func TestFlakeOutput(t *testing.T) {
	client, err := NewClient(ClientConfig{})
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}
	t.Cleanup(func() { _ = client.Close() })

	f, err := client.NewFlake(writePackageFlake(t))
	if err != nil {
		t.Fatalf("Client.NewFlake() error = %v", err)
	}
	t.Cleanup(func() { _ = f.Close() })

	t.Run("empty path", func(t *testing.T) {
		var root struct {
			Demo struct {
				Scalar int `nix:"scalar" validate:"required"`
			} `nix:"demo" validate:"required"`
		}
		if err := f.Output(nil, &root); err != nil {
			t.Fatalf("Flake.Output(empty) error = %v", err)
		}
		if root.Demo.Scalar != 42 {
			t.Fatalf("root.Demo.Scalar = %d, want 42", root.Demo.Scalar)
		}
	})

	t.Run("struct", func(t *testing.T) {
		var record struct {
			Name    string `nix:"name" validate:"required"`
			Enabled bool   `nix:"enabled" validate:"required"`
		}
		if err := f.Output([]string{"demo", "record"}, &record); err != nil {
			t.Fatalf("Flake.Output(struct) error = %v", err)
		}
		if record.Name != "gonix" || !record.Enabled {
			t.Fatalf("record = %#v, want gonix and enabled", record)
		}
	})

	t.Run("map", func(t *testing.T) {
		var numbers map[string]int
		if err := f.Output([]string{"demo", "numbers"}, &numbers); err != nil {
			t.Fatalf("Flake.Output(map) error = %v", err)
		}
		if numbers["one"] != 1 || numbers["two"] != 2 {
			t.Fatalf("numbers = %#v, want one=1 and two=2", numbers)
		}
	})

	t.Run("slice", func(t *testing.T) {
		var items []string
		if err := f.Output([]string{"demo", "items"}, &items); err != nil {
			t.Fatalf("Flake.Output(slice) error = %v", err)
		}
		if len(items) != 2 || items[0] != "a" || items[1] != "b" {
			t.Fatalf("items = %#v, want [a b]", items)
		}
	})

	t.Run("scalar", func(t *testing.T) {
		var scalar int
		if err := f.Output([]string{"demo", "scalar"}, &scalar); err != nil {
			t.Fatalf("Flake.Output(scalar) error = %v", err)
		}
		if scalar != 42 {
			t.Fatalf("scalar = %d, want 42", scalar)
		}
	})

	t.Run("dotted attribute", func(t *testing.T) {
		var value string
		if err := f.Output([]string{"demo", "dotted.name"}, &value); err != nil {
			t.Fatalf("Flake.Output(dotted attribute) error = %v", err)
		}
		if value != "exact" {
			t.Fatalf("dotted attribute = %q, want exact", value)
		}
	})

	t.Run("missing attribute", func(t *testing.T) {
		var value string
		err := f.Output([]string{"demo", "missing"}, &value)
		if err == nil {
			t.Fatal("Flake.Output(missing attribute) error = nil")
		}
		var nixErr *Error
		if !errors.As(err, &nixErr) {
			t.Fatalf("Flake.Output(missing attribute) error = %v, want Nix Error", err)
		}
	})

	t.Run("non-attribute traversal", func(t *testing.T) {
		var value string
		err := f.Output([]string{"demo", "scalar", "child"}, &value)
		if err == nil {
			t.Fatal("Flake.Output(non-attribute traversal) error = nil")
		}
		var typeErr *eval.ValueTypeError
		if !errors.As(err, &typeErr) {
			t.Fatalf("Flake.Output(non-attribute traversal) error = %v, want ValueTypeError", err)
		}
	})

	t.Run("invalid destinations", func(t *testing.T) {
		err := f.Output([]string{"demo", "scalar"}, nil)
		var invalid *eval.InvalidUnmarshalError
		if !errors.As(err, &invalid) {
			t.Fatalf("Flake.Output(nil) error = %v, want InvalidUnmarshalError", err)
		}

		err = f.Output([]string{"demo", "scalar"}, 0)
		invalid = nil
		if !errors.As(err, &invalid) {
			t.Fatalf("Flake.Output(non-pointer) error = %v, want InvalidUnmarshalError", err)
		}

		err = f.Output([]string{"demo", "scalar"}, new(chan int))
		var unsupported *eval.UnsupportedTypeError
		if !errors.As(err, &unsupported) {
			t.Fatalf("Flake.Output(unsupported) error = %v, want UnsupportedTypeError", err)
		}
	})

	t.Run("repeated calls", func(t *testing.T) {
		for i := range 100 {
			var value int
			if err := f.Output([]string{"demo", "scalar"}, &value); err != nil {
				t.Fatalf("Flake.Output() call %d error = %v", i, err)
			}
			if value != 42 {
				t.Fatalf("Flake.Output() call %d = %d, want 42", i, value)
			}
		}
	})

	if err := f.Close(); err != nil {
		t.Fatalf("Flake.Close() error = %v", err)
	}
	var value int
	if err := f.Output([]string{"demo", "scalar"}, &value); !errors.Is(err, ErrClosed) {
		t.Fatalf("Flake.Output() after Close error = %v, want ErrClosed", err)
	}
}

func TestFlakePackageAPIs(t *testing.T) {
	t.Run("legacyPackages only is rejected", func(t *testing.T) {
		client, err := NewClient(ClientConfig{})
		if err != nil {
			t.Fatalf("NewClient() error = %v", err)
		}
		t.Cleanup(func() { _ = client.Close() })

		f, err := client.NewFlake(writeLegacyPackageFlake(t))
		if err != nil {
			t.Fatalf("Client.NewFlake() error = %v", err)
		}
		t.Cleanup(func() { _ = f.Close() })

		if _, err := f.FetchPackage("demo", WithFetchPackageSystem(DefaultSystem())); err == nil {
			t.Fatal("Flake.FetchPackage(legacyPackages only) error = nil")
		}
	})

	t.Run("realization validation and closure", func(t *testing.T) {
		client, err := NewClient(ClientConfig{})
		if err != nil {
			t.Fatalf("NewClient() error = %v", err)
		}
		t.Cleanup(func() { _ = client.Close() })

		f, err := client.NewFlake(writePackageFlake(t))
		if err != nil {
			t.Fatalf("Client.NewFlake() error = %v", err)
		}

		if _, err := f.RealizePackage(Package{Type: PackageTypeApp}); err == nil {
			t.Fatal("Flake.RealizePackage(app) error = nil")
		}
		if _, err := f.RealizePackage(Package{}); err == nil {
			t.Fatal("Flake.RealizePackage(missing drvPath) error = nil")
		}

		if err := f.Close(); err != nil {
			t.Fatalf("Flake.Close() error = %v", err)
		}
		if _, err := f.RealizePackage(Package{}); !errors.Is(err, ErrClosed) {
			t.Fatalf("Flake.RealizePackage() after Close error = %v, want ErrClosed", err)
		}
	})
}

func TestFlakePartialCleanup(t *testing.T) {
	client, err := NewClient(ClientConfig{})
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}
	t.Cleanup(func() { _ = client.Close() })

	parsed, err := flakeapi.NewParsedRef(
		client.ctx,
		client.fetcherSettings,
		client.flakeSettings,
		writePackageFlake(t),
	)
	if err != nil {
		t.Fatalf("flake.NewParsedRef() error = %v", err)
	}

	partial := &Flake{parsedRef: parsed}
	if err := partial.Close(); err != nil {
		t.Fatalf("partial Flake.Close() error = %v", err)
	}
	if _, err := parsed.Borrow(); !errors.Is(err, ErrClosed) {
		t.Fatalf("parsed ref after partial Close error = %v, want ErrClosed", err)
	}
	if err := partial.Close(); err != nil {
		t.Fatalf("second partial Flake.Close() error = %v", err)
	}

	_, err = client.NewFlake(
		writePackageFlake(t),
		WithLockOpts(flakeapi.WithLockMode(flakeapi.LockMode(100))),
	)
	if err == nil {
		t.Fatal("Client.NewFlake(invalid lock mode) error = nil")
	}
}

func writePackageFlake(t *testing.T) string {
	t.Helper()

	dir := t.TempDir()
	resolvedDir, err := filepath.EvalSymlinks(dir)
	if err != nil {
		t.Fatalf("resolve flake directory: %v", err)
	}
	dir = resolvedDir
	contents := fmt.Sprintf(`{
  outputs = { self }: {
    demo = {
      scalar = 42;
      record = {
        name = "gonix";
        enabled = true;
      };
      numbers = {
        one = 1;
        two = 2;
      };
      items = [ "a" "b" ];
      "dotted.name" = "exact";
    };
    packages.%q.demo = rec {
      type = "derivation";
      name = "demo-1.0";
      pname = "demo";
      version = "1.0";
      system = %q;
      drvPath = "/nix/store/00000000000000000000000000000000-demo.drv";
      outPath = "/nix/store/11111111111111111111111111111111-demo";
      outputName = "out";
      outputs = [ "out" ];
      out = {
        inherit type name drvPath;
        outPath = "/nix/store/11111111111111111111111111111111-demo";
        outputName = "out";
      };
    };
  };
}
`, DefaultSystem(), DefaultSystem())
	if err := os.WriteFile(filepath.Join(dir, "flake.nix"), []byte(contents), 0o644); err != nil {
		t.Fatalf("write flake.nix: %v", err)
	}

	return "path:" + filepath.ToSlash(dir)
}

func writeLegacyPackageFlake(t *testing.T) string {
	t.Helper()

	dir := t.TempDir()
	resolvedDir, err := filepath.EvalSymlinks(dir)
	if err != nil {
		t.Fatalf("resolve flake directory: %v", err)
	}
	dir = resolvedDir
	contents := fmt.Sprintf(`{
  outputs = { self }: {
    legacyPackages.%q.demo = {
      type = "derivation";
      name = "demo";
      drvPath = "/nix/store/00000000000000000000000000000000-demo.drv";
      outPath = "/nix/store/11111111111111111111111111111111-demo";
      outputName = "out";
      outputs = [ "out" ];
    };
  };
}
`, DefaultSystem())
	if err := os.WriteFile(filepath.Join(dir, "flake.nix"), []byte(contents), 0o644); err != nil {
		t.Fatalf("write flake.nix: %v", err)
	}

	return "path:" + filepath.ToSlash(dir)
}
