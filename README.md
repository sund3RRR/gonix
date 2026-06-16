# gonix

Experimental high-level Go SDK for working with [Nix](https://github.com/NixOS/nix).

`gonix` is an ergonomic Go layer built on top of
[`nix-go-bindings`](https://github.com/sund3RRR/nix-go-bindings), which provides
low-level generated bindings to the Nix C API.

The goal of this package is to expose Nix concepts through Go-native APIs:
contexts, stores, store paths, evaluation states, values, derivations, flakes,
locked flakes, realised outputs, closures, settings, and structured errors.

## Implementation status

`gonix` is currently an early SDK foundation. The implemented surface is useful
for runtime and store-backed workflows, but the broader high-level SDK is still
in progress.

- [x] Runtime and settings foundation.
  - [x] Runtime initialization and shutdown.
  - [x] Runtime options for selected Nix settings, verbosity, and log format.
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
- [ ] Fetchers.
  - [x] Fetcher settings lifecycle.
  - [ ] Fetch tree and input resolution helpers.
  - [ ] Store path conversion for fetched inputs.
- [x] Flakes.
  - [x] Flake reference parsing.
  - [x] Locked flake workflows.
  - [x] Locked output access.
- [ ] Nix package convenience API.
  - [ ] Package-shaped value wrappers.
  - [ ] Build/install metadata helpers.
  - [ ] Store-backed package realization helpers.

## License

Apache-2.0.

Note that `gonix` depends on [`nix-go-bindings`](https://github.com/sund3RRR/nix-go-bindings) and the Nix C libraries, which
have their own license terms.
