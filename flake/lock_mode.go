package flake

// LockMode selects how Nix should handle a flake lock file.
type LockMode int

const (
	// LockModeVirtual resolves the lock in memory without writing flake.lock.
	LockModeVirtual LockMode = iota
	// LockModeCheck fails unless the existing lock file is already usable.
	LockModeCheck
	// LockModeWriteAsNeeded creates or updates flake.lock when needed.
	LockModeWriteAsNeeded
)
