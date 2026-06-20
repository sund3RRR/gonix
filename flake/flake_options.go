package flake

// Option configures flake parsing and locking.
type Option func(*flakeConfig)

type flakeConfig struct {
	baseDirectory  string
	mode           LockMode
	inputOverrides []inputOverride
}

type inputOverride struct {
	path string
	ref  string
}

// WithLockMode sets how Nix should handle the flake lock file.
func WithLockMode(mode LockMode) Option {
	return func(c *flakeConfig) {
		c.mode = mode
	}
}

// WithInputOverride overrides a flake input path while locking.
func WithInputOverride(inputPath string, ref string) Option {
	return func(c *flakeConfig) {
		c.inputOverrides = append(c.inputOverrides, inputOverride{
			path: inputPath,
			ref:  ref,
		})
	}
}

// WithBaseDirectory sets the base directory used to resolve relative flake references.
func WithBaseDirectory(path string) Option {
	return func(c *flakeConfig) {
		c.baseDirectory = path
	}
}
