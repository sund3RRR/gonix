package gonix

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
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
		wantMaintainers := []Maintainer{
			{
				Name:   "Gonix Maintainer",
				Email:  "gonix@example.com",
				GitHub: "gonix",
				GitLab: "gonix-gl",
				Matrix: "@gonix:example.com",
				Keys: []MaintainerKey{{
					Fingerprint: "0123456789ABCDEF",
					LongKeyID:   "89ABCDEF",
				}},
			},
			{Name: "plain-maintainer", Keys: []MaintainerKey{}},
		}
		if !reflect.DeepEqual(pkg.Meta.Maintainers, wantMaintainers) {
			t.Fatalf("package maintainers = %#v, want %#v", pkg.Meta.Maintainers, wantMaintainers)
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

	t.Run("maintainers are best effort", func(t *testing.T) {
		client, err := NewClient(ClientConfig{})
		if err != nil {
			t.Fatalf("NewClient() error = %v", err)
		}
		t.Cleanup(func() { _ = client.Close() })

		f, err := client.NewFlake(writeMaintainerFlake(t))
		if err != nil {
			t.Fatalf("Client.NewFlake() error = %v", err)
		}
		t.Cleanup(func() { _ = f.Close() })

		wantMaintainers := []Maintainer{{
			Name:   "Valid Maintainer",
			Email:  "valid@example.com",
			GitHub: "valid",
			GitLab: "valid-gl",
			Matrix: "@valid:example.com",
			Keys: []MaintainerKey{{
				Fingerprint: "FEDCBA9876543210",
				LongKeyID:   "76543210",
			}},
		}}
		valid, err := f.FetchPackage("valid", WithFetchPackageSystem(DefaultSystem()))
		if err != nil {
			t.Fatalf("Flake.FetchPackage(valid) error = %v", err)
		}
		if !reflect.DeepEqual(valid.Meta.Maintainers, wantMaintainers) {
			t.Fatalf("valid maintainers = %#v, want %#v", valid.Meta.Maintainers, wantMaintainers)
		}

		for _, name := range []string{"missing", "missingAttr", "undefined", "thrown", "malformed"} {
			t.Run(name, func(t *testing.T) {
				pkg, err := f.FetchPackage(name, WithFetchPackageSystem(DefaultSystem()))
				if err != nil {
					t.Fatalf("Flake.FetchPackage(%s) error = %v", name, err)
				}
				if pkg.Meta.Maintainers == nil || len(pkg.Meta.Maintainers) != 0 {
					t.Fatalf("Flake.FetchPackage(%s) maintainers = %#v, want non-nil empty slice", name, pkg.Meta.Maintainers)
				}

				again, err := f.FetchPackage("valid", WithFetchPackageSystem(DefaultSystem()))
				if err != nil {
					t.Fatalf("Flake.FetchPackage(valid) after %s error = %v", name, err)
				}
				if !reflect.DeepEqual(again.Meta.Maintainers, wantMaintainers) {
					t.Fatalf("valid maintainers after %s = %#v, want %#v", name, again.Meta.Maintainers, wantMaintainers)
				}
			})
		}

		if _, err := f.FetchPackage("coreBroken", WithFetchPackageSystem(DefaultSystem())); err == nil {
			t.Fatal("Flake.FetchPackage(coreBroken) error = nil")
		}
	})
}

func TestFlakeListPackages(t *testing.T) {
	t.Run("sorted names without forcing values", func(t *testing.T) {
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

		want := []PackageRef{
			{Name: "Z", System: DefaultSystem()},
			{Name: "a-b", System: DefaultSystem()},
			{Name: "demo", System: DefaultSystem()},
			{Name: "dotted.name", System: DefaultSystem()},
			{Name: "nested", System: DefaultSystem()},
		}

		for i := range 25 {
			got, err := f.ListPackages()
			if err != nil {
				t.Fatalf("Flake.ListPackages() call %d error = %v", i, err)
			}
			if !reflect.DeepEqual(got, want) {
				t.Fatalf("Flake.ListPackages() call %d = %#v, want %#v", i, got, want)
			}
		}
	})

	t.Run("missing packages", func(t *testing.T) {
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

		got, err := f.ListPackages()
		if err != nil {
			t.Fatalf("Flake.ListPackages() error = %v", err)
		}
		if got == nil || len(got) != 0 {
			t.Fatalf("Flake.ListPackages() = %#v, want non-nil empty result", got)
		}
	})

	t.Run("missing system", func(t *testing.T) {
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

		got, err := f.ListPackages(WithListPackagesSystem("missing-system"))
		if err != nil {
			t.Fatalf("Flake.ListPackages(missing system) error = %v", err)
		}
		if got == nil || len(got) != 0 {
			t.Fatalf("Flake.ListPackages(missing system) = %#v, want non-nil empty result", got)
		}
	})

	t.Run("invalid containers", func(t *testing.T) {
		tests := []struct {
			name     string
			contents string
		}{
			{
				name:     "packages",
				contents: `{ outputs = { self }: { packages = 1; }; }`,
			},
			{
				name: "system",
				contents: fmt.Sprintf(
					`{ outputs = { self }: { packages.%q = 1; }; }`,
					DefaultSystem(),
				),
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				client, err := NewClient(ClientConfig{})
				if err != nil {
					t.Fatalf("NewClient() error = %v", err)
				}
				t.Cleanup(func() { _ = client.Close() })

				f, err := client.NewFlake(writeFlake(t, tt.contents))
				if err != nil {
					t.Fatalf("Client.NewFlake() error = %v", err)
				}
				t.Cleanup(func() { _ = f.Close() })

				_, err = f.ListPackages()
				var typeErr *eval.ValueTypeError
				if !errors.As(err, &typeErr) {
					t.Fatalf("Flake.ListPackages() error = %v, want ValueTypeError", err)
				}
			})
		}
	})

	t.Run("system resolution", func(t *testing.T) {
		tests := []struct {
			system string
			opts   []ListPackagesOption
		}{
			{
				system: "explicit-system",
				opts:   []ListPackagesOption{WithListPackagesSystem("explicit-system")},
			},
			{
				system: DefaultSystem(),
			},
		}

		for _, tt := range tests {
			t.Run(tt.system, func(t *testing.T) {
				client, err := NewClient(ClientConfig{})
				if err != nil {
					t.Fatalf("NewClient() error = %v", err)
				}
				t.Cleanup(func() { _ = client.Close() })

				f, err := client.NewFlake(writePackageNamesFlake(t, tt.system))
				if err != nil {
					t.Fatalf("Client.NewFlake() error = %v", err)
				}
				t.Cleanup(func() { _ = f.Close() })

				got, err := f.ListPackages(tt.opts...)
				if err != nil {
					t.Fatalf("Flake.ListPackages() error = %v", err)
				}
				want := []PackageRef{{Name: "demo", System: tt.system}}
				if !reflect.DeepEqual(got, want) {
					t.Fatalf("Flake.ListPackages() = %#v, want %#v", got, want)
				}
			})
		}
	})

	t.Run("closed resources", func(t *testing.T) {
		client, err := NewClient(ClientConfig{})
		if err != nil {
			t.Fatalf("NewClient() error = %v", err)
		}
		t.Cleanup(func() { _ = client.Close() })

		f, err := client.NewFlake(writePackageFlake(t))
		if err != nil {
			t.Fatalf("Client.NewFlake() error = %v", err)
		}

		if err := f.Close(); err != nil {
			t.Fatalf("Flake.Close() error = %v", err)
		}
		if _, err := f.ListPackages(); !errors.Is(err, ErrClosed) {
			t.Fatalf("Flake.ListPackages() after Close error = %v, want ErrClosed", err)
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
    packages.%q = {
      demo = rec {
        type = "derivation";
        name = "demo-1.0";
        pname = "demo";
        version = "1.0";
        system = %q;
        drvPath = "/nix/store/00000000000000000000000000000000-demo.drv";
        outPath = "/nix/store/11111111111111111111111111111111-demo";
        outputName = "out";
        outputs = [ "out" ];
        meta.maintainers = [
          {
            name = "Gonix Maintainer";
            email = "gonix@example.com";
            github = "gonix";
            gitlab = "gonix-gl";
            matrix = "@gonix:example.com";
            keys = [{
              fingerprint = "0123456789ABCDEF";
              longkeyid = "89ABCDEF";
            }];
          }
          "plain-maintainer"
        ];
        out = {
          inherit type name drvPath;
          outPath = "/nix/store/11111111111111111111111111111111-demo";
          outputName = "out";
        };
      };
      Z = throw "Z package must stay lazy";
      "a-b" = throw "a-b package must stay lazy";
      "dotted.name" = throw "dotted package must stay lazy";
      nested = { child = throw "nested package must stay lazy"; };
    };
    legacyPackages.%q.legacy-only = throw "legacyPackages must not be inspected";
  };
}
`, DefaultSystem(), DefaultSystem(), DefaultSystem())
	if err := os.WriteFile(filepath.Join(dir, "flake.nix"), []byte(contents), 0o644); err != nil {
		t.Fatalf("write flake.nix: %v", err)
	}

	return "path:" + filepath.ToSlash(dir)
}

func writeMaintainerFlake(t *testing.T) string {
	t.Helper()

	return writeFlake(t, fmt.Sprintf(`{
  outputs = { self }:
    let
      package = packageName: meta: rec {
        type = "derivation";
        name = packageName;
        pname = packageName;
        version = "1.0";
        system = %q;
        drvPath = "/nix/store/00000000000000000000000000000000-${packageName}.drv";
        outPath = "/nix/store/11111111111111111111111111111111-${packageName}";
        outputName = "out";
        outputs = [ "out" ];
        inherit meta;
        out = {
          inherit type name drvPath outPath outputName;
        };
      };
    in {
      packages.%q = {
        valid = package "valid" {
          maintainers = [{
            name = "Valid Maintainer";
            email = "valid@example.com";
            github = "valid";
            gitlab = "valid-gl";
            matrix = "@valid:example.com";
            keys = [{
              fingerprint = "FEDCBA9876543210";
              longkeyid = "76543210";
            }];
          }];
        };
        missing = package "missing" {};
        missingAttr = package "missing-attr" {
          maintainers = [ {}.missing ];
        };
        undefined = package "undefined" {
          maintainers = with {}; [ undefinedMaintainer ];
        };
        thrown = package "thrown" {
          maintainers = throw "broken maintainer list";
        };
        malformed = package "malformed" {
          maintainers = [{
            keys = [{
              fingerprint = throw "malformed maintainer key";
            }];
          }];
        };
        coreBroken = package (throw "broken core package") {};
      };
    };
}
`, DefaultSystem(), DefaultSystem()))
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

func writePackageNamesFlake(t *testing.T, system string) string {
	t.Helper()

	return writeFlake(t, fmt.Sprintf(`{
  outputs = { self }: {
    packages.%q.demo = throw "package must stay lazy";
  };
}
`, system))
}

func writeFlake(t *testing.T, contents string) string {
	t.Helper()

	dir := t.TempDir()
	resolvedDir, err := filepath.EvalSymlinks(dir)
	if err != nil {
		t.Fatalf("resolve flake directory: %v", err)
	}
	if err := os.WriteFile(filepath.Join(resolvedDir, "flake.nix"), []byte(contents), 0o644); err != nil {
		t.Fatalf("write flake.nix: %v", err)
	}

	return "path:" + filepath.ToSlash(resolvedDir)
}
