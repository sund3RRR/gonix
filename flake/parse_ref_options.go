package flake

// ParseOption configures flake reference parsing.
type ParseOption func(*parseConfig)

type parseConfig struct {
	baseDirectory string
}

// WithBaseDirectory sets the base directory used to resolve relative flake references.
func WithBaseDirectory(path string) ParseOption {
	return func(c *parseConfig) {
		c.baseDirectory = path
	}
}
