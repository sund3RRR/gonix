package store

import (
	"fmt"

	"github.com/sund3RRR/gonix/internal/status"
	"github.com/sund3RRR/gonix/storepath"
	nix "github.com/sund3RRR/nix-go-bindings"
)

// DerivationFromJSON imports a Nix derivation from JSON.
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

// AddDerivation adds d to this store and returns its store path.
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

	return storepath.New(s.ctx, ptr), nil
}
