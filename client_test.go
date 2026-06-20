package gonix

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/sund3RRR/gonix/eval"
	"github.com/sund3RRR/gonix/flake"
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

	if _, err := client.OpenFlake("path:/does-not-matter"); !errors.Is(err, ErrClosed) {
		t.Fatalf("Client.OpenFlake() after Close error = %v, want ErrClosed", err)
	}
	if _, err := client.Realize("/nix/store/00000000000000000000000000000000-demo.drv"); !errors.Is(err, ErrClosed) {
		t.Fatalf("Client.Realize() after Close error = %v, want ErrClosed", err)
	}
	if err := client.Unmarshal(nil, new(int)); !errors.Is(err, ErrClosed) {
		t.Fatalf("Client.Unmarshal() after Close error = %v, want ErrClosed", err)
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
	ref, _ := writeOutputFlake(t)
	client := newTestClient(t)

	f, err := client.OpenFlake(ref + "#demo")
	if err != nil {
		t.Fatalf("Client.OpenFlake() error = %v", err)
	}

	if got := f.Fragment(); got != "demo" {
		t.Fatalf("Flake.Fragment() = %q, want demo", got)
	}

	var scalar int
	if err := f.Output([]string{"demo", "scalar"}, &scalar); err != nil {
		t.Fatalf("Flake.Output() error = %v", err)
	}
	if scalar != 42 {
		t.Fatalf("Flake.Output() = %d, want 42", scalar)
	}

	if err := f.Close(); err != nil {
		t.Fatalf("Flake.Close() error = %v", err)
	}
	if err := f.Close(); err != nil {
		t.Fatalf("second Flake.Close() error = %v", err)
	}
	if _, err := f.OutputValue([]string{"demo"}); !errors.Is(err, ErrClosed) {
		t.Fatalf("Flake.OutputValue() after Close error = %v, want ErrClosed", err)
	}
}

func TestClientOpensMultipleFlakes(t *testing.T) {
	ref, _ := writeOutputFlake(t)
	client := newTestClient(t)

	first, err := client.OpenFlake(ref)
	if err != nil {
		t.Fatalf("Client.OpenFlake(first) error = %v", err)
	}
	second, err := client.OpenFlake(ref)
	if err != nil {
		_ = first.Close()
		t.Fatalf("Client.OpenFlake(second) error = %v", err)
	}

	if err := second.Close(); err != nil {
		t.Fatalf("second Flake.Close() error = %v", err)
	}
	if err := first.Close(); err != nil {
		t.Fatalf("first Flake.Close() error = %v", err)
	}
}

func TestFlakeOutput(t *testing.T) {
	ref, _ := writeOutputFlake(t)
	client := newTestClient(t)
	f := openTestFlake(t, client, ref)

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
		var nixErr *Error
		if !errors.As(err, &nixErr) {
			t.Fatalf("Flake.Output(missing attribute) error = %v, want Nix Error", err)
		}
	})

	t.Run("non-attribute final parent", func(t *testing.T) {
		var value string
		err := f.Output([]string{"demo", "scalar", "child"}, &value)
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

		err = f.Output([]string{"demo", "scalar"}, new(chan int))
		var unsupported *eval.UnsupportedTypeError
		if !errors.As(err, &unsupported) {
			t.Fatalf("Flake.Output(unsupported) error = %v, want UnsupportedTypeError", err)
		}
	})
}

func TestFlakeOutputValueOwnsFinalReference(t *testing.T) {
	ref, _ := writeOutputFlake(t)
	client := newTestClient(t)
	f := openTestFlake(t, client, ref)

	t.Run("nested value survives intermediate closes", func(t *testing.T) {
		value, err := f.OutputValue([]string{"demo", "nested", "value"})
		if err != nil {
			t.Fatalf("Flake.OutputValue() error = %v", err)
		}
		defer value.Close() //nolint:errcheck

		var got string
		if err := client.Unmarshal(value, &got); err != nil {
			t.Fatalf("Client.Unmarshal() error = %v", err)
		}
		if got != "alive" {
			t.Fatalf("Client.Unmarshal() = %q, want alive", got)
		}
	})

	t.Run("empty path returns live root", func(t *testing.T) {
		value, err := f.OutputValue(nil)
		if err != nil {
			t.Fatalf("Flake.OutputValue(empty) error = %v", err)
		}
		defer value.Close() //nolint:errcheck

		var root struct {
			Demo struct {
				Scalar int `nix:"scalar" validate:"required"`
			} `nix:"demo" validate:"required"`
		}
		if err := client.Unmarshal(value, &root); err != nil {
			t.Fatalf("Client.Unmarshal(root) error = %v", err)
		}
		if root.Demo.Scalar != 42 {
			t.Fatalf("root.Demo.Scalar = %d, want 42", root.Demo.Scalar)
		}
	})

	t.Run("repeated traversal", func(t *testing.T) {
		for i := range 100 {
			value, err := f.OutputValue([]string{"demo", "scalar"})
			if err != nil {
				t.Fatalf("Flake.OutputValue() call %d error = %v", i, err)
			}

			var got int
			unmarshalErr := client.Unmarshal(value, &got)
			closeErr := value.Close()
			if unmarshalErr != nil {
				t.Fatalf("Client.Unmarshal() call %d error = %v", i, unmarshalErr)
			}
			if closeErr != nil {
				t.Fatalf("Value.Close() call %d error = %v", i, closeErr)
			}
			if got != 42 {
				t.Fatalf("Client.Unmarshal() call %d = %d, want 42", i, got)
			}
		}
	})
}

func TestClientUnmarshalRejectsForeignValue(t *testing.T) {
	ref, _ := writeOutputFlake(t)
	first := newTestClient(t)
	second := newTestClient(t)
	f := openTestFlake(t, first, ref)

	value, err := f.OutputValue([]string{"demo", "scalar"})
	if err != nil {
		t.Fatalf("Flake.OutputValue() error = %v", err)
	}
	defer value.Close() //nolint:errcheck

	var out int
	if err := second.Unmarshal(value, &out); err == nil {
		t.Fatal("Client.Unmarshal(foreign value) error = nil")
	}
}

func TestFlakeOptions(t *testing.T) {
	ref, dir := writeOutputFlake(t)
	client := newTestClient(t)

	t.Run("base directory", func(t *testing.T) {
		f, err := client.OpenFlake(".", flake.WithBaseDirectory(dir))
		if err != nil {
			t.Fatalf("Client.OpenFlake(relative) error = %v", err)
		}
		defer f.Close() //nolint:errcheck

		var got int
		if err := f.Output([]string{"demo", "scalar"}, &got); err != nil {
			t.Fatalf("Flake.Output() error = %v", err)
		}
		if got != 42 {
			t.Fatalf("Flake.Output() = %d, want 42", got)
		}
	})

	t.Run("invalid lock mode", func(t *testing.T) {
		if _, err := client.OpenFlake(ref, flake.WithLockMode(flake.LockMode(100))); err == nil {
			t.Fatal("Client.OpenFlake(invalid lock mode) error = nil")
		}
	})

	t.Run("input override", func(t *testing.T) {
		originalRef, _ := writeValueFlake(t, "original")
		overrideRef, _ := writeValueFlake(t, "override")
		mainRef, _ := writeFlake(t, fmt.Sprintf(`{
  inputs.dep.url = %q;
  outputs = { self, dep }: {
    value = dep.value;
  };
}
`, originalRef))

		f, err := client.OpenFlake(
			mainRef,
			flake.WithInputOverride("dep", overrideRef),
		)
		if err != nil {
			t.Fatalf("Client.OpenFlake(input override) error = %v", err)
		}
		defer f.Close() //nolint:errcheck

		var got string
		if err := f.Output([]string{"value"}, &got); err != nil {
			t.Fatalf("Flake.Output(overridden input) error = %v", err)
		}
		if got != "override" {
			t.Fatalf("Flake.Output(overridden input) = %q, want override", got)
		}
	})
}

func TestClientRealizeValidation(t *testing.T) {
	client := newTestClient(t)

	if _, err := client.Realize("not-a-store-path"); err == nil {
		t.Fatal("Client.Realize(invalid path) error = nil")
	}
}

func TestClientEvalAndRealizeFlow(t *testing.T) {
	client := newTestClient(t)

	var drvPath string
	expr := fmt.Sprintf(`(derivation {
  name = "gonix-client-flow";
  system = %q;
  builder = "/bin/sh";
  args = [ "-c" "printf gonix > \"$out\"" ];
}).drvPath`, DefaultSystem())
	if err := client.Eval(expr, &drvPath); err != nil {
		t.Fatalf("Client.Eval(derivation) error = %v", err)
	}

	outputs, err := client.Realize(drvPath)
	if err != nil {
		t.Fatalf("Client.Realize() error = %v", err)
	}
	if len(outputs) != 1 {
		t.Fatalf("Client.Realize() returned %d outputs, want 1", len(outputs))
	}
	if outputs[0].OutputName != "out" {
		t.Fatalf("output name = %q, want out", outputs[0].OutputName)
	}

	contents, err := os.ReadFile(outputs[0].RealPath)
	if err != nil {
		t.Fatalf("read realized output: %v", err)
	}
	if string(contents) != "gonix" {
		t.Fatalf("realized output = %q, want gonix", contents)
	}
}

func newTestClient(t *testing.T) *Client {
	t.Helper()

	client, err := NewClient(ClientConfig{})
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}
	t.Cleanup(func() {
		if err := client.Close(); err != nil {
			t.Errorf("Client.Close() error = %v", err)
		}
	})

	return client
}

func openTestFlake(t *testing.T, client *Client, ref string) *flake.Flake {
	t.Helper()

	f, err := client.OpenFlake(ref)
	if err != nil {
		t.Fatalf("Client.OpenFlake() error = %v", err)
	}
	t.Cleanup(func() {
		if err := f.Close(); err != nil {
			t.Errorf("Flake.Close() error = %v", err)
		}
	})

	return f
}

func writeOutputFlake(t *testing.T) (ref string, dir string) {
	t.Helper()

	return writeFlake(t, `{
  outputs = { self }: {
    demo = {
      scalar = 42;
      nested.value = "alive";
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
  };
}
`)
}

func writeValueFlake(t *testing.T, value string) (ref string, dir string) {
	t.Helper()

	return writeFlake(t, fmt.Sprintf(`{
  outputs = { self }: {
    value = %q;
  };
}
`, value))
}

func writeFlake(t *testing.T, contents string) (ref string, dir string) {
	t.Helper()

	dir = t.TempDir()
	resolvedDir, err := filepath.EvalSymlinks(dir)
	if err != nil {
		t.Fatalf("resolve flake directory: %v", err)
	}
	dir = resolvedDir

	if err := os.WriteFile(filepath.Join(dir, "flake.nix"), []byte(contents), 0o644); err != nil {
		t.Fatalf("write flake.nix: %v", err)
	}

	return fmt.Sprintf("path:%s", filepath.ToSlash(dir)), dir
}
