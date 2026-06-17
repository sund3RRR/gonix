// Package storepath wraps Nix store path handles.
package storepath

import (
	"fmt"

	"github.com/sund3RRR/gonix/internal/status"
	"github.com/sund3RRR/gonix/internal/utils"
	nix "github.com/sund3RRR/nix-go-bindings"
)

// Path owns a Nix store path handle.
//
// The Nix context passed to New or FromParts must remain valid for as long as
// the Path is used.
//
// A Path must be closed when the caller is done with it. Close is idempotent,
// but methods that need the raw Nix handle return an error wrapping
// gonix.ErrClosed after Close.
type Path struct {
	name string
	hash [20]byte
	ctx  *nix.NixCContext
	ptr  *nix.StorePath
}

// New wraps an owned raw Nix store path handle.
//
// The returned Path owns ptr and will release it with Close. Passing nil ptr
// creates a closed Path. Passing nil ctx is a caller error.
func New(ctx *nix.NixCContext, ptr *nix.StorePath) (*Path, error) {
	namePtr := nix.StorePathName(ptr)
	if namePtr == nil {
		return nil, fmt.Errorf("storepath: failed to get store path name: %w", status.FromContext(ctx))
	}
	name := utils.TakeCString(namePtr)

	var hash nix.StorePathHashPart
	defer hash.Free()

	if code := nix.StorePathHash(ctx, ptr, &hash); status.ErrorCode(code) != status.ErrorCodeOK {
		return nil, fmt.Errorf("storepath: failed to get store path hash: %w", status.FromContext(ctx))
	}

	hash.Deref()

	return &Path{
		name: name,
		hash: hash.Bytes,
		ctx:  ctx,
		ptr:  ptr,
	}, nil
}

// FromParts creates a store path from a raw 20-byte hash part and a name.
func FromParts(ctx *nix.NixCContext, hash [20]byte, name string) (*Path, error) {
	rawHash := nix.StorePathHashPart{Bytes: hash}
	defer rawHash.Free()

	ptr := nix.StoreCreateFromParts(ctx, &rawHash, name, uint64(len(name)))
	if ptr == nil {
		return nil, fmt.Errorf("storepath: failed to create store path from parts: %w", status.FromContext(ctx))
	}

	return &Path{
		name: name,
		hash: hash,
		ctx:  ctx,
		ptr:  ptr,
	}, nil
}

// Name returns the store path's human-readable name portion.
func (p *Path) Name() string {
	return p.name
}

// Hash returns the store path's 20-byte hash part.
func (p *Path) Hash() [20]byte {
	return p.hash
}

// Clone returns an independently owned copy of p.
func (p *Path) Clone() (*Path, error) {
	if p.ptr == nil {
		return nil, status.ErrClosed
	}

	clonePtr := nix.StorePathClone(p.ptr)
	if clonePtr == nil {
		return nil, fmt.Errorf("storepath: failed to clone store path: %w", status.FromContext(p.ctx))
	}

	path, err := New(p.ctx, clonePtr)
	if err != nil {
		nix.StorePathFree(clonePtr)
		return nil, fmt.Errorf("storepath: failed to create cloned store path: %w", err)
	}

	return path, nil
}

// Borrow returns the borrowed raw Nix store path handle.
//
// Callers must not free the returned pointer and must not retain it beyond the
// immediate raw Nix call that needs it.
func (p *Path) Borrow() (*nix.StorePath, error) {
	if p.ptr == nil {
		return nil, status.ErrClosed
	}

	return p.ptr, nil
}

// Close releases the owned Nix store path handle and is safe to call more than once.
func (p *Path) Close() error {
	if p.ptr == nil {
		return nil
	}

	nix.StorePathFree(p.ptr)
	p.ptr = nil

	return nil
}
