# AGENTS.md

Guidance for agents working on the `gonix` repository.

## Project Goal

`gonix` is a high-level Go SDK for working with Nix.

`gonix` is the ergonomic, high-level Go layer. The `pkg/raw` package is the
low-level generated bridge to the Nix C API and contains the C/C++ shims,
generator configuration, generated Go files, and binding tests.

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

## Raw Package Boundary

Treat `pkg/raw` as the only gateway to Nix.

High-level gonix packages should not edit generated files in `pkg/raw` by hand.
If a missing low-level API is required, add it through the `pkg/raw`
shim/config/codegen workflow, regenerate the bindings, then consume the new raw
API from the high-level package.

Do not work around missing bindings by:

- invoking private C++ Nix APIs directly;
- adding ad hoc cgo calls in `gonix`;
- reimplementing Nix evaluator, store, flake, or derivation behavior in Go;
- shelling out to `nix` for core library behavior.

Raw generated types from `pkg/raw` should normally remain behind high-level
gonix wrappers. Public high-level APIs should expose Go-native values, errors,
options, and lifecycle methods. Deliberate advanced integration points may use
`pkg/raw` types when their ownership is documented.

## Raw Binding Workflow

The low-level package is intentionally close to upstream Nix. It is not another
high-level SDK.

Current upstream C API packages covered by `pkg/raw` include:

- `nix-util-c`
- `nix-store-c`
- `nix-expr-c`
- `nix-fetchers-c`
- `nix-flake-c`
- `nix-main-c`

For each low-level API addition:

1. Inspect the upstream Nix headers and tests.
2. Add or extend a small shim under `pkg/raw`.
3. Update `pkg/raw/nix-go-bindings.yml` with explicit includes, pkg-config
   dependencies, accept/reject rules, rename rules, pointer tips, memory tips,
   and callback hints.
4. Regenerate with `make generate`.
5. Add raw binding tests organized by upstream C package area.
6. Run `make test` and `make lint` from the repository root.

The generated Go package name is `raw`. Generated files include `raw.go`,
`types.go`, `const.go`, `cgo_helpers.*`, and `doc.go`.

### C shim rules

Keep the shim small, transparent, and faithful to upstream semantics.

Use it to:

- adapt signatures that c-for-go cannot generate cleanly;
- normalize awkward arrays, callback results, or extreme pointer shapes;
- expose explicit ownership and package-specific free functions;
- provide narrow adapters for required Nix operations unavailable in the
  public C API.

Do not use it to:

- add high-level workflows or policy;
- reimplement evaluator, store, flake, or derivation behavior;
- hide ownership or error-handling requirements;
- bypass the binding generator with ad hoc cgo in high-level packages.

Prefer explicit generator rules over broad accidental generation. When changing
the generator, run `make generate` and inspect the generated diff for stale
files or unexpected API changes.

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

Do not expose generated `.Free()` helpers from `pkg/raw`. They may call
raw `C.free` and are not the correct lifecycle operation for many opaque Nix
objects. Use the API-specific lifecycle function instead.

Do not leak raw `unsafe.Pointer` or `pkg/raw` types through high-level APIs
unless there is a deliberate, documented escape hatch.

## Error Handling

Binding errors must be converted into Go errors consistently. Always preserve
the Nix context error with `%w` when wrapping.

When a binding call returns a status code, handle it in this style:

```go
if code := raw.SetVerbosity(r.ctx, raw.NixVerbosity(level)); status.ErrorCode(code) != status.ErrorCodeOK {
	return fmt.Errorf("runtime: set verbosity: %w", status.FromContext(r.ctx))
}
```

When a binding call does not return a status code but returns a pointer or
result that can signal failure, check that result immediately:

```go
namePtr := raw.StorePathName(p.ptr)
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
func (c *Client) Close() error {
	if c.ctx == nil {
		return nil
	}

	errs := make([]error, 0, len(c.resources))
	for i := len(c.resources) - 1; i >= 0; i-- {
		if err := c.resources[i].Close(); err != nil {
			errs = append(errs, err)
		}
	}

	if err := c.ctx.Close(); err != nil {
		errs = append(errs, err)
	}
	c.ctx = nil
	c.resources = nil

	if len(errs) != 0 {
		return fmt.Errorf("client: failed to close resources: %w", errors.Join(errs...))
	}

	return nil
}
```

For boolean-returning raw calls, distinguish a real `false` result from a
context error. Return `(false, nil)` only when the Nix context is still OK.

## Resource Ownership

Wrap and own lifecycle for all safe entities consumed from `pkg/raw`.

Every wrapper must make its ownership model clear in code and docs:

- **Owned:** this wrapper must release the underlying resource.
- **Borrowed:** this wrapper must not release the underlying resource.
- **Cloned:** this wrapper owns a cloned resource and must release it.
- **Refcounted:** this wrapper must use the matching Nix incref/decref API.

When ownership is ambiguous, stop and inspect the upstream Nix C API and the
`pkg/raw` shim before writing code.

## Testing

Tests run inside the root Nix development environment:

```sh
make test
```

High-level public API tests should import and use gonix packages, not `pkg/raw`,
unless the test specifically verifies a documented raw-handle integration
point.

Tests should cover resource ownership, idempotent `Close`, closed-object
behavior, Nix context error conversion, and public workflows.

Raw binding tests live in `pkg/raw` and should follow upstream package areas,
using names such as `nix_util_test.go`, `nix_store_test.go`,
`nix_expr_test.go`, `nix_fetchers_test.go`, `nix_flake_test.go`, and
`nix_main_test.go`. They should verify context/error behavior and
ownership-sensitive paths, not only happy paths.

After making changes, always run:

```sh
make test
make lint
```

If a check cannot be run, state that explicitly and explain why.

## Commit Style

Keep changes logical and reviewable. Binding additions should separate shim or
generator changes from unrelated high-level SDK work when practical. Use
descriptive commit messages such as `feat: add libstore bindings`,
`fix: generator hints for store path ownership`, or
`feat: expose store workflow`.

## Licensing

Code outside `pkg/raw` is Apache-2.0. Everything under `pkg/raw`, including
generated bindings and shims, is LGPL-2.1-or-later. Keep changes within the
license boundary documented by the nearest license file.

## Boundaries

`gonix` may be ergonomic and high-level. It should provide friendly Go APIs,
workflow helpers, ownership wrappers, and typed errors.

`gonix` should not become a full reimplementation of Nix. High-level packages
should orchestrate Nix through `pkg/raw`, not duplicate evaluator, store, flake,
or derivation behavior in Go.

Before adding a feature, ask:

- Does this belong in the high-level SDK?
- Can it be implemented through `pkg/raw` safely?
- Does it preserve upstream Nix semantics?
- Is ownership explicit and testable?

If the answer depends on missing low-level API, add that capability to
`pkg/raw` first through its shim/config/generation workflow.
