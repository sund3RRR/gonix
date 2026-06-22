package gonix

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"
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
	if _, err := client.Realize(context.Background(), "/nix/store/00000000000000000000000000000000-demo.drv"); !errors.Is(err, ErrClosed) {
		t.Fatalf("Client.Realize() after Close error = %v, want ErrClosed", err)
	}
	if err := client.Unmarshal(context.Background(), nil, new(int)); !errors.Is(err, ErrClosed) {
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
	if err := client.EvalFlakeOutput(context.Background(), f, []string{"demo", "scalar"}, &scalar); err != nil {
		t.Fatalf("Client.EvalFlakeOutput() error = %v", err)
	}
	if scalar != 42 {
		t.Fatalf("Client.EvalFlakeOutput() = %d, want 42", scalar)
	}

	if err := f.Close(); err != nil {
		t.Fatalf("Flake.Close() error = %v", err)
	}
	if err := f.Close(); err != nil {
		t.Fatalf("second Flake.Close() error = %v", err)
	}
	if _, err := client.GetFlakeOutputValue(context.Background(), f, []string{"demo"}); !errors.Is(err, ErrClosed) {
		t.Fatalf("Client.GetFlakeOutputValue() after Close error = %v, want ErrClosed", err)
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

func TestClientCloseClosesTrackedFlakes(t *testing.T) {
	ref, _ := writeOutputFlake(t)
	client, err := NewClient(ClientConfig{})
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}

	f, err := client.OpenFlake(ref)
	if err != nil {
		_ = client.Close()
		t.Fatalf("Client.OpenFlake() error = %v", err)
	}
	fingerprint := f.Fingerprint()

	if err := client.Close(); err != nil {
		t.Fatalf("Client.Close() error = %v", err)
	}
	if _, err := f.Borrow(); !errors.Is(err, ErrClosed) {
		t.Fatalf("tracked Flake.Borrow() after Client.Close error = %v, want ErrClosed", err)
	}
	if got := f.Fingerprint(); got != fingerprint {
		t.Fatalf("tracked Flake.Fingerprint() after Client.Close = %q, want %q", got, fingerprint)
	}
}

func TestFlakeConstructorMetadataFailure(t *testing.T) {
	ref, _ := writeOutputFlake(t)
	client := newTestClient(t)

	if err := client.store.Close(); err != nil {
		t.Fatalf("Store.Close() error = %v", err)
	}

	f, err := flake.New(
		client.ctx,
		client.store,
		client.fetcherSettings,
		client.flakeSettings,
		client.evaluator,
		ref,
	)
	if !errors.Is(err, ErrClosed) {
		t.Fatalf("flake.New(closed store) error = %v, want ErrClosed", err)
	}
	if f != nil {
		t.Fatalf("flake.New(closed store) = %v, want nil", f)
	}
}

func TestFlakeCachedLockMetadata(t *testing.T) {
	ref, dir := writeLockMetadataFlake(t)
	client := newTestClient(t)

	writer, err := client.OpenFlake(ref, flake.WithLockMode(flake.LockModeWriteAsNeeded))
	if err != nil {
		t.Fatalf("Client.OpenFlake(write lock) error = %v", err)
	}
	if err := writer.Close(); err != nil {
		t.Fatalf("writer Flake.Close() error = %v", err)
	}

	f, err := client.OpenFlake(ref, flake.WithLockMode(flake.LockModeCheck))
	if err != nil {
		t.Fatalf("Client.OpenFlake(check lock) error = %v", err)
	}

	writtenLockJSON, err := os.ReadFile(filepath.Join(dir, "flake.lock"))
	if err != nil {
		t.Fatalf("read written flake.lock: %v", err)
	}
	var writtenLock flake.LockInfo
	if err := json.Unmarshal(writtenLockJSON, &writtenLock); err != nil {
		t.Fatalf("decode written lock JSON: %v", err)
	}
	got, err := f.LockInfo()
	if err != nil {
		t.Fatalf("Flake.LockInfo() error = %v", err)
	}
	if !reflect.DeepEqual(got, writtenLock) {
		t.Fatalf("cached LockInfo = %#v, want decoded written lock %#v", got, writtenLock)
	}

	info, err := f.LockInfo()
	if err != nil {
		t.Fatalf("Flake.LockInfo() error = %v", err)
	}
	if info.Version == 0 || info.Root == "" || len(info.Nodes) == 0 {
		t.Fatalf("Flake.LockInfo() = %#v, want populated lock graph", info)
	}

	root := info.Nodes[info.Root]
	if got := lockFollowsPath(t, root.Inputs["alias"]); len(got) != 1 || got[0] != "dep" {
		t.Fatalf("root alias input = %#v, want follows [dep]", got)
	}
	dataInput := lockNodeName(t, root.Inputs["data"])
	dataNode, ok := info.Nodes[dataInput]
	if !ok {
		t.Fatalf("data input target %q is absent from lock nodes", dataInput)
	}
	if dataNode.Flake {
		t.Fatal("data node Flake = true, want explicit false")
	}
	depInput := lockNodeName(t, root.Inputs["dep"])
	if !info.Nodes[depInput].Flake {
		t.Fatal("dep node Flake = false, want omitted field to default to true")
	}
	if _, ok := dataNode.Locked["narHash"]; !ok {
		t.Fatalf("data locked reference = %#v, want narHash", dataNode.Locked)
	}

	info.Version = 0
	delete(info.Nodes, dataInput)
	delete(root.Inputs, "alias")
	if narHash := dataNode.Locked["narHash"]; len(narHash) != 0 {
		narHash[0] ^= 0xff
	}

	freshInfo, err := f.LockInfo()
	if err != nil {
		t.Fatalf("Flake.LockInfo() after mutation error = %v", err)
	}
	if !reflect.DeepEqual(freshInfo, writtenLock) {
		t.Fatalf("LockInfo() cache was mutated: got %#v, want %#v", freshInfo, writtenLock)
	}

	fingerprint := f.Fingerprint()
	assertFingerprint(t, fingerprint)

	equivalent, err := client.OpenFlake(ref, flake.WithLockMode(flake.LockModeCheck))
	if err != nil {
		t.Fatalf("Client.OpenFlake(equivalent) error = %v", err)
	}
	if got := equivalent.Fingerprint(); got != fingerprint {
		t.Fatalf("equivalent fingerprint = %q, want %q", got, fingerprint)
	}
	if err := equivalent.Close(); err != nil {
		t.Fatalf("equivalent Flake.Close() error = %v", err)
	}

	wantInfo, err := f.LockInfo()
	if err != nil {
		t.Fatalf("Flake.LockInfo() before closure error = %v", err)
	}
	if err := f.Close(); err != nil {
		t.Fatalf("Flake.Close() error = %v", err)
	}
	if err := client.Close(); err != nil {
		t.Fatalf("Client.Close() error = %v", err)
	}

	if got := f.Fingerprint(); got != fingerprint {
		t.Fatalf("Fingerprint() after closure = %q, want %q", got, fingerprint)
	}
	got, err = f.LockInfo()
	if err != nil {
		t.Fatalf("LockInfo() after closure error = %v", err)
	}
	if !reflect.DeepEqual(got, wantInfo) {
		t.Fatalf("LockInfo() after closure = %#v, want %#v", got, wantInfo)
	}
}

func TestFlakeFingerprintChangesWithInputOverride(t *testing.T) {
	originalRef, _ := writeValueFlake(t, "original")
	overrideRef, _ := writeValueFlake(t, "override")
	mainRef, _ := writeFlake(t, fmt.Sprintf(`{
  inputs.dep.url = %q;
  outputs = { self, dep }: {
    value = dep.value;
  };
}
`, originalRef))
	client := newTestClient(t)

	original := openTestFlake(t, client, mainRef)
	overridden, err := client.OpenFlake(
		mainRef,
		flake.WithInputOverride("dep", overrideRef),
	)
	if err != nil {
		t.Fatalf("Client.OpenFlake(input override) error = %v", err)
	}
	defer overridden.Close() //nolint:errcheck

	assertFingerprint(t, original.Fingerprint())
	assertFingerprint(t, overridden.Fingerprint())
	if original.Fingerprint() == overridden.Fingerprint() {
		t.Fatalf("override fingerprint = original fingerprint %q", original.Fingerprint())
	}
	originalInfo, err := original.LockInfo()
	if err != nil {
		t.Fatalf("original Flake.LockInfo() error = %v", err)
	}
	overrideInfo, err := overridden.LockInfo()
	if err != nil {
		t.Fatalf("overridden Flake.LockInfo() error = %v", err)
	}
	if reflect.DeepEqual(originalInfo, overrideInfo) {
		t.Fatal("override LockInfo equals original LockInfo")
	}
	originalDepInput := lockNodeName(t, originalInfo.Nodes[originalInfo.Root].Inputs["dep"])
	overrideDepInput := lockNodeName(t, overrideInfo.Nodes[overrideInfo.Root].Inputs["dep"])
	originalDep := originalInfo.Nodes[originalDepInput]
	overrideDep := overrideInfo.Nodes[overrideDepInput]
	if reflect.DeepEqual(originalDep.Locked, overrideDep.Locked) {
		t.Fatalf("override locked reference = original locked reference %#v", originalDep.Locked)
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
		if err := client.EvalFlakeOutput(context.Background(), f, nil, &root); err != nil {
			t.Fatalf("Client.EvalFlakeOutput(empty) error = %v", err)
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
		if err := client.EvalFlakeOutput(context.Background(), f, []string{"demo", "record"}, &record); err != nil {
			t.Fatalf("Client.EvalFlakeOutput(struct) error = %v", err)
		}
		if record.Name != "gonix" || !record.Enabled {
			t.Fatalf("record = %#v, want gonix and enabled", record)
		}
	})

	t.Run("map", func(t *testing.T) {
		var numbers map[string]int
		if err := client.EvalFlakeOutput(context.Background(), f, []string{"demo", "numbers"}, &numbers); err != nil {
			t.Fatalf("Client.EvalFlakeOutput(map) error = %v", err)
		}
		if numbers["one"] != 1 || numbers["two"] != 2 {
			t.Fatalf("numbers = %#v, want one=1 and two=2", numbers)
		}
	})

	t.Run("slice", func(t *testing.T) {
		var items []string
		if err := client.EvalFlakeOutput(context.Background(), f, []string{"demo", "items"}, &items); err != nil {
			t.Fatalf("Client.EvalFlakeOutput(slice) error = %v", err)
		}
		if len(items) != 2 || items[0] != "a" || items[1] != "b" {
			t.Fatalf("items = %#v, want [a b]", items)
		}
	})

	t.Run("dotted attribute", func(t *testing.T) {
		var value string
		if err := client.EvalFlakeOutput(context.Background(), f, []string{"demo", "dotted.name"}, &value); err != nil {
			t.Fatalf("Client.EvalFlakeOutput(dotted attribute) error = %v", err)
		}
		if value != "exact" {
			t.Fatalf("dotted attribute = %q, want exact", value)
		}
	})

	t.Run("missing attribute", func(t *testing.T) {
		var value string
		err := client.EvalFlakeOutput(context.Background(), f, []string{"demo", "missing"}, &value)
		var nixErr *Error
		if !errors.As(err, &nixErr) {
			t.Fatalf("Flake.Output(missing attribute) error = %v, want Nix Error", err)
		}
	})

	t.Run("non-attribute final parent", func(t *testing.T) {
		var value string
		err := client.EvalFlakeOutput(context.Background(), f, []string{"demo", "scalar", "child"}, &value)
		var typeErr *eval.ValueTypeError
		if !errors.As(err, &typeErr) {
			t.Fatalf("Flake.Output(non-attribute traversal) error = %v, want ValueTypeError", err)
		}
	})

	t.Run("invalid destinations", func(t *testing.T) {
		err := client.EvalFlakeOutput(context.Background(), f, []string{"demo", "scalar"}, nil)
		var invalid *eval.InvalidUnmarshalError
		if !errors.As(err, &invalid) {
			t.Fatalf("Flake.Output(nil) error = %v, want InvalidUnmarshalError", err)
		}

		err = client.EvalFlakeOutput(context.Background(), f, []string{"demo", "scalar"}, new(chan int))
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
		value, err := client.GetFlakeOutputValue(context.Background(), f, []string{"demo", "nested", "value"})
		if err != nil {
			t.Fatalf("Flake.OutputValue() error = %v", err)
		}
		defer value.Close() //nolint:errcheck

		var got string
		if err := client.Unmarshal(context.Background(), value, &got); err != nil {
			t.Fatalf("Client.Unmarshal() error = %v", err)
		}
		if got != "alive" {
			t.Fatalf("Client.Unmarshal() = %q, want alive", got)
		}
	})

	t.Run("empty path returns live root", func(t *testing.T) {
		value, err := client.GetFlakeOutputValue(context.Background(), f, nil)
		if err != nil {
			t.Fatalf("Flake.OutputValue(empty) error = %v", err)
		}
		defer value.Close() //nolint:errcheck

		var root struct {
			Demo struct {
				Scalar int `nix:"scalar" validate:"required"`
			} `nix:"demo" validate:"required"`
		}
		if err := client.Unmarshal(context.Background(), value, &root); err != nil {
			t.Fatalf("Client.Unmarshal(root) error = %v", err)
		}
		if root.Demo.Scalar != 42 {
			t.Fatalf("root.Demo.Scalar = %d, want 42", root.Demo.Scalar)
		}
	})

	t.Run("repeated traversal", func(t *testing.T) {
		for i := range 100 {
			value, err := client.GetFlakeOutputValue(context.Background(), f, []string{"demo", "scalar"})
			if err != nil {
				t.Fatalf("Flake.OutputValue() call %d error = %v", i, err)
			}

			var got int
			unmarshalErr := client.Unmarshal(context.Background(), value, &got)
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

	value, err := first.GetFlakeOutputValue(context.Background(), f, []string{"demo", "scalar"})
	if err != nil {
		t.Fatalf("Flake.OutputValue() error = %v", err)
	}
	defer value.Close() //nolint:errcheck

	var out int
	if err := second.Unmarshal(context.Background(), value, &out); err == nil {
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
		if err := client.EvalFlakeOutput(context.Background(), f, []string{"demo", "scalar"}, &got); err != nil {
			t.Fatalf("Client.EvalFlakeOutput() error = %v", err)
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
		if err := client.EvalFlakeOutput(context.Background(), f, []string{"value"}, &got); err != nil {
			t.Fatalf("Client.EvalFlakeOutput(overridden input) error = %v", err)
		}
		if got != "override" {
			t.Fatalf("Flake.Output(overridden input) = %q, want override", got)
		}
	})
}

func TestClientRealizeValidation(t *testing.T) {
	client := newTestClient(t)

	if _, err := client.Realize(context.Background(), "not-a-store-path"); err == nil {
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
	if err := client.Eval(context.Background(), expr, &drvPath); err != nil {
		t.Fatalf("Client.Eval(derivation) error = %v", err)
	}

	outputs, err := client.Realize(context.Background(), drvPath)
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

func writeLockMetadataFlake(t *testing.T) (ref string, dir string) {
	t.Helper()

	dependencyRef, _ := writeValueFlake(t, "dependency")
	dataDir := t.TempDir()
	resolvedDataDir, err := filepath.EvalSymlinks(dataDir)
	if err != nil {
		t.Fatalf("resolve data input directory: %v", err)
	}
	if err := os.WriteFile(filepath.Join(resolvedDataDir, "value.txt"), []byte("data"), 0o644); err != nil {
		t.Fatalf("write non-flake input: %v", err)
	}

	return writeFlake(t, fmt.Sprintf(`{
  inputs.dep.url = %q;
  inputs.alias.follows = "dep";
  inputs.data = {
    url = "path:%s";
    flake = false;
  };
  outputs = { self, dep, ... }: {
    value = dep.value;
  };
}
`, dependencyRef, filepath.ToSlash(resolvedDataDir)))
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

func assertFingerprint(t *testing.T, fingerprint string) {
	t.Helper()

	if len(fingerprint) != 64 {
		t.Fatalf("fingerprint length = %d, want 64: %q", len(fingerprint), fingerprint)
	}
	for _, r := range fingerprint {
		if !strings.ContainsRune("0123456789abcdef", r) {
			t.Fatalf("fingerprint = %q, want lowercase base16", fingerprint)
		}
	}
}

func lockNodeName(t *testing.T, input flake.LockInput) string {
	t.Helper()

	node, ok := input.GetNode()
	if !ok {
		t.Fatalf("lock input = %#v, want direct node string", input)
	}
	return node
}

func lockFollowsPath(t *testing.T, input flake.LockInput) []string {
	t.Helper()

	path, ok := input.GetFollows()
	if !ok {
		t.Fatalf("lock input = %#v, want follows path array", input)
	}
	return path
}
