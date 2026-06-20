# gonix

Experimental high-level Go SDK for working with [Nix](https://github.com/NixOS/nix).

`gonix` is an ergonomic Go layer built on top of
[`nix-go-bindings`](https://github.com/sund3RRR/nix-go-bindings), which provides
low-level generated bindings to the Nix C API and narrow binding-owned adapters
for Nix functionality that has no public C equivalent.

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

f, err := client.OpenFlake("github:NixOS/nixpkgs/nixos-unstable")
if err != nil {
	return err
}
defer f.Close()

var pkg struct {
	Name    string `nix:"name" validate:"required"`
	DrvPath string `nix:"drvPath" validate:"required"`
}
err = f.Output(
	[]string{"legacyPackages", gonix.DefaultSystem(), "hello"},
	&pkg,
)
outputs, err := client.Realize(pkg.DrvPath)
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
  - [x] Cached lock graph metadata and fingerprints.
  - [x] Generic typed locked-output access.
- [ ] Package-manager workflows.
  - [x] Generic flake output traversal.
  - [x] Store-backed derivation realization.
  - [ ] Package discovery, indexing, and metadata policy belong in a package
        manager built on gonix.

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

## Flake outputs and realization

`Flake.Output` traverses the output attributes of the locked flake and decodes
the selected value into Go data:

```go
var packageName string
err := f.Output(
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
value, err := f.OutputValue(
	[]string{"legacyPackages", gonix.DefaultSystem(), "hello"},
)
defer value.Close()

var pkg struct {
	Name    string `nix:"name" validate:"required"`
	DrvPath string `nix:"drvPath" validate:"required"`
}
err = client.Unmarshal(value, &pkg)
```

Every attribute lookup returns an independently referenced Nix value.
`OutputValue` closes intermediate values during traversal and transfers
ownership of only the final value to the caller. The value must be closed before
its Client.

`Client.Realize` builds or substitutes every output of a derivation path and
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

## License

Apache-2.0.

Note that `gonix` depends on [`nix-go-bindings`](https://github.com/sund3RRR/nix-go-bindings) and the Nix C libraries, which
have their own license terms.
