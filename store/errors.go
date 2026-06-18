package store

import "errors"

// ErrPathNotFound reports that a store has no path matching a hash part.
var ErrPathNotFound = errors.New("store path not found")
