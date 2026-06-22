// Package eval wraps Nix evaluation states and values.
//
// The package exposes Go-native evaluators, values, builders, and realized
// strings over the generated pkg/raw expression API. Evaluators own the
// underlying Nix evaluation state and values are tied to the evaluator that
// created them.
package eval

import (
	"github.com/sund3RRR/gonix/flakesettings"
)

// Option configures Evaluator creation.
type Option func(*config)

type config struct {
	lookupPath    []string
	flakesettings *flakesettings.Settings
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

// WithFlakeSettings adds flake evaluator integration to the state builder.
func WithFlakeSettings(settings *flakesettings.Settings) Option {
	return func(c *config) {
		c.flakesettings = settings
	}
}
