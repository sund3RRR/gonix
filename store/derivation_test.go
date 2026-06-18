package store

import (
	"bytes"
	"encoding/json"
	"errors"
	"reflect"
	"testing"

	"github.com/sund3RRR/gonix/internal/status"
	"github.com/sund3RRR/gonix/nixcontext"
	nix "github.com/sund3RRR/nix-go-bindings"
)

const testDerivationJSON = `{
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
}`

const testDerivationVariantsJSON = `{
  "name": "variants",
  "version": 4,
  "outputs": {
    "input": {
      "path": "00000000000000000000000000000000-input"
    },
    "fixed": {
      "method": "nar",
      "hash": "sha256-iUUXyRY8iW7DGirb0zwGgf1fRbLA7wimTJKgP7l/OQ8="
    },
    "floating": {
      "method": "nar",
      "hashAlgo": "sha256"
    },
    "deferred": {},
    "impure": {
      "method": "nar",
      "hashAlgo": "sha256",
      "impure": true
    }
  },
  "inputs": {
    "srcs": [
      "00000000000000000000000000000000-source"
    ],
    "drvs": {
      "00000000000000000000000000000000-dependency.drv": {
        "outputs": [
          "out"
        ],
        "dynamicOutputs": {
          "out": {
            "outputs": [
              "nested"
            ],
            "dynamicOutputs": {}
          }
        }
      }
    }
  },
  "system": "x86_64-linux",
  "builder": "/bin/sh",
  "args": [
    "-c",
    "echo test"
  ],
  "env": {
    "name": "variants"
  },
  "structuredAttrs": {
    "enabled": true,
    "nested": {
      "value": 42
    }
  }
}`

func newDerivationTestContext(t *testing.T) *nixcontext.Context {
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

	return ctx
}

func newDerivationTestStore(t *testing.T, ctx *nixcontext.Context) *nix.Store {
	t.Helper()

	rawCtx, err := ctx.Borrow()
	if err != nil {
		t.Fatalf("Context.Borrow() error = %v", err)
	}

	store := nix.StoreOpen(rawCtx, "dummy://", nix.StoreParams{})
	if store == nil {
		t.Fatalf("StoreOpen(dummy://) returned nil: %v", status.FromContext(rawCtx))
	}
	t.Cleanup(func() {
		nix.StoreFree(store)
	})

	return store
}

func newRawTestDerivation(t *testing.T, ctx *nixcontext.Context, store *nix.Store) *nix.NixDerivation {
	t.Helper()

	rawCtx, err := ctx.Borrow()
	if err != nil {
		t.Fatalf("Context.Borrow() error = %v", err)
	}

	ptr := nix.DerivationFromJson(rawCtx, store, testDerivationJSON)
	if ptr == nil {
		t.Fatalf("DerivationFromJson returned nil: %v", status.FromContext(rawCtx))
	}

	return ptr
}

func newTestDerivationWithContext(t *testing.T) (*nixcontext.Context, *Derivation) {
	t.Helper()

	ctx := newDerivationTestContext(t)
	store := newDerivationTestStore(t, ctx)
	d, err := NewDerivation(ctx, newRawTestDerivation(t, ctx, store))
	if err != nil {
		t.Fatalf("NewDerivation() error = %v", err)
	}
	t.Cleanup(func() {
		if err := d.Close(); err != nil {
			t.Fatalf("Derivation.Close() error = %v", err)
		}
	})

	return ctx, d
}

func newTestDerivation(t *testing.T) *Derivation {
	t.Helper()

	_, d := newTestDerivationWithContext(t)
	return d
}

func TestNewDerivation(t *testing.T) {
	t.Run("nil_raw_derivation", func(t *testing.T) {
		got, err := NewDerivation(newDerivationTestContext(t), nil)
		if err == nil {
			t.Fatal("NewDerivation() error = nil, want error")
		}
		if got != nil {
			t.Fatalf("NewDerivation() = %v, want nil", got)
		}
	})

	t.Run("closed_context", func(t *testing.T) {
		ctx := newDerivationTestContext(t)
		store := newDerivationTestStore(t, ctx)
		ptr := newRawTestDerivation(t, ctx, store)

		if err := ctx.Close(); err != nil {
			t.Fatalf("Context.Close() error = %v", err)
		}

		got, err := NewDerivation(ctx, ptr)
		if !errors.Is(err, status.ErrClosed) {
			t.Fatalf("NewDerivation() error = %v, want status.ErrClosed", err)
		}
		if got != nil {
			t.Fatalf("NewDerivation() = %v, want nil", got)
		}
	})
}

func TestDerivation_SerializeJSON(t *testing.T) {
	ctx, d := newTestDerivationWithContext(t)

	first := d.SerializeJSON()
	if !json.Valid(first) {
		t.Fatalf("SerializeJSON() returned invalid JSON: %s", first)
	}
	if bytes.Equal(first, []byte(testDerivationJSON)) {
		t.Fatal("SerializeJSON() returned caller formatting instead of Nix-normalized JSON")
	}

	first[0] = 'x'
	second := d.SerializeJSON()
	if !json.Valid(second) {
		t.Fatalf("SerializeJSON() cache was mutated: %s", second)
	}

	if err := d.Close(); err != nil {
		t.Fatalf("Derivation.Close() error = %v", err)
	}
	if err := ctx.Close(); err != nil {
		t.Fatalf("Context.Close() error = %v", err)
	}
	if got := d.SerializeJSON(); !bytes.Equal(got, second) {
		t.Fatalf("SerializeJSON() after close = %s, want %s", got, second)
	}
	if got, err := d.Deserialize(); err != nil {
		t.Fatalf("Deserialize() after close error = %v", err)
	} else if got.Name != "gonix-test" {
		t.Fatalf("Deserialize() after close name = %q, want gonix-test", got.Name)
	}
}

func TestDerivation_Deserialize(t *testing.T) {
	d := &Derivation{json: []byte(testDerivationVariantsJSON)}

	got, err := d.Deserialize()
	if err != nil {
		t.Fatalf("Derivation.Deserialize() error = %v", err)
	}
	if got.Name != "variants" || got.Version != 4 {
		t.Fatalf("Deserialize() identity = %q version %d", got.Name, got.Version)
	}
	if got.System != "x86_64-linux" || got.Builder != "/bin/sh" {
		t.Fatalf("Deserialize() system/builder = %q/%q", got.System, got.Builder)
	}
	if !reflect.DeepEqual(got.Args, []string{"-c", "echo test"}) {
		t.Fatalf("Deserialize() args = %#v", got.Args)
	}
	if got.Environment["name"] != "variants" {
		t.Fatalf("Deserialize() environment = %#v", got.Environment)
	}

	if got.Outputs["input"].Path != "00000000000000000000000000000000-input" {
		t.Fatalf("input-addressed output = %#v", got.Outputs["input"])
	}
	if fixed := got.Outputs["fixed"]; fixed.Method != "nar" || fixed.Hash == "" {
		t.Fatalf("fixed output = %#v", fixed)
	}
	if floating := got.Outputs["floating"]; floating.Method != "nar" || floating.HashAlgo != "sha256" {
		t.Fatalf("floating output = %#v", floating)
	}
	if deferred := got.Outputs["deferred"]; deferred != (DerivationOutput{}) {
		t.Fatalf("deferred output = %#v, want zero value", deferred)
	}
	if impure := got.Outputs["impure"]; impure.Method != "nar" || impure.HashAlgo != "sha256" || !impure.Impure {
		t.Fatalf("impure output = %#v", impure)
	}

	if !reflect.DeepEqual(got.Inputs.Sources, []string{"00000000000000000000000000000000-source"}) {
		t.Fatalf("input sources = %#v", got.Inputs.Sources)
	}
	input := got.Inputs.Derivations["00000000000000000000000000000000-dependency.drv"]
	if !reflect.DeepEqual(input.Outputs, []string{"out"}) {
		t.Fatalf("input outputs = %#v", input.Outputs)
	}
	if !reflect.DeepEqual(input.DynamicOutputs["out"].Outputs, []string{"nested"}) {
		t.Fatalf("dynamic input outputs = %#v", input.DynamicOutputs)
	}

	var structured map[string]any
	if err := json.Unmarshal(got.StructuredAttrs, &structured); err != nil {
		t.Fatalf("json.Unmarshal(StructuredAttrs) error = %v", err)
	}
	if structured["enabled"] != true {
		t.Fatalf("structuredAttrs = %#v", structured)
	}
}

func TestDerivation_DeserializeMalformedCache(t *testing.T) {
	d := &Derivation{json: []byte(`{"name":`)}
	if _, err := d.Deserialize(); err == nil {
		t.Fatal("Derivation.Deserialize() error = nil, want error")
	}
}

func TestDerivation_Clone(t *testing.T) {
	d := newTestDerivation(t)

	clone, err := d.Clone()
	if err != nil {
		t.Fatalf("Derivation.Clone() error = %v", err)
	}
	t.Cleanup(func() {
		if err := clone.Close(); err != nil {
			t.Fatalf("clone.Close() error = %v", err)
		}
	})

	if !bytes.Equal(clone.SerializeJSON(), d.SerializeJSON()) {
		t.Fatalf("clone JSON = %s, want %s", clone.SerializeJSON(), d.SerializeJSON())
	}

	if err := clone.Close(); err != nil {
		t.Fatalf("clone.Close() error = %v", err)
	}
	if !json.Valid(clone.SerializeJSON()) {
		t.Fatal("clone SerializeJSON() unavailable after clone.Close()")
	}
	if _, err := d.Borrow(); err != nil {
		t.Fatalf("original.Borrow() after clone.Close() error = %v", err)
	}
}

func TestDerivation_RawMethodsRejectClosed(t *testing.T) {
	d := newTestDerivation(t)
	if err := d.Close(); err != nil {
		t.Fatalf("Derivation.Close() error = %v", err)
	}

	if clone, err := d.Clone(); !errors.Is(err, status.ErrClosed) || clone != nil {
		t.Fatalf("Derivation.Clone() = %v, %v; want nil, status.ErrClosed", clone, err)
	}
	if ptr, err := d.Borrow(); !errors.Is(err, status.ErrClosed) || ptr != nil {
		t.Fatalf("Derivation.Borrow() = %v, %v; want nil, status.ErrClosed", ptr, err)
	}
	if _, err := d.Deserialize(); err != nil {
		t.Fatalf("Derivation.Deserialize() after close error = %v", err)
	}
}

func TestDerivation_Close(t *testing.T) {
	d := newTestDerivation(t)

	if err := d.Close(); err != nil {
		t.Fatalf("Derivation.Close() error = %v", err)
	}
	if err := d.Close(); err != nil {
		t.Fatalf("second Derivation.Close() error = %v", err)
	}
	if d.ptr != nil {
		t.Fatalf("d.ptr = %p, want nil", d.ptr)
	}
}
