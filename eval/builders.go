package eval

import (
	"fmt"
	"maps"
	"sort"

	"github.com/sund3RRR/gonix/internal/status"
	"github.com/sund3RRR/gonix/pkg/raw"
)

// GoValue initializes a Nix value from Go-native data.
//
// Use the helper functions in this package, such as Int, String, List, and
// Attrs, to construct GoValue values for Evaluator.NewValue.
type GoValue func(*Evaluator, *raw.NixValue) error

// Null returns a GoValue that initializes Nix null.
func Null() GoValue {
	return GoValue(func(e *Evaluator, out *raw.NixValue) error {
		rawCtx, err := e.ctx.Borrow()
		if err != nil {
			return fmt.Errorf("eval: failed to borrow context: %w", err)
		}

		if code := raw.InitNull(rawCtx, out); status.ErrorCode(code) != status.ErrorCodeOK {
			return fmt.Errorf("eval: failed to initialize null: %w", status.FromContext(rawCtx))
		}

		return nil
	})
}

// Bool returns a GoValue that initializes a Nix boolean.
func Bool(value bool) GoValue {
	return GoValue(func(e *Evaluator, out *raw.NixValue) error {
		rawCtx, err := e.ctx.Borrow()
		if err != nil {
			return fmt.Errorf("eval: failed to borrow context: %w", err)
		}

		if code := raw.InitBool(rawCtx, out, value); status.ErrorCode(code) != status.ErrorCodeOK {
			return fmt.Errorf("eval: failed to initialize bool: %w", status.FromContext(rawCtx))
		}

		return nil
	})
}

// Int returns a GoValue that initializes a Nix integer.
func Int(value int64) GoValue {
	return GoValue(func(e *Evaluator, out *raw.NixValue) error {
		rawCtx, err := e.ctx.Borrow()
		if err != nil {
			return fmt.Errorf("eval: failed to borrow context: %w", err)
		}

		if code := raw.InitInt(rawCtx, out, value); status.ErrorCode(code) != status.ErrorCodeOK {
			return fmt.Errorf("eval: failed to initialize int: %w", status.FromContext(rawCtx))
		}

		return nil
	})
}

// Float returns a GoValue that initializes a Nix floating-point number.
func Float(value float64) GoValue {
	return GoValue(func(e *Evaluator, out *raw.NixValue) error {
		rawCtx, err := e.ctx.Borrow()
		if err != nil {
			return fmt.Errorf("eval: failed to borrow context: %w", err)
		}

		if code := raw.InitFloat(rawCtx, out, value); status.ErrorCode(code) != status.ErrorCodeOK {
			return fmt.Errorf("eval: failed to initialize float: %w", status.FromContext(rawCtx))
		}

		return nil
	})
}

// String returns a GoValue that initializes a Nix string.
func String(value string) GoValue {
	return GoValue(func(e *Evaluator, out *raw.NixValue) error {
		rawCtx, err := e.ctx.Borrow()
		if err != nil {
			return fmt.Errorf("eval: failed to borrow context: %w", err)
		}

		if code := raw.InitString(rawCtx, out, value); status.ErrorCode(code) != status.ErrorCodeOK {
			return fmt.Errorf("eval: failed to initialize string: %w", status.FromContext(rawCtx))
		}

		return nil
	})
}

// PathString returns a GoValue that initializes a Nix path string.
func PathString(value string) GoValue {
	return GoValue(func(e *Evaluator, out *raw.NixValue) error {
		rawCtx, err := e.ctx.Borrow()
		if err != nil {
			return fmt.Errorf("eval: failed to borrow context: %w", err)
		}

		if code := raw.InitPathString(rawCtx, e.state, out, value); status.ErrorCode(code) != status.ErrorCodeOK {
			return fmt.Errorf("eval: failed to initialize path string: %w", status.FromContext(rawCtx))
		}

		return nil
	})
}

// List returns a GoValue that initializes a Nix list.
func List(values ...GoValue) GoValue {
	copied := append([]GoValue(nil), values...)
	return GoValue(func(e *Evaluator, out *raw.NixValue) error {
		rawCtx, err := e.ctx.Borrow()
		if err != nil {
			return fmt.Errorf("eval: failed to borrow context: %w", err)
		}

		builder := raw.MakeListBuilder(rawCtx, e.state, uint64(len(copied)))
		if builder == nil {
			return fmt.Errorf("eval: failed to create list builder: %w", status.FromContext(rawCtx))
		}
		defer raw.ListBuilderFree(builder)

		for i, item := range copied {
			value, err := e.NewValue(item)
			if err != nil {
				return fmt.Errorf("eval: failed to build list item %d: %w", i, err)
			}
			defer value.Close() //nolint:errcheck

			if code := raw.ListBuilderInsert(rawCtx, builder, uint32(i), value.ptr); status.ErrorCode(code) != status.ErrorCodeOK {
				return fmt.Errorf("eval: failed to insert list item %d: %w", i, status.FromContext(rawCtx))
			}
		}

		if code := raw.MakeList(rawCtx, builder, out); status.ErrorCode(code) != status.ErrorCodeOK {
			return fmt.Errorf("eval: failed to initialize list: %w", status.FromContext(rawCtx))
		}

		return nil
	})
}

// Attrs returns a GoValue that initializes a Nix attribute set.
func Attrs(values map[string]GoValue) GoValue {
	copied := make(map[string]GoValue, len(values))
	maps.Copy(copied, values)

	return GoValue(func(e *Evaluator, out *raw.NixValue) error {
		rawCtx, err := e.ctx.Borrow()
		if err != nil {
			return fmt.Errorf("eval: failed to borrow context: %w", err)
		}

		builder := raw.MakeBindingsBuilder(rawCtx, e.state, uint64(len(copied)))
		if builder == nil {
			return fmt.Errorf("eval: failed to create attrs builder: %w", status.FromContext(rawCtx))
		}
		defer raw.BindingsBuilderFree(builder)

		names := make([]string, 0, len(copied))
		for name := range copied {
			names = append(names, name)
		}
		sort.Strings(names)

		for _, name := range names {
			value, err := e.NewValue(copied[name])
			if err != nil {
				return fmt.Errorf("eval: failed to build attr %q: %w", name, err)
			}
			defer value.Close() //nolint:errcheck

			if code := raw.BindingsBuilderInsert(rawCtx, builder, name, value.ptr); status.ErrorCode(code) != status.ErrorCodeOK {
				return fmt.Errorf("eval: failed to insert attr %q: %w", name, status.FromContext(rawCtx))
			}
		}

		if code := raw.MakeAttrs(rawCtx, out, builder); status.ErrorCode(code) != status.ErrorCodeOK {
			return fmt.Errorf("eval: failed to initialize attrs: %w", status.FromContext(rawCtx))
		}

		return nil
	})
}

// Apply returns a GoValue that initializes an unapplied function call.
func Apply(fn, arg *Value) GoValue {
	return GoValue(func(e *Evaluator, out *raw.NixValue) error {
		rawCtx, err := e.ctx.Borrow()
		if err != nil {
			return fmt.Errorf("eval: failed to borrow context: %w", err)
		}

		if err := e.validateValue(fn); err != nil {
			return fmt.Errorf("eval: failed to validate function value: %w", err)
		}

		if err := e.validateValue(arg); err != nil {
			return fmt.Errorf("eval: failed to validate argument value: %w", err)
		}

		if code := raw.InitApply(rawCtx, out, fn.ptr, arg.ptr); status.ErrorCode(code) != status.ErrorCodeOK {
			return fmt.Errorf("eval: failed to initialize apply: %w", status.FromContext(rawCtx))
		}

		return nil
	})
}

// Copy returns a GoValue that initializes a value by copying src.
func Copy(src *Value) GoValue {
	return GoValue(func(e *Evaluator, out *raw.NixValue) error {
		rawCtx, err := e.ctx.Borrow()
		if err != nil {
			return fmt.Errorf("eval: failed to borrow context: %w", err)
		}

		if err := e.validateValue(src); err != nil {
			return fmt.Errorf("eval: failed to validate source value: %w", err)
		}

		if code := raw.CopyValue(rawCtx, out, src.ptr); status.ErrorCode(code) != status.ErrorCodeOK {
			return fmt.Errorf("eval: failed to copy value: %w", status.FromContext(rawCtx))
		}

		return nil
	})
}
