package eval

import (
	"errors"
	"fmt"
	"unsafe"

	"github.com/sund3RRR/gonix/internal/status"
	"github.com/sund3RRR/gonix/storepath"
	nix "github.com/sund3RRR/nix-go-bindings"
)

// RealisedString is a Nix string with its referenced store paths.
//
// RealiseString returns Go-owned string data and owned store path clones. Call
// Close when finished to release the paths.
type RealisedString struct {
	// Value is the realized string content.
	Value string

	// Paths are owned store paths referenced by Value.
	Paths []*storepath.Path
}

// Close releases every referenced store path.
//
// Close attempts to close every path and returns a joined error if one or more
// closes fail. It is safe to call more than once.
func (r *RealisedString) Close() error {
	if len(r.Paths) == 0 {
		return nil
	}

	errs := make([]error, 0, len(r.Paths))
	for _, path := range r.Paths {
		if err := path.Close(); err != nil {
			errs = append(errs, err)
		}
	}
	r.Paths = nil

	if len(errs) != 0 {
		return fmt.Errorf("eval: failed to close realised string paths: %w", errors.Join(errs...))
	}

	return nil
}

// RealiseString realizes v as a string and returns referenced store paths.
func (e *Evaluator) RealiseString(v *Value) (*RealisedString, error) {
	if e.state == nil {
		return nil, status.ErrClosed
	}

	if err := e.validateValue(v); err != nil {
		return nil, fmt.Errorf("eval: failed to validate value: %w", err)
	}

	realised := nix.StringRealise(e.ctx, e.state, v.ptr, false)
	if realised == nil {
		return nil, fmt.Errorf("eval: failed to realise string: %w", status.FromContext(e.ctx))
	}
	defer nix.RealisedStringFree(realised)

	valuePtr := nix.RealisedStringGetBuffer(realised)
	if valuePtr == nil {
		return nil, fmt.Errorf("eval: failed to get realised string buffer: %w", status.FromContext(e.ctx))
	}
	size := nix.RealisedStringGetBufferSize(realised)
	value := string(unsafe.Slice(valuePtr, size))
	nix.StringFree(valuePtr)

	count := nix.RealisedStringGetStorePathCount(realised)
	out := &RealisedString{
		Value: value,
		Paths: make([]*storepath.Path, 0, count),
	}

	for i := range count {
		pathPtr := nix.RealisedStringGetStorePath(realised, i)
		if pathPtr == nil {
			_ = out.Close()
			return nil, fmt.Errorf("eval: failed to clone realised string path %d: %w", i, status.FromContext(e.ctx))
		}
		out.Paths = append(out.Paths, storepath.New(e.ctx, pathPtr))
	}

	return out, nil
}
