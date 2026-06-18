package eval

import (
	"fmt"

	"github.com/sund3RRR/gonix/internal/status"
	"github.com/sund3RRR/gonix/nixcontext"
	"github.com/sund3RRR/gonix/store"
	nix "github.com/sund3RRR/nix-go-bindings"
)

// Evaluator owns a Nix evaluation state.
//
// An Evaluator borrows the Store and Nix context used to create it. The store
// and context must outlive the evaluator. Evaluator is not goroutine-safe.
// Values returned by an Evaluator are caller-owned, tied to that evaluator,
// and must not be used with another evaluator. Callers should close every
// Value before closing the Evaluator.
type Evaluator struct {
	ctx   *nixcontext.Context
	store *store.Store
	state *nix.EvalState
}

// New creates an evaluator using an initialized Nix context and open store.
//
// The returned Evaluator owns the raw EvalState and borrows ctx and s. The
// caller must close the evaluator when finished.
func New(ctx *nixcontext.Context, s *store.Store, opts ...Option) (*Evaluator, error) {
	rawCtx, err := ctx.Borrow()
	if err != nil {
		return nil, fmt.Errorf("eval: failed to borrow context: %w", err)
	}

	var cfg config
	for _, opt := range opts {
		opt(&cfg)
	}

	storePtr, err := s.Borrow()
	if err != nil {
		return nil, fmt.Errorf("eval: failed to borrow store: %w", err)
	}

	builder := nix.EvalStateBuilderNew(rawCtx, storePtr)
	if builder == nil {
		return nil, fmt.Errorf("eval: failed to create eval state builder: %w", status.FromContext(rawCtx))
	}
	defer nix.EvalStateBuilderFree(builder)

	lookupPath := stringArray(cfg.lookupPath)
	if code := nix.EvalStateBuilderSetLookupPath(rawCtx, builder, lookupPath); status.ErrorCode(code) != status.ErrorCodeOK {
		return nil, fmt.Errorf("eval: failed to set lookup path: %w", status.FromContext(rawCtx))
	}

	if cfg.flakesettings != nil {
		flakesettingsPtr, err := cfg.flakesettings.Borrow()
		if err != nil {
			return nil, fmt.Errorf("eval: failed to borrow flake settings: %w", err)
		}

		if code := nix.FlakeSettingsAddToEvalStateBuilder(rawCtx, flakesettingsPtr, builder); status.ErrorCode(code) != status.ErrorCodeOK {
			return nil, fmt.Errorf("eval: failed to add flake settings: %w", status.FromContext(rawCtx))
		}
	}

	state := nix.EvalStateBuild(rawCtx, builder)
	if state == nil {
		return nil, fmt.Errorf("eval: failed to build eval state: %w", status.FromContext(rawCtx))
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
// call. The returned Value must be closed by the caller.
func (e *Evaluator) WrapValue(ptr *nix.NixValue) (*Value, error) {
	if e.state == nil {
		return nil, status.ErrClosed
	}

	value := &Value{
		ctx:   e.ctx,
		ptr:   ptr,
		owner: e,
	}

	return value, nil
}

// Close releases the owned EvalState.
//
// Close is safe to call more than once. Once Close returns, methods that need
// the raw evaluation state and operations on Values created by e report
// status.ErrClosed. Caller-owned Values may still be closed while their Context
// remains open.
func (e *Evaluator) Close() error {
	if e == nil || e.state == nil {
		return nil
	}

	nix.StateFree(e.state)
	e.state = nil
	e.store = nil
	e.ctx = nil

	return nil
}

func (e *Evaluator) allocValue() (*Value, error) {
	if e.state == nil {
		return nil, status.ErrClosed
	}

	rawCtx, err := e.ctx.Borrow()
	if err != nil {
		return nil, fmt.Errorf("eval: failed to borrow context: %w", err)
	}

	ptr := nix.AllocValue(rawCtx, e.state)
	if ptr == nil {
		return nil, fmt.Errorf("eval: failed to allocate value: %w", status.FromContext(rawCtx))
	}

	value := &Value{
		ctx:   e.ctx,
		ptr:   ptr,
		owner: e,
	}

	return value, nil
}

func (e *Evaluator) validateValue(v *Value) error {
	if e.state == nil {
		return status.ErrClosed
	}

	if v == nil {
		return fmt.Errorf("eval: value is nil")
	}

	if v.ptr == nil {
		return status.ErrClosed
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
