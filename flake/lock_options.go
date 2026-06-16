package flake

// LockOption configures flake locking.
type LockOption func(*lockConfig)

type lockConfig struct {
	mode           LockMode
	inputOverrides []inputOverride
}

type inputOverride struct {
	path string
	ref  *Ref
}

// WithLockMode sets how Nix should handle the flake lock file.
func WithLockMode(mode LockMode) LockOption {
	return func(c *lockConfig) {
		c.mode = mode
	}
}

// WithInputOverride overrides a flake input path while locking.
func WithInputOverride(inputPath string, ref *Ref) LockOption {
	return func(c *lockConfig) {
		c.inputOverrides = append(c.inputOverrides, inputOverride{
			path: inputPath,
			ref:  ref,
		})
	}
}
