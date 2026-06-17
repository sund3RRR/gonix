package gonix

import "github.com/sund3RRR/gonix/store"

// ClientOption configures Client creation.
type ClientOption func(*clientConfig)

type clientConfig struct {
	store *store.Store
}

// WithClientStore configures the store opened for the client evaluator.
func WithClientStore(s *store.Store) ClientOption {
	return func(c *clientConfig) {
		c.store = s
	}
}

// FetchPackageOption configures FetchPackage.
type FetchPackageOption func(*fetchPackageConfig)

type fetchPackageConfig struct {
	system string
}

// WithFetchPackageSystem overrides the package system for one FetchPackage call.
func WithFetchPackageSystem(system SystemIdent) FetchPackageOption {
	return func(c *fetchPackageConfig) {
		c.system = string(system)
	}
}
