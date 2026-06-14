package store

// Option configures store opening.
type Option func(*Config)

// WithStoreDir sets the logical Nix store directory, usually /nix/store.
//
// Store paths can only be copied between stores with the same logical store.
func WithStoreDir(path string) Option {
	return func(c *Config) {
		c.StoreDir = path
	}
}

// WithPathInfoCacheSize sets the in-memory store path metadata cache size.
func WithPathInfoCacheSize(size int) Option {
	return func(c *Config) {
		c.PathInfoCacheSize = size
	}
}

// WithTrusted allows paths from this store to be used as substitutes even when
// they are not signed by a key listed in trusted-public-keys.
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

// WithSystemFeatures sets system features available for builds on this store,
// such as "kvm".
func WithSystemFeatures(features ...string) Option {
	return func(c *Config) {
		c.SystemFeatures = append([]string(nil), features...)
	}
}

// WithReadOnly opens a store in read-only mode when the store backend supports it.
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

// WithLogDir sets the directory where Nix stores log files.
func WithLogDir(path string) Option {
	return func(c *Config) {
		c.LogDir = path
	}
}

// WithRealStoreDir sets the physical filesystem path of the Nix store.
func WithRealStoreDir(path string) Option {
	return func(c *Config) {
		c.RealStoreDir = path
	}
}

// WithRequireSignatures controls whether store paths copied into this store
// must have a trusted signature.
func WithRequireSignatures(require bool) Option {
	return func(c *Config) {
		c.RequireSignatures = require
	}
}

// WithRootDir sets the directory prefixed to other local filesystem store paths.
func WithRootDir(path string) Option {
	return func(c *Config) {
		c.RootDir = path
	}
}

// WithStateDir sets the directory where Nix stores state.
func WithStateDir(path string) Option {
	return func(c *Config) {
		c.StateDir = path
	}
}
