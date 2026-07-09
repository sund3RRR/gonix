package eval

import (
	"fmt"

	"github.com/sund3RRR/gonix/internal/status"
	"github.com/sund3RRR/gonix/pkg/raw"
	"github.com/sund3RRR/gonix/pkg/utils"
)

// EvalString evaluates a Nix expression string at path.
//
// The path is used by Nix for diagnostics and relative path resolution. The
// returned Value is owned by the caller and must be closed.
func (e *Evaluator) EvalString(expr, path string) (*Value, error) {
	if e.state == nil {
		return nil, status.ErrClosed
	}

	rawCtx, err := e.ctx.Borrow()
	if err != nil {
		return nil, fmt.Errorf("eval: failed to borrow context: %w", err)
	}

	value, err := e.allocValue()
	if err != nil {
		return nil, err
	}

	if code := raw.ExprEvalFromString(rawCtx, e.state, expr, path, value.ptr); status.ErrorCode(code) != status.ErrorCodeOK {
		_ = value.Close()
		return nil, fmt.Errorf("eval: failed to evaluate string: %w", status.FromContext(rawCtx))
	}

	return value, nil
}

// EvalWithArgs evaluates expr as a function and applies arg.
//
// Arg may be nil, a boolean, string, integer, float, slice, or map with string
// keys. Slices and maps may contain nested combinations of those types. The
// path is used by Nix for diagnostics and relative path resolution. The
// returned Value is owned by the caller and must be closed.
func (e *Evaluator) EvalWithArgs(expr, path string, arg any) (*Value, error) {
	fn, err := e.EvalString(expr, path)
	if err != nil {
		return nil, fmt.Errorf("eval: failed to evaluate function: %w", err)
	}
	defer fn.Close() //nolint:errcheck

	goValue, err := goValueFromAny(arg, "$")
	if err != nil {
		return nil, fmt.Errorf("eval: failed to convert function argument: %w", err)
	}

	argValue, err := e.NewValue(goValue)
	if err != nil {
		return nil, fmt.Errorf("eval: failed to build function argument: %w", err)
	}
	defer argValue.Close() //nolint:errcheck

	result, err := e.Call(fn, argValue)
	if err != nil {
		return nil, fmt.Errorf("eval: failed to call function: %w", err)
	}

	return result, nil
}

// NewValue creates a caller-owned Nix value from a typed GoValue.
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

	rawCtx, err := e.ctx.Borrow()
	if err != nil {
		return fmt.Errorf("eval: failed to borrow context: %w", err)
	}

	if err := e.validateValue(v); err != nil {
		return fmt.Errorf("eval: failed to validate value: %w", err)
	}

	if code := raw.ValueForce(rawCtx, e.state, v.ptr); status.ErrorCode(code) != status.ErrorCodeOK {
		return fmt.Errorf("eval: failed to force value: %w", status.FromContext(rawCtx))
	}

	return nil
}

// ForceDeep recursively evaluates v.
func (e *Evaluator) ForceDeep(v *Value) error {
	if e.state == nil {
		return status.ErrClosed
	}

	rawCtx, err := e.ctx.Borrow()
	if err != nil {
		return fmt.Errorf("eval: failed to borrow context: %w", err)
	}

	if err := e.validateValue(v); err != nil {
		return fmt.Errorf("eval: failed to validate value: %w", err)
	}

	if code := raw.ValueForceDeep(rawCtx, e.state, v.ptr); status.ErrorCode(code) != status.ErrorCodeOK {
		return fmt.Errorf("eval: failed to force value deeply: %w", status.FromContext(rawCtx))
	}

	return nil
}

// Call applies fn to arg and returns a caller-owned result value.
func (e *Evaluator) Call(fn, arg *Value) (*Value, error) {
	if e.state == nil {
		return nil, status.ErrClosed
	}

	rawCtx, err := e.ctx.Borrow()
	if err != nil {
		return nil, fmt.Errorf("eval: failed to borrow context: %w", err)
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

	if code := raw.ValueCall(rawCtx, e.state, fn.ptr, arg.ptr, out.ptr); status.ErrorCode(code) != status.ErrorCodeOK {
		_ = out.Close()
		return nil, fmt.Errorf("eval: failed to call function: %w", status.FromContext(rawCtx))
	}

	return out, nil
}

// CallMulti applies fn to args and returns a caller-owned result value.
func (e *Evaluator) CallMulti(fn *Value, args ...*Value) (*Value, error) {
	if e.state == nil {
		return nil, status.ErrClosed
	}

	rawCtx, err := e.ctx.Borrow()
	if err != nil {
		return nil, fmt.Errorf("eval: failed to borrow context: %w", err)
	}

	if err := e.validateValue(fn); err != nil {
		return nil, fmt.Errorf("eval: failed to validate function value: %w", err)
	}

	items := make([]raw.ValueItem, len(args))
	for i, arg := range args {
		if err := e.validateValue(arg); err != nil {
			return nil, fmt.Errorf("eval: failed to validate argument value: %w", err)
		}
		items[i] = raw.ValueItem{Value: arg.ptr}
	}

	out, err := e.allocValue()
	if err != nil {
		return nil, err
	}

	valueArray := raw.ValueArray{
		Items: items,
		Len:   uint64(len(items)),
	}

	if code := raw.ValueCallMulti(rawCtx, e.state, fn.ptr, valueArray, out.ptr); status.ErrorCode(code) != status.ErrorCodeOK {
		_ = out.Close()
		return nil, fmt.Errorf("eval: failed to call function with arguments: %w", status.FromContext(rawCtx))
	}

	return out, nil
}

// Index returns the caller-owned forced list item at index.
func (e *Evaluator) Index(v *Value, index uint32) (*Value, error) {
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

	child := raw.GetListByidx(rawCtx, v.ptr, e.state, index)
	if child == nil {
		return nil, fmt.Errorf("eval: failed to get list item %d: %w", index, status.FromContext(rawCtx))
	}

	value, err := e.WrapValue(child)
	if err != nil {
		return nil, fmt.Errorf("eval: failed to wrap value: %w", err)
	}

	return value, nil
}

// Attr returns the caller-owned forced attribute named name.
func (e *Evaluator) Attr(v *Value, name string) (*Value, error) {
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

	child := raw.GetAttrByname(rawCtx, v.ptr, e.state, name)
	if child == nil {
		return nil, fmt.Errorf("eval: failed to get attr %q: %w", name, status.FromContext(rawCtx))
	}

	value, err := e.WrapValue(child)
	if err != nil {
		return nil, fmt.Errorf("eval: failed to wrap value: %w", err)
	}

	return value, nil
}

// AttrLazy returns a caller-owned attribute without forcing its value.
func (e *Evaluator) AttrLazy(v *Value, name string) (*Value, error) {
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

	if err := e.Force(v); err != nil {
		return nil, fmt.Errorf("eval: failed to force parent value: %w", err)
	}
	typ, err := v.Type()
	if err != nil {
		return nil, fmt.Errorf("eval: failed to get parent value type: %w", err)
	}
	if typ != ValueTypeAttrs {
		return nil, &ValueTypeError{Actual: typ, Expected: ValueTypeAttrs}
	}

	child := raw.GetAttrBynameLazy(rawCtx, v.ptr, e.state, name)
	if child == nil {
		return nil, fmt.Errorf("eval: failed to get lazy attr %q: %w", name, status.FromContext(rawCtx))
	}

	value, err := e.WrapValue(child)
	if err != nil {
		if code := raw.ValueDecref(rawCtx, child); status.ErrorCode(code) != status.ErrorCodeOK {
			return nil, fmt.Errorf("eval: failed to wrap lazy attr %q and decref value: %w", name, status.FromContext(rawCtx))
		}
		return nil, fmt.Errorf("eval: failed to wrap lazy attr %q: %w", name, err)
	}

	return value, nil
}

// AttrByIndex returns the caller-owned forced attribute value at index.
func (e *Evaluator) AttrByIndex(v *Value, index uint32) (*Value, error) {
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

	child := raw.GetAttrByidx(rawCtx, v.ptr, e.state, index)
	if child == nil {
		return nil, fmt.Errorf("eval: failed to get attr by index %d: %w", index, status.FromContext(rawCtx))
	}

	value, err := e.WrapValue(child)
	if err != nil {
		return nil, fmt.Errorf("eval: failed to wrap value: %w", err)
	}

	return value, nil
}

// AttrName returns the attribute name at index.
func (e *Evaluator) AttrName(v *Value, index uint32) (string, error) {
	if e.state == nil {
		return "", status.ErrClosed
	}

	rawCtx, err := e.ctx.Borrow()
	if err != nil {
		return "", fmt.Errorf("eval: failed to borrow context: %w", err)
	}

	if err := e.validateValue(v); err != nil {
		return "", fmt.Errorf("eval: failed to validate value: %w", err)
	}

	name := raw.GetAttrNameByidx(rawCtx, v.ptr, e.state, index)
	if name == nil {
		return "", fmt.Errorf("eval: failed to get attr name by index %d: %w", index, status.FromContext(rawCtx))
	}

	return utils.TakeCString(name), nil
}

// HasAttr reports whether v has an attribute named name.
func (e *Evaluator) HasAttr(v *Value, name string) (bool, error) {
	if e.state == nil {
		return false, status.ErrClosed
	}

	rawCtx, err := e.ctx.Borrow()
	if err != nil {
		return false, fmt.Errorf("eval: failed to borrow context: %w", err)
	}

	if err := e.validateValue(v); err != nil {
		return false, fmt.Errorf("eval: failed to validate value: %w", err)
	}

	got := raw.HasAttrByname(rawCtx, v.ptr, e.state, name)
	if err := status.FromContext(rawCtx); err != nil {
		return false, fmt.Errorf("eval: failed to check attr %q: %w", name, err)
	}

	return got, nil
}
