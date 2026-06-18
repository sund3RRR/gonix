package gonix

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"testing"

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
