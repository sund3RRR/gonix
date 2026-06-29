# gonix

Experimental high-level Go SDK for working with [Nix](https://github.com/NixOS/nix).

`gonix` is an ergonomic Go layer built on top of its bundled
[`pkg/raw`](pkg/raw) package. That package contains low-level generated bindings
to the Nix C API and narrow binding-owned adapters for Nix functionality that
has no public C equivalent.

The goal of this package is to expose Nix concepts through Go-native APIs:
contexts, stores, store paths, evaluation states, values, derivations, flakes,
locked flakes, realised outputs, closures, settings, and structured errors.

## Quick start

```go
ctx := context.Background()

client, err := gonix.NewClient(gonix.ClientConfig{})
if err != nil {
	return err
}
defer client.Close()

f, err := client.OpenFlake("github:NixOS/nixpkgs/nixos-unstable")
if err != nil {
	return err
}
defer f.Close()

var pkg struct {
	Name    string `nix:"name" validate:"required"`
	DrvPath string `nix:"drvPath" validate:"required"`
}
err = client.EvalFlakeOutput(
	ctx,
	f,
	[]string{"legacyPackages", gonix.DefaultSystem(), "hello"},
	&pkg,
)
outputs, err := client.Realize(ctx, pkg.DrvPath)
```

## Core features and concepts

`gonix` is currently an early SDK foundation. `Client` provides a flake-first
quick start, while `nixcontext` and the public subpackages support lower-level
composition.

- Flake-first `Client` API with automatic resource orchestration.
- Composable contexts, stores, evaluators, settings, and values for advanced
  workflows.
- Explicit ownership with idempotent `Close` methods and documented borrowed,
  cloned, and refcounted resources.
- Nix settings, verbosity, log formatting, structured errors, and cooperative
  cancellation.
- Store paths, derivations, realization, closures, copying, GC roots, and
  garbage collection.
- Expression evaluation, value traversal and construction, function calls, and
  unmarshalling into Go values.
- Flake reference parsing, locking, input overrides, output traversal, lock
  metadata, and fingerprints.
- Go-native APIs over the bundled low-level `pkg/raw` package.
- Package discovery, indexing, metadata normalization, and policy remain the
  responsibility of software built on gonix.

## Evaluation and unmarshalling

`Client.Eval` runs expressions through the real Nix evaluator and decodes the
result directly into Go data:

```go
var result struct {
	Message string         `nix:"message" validate:"required"`
	Ports   []int          `nix:"ports" validate:"required"`
	Flags   map[string]bool `nix:"flags" validate:"required"`
}

err := client.Eval(ctx, `{
  message = "hello from Nix";
  ports = [ 80 443 ];
  flags = { tls = true; };
}`, &result)
```

Evaluation supports ordinary Nix language semantics, including functions,
imports, laziness, builtins, conditionals, attribute sets, and lists, subject
to the configured evaluator and enabled Nix features.

High-level operations accept a `context.Context`. Cancellation requests Nix
interruption, waits for the native operation to stop, and closes the Client
because interrupted remote stores cannot be reused safely. A Client permits
only one operation at a time and reports `ErrConcurrentUse` for overlaps.

The unmarshaller currently supports:

- strings and paths into Go strings;
- integers, floats, and booleans;
- `null` into nil pointers;
- lists into slices and fixed-length arrays;
- attribute sets into structs and maps with string keys;
- nested combinations of supported values;
- `nix` field names and `validate:"required"` fields.

It does not currently decode functions, derivations as dedicated Go types,
string contexts, external values, interfaces or union types, arbitrary map key
types, or custom decoder implementations. Advanced callers can use the
lower-level `eval` package to work with caller-owned Nix values directly.

## Flake outputs and realization

`Client.EvalFlakeOutput` traverses the output attributes of a locked flake and
decodes the selected value into Go data:

```go
var packageName string
err := client.EvalFlakeOutput(
	ctx,
	f,
	[]string{"legacyPackages", gonix.DefaultSystem(), "hello", "name"},
	&packageName,
)
```

Each path element is one exact attribute name, so an attribute containing a dot
is addressed as a single slice element. An empty path decodes the complete
flake output attribute set.

Advanced workflows can keep the selected output as a caller-owned
`*eval.Value`:

```go
value, err := client.GetFlakeOutputValue(
	ctx,
	f,
	[]string{"legacyPackages", gonix.DefaultSystem(), "hello"},
)
defer value.Close()

var pkg struct {
	Name    string `nix:"name" validate:"required"`
	DrvPath string `nix:"drvPath" validate:"required"`
}
err = client.Unmarshal(ctx, value, &pkg)
```

Every attribute lookup returns an independently referenced Nix value.
`GetFlakeOutputValue` closes intermediate values during traversal and transfers
ownership of only the final value to the caller. The value must be closed before
its Client.

`Client.Realize(ctx, drvPath)` builds or substitutes every output of a
derivation path and
returns Go-owned `RealizedOutput` values. Gonix intentionally leaves package
discovery, package metadata normalization, indexing, and policy to higher-level
package managers.

## Flake lock metadata

Each opened Flake caches Nix's normalized resolved lock graph and optional
fingerprint during construction:

```go
lock, err := f.LockInfo()
fingerprint := f.Fingerprint()
```

The cached graph can be supplied later as a reference lock while opening the
same root flake:

```go
f, err := client.OpenFlakeFromLock(ref, lock)
```

`LockInfo` types the lock graph, nodes, non-flake flags, and override parent
paths. `LockInput.GetNode` and `LockInput.GetFollows` safely distinguish Nix's
direct-node and follows-path JSON variants. Fetcher-specific `original` and
`locked` reference attributes remain `json.RawMessage` values so new Nix input
schemes do not require gonix API changes. `LockNode.Flake` normalizes Nix's
omitted-true JSON convention into an ordinary Go bool. The returned graph shares
no mutable data with the cache and may be modified by the caller.

The fingerprint is Nix's lowercase hexadecimal locked-flake cache key. It is
empty when Nix cannot fingerprint the source or considers the graph unlocked.
The fragment, lock information, and fingerprint remain available after the
Flake or Client is closed.

## Development

The root flake provides Go, cgo, c-for-go, pkg-config, golangci-lint, and the Nix
libraries used by both SDK layers.

```sh
nix develop
make generate
make test
make lint
make check
```

`make generate` regenerates the low-level Go package in `pkg/raw`.

## License

This repository contains separately licensed parts:

- Gonix code outside `pkg/raw` is licensed under Apache-2.0; see
  [`LICENSE`](LICENSE).
- Everything under `pkg/raw` is licensed under LGPL-2.1-or-later; see
  [`pkg/raw/LICENSE`](pkg/raw/LICENSE).
- The linked Nix libraries and other dependencies retain their own license
  terms.
