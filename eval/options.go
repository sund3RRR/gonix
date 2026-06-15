// Package eval wraps Nix evaluation states and values.
//
// The package exposes Go-native evaluators, values, builders, and realized
// strings over the generated nix-go-bindings expression API. Evaluators own the
// underlying Nix evaluation state and values are tied to the evaluator that
// created them.
package eval

// Option configures Evaluator creation.
type Option func(*config)

type config struct {
	lookupPath []string
}

// WithLookupPath sets the Nix evaluator lookup path.
//
// Entries use Nix lookup-path syntax, for example "nixpkgs=/path/to/nixpkgs".
// Repeated options append entries in order.
func WithLookupPath(entries ...string) Option {
	return func(c *config) {
		c.lookupPath = append(c.lookupPath, entries...)
	}
}
