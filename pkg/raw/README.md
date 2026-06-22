# gonix raw bindings

`github.com/sund3RRR/gonix/pkg/raw` contains the low-level Go bindings for the
[Nix](https://github.com/NixOS/nix) C API.

The Nix project is implemented in C++, and its public C API packages are thin C
facades over those C++ libraries. This package turns that C API into Go via
[c-for-go](https://github.com/xlab/c-for-go). Small C shims translate awkward C
API shapes into signatures that c-for-go can generate cleanly. Narrow C++
store and flake shims also expose selected upstream operations that are not in
the public Nix C API.

This is the low-level layer of the gonix repository, not the idiomatic SDK.
Most callers should use the high-level gonix packages instead.

## Development environment

From the gonix repository root:

```sh
nix develop
```

The root flake sets up Go, cgo, `pkg-config`, c-for-go, and the Nix libraries
needed to build, test, and regenerate this package.

## Repository layout

- `/flake.nix` provides the development shell, Nix C API libraries, the Nix
  flake C++ library, pkg-config paths, Go, c-for-go, and the binding generation
  app.
- `pkg/raw/nix-go-bindings.yml` is the c-for-go configuration.
- `nix_go_*.h`, `nix_go_*.c`, and `nix_go_*_cpp.cc` are the local shim layer.
- `raw.go`, `types.go`, `const.go`, `cgo_helpers.*`, and `doc.go` are generated.

## Regenerating bindings

```sh
make generate
```

The root flake runs c-for-go against `pkg/raw/nix-go-bindings.yml`, writes into a
temporary directory, and copies the generated `raw` package files back into
`pkg/raw`.

Run the complete repository checks from the root:

```sh
make test
make lint
make check
```

## Usage Notes

Current bindings are intentionally close to the C layer. Strings returned by the
shim are C-owned `*byte` values and must be released with `StringFree`.
Store handles should be released with `StoreFree`.

`LogSinkInstall` adds a process-global logger that copies Nix events to either
an append-only file or an existing Unix socket as newline-delimited
`internal-json` records without the `@nix ` prefix. The sink remains installed
for the lifetime of the process, and additional calls add additional sinks.
Initialize Nix, call `SetLogFormat` if needed, install sinks, and then run Nix
operations. Calling `SetLogFormat` after installation replaces the
process-global logger and discards all installed sinks.

`InterruptRequest`, `InterruptClear`, and `InterruptRequested` expose Nix's
process-global logical interrupt state without installing signal handlers or
raising `SIGINT`. Requesting interruption also wakes Nix's registered interrupt
callbacks, including subsystems such as the curl file-transfer worker.
`StoreInterrupt` additionally shuts down active `RemoteStore` connections so
blocking daemon protocol I/O wakes up.

Only one cancellable Nix operation may be active in the process. Call
`StoreInterrupt` with a separate error context from the context used by the
running operation. Wait for the active native call to return before calling
`InterruptClear`, closing its resources, or starting another operation. A
remotely interrupted store is unusable afterward and must be closed and
replaced; a worker-based integration may recycle the entire worker process.

Interruption is cooperative: builds, downloads, daemon operations, garbage
collection, and store traversal generally check the flag promptly, while pure
evaluation may respond later because evaluator checks are sparse. If the native
call does not finish within the higher layer's grace period, terminating its
worker process remains the hard-cancellation fallback.

GC root discovery and collection return opaque `StoreRoots` and
`StoreGCResults` handles. Release them with their matching free functions;
strings and cloned store paths returned by their accessors remain separately
owned. `StoreGCOptions.IgnoreLiveness` preserves upstream's dangerous behavior,
and `MaxFreed` should be `^uint64(0)` when no limit is wanted.

### Known limitations

- Go-facing callback APIs are intentionally not generated. This excludes custom
  primop callbacks, external value callback descriptors, arbitrary GC
  finalizers, and raw store callbacks. Store realisation and closure traversal
  are exposed through callback-free shim result handles instead.
- Some generated helper methods call `C.free` on opaque Nix objects. Do not use
  those raw `.Free()` helpers for Nix-owned opaque values; use the
  API-specific free functions such as `StoreFree`, `StorePathFree`,
  `DerivationFree`, and the package-specific result free functions.
- Generated array structs contain both `Items` and `Len`. `Len` must match the
  number of Go items supplied. The C shim can reject null pointers paired with a
  non-zero length, but it cannot recover the original Go slice length from C.
- Interruption state is process-global. These bindings intentionally do not
  enforce operation serialization, provide Go context integration, or guard
  caller output values against partial decoding; higher-level consumers must
  provide those policies, cancellation grace periods, and remote-store
  replacement.
- Nix expression GC and value reference counts remain caller-managed. Pair
  values returned by allocation/getter APIs with the upstream refcount
  functions documented by the generated binding names and Nix C API ownership
  rules.
- The generator copies newly generated files into the repository. When removing
  generated symbols, verify the resulting diff so stale generated files that are
  no longer emitted are removed from version control.

## Upstream C API Surface

The upstream C API packages are:

- [`nix-util-c`](https://github.com/NixOS/nix/tree/master/src/libutil-c): common
  initialization, contexts, errors, settings, version, and verbosity.
- [`nix-store-c`](https://github.com/NixOS/nix/tree/master/src/libstore-c):
  stores, store paths, derivations, realization, closure traversal, and copying.
- [`nix-expr-c`](https://github.com/NixOS/nix/tree/master/src/libexpr-c):
  evaluation state, values, primops, external values, and GC hooks.
- [`nix-fetchers-c`](https://github.com/NixOS/nix/tree/master/src/libfetchers-c):
  fetcher settings.
- [`nix-flake-c`](https://github.com/NixOS/nix/tree/master/src/libflake-c):
  flake settings, references, lock flags, locking, and output lookup.
- [`nix-main-c`](https://github.com/NixOS/nix/tree/master/src/libmain-c):
  plugin initialization, log format, and the local process-lifetime JSON log
  sink adapter.

## Core features and concepts

- Generated bindings for the `nix-util-c`, `nix-store-c`, `nix-expr-c`,
  `nix-fetchers-c`, `nix-flake-c`, and `nix-main-c` API areas.
- Nix context, library initialization, settings, version, verbosity, errors,
  and process-global interruption primitives.
- Store and store-path lifecycle, metadata, parsing, validity checks,
  derivations, realization, closures, copying, roots, and garbage collection.
- Evaluation states and values, including forcing, traversal, construction,
  function calls, lists, attribute sets, realized strings, and reference
  counting.
- Fetcher and flake settings, reference parsing, locking, input overrides,
  locked output access, lock JSON, and fingerprints.
- Plugin initialization, log formatting, and process-lifetime JSON log sinks.
- Small C and C++ shims for generator compatibility and narrow callback-free
  adapters where the public Nix C API is insufficient.
- Explicit caller-managed ownership that follows the matching Nix lifecycle
  and reference-counting functions.
- Reproducible generation through c-for-go and the repository root flake.

## License

This repository contains separately licensed parts:

- Everything under `pkg/raw` is licensed under LGPL-2.1-or-later; see
  [`LICENSE`](LICENSE).
- Gonix code outside `pkg/raw` is licensed under Apache-2.0; see the root
  [`LICENSE`](../../LICENSE).
- The linked Nix libraries and other dependencies retain their own license
  terms.
