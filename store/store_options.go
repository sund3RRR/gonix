package store

// Option configures store opening.
//
// Options are applied to a Config and then serialized into Nix store
// parameters by New.
type Option func(*Config)

// WithStoreDir sets the logical Nix store directory, usually DefaultDir.
//
// Store paths can only be copied between stores with the same logical store.
func WithStoreDir(path string) Option {
	return func(c *Config) {
		c.StoreDir = path
	}
}

// WithPathInfoCacheSize sets the in-memory store path metadata cache size.
//
// Nix uses this cache for path information lookups on stores that support path
// metadata queries.
func WithPathInfoCacheSize(size int) Option {
	return func(c *Config) {
		c.PathInfoCacheSize = size
	}
}

// WithTrusted marks this store as trusted for substitution decisions.
//
// Trusted stores may provide paths that are not signed by a key listed in
// trusted-public-keys. Whether the setting is accepted depends on the backend
// and Nix configuration.
func WithTrusted(trusted bool) Option {
	return func(c *Config) {
		c.Trusted = trusted
	}
}

// WithPriority sets this store's substituter priority.
//
// Lower values have higher priority.
func WithPriority(priority int) Option {
	return func(c *Config) {
		c.Priority = priority
	}
}

// WithWantMassQuery reports whether this store can efficiently answer bulk path
// validity queries when used as a substituter.
func WithWantMassQuery(want bool) Option {
	return func(c *Config) {
		c.WantMassQuery = want
	}
}

// WithSystemFeatures sets system features available for builds on this store.
//
// Common values include "kvm", "big-parallel", and "benchmark". Nix uses these
// features when deciding whether a derivation can be built locally.
func WithSystemFeatures(features ...string) Option {
	return func(c *Config) {
		c.SystemFeatures = append([]string(nil), features...)
	}
}

// WithReadOnly opens a store in read-only mode when the backend supports it.
//
// For dummy stores it makes writes fail. For local stores it disables database
// locking and should only be used when the filesystem is actually read-only.
func WithReadOnly(readOnly bool) Option {
	return func(c *Config) {
		c.ReadOnly = readOnly
	}
}

// WithBuildDir sets the host directory used for temporary derivation build directories.
//
// Do not point this at a world-writable directory.
func WithBuildDir(path string) Option {
	return func(c *Config) {
		c.BuildDir = path
	}
}

// WithLogDir sets the directory where Nix stores build logs.
func WithLogDir(path string) Option {
	return func(c *Config) {
		c.LogDir = path
	}
}

// WithRealStoreDir sets the physical filesystem path of the Nix store.
//
// This may differ from StoreDir when a store is rooted somewhere other than the
// logical /nix/store path.
func WithRealStoreDir(path string) Option {
	return func(c *Config) {
		c.RealStoreDir = path
	}
}

// WithRequireSignatures controls whether copied paths must have a trusted
// signature before entering this store.
func WithRequireSignatures(require bool) Option {
	return func(c *Config) {
		c.RequireSignatures = require
	}
}

// WithRootDir sets the root directory for local filesystem store paths.
//
// Nix prefixes store, state, log, and related directories with this path for
// store backends that support a root directory.
func WithRootDir(path string) Option {
	return func(c *Config) {
		c.RootDir = path
	}
}

// WithStateDir sets the directory where Nix stores mutable state.
func WithStateDir(path string) Option {
	return func(c *Config) {
		c.StateDir = path
	}
}
