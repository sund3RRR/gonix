# gonix

Experimental high-level Go SDK for working with [Nix](https://github.com/NixOS/nix).

`gonix` is an ergonomic Go layer built on top of
[`nix-go-bindings`](https://github.com/sund3RRR/nix-go-bindings), which provides
low-level generated bindings to the Nix C API.

The goal of this package is to expose Nix concepts through Go-native APIs:
contexts, stores, store paths, evaluation states, values, derivations, flakes,
locked flakes, realised outputs, closures, settings, and structured errors.

## Quick start

```go
client, err := gonix.NewClient(gonix.ClientConfig{})
if err != nil {
	return err
}
defer client.Close()

f, err := client.NewFlake("github:NixOS/nixpkgs/nixos-unstable")
if err != nil {
	return err
}
defer f.Close()

var name string
err = f.Output([]string{"packages", gonix.DefaultSystem(), "hello", "name"}, &name)

packages, err := f.ListPackages()
pkg, err := f.FetchPackage("hello")
outputs, err := f.RealizePackage(pkg)
```

## Implementation status

`gonix` is currently an early SDK foundation. `Client` provides a flake-first
quick start, while `nixcontext` and the public subpackages support lower-level
composition.

- [x] Context, Client, and settings foundation.
  - [x] Nix context initialization and shutdown.
  - [x] Flake-ready Client configuration.
  - [x] Selected Nix settings, verbosity, and log format.
  - [x] Structured Nix error conversion.
- [x] Store and store path foundation.
  - [x] Store opening and store metadata.
  - [x] Store path parsing, cloning, hash/name access, and lifecycle handling.
  - [x] Filesystem closure traversal.
  - [x] Path and closure copying between stores.
- [x] Initial derivation support.
  - [x] Derivation import/export and cloning.
  - [x] Store-backed derivation helpers.
  - [x] Realised output result conversion.
- [x] Evaluation and values.
  - [x] Evaluation state API.
  - [x] Nix value API.
  - [x] Value forcing, traversal, calls, and conversion helpers.
  - [x] High-level `Client.Eval` evaluation and unmarshalling.
- [ ] Fetchers.
  - [x] Fetcher settings lifecycle.
  - [ ] Fetch tree and input resolution helpers.
  - [ ] Store path conversion for fetched inputs.
- [x] Flakes.
  - [x] Flake reference parsing.
  - [x] Locked flake workflows.
  - [x] Generic typed locked-output access.
- [ ] Nix package convenience API.
  - [x] Fast top-level package listing.
  - [x] Package metadata projection.
  - [ ] Build/install metadata helpers.
  - [x] Store-backed package realization helpers.

## Evaluation and unmarshalling

`Client.Eval` runs expressions through the real Nix evaluator and decodes the
result directly into Go data:

```go
var result struct {
	Message string         `nix:"message" validate:"required"`
	Ports   []int          `nix:"ports" validate:"required"`
	Flags   map[string]bool `nix:"flags" validate:"required"`
}

err := client.Eval(`{
  message = "hello from Nix";
  ports = [ 80 443 ];
  flags = { tls = true; };
}`, &result)
```

Evaluation supports ordinary Nix language semantics, including functions,
imports, laziness, builtins, conditionals, attribute sets, and lists, subject
to the configured evaluator and enabled Nix features.

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

## Flake outputs and packages

`Flake.Output` traverses the output attributes of the locked flake and decodes
the selected value into Go data:

```go
var packageName string
err := f.Output(
	[]string{"packages", gonix.DefaultSystem(), "hello", "name"},
	&packageName,
)
```

Each path element is one exact attribute name, so an attribute containing a dot
is addressed as a single slice element. An empty path decodes the complete
flake output attribute set.

`ListPackages` returns sorted top-level names from `packages.<system>` without
forcing or decoding the individual package values:

```go
packages, err := f.ListPackages()
packages, err = f.ListPackages(gonix.WithListPackagesSystem("aarch64-linux"))
```

The default system is the flake evaluator's `builtins.currentSystem`, cached
when the Flake is constructed. Missing `packages` or system attributes produce
an empty result. Listing is intentionally shallow: nested package sets appear
as one top-level `PackageRef`.

`FetchPackage` is the normalized package convenience API for
`packages.<system>` outputs. It intentionally does not fall back to
`legacyPackages`. `RealizePackage` builds or substitutes the selected package
and returns Go-owned `RealizedPackageOutput` values.

## License

Apache-2.0.

Note that `gonix` depends on [`nix-go-bindings`](https://github.com/sund3RRR/nix-go-bindings) and the Nix C libraries, which
have their own license terms.
