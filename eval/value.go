package eval

import (
	"fmt"

	"github.com/sund3RRR/gonix/internal/status"
	"github.com/sund3RRR/gonix/nixcontext"
	"github.com/sund3RRR/gonix/pkg/raw"
	"github.com/sund3RRR/gonix/pkg/utils"
)

// ValueType identifies a Nix value's runtime type.
type ValueType int

const (
	// ValueTypeThunk is a deferred Nix expression.
	ValueTypeThunk ValueType = iota
	// ValueTypeInt is a Nix integer.
	ValueTypeInt
	// ValueTypeFloat is a Nix floating-point number.
	ValueTypeFloat
	// ValueTypeBool is a Nix boolean.
	ValueTypeBool
	// ValueTypeString is a Nix string.
	ValueTypeString
	// ValueTypePath is a Nix path.
	ValueTypePath
	// ValueTypeNull is Nix null.
	ValueTypeNull
	// ValueTypeAttrs is a Nix attribute set.
	ValueTypeAttrs
	// ValueTypeList is a Nix list.
	ValueTypeList
	// ValueTypeFunction is a Nix function.
	ValueTypeFunction
	// ValueTypeExternal is an external value.
	ValueTypeExternal
	// ValueTypeFailed is a failed value.
	ValueTypeFailed
)

func (vt ValueType) String() string {
	switch vt {
	case ValueTypeThunk:
		return "thunk"
	case ValueTypeInt:
		return "int"
	case ValueTypeFloat:
		return "float"
	case ValueTypeBool:
		return "bool"
	case ValueTypeString:
		return "string"
	case ValueTypePath:
		return "path"
	case ValueTypeNull:
		return "null"
	case ValueTypeAttrs:
		return "attrs"
	case ValueTypeList:
		return "list"
	case ValueTypeFunction:
		return "function"
	case ValueTypeExternal:
		return "external"
	case ValueTypeFailed:
		return "failed"
	default:
		return ""
	}
}

// Value owns a reference to a Nix value.
//
// Values are tied to the Evaluator that created them and must be closed by the
// caller before the Context. Value operations require the Evaluator to remain
// open. Close releases the owned value reference and is idempotent.
type Value struct {
	ctx   *nixcontext.Context
	ptr   *raw.NixValue
	owner *Evaluator
}

// Type returns the Nix value type.
func (v *Value) Type() (ValueType, error) {
	if err := v.validate(); err != nil {
		return 0, err
	}

	rawCtx, err := v.ctx.Borrow()
	if err != nil {
		return 0, fmt.Errorf("eval: failed to borrow context: %w", err)
	}

	typ := raw.GetType(rawCtx, v.ptr)
	if err := status.FromContext(rawCtx); err != nil {
		return 0, fmt.Errorf("eval: failed to get value type: %w", err)
	}

	return ValueType(typ), nil
}

// TypeName returns Nix's human-readable type name for v.
func (v *Value) TypeName() (string, error) {
	if err := v.validate(); err != nil {
		return "", err
	}

	rawCtx, err := v.ctx.Borrow()
	if err != nil {
		return "", fmt.Errorf("eval: failed to borrow context: %w", err)
	}

	ptr := raw.GetTypename(rawCtx, v.ptr)
	if ptr == nil {
		return "", fmt.Errorf("eval: failed to get value type name: %w", status.FromContext(rawCtx))
	}

	return utils.TakeCString(ptr), nil
}

// Bool returns v as a Go bool.
func (v *Value) Bool() (bool, error) {
	if err := v.validate(); err != nil {
		return false, err
	}

	rawCtx, err := v.ctx.Borrow()
	if err != nil {
		return false, fmt.Errorf("eval: failed to borrow context: %w", err)
	}
	if err := v.requireType(rawCtx, ValueTypeBool); err != nil {
		return false, err
	}

	got := raw.GetBool(rawCtx, v.ptr)
	if err := status.FromContext(rawCtx); err != nil {
		return false, fmt.Errorf("eval: failed to get bool: %w", err)
	}

	return got, nil
}

// Int returns v as a Go int64.
func (v *Value) Int() (int64, error) {
	if err := v.validate(); err != nil {
		return 0, err
	}

	rawCtx, err := v.ctx.Borrow()
	if err != nil {
		return 0, fmt.Errorf("eval: failed to borrow context: %w", err)
	}
	if err := v.requireType(rawCtx, ValueTypeInt); err != nil {
		return 0, err
	}

	got := raw.GetInt(rawCtx, v.ptr)
	if err := status.FromContext(rawCtx); err != nil {
		return 0, fmt.Errorf("eval: failed to get int: %w", err)
	}

	return got, nil
}

// Float returns v as a Go float64.
func (v *Value) Float() (float64, error) {
	if err := v.validate(); err != nil {
		return 0, err
	}

	rawCtx, err := v.ctx.Borrow()
	if err != nil {
		return 0, fmt.Errorf("eval: failed to borrow context: %w", err)
	}
	if err := v.requireType(rawCtx, ValueTypeFloat); err != nil {
		return 0, err
	}

	got := raw.GetFloat(rawCtx, v.ptr)
	if err := status.FromContext(rawCtx); err != nil {
		return 0, fmt.Errorf("eval: failed to get float: %w", err)
	}

	return got, nil
}

// String returns v as a Go string.
func (v *Value) String() (string, error) {
	if err := v.validate(); err != nil {
		return "", err
	}

	rawCtx, err := v.ctx.Borrow()
	if err != nil {
		return "", fmt.Errorf("eval: failed to borrow context: %w", err)
	}
	if err := v.requireType(rawCtx, ValueTypeString); err != nil {
		return "", err
	}

	ptr := raw.GetString(rawCtx, v.ptr)
	if ptr == nil {
		return "", fmt.Errorf("eval: failed to get string: %w", status.FromContext(rawCtx))
	}

	return utils.TakeCString(ptr), nil
}

// PathString returns v as a Nix path string.
func (v *Value) PathString() (string, error) {
	if err := v.validate(); err != nil {
		return "", err
	}

	rawCtx, err := v.ctx.Borrow()
	if err != nil {
		return "", fmt.Errorf("eval: failed to borrow context: %w", err)
	}
	if err := v.requireType(rawCtx, ValueTypePath); err != nil {
		return "", err
	}

	ptr := raw.GetPathString(rawCtx, v.ptr)
	if ptr == nil {
		return "", fmt.Errorf("eval: failed to get path string: %w", status.FromContext(rawCtx))
	}

	return utils.TakeCString(ptr), nil
}

// ListLen returns the number of items in a Nix list.
func (v *Value) ListLen() (uint32, error) {
	if err := v.validate(); err != nil {
		return 0, err
	}

	rawCtx, err := v.ctx.Borrow()
	if err != nil {
		return 0, fmt.Errorf("eval: failed to borrow context: %w", err)
	}
	if err := v.requireType(rawCtx, ValueTypeList); err != nil {
		return 0, err
	}

	got := raw.GetListSize(rawCtx, v.ptr)
	if err := status.FromContext(rawCtx); err != nil {
		return 0, fmt.Errorf("eval: failed to get list size: %w", err)
	}

	return got, nil
}

// AttrLen returns the number of attributes in a Nix attribute set.
func (v *Value) AttrLen() (uint32, error) {
	if err := v.validate(); err != nil {
		return 0, err
	}

	rawCtx, err := v.ctx.Borrow()
	if err != nil {
		return 0, fmt.Errorf("eval: failed to borrow context: %w", err)
	}
	if err := v.requireType(rawCtx, ValueTypeAttrs); err != nil {
		return 0, err
	}

	got := raw.GetAttrsSize(rawCtx, v.ptr)
	if err := status.FromContext(rawCtx); err != nil {
		return 0, fmt.Errorf("eval: failed to get attrs size: %w", err)
	}

	return got, nil
}

// Borrow returns the borrowed raw Nix value handle.
//
// Callers must not free the returned pointer and must not retain it beyond the
// immediate raw Nix call that needs it. This is an escape hatch for integration
// with lower-level bindings.
func (v *Value) Borrow() (*raw.NixValue, error) {
	if err := v.validate(); err != nil {
		return nil, err
	}

	return v.ptr, nil
}

// Close releases the owned Nix value reference.
//
// Close is safe to call more than once. It may be called after the Evaluator is
// closed, but the Context must remain open. Once Close returns, methods that
// need the raw value handle report status.ErrClosed.
func (v *Value) Close() error {
	if v == nil || v.ptr == nil {
		return nil
	}

	rawCtx, err := v.ctx.Borrow()
	if err != nil {
		return fmt.Errorf("eval: failed to borrow context: %w", err)
	}

	defer func() {
		v.ptr = nil
	}()

	if code := raw.ValueDecref(rawCtx, v.ptr); status.ErrorCode(code) != status.ErrorCodeOK {
		return fmt.Errorf("eval: failed to decref value: %w", status.FromContext(rawCtx))
	}

	return nil
}

func (v *Value) validate() error {
	if v == nil || v.ptr == nil || v.owner == nil || v.owner.state == nil {
		return status.ErrClosed
	}

	return nil
}

func (v *Value) requireType(rawCtx *raw.NixCContext, expected ValueType) error {
	actual := ValueType(raw.GetType(rawCtx, v.ptr))
	if err := status.FromContext(rawCtx); err != nil {
		return fmt.Errorf("eval: failed to get value type: %w", err)
	}
	if actual != expected {
		return &ValueTypeError{Actual: actual, Expected: expected}
	}

	return nil
}
