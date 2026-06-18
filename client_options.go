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
func WithFetchPackageSystem(system string) FetchPackageOption {
	return func(c *fetchPackageConfig) {
		c.system = system
	}
}

// DownloadPackageOption configures DownloadPackage.
type DownloadPackageOption func(*downloadPackageConfig)

type downloadPackageConfig struct {
	system string
}

// WithDownloadPackageSystem overrides the package system for one DownloadPackage call.
func WithDownloadPackageSystem(system string) DownloadPackageOption {
	return func(c *downloadPackageConfig) {
		c.system = system
	}
}
