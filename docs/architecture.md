# gonix architecture

`gonix` is a high-level Go SDK for Nix. It wraps
`github.com/sund3RRR/nix-go-bindings`, which is the generated, low-level bridge
to the Nix C API.

The SDK goal is to expose Nix through Go-native APIs: runtime initialization,
settings, stores, store paths, derivations, evaluation states, values, flakes,
realized outputs, closures, ownership-safe resource lifecycle, and structured
errors.

## Boundary with nix-go-bindings

`nix-go-bindings` is the only direct portal to Nix. It exposes raw opaque types
and raw lifecycle functions. `gonix` should wrap those resources before they
cross a public API boundary.

The public SDK should not:

- duplicate generated bindings;
- expose generated `.Free()` helpers as public API;
- call private Nix C++ APIs directly;
- add ad hoc cgo calls around missing bindings;
- shell out to the `nix` CLI for core SDK behavior.

Shelling out is acceptable for tests, diagnostics, and compatibility checks.

Use API-specific free functions instead of generated `.Free()` methods:
`StoreFree`, `StorePathFree`, `DerivationFree`, `StateFree`, `ValueDecref`,
`FlakeReferenceFree`, `LockedFlakeFree`, and similar functions from
`nix-go-bindings`.

Callback-heavy APIs are out of scope for v1 unless a safe `cgo.Handle` registry
and lifetime model is designed. This includes custom primops, external value
callback descriptors, raw store callbacks, and arbitrary GC finalizers.

## Entrypoint model

The selected v1 entrypoint is `gonix.Runtime`.

`Runtime` initializes Nix, owns the Nix C context, applies settings, and creates
high-level objects such as stores, evaluators, flake references, and locked
flakes. It does not proxy domain operations through itself. After a child object
is returned, callers use the child object's own methods.

This gives users one clear bootstrap object without turning the root package
into a large proxy facade.

### Runtime responsibilities

- Create and free `nix_c_context`.
- Initialize selected Nix libraries.
- Apply process/global settings, verbosity, and log format.
- Open stores.
- Create evaluators.
- Parse and lock flakes.
- Lazily own fetcher and flake settings when flake APIs need them.
- Track resources created through the runtime and close them in reverse order.

### Runtime lifetime

The runtime must outlive every object it creates or helps construct:

- `store.Store`
- `storepath.Path`
- `store.Derivation`
- `eval.Evaluator`
- `eval.Value`
- `flake.Ref`
- `flake.LockedFlake`
- `nixpkg.Package`

Closing the runtime ends the object graph. Child objects should reject use after
close with `status.ErrClosed` wrapped by the public method that detected it.

### Concurrency

One `Runtime` is one execution stream. Public wrappers are not goroutine-safe
unless a type explicitly documents otherwise.

Parallel Nix work should use separate `Runtime` instances, each with its own
stores, evaluators, and values. This keeps the API honest about mutable Nix
contexts and eval states and avoids hidden locking semantics in v1.

## Package layout

Packages are split by ownership boundary and dependency direction, not by every
Nix noun.

| Go package | Public abstractions | Responsibility |
| --- | --- | --- |
| `gonix` | `Runtime`, root error aliases, runtime options | SDK entrypoint, Nix initialization, settings, resource tracking, factory methods. |
| `storepath` | `Path` | Independent owned wrapper for `*nix.StorePath`. Does not depend on `store`. |
| `store` | `Store`, `Derivation`, `Realization`, `Closure` | Store-backed workflows: store metadata, path parsing, derivations, realization, closure traversal, copying. |
| `eval` | `Evaluator`, `Value`, `ValueType`, builders, realized strings | Evaluation state and values. `Value` lives here because many value operations require an `EvalState`. |
| `flake` | `Ref`, `LockedFlake`, parse and lock options | Flake references, locking, and locked output access. |
| `nixpkg` | `Package` | Later convenience layer around package-shaped Nix values. `package` is a Go keyword. |
| `internal/status` | `NixError`, `ErrorCode`, `ErrClosed` | Conversion from mutable Nix context errors into stable Go errors. |
| `internal/utils` | C string and small raw adapters | Shared implementation details for wrappers. |

Import direction must stay one-way:

- root `gonix` may import `store`, `storepath`, `eval`, `flake`, and `nixpkg`;
- `store` may import `storepath`;
- `eval` may import `store` and `storepath`;
- `flake` may import `eval`;
- `nixpkg` may import `eval`, `store`, and `storepath`;
- subpackages must not import root `gonix`.

## Public call boundaries

Public workflow methods should accept and return high-level wrappers:
`*storepath.Path`, `*store.Store`, `*store.Derivation`, `*eval.Value`, and so
on.

Low-level constructors may accept raw pointers only when they are integration
points between sibling packages or tests. For example, `storepath.New(ctx, ptr)`
accepts a raw Nix context plus an owned raw `*nix.StorePath`.

Do not expose raw `*nix.EvalState` in public value APIs. In v1,
state-dependent value operations should be methods on `*eval.Evaluator`, such
as `Evaluator.Force(value)` or `Evaluator.Attr(value, name)`.

## Core abstractions

| Abstraction | Owns | Borrows or depends on | Public inputs |
| --- | --- | --- | --- |
| `gonix.Runtime` | `*nix.NixCContext`, process settings, lazy fetcher/flake settings | nothing above it | runtime options, store URIs, high-level option structs |
| `store.Store` | `*nix.Store` | runtime context | `*storepath.Path`, `*store.Derivation`, store options |
| `storepath.Path` | `*nix.StorePath` | Nix context for error-producing methods | no `Store`; raw pointer only through `New` and `Borrow` escape hatches |
| `store.Derivation` | `*nix.NixDerivation` | runtime/store context for JSON and store operations | JSON strings, cloned derivations |
| `eval.Evaluator` | `*nix.EvalState` | `*store.Store`, runtime context | `*eval.Value` for state-dependent value operations |
| `eval.Value` | `*nix.NixValue` reference | evaluator identity and context | state-independent getters only; state-dependent operations go through `Evaluator` |
| `flake.Ref` | `*nix.NixFlakeReference`, fragment string | fetcher/flake settings context | parse options, input override paths |
| `flake.LockedFlake` | `*nix.NixLockedFlake` | `*eval.Evaluator`, flake settings | lock options, `*flake.Ref`; returns `*eval.Value` outputs |
| `nixpkg.Package` | usually no raw Nix object; may wrap an `eval.Value` | `*eval.Evaluator`, optional `*store.Store` | package value plus evaluator/store-backed helpers |

### Runtime

Typical creation surface:

- `gonix.NewRuntime(opts ...Option) (*Runtime, error)`
- `Runtime.Close() error`
- `Runtime.OpenStore(uri string, opts ...store.Option) (*store.Store, error)`
- `Runtime.NewEvaluator(store *store.Store, opts ...eval.Option) (*eval.Evaluator, error)`
- `Runtime.ParseFlakeRef(ref string, opts ...flake.ParseOption) (*flake.Ref, error)`
- `Runtime.LockFlake(e *eval.Evaluator, ref *flake.Ref, opts ...flake.LockOption) (*flake.LockedFlake, error)`

Runtime methods should stop at bootstrapping and composition. Store operations
belong on `store.Store`; value operations belong on `eval.Evaluator` or
`eval.Value`; flake operations belong on `flake.Ref` and `flake.LockedFlake`.

### Store

`store.Store` owns a raw `*nix.Store` and borrows the runtime context.

Store creation:

- `Runtime.OpenStore(uri string, opts ...store.Option)`
- package-level constructors for internal integration when a raw handle is
  already owned.

Core methods:

- metadata: `URI`, `StoreDir`, `Version`;
- paths: `ParsePath`, `PathFromHash`, `RealPath`, `IsValidPath`;
- derivations: `DerivationFromJSON`, `DerivationFromPath`, `AddDerivation`;
- realization: `Realise`;
- closure: `Closure`, `CopyClosure`;
- copying: `CopyPathTo`.

`Store` creates many `storepath.Path` and `store.Derivation` wrappers. Returned
wrappers own their raw resources and must be closed independently unless they
are tracked by a parent object that documents otherwise.

### StorePath

`storepath.Path` is an owned Nix store path handle.

Creation paths:

- `Store.ParsePath`;
- `Path.Clone`;
- `Store.PathFromHash`;
- `storepath.FromParts`;
- `Store.AddDerivation`;
- closure and realization result conversion.

Core methods:

- `Name() (string, error)`;
- `Hash() ([20]byte, error)`;
- `Borrow() (*nix.StorePath, error)`;
- `Clone() (*storepath.Path, error)`;
- `Close() error`.

`Path` intentionally has no `String` or `RealPath` method. Formatting a full
store path depends on `Store`, and `storepath` must not depend on `store`.

### Derivation

`store.Derivation` owns a raw `*nix.NixDerivation`.

It lives in the `store` package for v1 because importing, exporting, querying,
and adding derivations are store-backed workflows.

Creation paths:

- `Store.DerivationFromJSON`;
- `Store.DerivationFromPath`;
- `Derivation.Clone`.

Core methods:

- `JSON() (string, error)`;
- `Clone() (*Derivation, error)`;
- `Borrow() (*nix.NixDerivation, error)`;
- `Close() error`.

`Store.AddDerivation(derivation)` returns an owned `*storepath.Path`.

### Evaluator

`eval.Evaluator` owns a raw `*nix.EvalState` and borrows a `store.Store` plus
the runtime context.

Creation:

- `Runtime.NewEvaluator(store *store.Store, opts ...eval.Option)`;
- package-level construction for integration when a caller already owns the
  context.

`eval.Options` should include lookup path entries and feature toggles needed by
the eval-state builder.

Core methods:

- `EvalString(expr, path string) (*eval.Value, error)`;
- `NewValue(v eval.GoValue) (*eval.Value, error)`;
- `Force(value)`;
- `ForceDeep(value)`;
- `Call(fn, arg)`;
- `CallMulti(fn, args...)`;
- `Index(value, i)`;
- `Attr(value, name)`;
- `RealiseString(value)`;
- `Close() error`.

`Evaluator` creates `Value` wrappers. Values are tied to the evaluator because
forcing, calls, list traversal, attr traversal, path strings, and realized
strings require an `EvalState`.

### Value

`eval.Value` wraps an owned or refcounted `*nix.NixValue`.

The raw value handle is not enough for many useful operations. Nix requires the
`EvalState` for forcing, function calls, list and attrset traversal,
path-string initialization and getters, and realized strings.

Decision for v1:

- keep `Evaluator` and `Value` in the same package;
- implement state-dependent operations as methods on `*eval.Evaluator`;
- store unexported origin metadata in `Value` so evaluator methods can reject
  values from a different evaluator;
- do not accept raw `*nix.EvalState` in public APIs.

Value methods:

- `Type() (ValueType, error)`;
- `TypeName() (string, error)`;
- primitive getters such as `Bool`, `Int`, `Float`, and `String`;
- `Borrow() (*nix.NixValue, error)`;
- `Close() error`.

`Value.Close` should call `ValueDecref`. Values returned by list and attr
lookups should be treated as owned references unless upstream ownership proves
otherwise.

### Realization and Closure

Realization and closure results should be Go-native result structs, not
long-lived raw wrappers.

```go
type Realization struct {
	OutputName string
	Path       *storepath.Path
}

type Closure struct {
	Paths []*storepath.Path
}
```

`Store.Realise` converts raw realization results into `[]Realization`, clones
paths as owned `storepath.Path` wrappers, and frees the raw result handle before
returning.

`Store.Closure` converts raw `StorePathArray` handles into owned paths and frees
the raw array before returning.

### Flakes

Flake APIs need runtime-owned fetcher settings, runtime-owned flake settings,
an evaluator, and often a store.

Creation:

- `Runtime.ParseFlakeRef(ref string, opts ...flake.ParseOption) (*flake.Ref, error)`
- `Runtime.LockFlake(e *eval.Evaluator, ref *flake.Ref, opts ...flake.LockOption) (*flake.LockedFlake, error)`

`flake.Ref` owns a raw `*nix.NixFlakeReference` and stores the parsed fragment
as a Go string.

`flake.LockedFlake` owns a raw `*nix.NixLockedFlake` and borrows the runtime
context, `eval.Evaluator`, and flake settings.

The main v1 method can be:

- `OutputAttrs() (*eval.Value, error)`

Higher-level traversal should build on `Evaluator.Attr(value, name)` and later
package helpers.

### Package

`nixpkg.Package` is a convenience layer, not a raw Nix C concept.

Expected shape:

- wraps a `Value` that represents a package or derivation output;
- borrows the same `eval.Evaluator` as that value;
- uses `Store` for realization and output path inspection;
- exposes name, system, output, and derivation helpers when those are safely
  available.

Package helpers should wait until `Value` and flake output traversal are stable.

## Dependency graph

Edges describe ownership, factory calls, or required state.

```mermaid
flowchart TD
    Runtime["gonix.Runtime\nowns Nix context + library init"]
    Settings["Settings API\nsettings/version/verbosity/log/plugins"]
    Fetchers["FetcherSettings\nowned by Runtime"]
    FlakeSettings["FlakeSettings\nowned by Runtime"]
    Store["store.Store\nowns *nix.Store"]
    StorePath["storepath.Path\nowns *nix.StorePath"]
    Derivation["store.Derivation\nowns *nix.NixDerivation"]
    EvalBuilder["EvalStateBuilder\ninternal temporary"]
    Eval["eval.Evaluator\nowns *nix.EvalState"]
    Value["eval.Value\nowns/refcounts *nix.NixValue\nbound to evaluator"]
    ValueBuilders["ListBuilder / BindingsBuilder\ninternal temporary"]
    RealisedString["RealisedString\nconverted to Go data"]
    Realization["store.Realization\nGo-native output result"]
    Closure["store.Closure\nGo-native path set"]
    FlakeParseFlags["FlakeParseFlags\ninternal temporary"]
    FlakeRef["flake.Ref\nowns *nix.NixFlakeReference\nplus fragment"]
    FlakeLockFlags["FlakeLockFlags\ninternal temporary"]
    LockedFlake["flake.LockedFlake\nowns *nix.NixLockedFlake\nborrows evaluator"]
    Package["nixpkg.Package\nconvenience wrapper around package Value"]

    Runtime -->|exposes| Settings
    Runtime -->|owns lazily| Fetchers
    Runtime -->|owns lazily| FlakeSettings
    Runtime -->|opens| Store
    Runtime -->|creates| EvalBuilder
    Runtime -->|creates| Eval

    Store -->|borrows Runtime context| Runtime
    Store -->|ParsePath / PathFromHash / FromParts| StorePath
    Store -->|RealPath / IsValid / CopyPath consume| StorePath
    Store -->|DerivationFromJSON| Derivation
    Store -->|DerivationFromPath consumes| StorePath
    Store -->|DerivationFromPath returns| Derivation
    Store -->|AddDerivation consumes| Derivation
    Store -->|AddDerivation returns| StorePath
    Store -->|Realise consumes| StorePath
    Store -->|Realise returns| Realization
    Store -->|Closure consumes| StorePath
    Store -->|Closure returns| Closure
    Store -->|CopyClosure / CopyPath use dst Store| Store

    StorePath -->|Clone returns| StorePath
    Derivation -->|Clone returns| Derivation
    Realization -->|contains owned path| StorePath
    Closure -->|contains owned paths| StorePath

    EvalBuilder -->|built with Store| Store
    FlakeSettings -->|AddToEvalStateBuilder for getFlake| EvalBuilder
    EvalBuilder -->|Build| Eval
    Eval -->|borrows Store| Store
    Eval -->|EvalString / AllocValue / NewValue| Value
    Eval -->|NewValue list/attrs uses| ValueBuilders
    ValueBuilders -->|produce| Value

    Value -->|records origin evaluator for validation| Eval
    Eval -->|Force / ForceDeep require EvalState| Value
    Eval -->|Call / CallMulti require EvalState| Value
    Eval -->|Index / Attr traversal require EvalState| Value
    Eval -->|PathString / RealiseString require EvalState| Value
    Eval -->|list/attr traversal returns child refs| Value
    Eval -->|RealiseString returns| RealisedString
    RealisedString -->|contains cloned referenced paths| StorePath

    Fetchers -->|parse ref uses| FlakeRef
    FlakeSettings -->|parse ref uses| FlakeRef
    FlakeSettings -->|creates| FlakeParseFlags
    FlakeParseFlags -->|base directory affects| FlakeRef
    FlakeSettings -->|creates| FlakeLockFlags
    FlakeRef -->|Lock consumes| LockedFlake
    FlakeLockFlags -->|Lock mode and overrides| LockedFlake
    Fetchers -->|Lock uses| LockedFlake
    FlakeSettings -->|Lock and OutputAttrs use| LockedFlake
    Eval -->|Lock / OutputAttrs require EvalState| LockedFlake
    LockedFlake -->|OutputAttrs returns| Value

    Value -->|package output attrs can be wrapped as| Package
    Package -->|wraps / borrows| Value
    Package -->|uses evaluator through Value| Eval
    Package -->|uses store for realization/output paths| Store
    Package -->|may expose derivation metadata through| Derivation
    Package -->|realized outputs return| StorePath
```

## Layer diagram

The API is layered from user-facing orchestration down to raw bindings.

```mermaid
flowchart TB
    subgraph L1["Root entrypoint"]
        Runtime["gonix.Runtime"]
    end

    subgraph L2["Session and service objects"]
        Store["store.Store"]
        Eval["eval.Evaluator"]
        FlakeRef["flake.Ref"]
        LockedFlake["flake.LockedFlake"]
    end

    subgraph L3["Resources and values"]
        StorePath["storepath.Path"]
        Derivation["store.Derivation"]
        Value["eval.Value"]
        Realization["store.Realization"]
        Closure["store.Closure"]
        Package["nixpkg.Package"]
    end

    subgraph L4["Internal helpers and temporary wrappers"]
        Status["internal/status"]
        Utils["internal/utils"]
        EvalBuilder["EvalStateBuilder"]
        Builders["value/list/attr builders"]
        ResultAdapters["realise/closure adapters"]
        Fetchers["fetcher settings"]
        FlakeSettings["flake settings"]
        FlakeParseFlags["flake parse flags"]
        FlakeLockFlags["flake lock flags"]
    end

    subgraph L5["nix-go-bindings raw API"]
        RawContext["NixCContext"]
        RawStore["Store / StorePath / Derivation"]
        RawEval["EvalState / NixValue / builders"]
        RawFlake["fetchers / flakes / locked flakes"]
    end

    Runtime --> Store
    Runtime --> Eval
    Runtime --> FlakeRef
    Runtime --> LockedFlake
    Runtime --> Fetchers
    Runtime --> FlakeSettings

    Store --> StorePath
    Store --> Derivation
    Store --> Realization
    Store --> Closure
    Eval --> Value
    LockedFlake --> Value
    Value --> Package
    Package --> StorePath
    Package --> Derivation

    Runtime --> Status
    Store --> Status
    Eval --> Status
    Value --> Status
    Store --> ResultAdapters
    Eval --> EvalBuilder
    Eval --> Builders
    FlakeRef --> FlakeParseFlags
    LockedFlake --> FlakeLockFlags
    FlakeSettings --> EvalBuilder
    Status --> Utils

    Runtime --> RawContext
    Store --> RawStore
    StorePath --> RawStore
    Derivation --> RawStore
    Eval --> RawEval
    Value --> RawEval
    EvalBuilder --> RawEval
    Builders --> RawEval
    FlakeRef --> RawFlake
    LockedFlake --> RawFlake
    Fetchers --> RawFlake
    FlakeSettings --> RawFlake
    FlakeParseFlags --> RawFlake
    FlakeLockFlags --> RawFlake
```

## Ownership table

| gonix type | Raw object | Ownership | Close operation |
| --- | --- | --- | --- |
| `gonix.Runtime` | `*nix.NixCContext` | owned | `CContextFree` |
| `store.Store` | `*nix.Store` | owned, borrows runtime context | `StoreFree` |
| `storepath.Path` | `*nix.StorePath` | owned or cloned before public return | `StorePathFree` |
| `store.Derivation` | `*nix.NixDerivation` | owned or cloned | `DerivationFree` |
| `eval.Evaluator` | `*nix.EvalState` | owned, borrows `store.Store` | `StateFree` |
| `eval.Value` | `*nix.NixValue` | owned/refcounted, tied to an evaluator | `ValueDecref` |
| list/attr builders | raw builders | internal temporary | matching builder free function |
| realized strings | `*nix.NixRealisedString` | internal temporary | `RealisedStringFree` |
| `flake.Ref` | `*nix.NixFlakeReference` | owned | `FlakeReferenceFree` |
| `flake.LockedFlake` | `*nix.NixLockedFlake` | owned, borrows `eval.Evaluator` | `LockedFlakeFree` |
| `nixpkg.Package` | none directly | wraps or borrows package `eval.Value`; may borrow `store.Store` | no raw free; close owned `Value` if it owns one |
| fetcher/flake settings | raw settings | owned by `Runtime` | matching settings free functions |

Every owned wrapper must implement idempotent `Close() error`. Public operations
after `Close` should return a wrapped `status.ErrClosed`.

Borrowed raw pointers may be exposed only through narrow, documented escape
hatches such as `Borrow`. Callers must not free borrowed pointers and must not
retain them beyond the immediate raw call.

## Error handling

All public methods should return Go `error` values when failure is possible.

Rules:

- Check every raw call that returns `nix.NixErr`.
- Convert raw context errors through `status.FromContext`.
- Wrap errors with operation context using `fmt.Errorf("...: %w", err)`.
- Check nil pointers returned by raw APIs and convert the Nix context error.
- For boolean raw calls, distinguish a real `false` result from a context error.
- Convert C-owned strings to Go strings immediately with `utils.TakeCString`.
- Convert raw result handles into Go structs/slices immediately, then free the
  raw handle before returning.

## File layout

Expected layout for v1:

| Path | Contents |
| --- | --- |
| `errors.go` | Public aliases for structured errors and `ErrClosed`. |
| `runtime.go`, `runtime_config.go`, `runtime_options.go` | `Runtime`, initialization, settings, verbosity, log format, typed runtime options. |
| `storepath/path.go` | `storepath.Path`, constructors, metadata, clone, borrow, close. |
| `store/store.go` | `store.Store`, open options, metadata, path operations, copy operations. |
| `store/derivation.go` | `store.Derivation` and store-backed derivation workflows. |
| `store/realization.go` | `store.Realization`, `store.Closure`, raw result conversion. |
| `eval/evaluator.go` | `eval.Evaluator`, options, eval-state builder integration. |
| `eval/value.go` | `eval.Value`, `ValueType`, primitive getters, ownership, borrow. |
| `eval/ops.go` | State-dependent value operations as `Evaluator` methods. |
| `eval/builders.go` | Go-to-Nix values, list builders, attr builders. |
| `eval/realised_string.go` | Realized string conversion and referenced path cloning. |
| `flake/ref.go`, `flake/lock.go` | Flake references, parse options, lock options, locked output attrs. |
| `nixpkg/package.go` | Later package convenience layer. |
| `internal/status` | Error conversion, error codes, `ErrClosed`. |
| `internal/utils` | C string conversion and small raw adapters. |
