# AGENTS.md

Guidance for agents working on the `gonix` repository.

## Project Goal

`gonix` is a high-level Go SDK for working with Nix.

`nix-go-bindings` is the low-level, generated bridge to the Nix C API.
`gonix` is the ergonomic Go layer built on top of that bridge.

Use `gonix` to provide Go-native APIs for:

- Nix runtime initialization and settings.
- Stores and store paths.
- Evaluation states and Nix values.
- Derivations.
- Flakes and locked flakes.
- Realized outputs and closures.
- Error handling and ownership-safe resource lifecycle.

Do not use `gonix` to duplicate generated bindings, call private Nix C++ APIs
directly, or shell out to the `nix` CLI for core SDK behavior.

Shelling out to `nix` is acceptable only for tests, diagnostics, compatibility
checks, or explicitly documented fallback tools.

## Documentation

Use the repository docs as the source of truth for design decisions:

- [docs/architecture.md](docs/architecture.md): SDK architecture, ownership,
  package boundaries, dependency direction, object lifetimes, and diagrams.
- [docs/nix-settings.md](docs/nix-settings.md): Nix settings table with keys,
  types, defaults, experimental gates, aliases, and descriptions.

Read the relevant document before changing architecture, ownership, package
boundaries, runtime settings, or Nix settings behavior.

If a change alters architecture, ownership rules, package boundaries, public
resource lifecycle, runtime settings semantics, or the supported settings
surface, update the relevant file in `docs/` in the same change.

All public Go entities must have concise godoc comments. After adding or
changing public API documentation, verify the rendered output with `go doc` for
the affected package or symbols.

## Nix-Go-Bindings Boundary

Treat `nix-go-bindings` as the only gateway to Nix.

Agents working on `gonix` should not modify generated bindings inside this
repository. If a missing low-level API is required, add it first to
`nix-go-bindings` through its shim/config/codegen workflow, then consume the new
generated API from `gonix`.

Do not work around missing bindings by:

- invoking private C++ Nix APIs directly;
- adding ad hoc cgo calls in `gonix`;
- reimplementing Nix evaluator, store, flake, or derivation behavior in Go;
- shelling out to `nix` for core library behavior.

Raw generated types from `nix-go-bindings` should normally remain internal to
`gonix`. Public APIs should expose Go-native values, errors, options, and
lifecycle methods.

## API Principles

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
objects. Use the API-specific lifecycle function instead.

Do not leak raw `unsafe.Pointer` or `nix-go-bindings` types through public APIs
unless there is a deliberate, documented escape hatch.

## Error Handling

Binding errors must be converted into Go errors consistently. Always preserve
the Nix context error with `%w` when wrapping.

When a binding call returns a status code, handle it in this style:

```go
if code := nix.SetVerbosity(r.ctx, nix.NixVerbosity(level)); status.ErrorCode(code) != status.ErrorCodeOK {
	return fmt.Errorf("runtime: set verbosity: %w", status.FromContext(r.ctx))
}
```

When a binding call does not return a status code but returns a pointer or
result that can signal failure, check that result immediately:

```go
namePtr := nix.StorePathName(p.ptr)
if namePtr == nil {
	return "", fmt.Errorf("storepath: failed to get store path name: %w", status.FromContext(p.ctx))
}
```

Methods that depend on an owned raw pointer must check the pointer at the start
of the method. Use the correct zero value for the method's return type:

```go
if p.ptr == nil {
	return nil, status.ErrClosed
}
```

When a method closes several resources and can collect multiple errors, keep
closing every resource, join errors, and return one wrapped error:

```go
func (r *Runtime) Close() error {
	if r.ctx == nil {
		return nil
	}

	errs := make([]error, 0, len(r.resources))
	for i := len(r.resources) - 1; i >= 0; i-- {
		if err := r.resources[i].Close(); err != nil {
			errs = append(errs, err)
		}
	}

	nix.CContextFree(r.ctx)
	r.ctx = nil
	r.resources = nil

	if len(errs) != 0 {
		return fmt.Errorf("runtime: failed to close resources: %w", errors.Join(errs...))
	}

	return nil
}
```

For boolean-returning raw calls, distinguish a real `false` result from a
context error. Return `(false, nil)` only when the Nix context is still OK.

## Resource Ownership

Wrap and own lifecycle for all safe entities exported by `nix-go-bindings`.

Every wrapper must make its ownership model clear in code and docs:

- **Owned:** this wrapper must release the underlying resource.
- **Borrowed:** this wrapper must not release the underlying resource.
- **Cloned:** this wrapper owns a cloned resource and must release it.
- **Refcounted:** this wrapper must use the matching Nix incref/decref API.

When ownership is ambiguous, stop and inspect the upstream Nix C API and
`nix-go-bindings` shim before writing code.

## Testing

Tests should run inside the Nix development environment:

```sh
nix develop github:sund3RRR/nix-go-bindings --command go test ./...
```

Public API tests should import and use `gonix`, not raw `nix-go-bindings`.

Tests should cover resource ownership, idempotent `Close`, closed-object
behavior, Nix context error conversion, and public workflows.

After making changes, always run:

```sh
make test
make lint
```

If a check cannot be run, state that explicitly and explain why.

## Boundaries

`gonix` may be ergonomic and high-level. It should provide friendly Go APIs,
workflow helpers, ownership wrappers, and typed errors.

`gonix` should not become a full reimplementation of Nix. It should orchestrate
Nix through `nix-go-bindings`, not duplicate evaluator, store, flake, or
derivation behavior in Go.

Before adding a feature, ask:

- Does this belong in the high-level SDK?
- Can it be implemented through `nix-go-bindings` safely?
- Does it preserve upstream Nix semantics?
- Is ownership explicit and testable?

If the answer depends on missing low-level API, add that capability to
`nix-go-bindings` first.
