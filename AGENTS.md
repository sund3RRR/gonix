# AGENTS.md

Guidance for agents working on the `gonix` repository.

## Project Goal

`gonix` is a high-level Go SDK for working with Nix.

This repository is intentionally different from `nix-go-bindings`.
`nix-go-bindings` is the low-level, generated bridge to the Nix C API.
`gonix` is the ergonomic Go layer built on top of that bridge.

Use `gonix` to provide Go-native APIs for:

- Nix runtime initialization and settings.
- Stores and store paths.
- Evaluation states and Nix values.
- Derivations.
- Flakes and locked flakes.
- Realised outputs and closures.
- Error handling and ownership-safe resource lifecycle.

Do not use `gonix` to duplicate generated bindings, call private Nix C++ APIs
directly, or shell out to the `nix` CLI for core SDK behavior.

The core pieces are:

- Nix as the build system and development environment.
- `nix-go-bindings` as the only direct portal to the Nix C API.
- Go wrapper types that own or borrow raw Nix resources deliberately.
- Public APIs that expose Go-native values, errors, and lifecycle methods.
- Tests that prove users can work through `gonix` without importing raw
  bindings directly.

## Architecture

Use Nix for the development shell, tests, and builds. The repository should be
usable through:

- `nix develop`
- `nix develop -c go test ./...`
- `nix build`

Use `nix-go-bindings` as the low-level dependency. Raw generated types from that
package should normally remain internal to `gonix`.

Public API should expose stable Go types such as:

- `Client`
- `Store`
- `Eval`
- `Value`
- `StorePath`
- `Derivation`
- `FlakeRef`
- `LockedFlake`
- `Package`
- `Realization`
- `Closure`

All ownership-sensitive raw pointers must be wrapped before they cross a public
API boundary.

## High-Level API Principles

Prefer Go-native data and control flow:

- Use `string` instead of C strings.
- Use slices and maps instead of generated array structs.
- Use structs and option types instead of long C-style argument lists.
- Use Go `error` values instead of exposing raw `nix_err`.

Convert C-owned strings to Go strings immediately and release the C allocation
with the appropriate low-level free function.

Use explicit ownership:

- Owned wrapper types should implement `Close() error`.
- `Close` must be idempotent.
- Borrowed wrapper types must be clearly documented.
- Cloned wrapper types must free their clone.
- Do not rely on Go finalizers for correctness.

Do not expose generated `.Free()` helpers from `nix-go-bindings`. They may call
raw `C.free` and are not the correct lifecycle operation for many opaque Nix
objects.

Do not leak raw `unsafe.Pointer` or `nix-go-bindings` types through public APIs
unless there is a deliberate, documented escape hatch. Preserve Nix semantics
instead of inventing different behavior.

## Nix-Go-Bindings Boundary

Treat `nix-go-bindings` as the only gateway to Nix.

Agents working on `gonix` should not modify generated bindings inside this
repository. If a missing low-level API is required, add it first to
`nix-go-bindings` through its shim/config/codegen workflow, then consume the new
generated API from `gonix`.

Do not work around missing bindings by:

- Invoking private C++ Nix APIs directly.
- Adding ad hoc cgo calls in `gonix`.
- Reimplementing Nix evaluator or store behavior in Go.
- Shelling out to `nix` for core library behavior.

Shelling out to `nix` is acceptable only for tests, diagnostics, compatibility
checks, or explicitly documented fallback tools.

## Resource Ownership

Wrap and own lifecycle for all safe entities exported by `nix-go-bindings`,
including:

- `nix_c_context`
- `Store`
- `StorePath`
- `nix_derivation`
- `EvalState`
- `nix_value`
- list and bindings builders
- fetcher settings
- flake settings
- flake references
- lock flags
- locked flakes
- realised strings
- store realisation results
- store path arrays and closures

Every wrapper must make its ownership model clear in code and docs:

- **Owned:** this wrapper must release the underlying resource.
- **Borrowed:** this wrapper must not release the underlying resource.
- **Cloned:** this wrapper owns a cloned resource and must release it.
- **Refcounted:** this wrapper must use the matching Nix incref/decref API.

When ownership is ambiguous, stop and inspect the upstream Nix C API and
`nix-go-bindings` shim before writing code.

## Error Handling

Convert Nix context errors into a Go error type.

The SDK error type should include:

- error code
- error name
- message
- optional detailed info

All public methods should return `error` when failure is possible. SDK users
should not need to inspect raw `nix_c_context` values.

Prefer wrapping errors with enough operation context to diagnose failures, while
preserving access to the original Nix error.

## API Coverage Expectations

Provide wrappers for all safe entities currently exported by `nix-go-bindings`.

The high-level API should include methods to:

- Configure and initialize Nix.
- Open stores and inspect store metadata.
- Parse, clone, hash, name, and free store paths.
- Read and write derivations through supported Nix C API surfaces.
- Evaluate Nix expressions.
- Traverse lists and attrsets.
- Build Go-native values and attrsets.
- Traverse flake outputs.
- Lock flakes and read locked flake outputs.
- Realise store paths.
- Query closures.
- Read store path and derivation metadata.

Keep unsupported callback-based APIs out of v1 unless a safe `cgo.Handle`
registry and lifetime model is explicitly designed. This includes custom
primops, external value callback descriptors, and arbitrary GC finalizers.

## Testing Rules

Tests should run inside the Nix development environment:

```sh
nix develop -c go test ./...
```

Also keep these checks healthy:

```sh
GOEXPERIMENT=cgocheck2 go test ./...
go test -race ./...
```

Tests should cover:

- Resource ownership and idempotent `Close`.
- Error conversion from Nix contexts.
- Store opening, path parsing, path metadata, closures, and realisation.
- Derivation lifecycle and JSON round trips where supported.
- Evaluation of simple expressions.
- Value forcing, primitive getters, list traversal, attrset traversal, and
  builders.
- Realised strings and referenced store paths.
- Flake reference parsing, locking, output traversal, and local test flakes.
- Isolated temporary stores where possible.

Public API tests should import and use `gonix`, not raw `nix-go-bindings`.

## Commit Style

Make logical, reviewable commits. Good examples:

- `feat: add store wrapper`
- `feat: add eval value API`
- `feat: add flake traversal`
- `fix: close realised string paths`
- `test: cover store realisation`

Avoid mixing unrelated API design, binding dependency updates, ownership fixes,
and broad behavior changes in one commit.

## Boundaries

`gonix` may be ergonomic and high-level. It should provide friendly Go APIs,
workflow helpers, ownership wrappers, and typed errors.

`gonix` should not become a full reimplementation of Nix. It should orchestrate
Nix through `nix-go-bindings`, not duplicate evaluator, store, flake, or
derivation behavior in Go.

Profile or state-management features may exist as Go-level abstractions only
when they are backed by clear Nix store/eval semantics.

Before adding a feature, ask:

- Does this belong in the high-level SDK?
- Can it be implemented through `nix-go-bindings` safely?
- Does it preserve upstream Nix semantics?
- Is ownership explicit and testable?

If the answer depends on missing low-level API, add that capability to
`nix-go-bindings` first.
