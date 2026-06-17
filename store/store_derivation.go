package store

import (
	"fmt"

	"github.com/sund3RRR/gonix/internal/status"
	"github.com/sund3RRR/gonix/storepath"
	nix "github.com/sund3RRR/nix-go-bindings"
)

// DerivationFromJSON imports a Nix derivation from JSON.
//
// The JSON must use Nix's derivation JSON shape. Nix validates the derivation,
// including output paths, while importing it. The returned Derivation owns its
// raw handle and must be closed by the caller.
func (s *Store) DerivationFromJSON(data []byte) (*Derivation, error) {
	if s.ptr == nil {
		return nil, status.ErrClosed
	}

	ptr := nix.DerivationFromJson(s.ctx, s.ptr, string(data))
	if ptr == nil {
		return nil, fmt.Errorf("store: failed to import derivation from json: %w", status.FromContext(s.ctx))
	}

	return NewDerivation(s.ctx, ptr), nil
}

// DerivationFromPath loads a derivation from a store path.
//
// The path must refer to a derivation known to this store. The returned
// Derivation owns its raw handle and must be closed by the caller.
func (s *Store) DerivationFromPath(path *storepath.Path) (*Derivation, error) {
	if s.ptr == nil {
		return nil, status.ErrClosed
	}

	pathPtr, err := path.Borrow()
	if err != nil {
		return nil, fmt.Errorf("store: failed to borrow path: %w", err)
	}

	ptr := nix.StoreDrvFromStorePath(s.ctx, s.ptr, pathPtr)
	if ptr == nil {
		return nil, fmt.Errorf("store: failed to load derivation from path: %w", status.FromContext(s.ctx))
	}

	return NewDerivation(s.ctx, ptr), nil
}

// AddDerivation adds d to this store and returns its derivation store path.
//
// The derivation is borrowed for the duration of the call; ownership of d stays
// with the caller. The returned StorePath is owned by the caller.
func (s *Store) AddDerivation(d *Derivation) (*storepath.Path, error) {
	if s.ptr == nil {
		return nil, status.ErrClosed
	}

	drvPtr, err := d.Borrow()
	if err != nil {
		return nil, fmt.Errorf("store: failed to borrow derivation: %w", err)
	}

	ptr := nix.AddDerivation(s.ctx, s.ptr, drvPtr)
	if ptr == nil {
		return nil, fmt.Errorf("store: failed to add derivation: %w", status.FromContext(s.ctx))
	}

	path, err := storepath.New(s.ctx, ptr)
	if err != nil {
		nix.StorePathFree(ptr)
		return nil, fmt.Errorf("store: failed to create store path: %w", err)
	}

	return path, nil
}
