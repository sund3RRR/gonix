# gonix

Experimental high-level Go SDK for working with [Nix](https://github.com/NixOS/nix).

`gonix` is an ergonomic Go layer built on top of
[`nix-go-bindings`](https://github.com/sund3RRR/nix-go-bindings), which provides
low-level generated bindings to the Nix C API.

The goal of this package is to expose Nix concepts through Go-native APIs:
contexts, stores, store paths, evaluation states, values, derivations, flakes,
locked flakes, realised outputs, closures, settings, and structured errors.

## License

Apache-2.0.

Note that `gonix` depends on [`nix-go-bindings`](https://github.com/sund3RRR/nix-go-bindings) and the Nix C libraries, which
have their own license terms.
