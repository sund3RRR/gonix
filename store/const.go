package store

const (
	// DefaultDir is the default local Nix store directory.
	DefaultDir = "/nix/store"
	// Auto lets Nix choose the appropriate store backend.
	Auto = "auto"
	// Local identifies the local store backend.
	Local = "local"
	// Daemon identifies the local Nix daemon store backend.
	Daemon = "daemon"
	// Dummy identifies the dummy store backend.
	Dummy = "dummy://"
	// Unix identifies a Unix-domain socket daemon store.
	Unix = "unix://"
	// File identifies a file-backed binary cache store.
	File = "file://"
	// HTTP identifies an HTTP binary cache store.
	HTTP = "http://"
	// HTTPS identifies an HTTPS binary cache store.
	HTTPS = "https://"
	// SSH identifies the legacy SSH store backend.
	SSH = "ssh://"
	// SSHNG identifies the SSH-NG store backend.
	SSHNG = "ssh-ng://"
	// MountedSSH identifies a mounted SSH store backend.
	MountedSSH = "mounted-ssh://"
	// S3 identifies an S3 binary cache store.
	S3 = "s3://"
)
