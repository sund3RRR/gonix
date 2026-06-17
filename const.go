package gonix

import "github.com/sund3RRR/gonix/store"

const (
	// DefaultStoreDir is the default local Nix store directory.
	DefaultStoreDir = store.DefaultDir
	// AutoStore lets Nix choose the appropriate store backend.
	AutoStore = "auto"
	// LocalStore identifies the local store backend.
	LocalStore = "local"
	// DaemonStore identifies the local Nix daemon store backend.
	DaemonStore = "daemon"
	// DummyStore identifies the dummy store backend.
	DummyStore = "dummy://"
	// UnixStore identifies a Unix-domain socket daemon store.
	UnixStore = "unix://"
	// FileStore identifies a file-backed binary cache store.
	FileStore = "file://"
	// HTTPStore identifies an HTTP binary cache store.
	HTTPStore = "http://"
	// HTTPSStore identifies an HTTPS binary cache store.
	HTTPSStore = "https://"
	// SSHStore identifies the legacy SSH store backend.
	SSHStore = "ssh://"
	// SSHNGStore identifies the SSH-NG store backend.
	SSHNGStore = "ssh-ng://"
	// MountedSSHStore identifies a mounted SSH store backend.
	MountedSSHStore = "mounted-ssh://"
	// S3Store identifies an S3 binary cache store.
	S3Store = "s3://"
)
