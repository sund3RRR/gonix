package eval

import (
	"errors"
	"fmt"

	"github.com/sund3RRR/gonix/internal/status"
	"github.com/sund3RRR/gonix/store"
	nix "github.com/sund3RRR/nix-go-bindings"
)

// Evaluator owns a Nix evaluation state.
//
// An Evaluator borrows the Store and Nix context used to create it. The store
// and context must outlive the evaluator. Evaluator is not goroutine-safe.
// Values returned by an Evaluator are tied to that evaluator and must not be
// used with another evaluator.
type Evaluator struct {
	ctx    *nix.NixCContext
	store  *store.Store
	state  *nix.EvalState
	values []*Value
}

// New creates an evaluator using an initialized Nix context and open store.
//
// The returned Evaluator owns the raw EvalState and borrows ctx and s. The
// caller must close the evaluator when finished.
func New(ctx *nix.NixCContext, s *store.Store, opts ...Option) (*Evaluator, error) {
	var cfg config
	for _, opt := range opts {
		opt(&cfg)
	}

	storePtr, err := s.Borrow()
	if err != nil {
		return nil, fmt.Errorf("eval: failed to borrow store: %w", err)
	}

	builder := nix.EvalStateBuilderNew(ctx, storePtr)
	if builder == nil {
		return nil, fmt.Errorf("eval: failed to create eval state builder: %w", status.FromContext(ctx))
	}
	defer nix.EvalStateBuilderFree(builder)

	lookupPath := stringArray(cfg.lookupPath)
	if code := nix.EvalStateBuilderSetLookupPath(ctx, builder, lookupPath); status.ErrorCode(code) != status.ErrorCodeOK {
		return nil, fmt.Errorf("eval: failed to set lookup path: %w", status.FromContext(ctx))
	}

	state := nix.EvalStateBuild(ctx, builder)
	if state == nil {
		return nil, fmt.Errorf("eval: failed to build eval state: %w", status.FromContext(ctx))
	}

	return &Evaluator{
		ctx:   ctx,
		store: s,
		state: state,
	}, nil
}

// Borrow returns the borrowed raw Nix evaluation state.
//
// Callers must not free the returned pointer and must not retain it beyond the
// immediate raw Nix call that needs it. This is an escape hatch for integration
// with lower-level bindings.
func (e *Evaluator) Borrow() (*nix.EvalState, error) {
	if e.state == nil {
		return nil, status.ErrClosed
	}

	return e.state, nil
}

// WrapValue adopts an owned raw Nix value as a Value tied to e.
//
// This is an integration point for sibling gonix packages that receive owned or
// refcounted values from lower-level Nix APIs. Callers transfer ownership of
// ptr to the returned Value and must not decref ptr directly after a successful
// call.
func (e *Evaluator) WrapValue(ptr *nix.NixValue) (*Value, error) {
	if e.state == nil {
		return nil, status.ErrClosed
	}

	value := &Value{
		ctx:   e.ctx,
		ptr:   ptr,
		owner: e,
	}
	e.values = append(e.values, value)

	return value, nil
}

// Close releases values created by e and then releases the owned EvalState.
//
// Close is safe to call more than once. Once Close returns, methods that need
// the raw evaluation state report status.ErrClosed.
func (e *Evaluator) Close() error {
	if e.state == nil {
		return nil
	}

	errs := make([]error, 0, len(e.values))
	for i := len(e.values) - 1; i >= 0; i-- {
		if err := e.values[i].Close(); err != nil {
			errs = append(errs, err)
		}
	}

	nix.StateFree(e.state)
	e.state = nil
	e.values = nil
	e.store = nil

	if len(errs) != 0 {
		return fmt.Errorf("eval: failed to close evaluator: %w", errors.Join(errs...))
	}

	return nil
}

func (e *Evaluator) allocValue() (*Value, error) {
	if e.state == nil {
		return nil, status.ErrClosed
	}

	ptr := nix.AllocValue(e.ctx, e.state)
	if ptr == nil {
		return nil, fmt.Errorf("eval: failed to allocate value: %w", status.FromContext(e.ctx))
	}

	value := &Value{
		ctx:   e.ctx,
		ptr:   ptr,
		owner: e,
	}
	e.values = append(e.values, value)

	return value, nil
}

func (e *Evaluator) validateValue(v *Value) error {
	if v.ptr == nil {
		return fmt.Errorf("eval: value is closed")
	}

	if v.owner != e {
		return fmt.Errorf("eval: value belongs to a different evaluator")
	}

	return nil
}

func stringArray(values []string) nix.StringArray {
	items := make([]nix.StringItem, len(values))
	for i, value := range values {
		items[i] = nix.StringItem{
			Value: []byte(value),
			Len:   uint64(len(value)),
		}
	}

	return nix.StringArray{
		Items: items,
		Len:   uint64(len(items)),
	}
}
