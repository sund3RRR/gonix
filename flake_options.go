package gonix

import "github.com/sund3RRR/gonix/flake"

// FetchPackageOption configures FetchPackage.
type FetchPackageOption func(*fetchPackageConfig)

type fetchPackageConfig struct {
	system string
}

// WithFetchPackageSystem overrides the package system for one FetchPackage call.
func WithFetchPackageSystem(system string) FetchPackageOption {
	return func(c *fetchPackageConfig) {
		c.system = system
	}
}

// ListPackagesOption configures ListPackages.
type ListPackagesOption func(*listPackagesConfig)

type listPackagesConfig struct {
	system string
}

// WithListPackagesSystem overrides the package system for one ListPackages call.
func WithListPackagesSystem(system string) ListPackagesOption {
	return func(c *listPackagesConfig) {
		c.system = system
	}
}

type flakeConfig struct {
	parseOpts []flake.ParseOption
	lockOpts  []flake.LockOption
}

// FlakeOption configures Flake creation.
type FlakeOption func(*flakeConfig)

// WithParseOpts configures flake-reference parsing.
func WithParseOpts(opts ...flake.ParseOption) FlakeOption {
	copied := append([]flake.ParseOption(nil), opts...)
	return func(c *flakeConfig) {
		c.parseOpts = copied
	}
}

// WithLockOpts configures flake locking.
func WithLockOpts(opts ...flake.LockOption) FlakeOption {
	copied := append([]flake.LockOption(nil), opts...)
	return func(c *flakeConfig) {
		c.lockOpts = copied
	}
}
