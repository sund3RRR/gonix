package eval

import (
	"fmt"

	"github.com/sund3RRR/gonix/internal/status"
	"github.com/sund3RRR/gonix/internal/utils"
	nix "github.com/sund3RRR/nix-go-bindings"
)

// EvalString evaluates a Nix expression string at path.
//
// The path is used by Nix for diagnostics and relative path resolution. The
// returned Value is tied to e and is closed by e.Close if the caller has not
// already closed it.
func (e *Evaluator) EvalString(expr, path string) (*Value, error) {
	if e.state == nil {
		return nil, status.ErrClosed
	}

	value, err := e.allocValue()
	if err != nil {
		return nil, err
	}

	if code := nix.ExprEvalFromString(e.ctx, e.state, expr, path, value.ptr); status.ErrorCode(code) != status.ErrorCodeOK {
		_ = value.Close()
		return nil, fmt.Errorf("eval: failed to evaluate string: %w", status.FromContext(e.ctx))
	}

	return value, nil
}

// NewValue creates a Nix value from a typed GoValue.
func (e *Evaluator) NewValue(gv GoValue) (*Value, error) {
	if e.state == nil {
		return nil, status.ErrClosed
	}

	value, err := e.allocValue()
	if err != nil {
		return nil, err
	}

	if err := gv(e, value.ptr); err != nil {
		_ = value.Close()
		return nil, err
	}

	return value, nil
}

// Force evaluates v to weak head normal form.
func (e *Evaluator) Force(v *Value) error {
	if e.state == nil {
		return status.ErrClosed
	}

	if err := e.validateValue(v); err != nil {
		return fmt.Errorf("eval: failed to validate value: %w", err)
	}

	if code := nix.ValueForce(e.ctx, e.state, v.ptr); status.ErrorCode(code) != status.ErrorCodeOK {
		return fmt.Errorf("eval: failed to force value: %w", status.FromContext(e.ctx))
	}

	return nil
}

// ForceDeep recursively evaluates v.
func (e *Evaluator) ForceDeep(v *Value) error {
	if e.state == nil {
		return status.ErrClosed
	}

	if err := e.validateValue(v); err != nil {
		return fmt.Errorf("eval: failed to validate value: %w", err)
	}

	if code := nix.ValueForceDeep(e.ctx, e.state, v.ptr); status.ErrorCode(code) != status.ErrorCodeOK {
		return fmt.Errorf("eval: failed to force value deeply: %w", status.FromContext(e.ctx))
	}

	return nil
}

// Call applies fn to arg and returns the result value.
func (e *Evaluator) Call(fn, arg *Value) (*Value, error) {
	if e.state == nil {
		return nil, status.ErrClosed
	}

	if err := e.validateValue(fn); err != nil {
		return nil, fmt.Errorf("eval: failed to validate function value: %w", err)
	}

	if err := e.validateValue(arg); err != nil {
		return nil, fmt.Errorf("eval: failed to validate argument value: %w", err)
	}

	out, err := e.allocValue()
	if err != nil {
		return nil, err
	}
	if code := nix.ValueCall(e.ctx, e.state, fn.ptr, arg.ptr, out.ptr); status.ErrorCode(code) != status.ErrorCodeOK {
		_ = out.Close()
		return nil, fmt.Errorf("eval: failed to call function: %w", status.FromContext(e.ctx))
	}

	return out, nil
}

// CallMulti applies fn to args and returns the result value.
func (e *Evaluator) CallMulti(fn *Value, args ...*Value) (*Value, error) {
	if e.state == nil {
		return nil, status.ErrClosed
	}

	if err := e.validateValue(fn); err != nil {
		return nil, fmt.Errorf("eval: failed to validate function value: %w", err)
	}

	items := make([]nix.ValueItem, len(args))
	for i, arg := range args {
		if err := e.validateValue(arg); err != nil {
			return nil, fmt.Errorf("eval: failed to validate argument value: %w", err)
		}
		items[i] = nix.ValueItem{Value: arg.ptr}
	}

	out, err := e.allocValue()
	if err != nil {
		return nil, err
	}
	valueArray := nix.ValueArray{
		Items: items,
		Len:   uint64(len(items)),
	}
	if code := nix.ValueCallMulti(e.ctx, e.state, fn.ptr, valueArray, out.ptr); status.ErrorCode(code) != status.ErrorCodeOK {
		_ = out.Close()
		return nil, fmt.Errorf("eval: failed to call function with arguments: %w", status.FromContext(e.ctx))
	}

	return out, nil
}

// Index returns the forced list item at index.
func (e *Evaluator) Index(v *Value, index uint32) (*Value, error) {
	if e.state == nil {
		return nil, status.ErrClosed
	}

	if err := e.validateValue(v); err != nil {
		return nil, fmt.Errorf("eval: failed to validate value: %w", err)
	}

	child := nix.GetListByidx(e.ctx, v.ptr, e.state, index)
	if child == nil {
		return nil, fmt.Errorf("eval: failed to get list item %d: %w", index, status.FromContext(e.ctx))
	}

	return e.wrapValue(child), nil
}

// Attr returns the forced attribute named name.
func (e *Evaluator) Attr(v *Value, name string) (*Value, error) {
	if e.state == nil {
		return nil, status.ErrClosed
	}

	if err := e.validateValue(v); err != nil {
		return nil, fmt.Errorf("eval: failed to validate value: %w", err)
	}

	child := nix.GetAttrByname(e.ctx, v.ptr, e.state, name)
	if child == nil {
		return nil, fmt.Errorf("eval: failed to get attr %q: %w", name, status.FromContext(e.ctx))
	}

	return e.wrapValue(child), nil
}

// AttrByIndex returns the forced attribute value at index.
func (e *Evaluator) AttrByIndex(v *Value, index uint32) (*Value, error) {
	if e.state == nil {
		return nil, status.ErrClosed
	}

	if err := e.validateValue(v); err != nil {
		return nil, fmt.Errorf("eval: failed to validate value: %w", err)
	}

	child := nix.GetAttrByidx(e.ctx, v.ptr, e.state, index)
	if child == nil {
		return nil, fmt.Errorf("eval: failed to get attr by index %d: %w", index, status.FromContext(e.ctx))
	}

	return e.wrapValue(child), nil
}

// AttrName returns the attribute name at index.
func (e *Evaluator) AttrName(v *Value, index uint32) (string, error) {
	if e.state == nil {
		return "", status.ErrClosed
	}

	if err := e.validateValue(v); err != nil {
		return "", fmt.Errorf("eval: failed to validate value: %w", err)
	}

	name := nix.GetAttrNameByidx(e.ctx, v.ptr, e.state, index)
	if name == nil {
		return "", fmt.Errorf("eval: failed to get attr name by index %d: %w", index, status.FromContext(e.ctx))
	}

	return utils.TakeCString(name), nil
}

// HasAttr reports whether v has an attribute named name.
func (e *Evaluator) HasAttr(v *Value, name string) (bool, error) {
	if e.state == nil {
		return false, status.ErrClosed
	}

	if err := e.validateValue(v); err != nil {
		return false, fmt.Errorf("eval: failed to validate value: %w", err)
	}

	got := nix.HasAttrByname(e.ctx, v.ptr, e.state, name)
	if err := status.FromContext(e.ctx); err != nil {
		return false, fmt.Errorf("eval: failed to check attr %q: %w", name, err)
	}

	return got, nil
}

func (e *Evaluator) wrapValue(ptr *nix.NixValue) *Value {
	value := &Value{
		ctx:   e.ctx,
		ptr:   ptr,
		owner: e,
	}
	e.values = append(e.values, value)

	return value
}
