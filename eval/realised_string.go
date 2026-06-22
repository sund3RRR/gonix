package eval

import (
	"errors"
	"fmt"
	"unsafe"

	"github.com/sund3RRR/gonix/internal/status"
	"github.com/sund3RRR/gonix/pkg/raw"
	"github.com/sund3RRR/gonix/storepath"
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
	if r == nil || len(r.Paths) == 0 {
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

	rawCtx, err := e.ctx.Borrow()
	if err != nil {
		return nil, fmt.Errorf("eval: failed to borrow context: %w", err)
	}

	if err := e.validateValue(v); err != nil {
		return nil, fmt.Errorf("eval: failed to validate value: %w", err)
	}

	realised := raw.StringRealise(rawCtx, e.state, v.ptr, false)
	if realised == nil {
		return nil, fmt.Errorf("eval: failed to realise string: %w", status.FromContext(rawCtx))
	}
	defer raw.RealisedStringFree(realised)

	valuePtr := raw.RealisedStringGetBuffer(realised)
	if valuePtr == nil {
		return nil, fmt.Errorf("eval: failed to get realised string buffer: %w", status.FromContext(rawCtx))
	}
	size := raw.RealisedStringGetBufferSize(realised)
	value := string(unsafe.Slice(valuePtr, size))
	raw.StringFree(valuePtr)

	count := raw.RealisedStringGetStorePathCount(realised)
	out := &RealisedString{
		Value: value,
		Paths: make([]*storepath.Path, 0, count),
	}

	for i := range count {
		pathPtr := raw.RealisedStringGetStorePath(realised, i)
		if pathPtr == nil {
			_ = out.Close()
			return nil, fmt.Errorf("eval: failed to clone realised string path %d: %w", i, status.FromContext(rawCtx))
		}
		path, err := storepath.New(e.ctx, pathPtr)
		if err != nil {
			_ = out.Close()
			return nil, fmt.Errorf("eval: failed to clone realised string path %d: %w", i, err)
		}
		out.Paths = append(out.Paths, path)
	}

	return out, nil
}
