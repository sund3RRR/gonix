package eval

import (
	"fmt"
	"sort"

	"github.com/sund3RRR/gonix/internal/status"
	nix "github.com/sund3RRR/nix-go-bindings"
)

// GoValue initializes a Nix value from Go-native data.
//
// Use the helper functions in this package, such as Int, String, List, and
// Attrs, to construct GoValue values for Evaluator.NewValue.
type GoValue func(*Evaluator, *nix.NixValue) error

// Null returns a GoValue that initializes Nix null.
func Null() GoValue {
	return GoValue(func(e *Evaluator, out *nix.NixValue) error {
		if code := nix.InitNull(e.ctx, out); status.ErrorCode(code) != status.ErrorCodeOK {
			return fmt.Errorf("eval: failed to initialize null: %w", status.FromContext(e.ctx))
		}
		return nil
	})
}

// Bool returns a GoValue that initializes a Nix boolean.
func Bool(value bool) GoValue {
	return GoValue(func(e *Evaluator, out *nix.NixValue) error {
		if code := nix.InitBool(e.ctx, out, value); status.ErrorCode(code) != status.ErrorCodeOK {
			return fmt.Errorf("eval: failed to initialize bool: %w", status.FromContext(e.ctx))
		}
		return nil
	})
}

// Int returns a GoValue that initializes a Nix integer.
func Int(value int64) GoValue {
	return GoValue(func(e *Evaluator, out *nix.NixValue) error {
		if code := nix.InitInt(e.ctx, out, value); status.ErrorCode(code) != status.ErrorCodeOK {
			return fmt.Errorf("eval: failed to initialize int: %w", status.FromContext(e.ctx))
		}
		return nil
	})
}

// Float returns a GoValue that initializes a Nix floating-point number.
func Float(value float64) GoValue {
	return GoValue(func(e *Evaluator, out *nix.NixValue) error {
		if code := nix.InitFloat(e.ctx, out, value); status.ErrorCode(code) != status.ErrorCodeOK {
			return fmt.Errorf("eval: failed to initialize float: %w", status.FromContext(e.ctx))
		}
		return nil
	})
}

// String returns a GoValue that initializes a Nix string.
func String(value string) GoValue {
	return GoValue(func(e *Evaluator, out *nix.NixValue) error {
		if code := nix.InitString(e.ctx, out, value); status.ErrorCode(code) != status.ErrorCodeOK {
			return fmt.Errorf("eval: failed to initialize string: %w", status.FromContext(e.ctx))
		}
		return nil
	})
}

// PathString returns a GoValue that initializes a Nix path string.
func PathString(value string) GoValue {
	return GoValue(func(e *Evaluator, out *nix.NixValue) error {
		if code := nix.InitPathString(e.ctx, e.state, out, value); status.ErrorCode(code) != status.ErrorCodeOK {
			return fmt.Errorf("eval: failed to initialize path string: %w", status.FromContext(e.ctx))
		}
		return nil
	})
}

// List returns a GoValue that initializes a Nix list.
func List(values ...GoValue) GoValue {
	copied := append([]GoValue(nil), values...)
	return GoValue(func(e *Evaluator, out *nix.NixValue) error {
		builder := nix.MakeListBuilder(e.ctx, e.state, uint64(len(copied)))
		if builder == nil {
			return fmt.Errorf("eval: failed to create list builder: %w", status.FromContext(e.ctx))
		}
		defer nix.ListBuilderFree(builder)

		temps := make([]*Value, 0, len(copied))
		defer closeValues(temps)

		for i, item := range copied {
			value, err := e.NewValue(item)
			if err != nil {
				return fmt.Errorf("eval: failed to build list item %d: %w", i, err)
			}
			temps = append(temps, value)

			if code := nix.ListBuilderInsert(e.ctx, builder, uint32(i), value.ptr); status.ErrorCode(code) != status.ErrorCodeOK {
				return fmt.Errorf("eval: failed to insert list item %d: %w", i, status.FromContext(e.ctx))
			}
		}

		if code := nix.MakeList(e.ctx, builder, out); status.ErrorCode(code) != status.ErrorCodeOK {
			return fmt.Errorf("eval: failed to initialize list: %w", status.FromContext(e.ctx))
		}

		return nil
	})
}

// Attrs returns a GoValue that initializes a Nix attribute set.
func Attrs(values map[string]GoValue) GoValue {
	copied := make(map[string]GoValue, len(values))
	for k, v := range values {
		copied[k] = v
	}

	return GoValue(func(e *Evaluator, out *nix.NixValue) error {
		builder := nix.MakeBindingsBuilder(e.ctx, e.state, uint64(len(copied)))
		if builder == nil {
			return fmt.Errorf("eval: failed to create attrs builder: %w", status.FromContext(e.ctx))
		}
		defer nix.BindingsBuilderFree(builder)

		names := make([]string, 0, len(copied))
		for name := range copied {
			names = append(names, name)
		}
		sort.Strings(names)

		temps := make([]*Value, 0, len(copied))
		defer closeValues(temps)

		for _, name := range names {
			value, err := e.NewValue(copied[name])
			if err != nil {
				return fmt.Errorf("eval: failed to build attr %q: %w", name, err)
			}
			temps = append(temps, value)

			if code := nix.BindingsBuilderInsert(e.ctx, builder, name, value.ptr); status.ErrorCode(code) != status.ErrorCodeOK {
				return fmt.Errorf("eval: failed to insert attr %q: %w", name, status.FromContext(e.ctx))
			}
		}

		if code := nix.MakeAttrs(e.ctx, out, builder); status.ErrorCode(code) != status.ErrorCodeOK {
			return fmt.Errorf("eval: failed to initialize attrs: %w", status.FromContext(e.ctx))
		}

		return nil
	})
}

// Apply returns a GoValue that initializes an unapplied function call.
func Apply(fn, arg *Value) GoValue {
	return GoValue(func(e *Evaluator, out *nix.NixValue) error {
		if err := e.validateValue(fn); err != nil {
			return fmt.Errorf("eval: failed to validate function value: %w", err)
		}

		if err := e.validateValue(arg); err != nil {
			return fmt.Errorf("eval: failed to validate argument value: %w", err)
		}

		if code := nix.InitApply(e.ctx, out, fn.ptr, arg.ptr); status.ErrorCode(code) != status.ErrorCodeOK {
			return fmt.Errorf("eval: failed to initialize apply: %w", status.FromContext(e.ctx))
		}

		return nil
	})
}

// Copy returns a GoValue that initializes a value by copying src.
func Copy(src *Value) GoValue {
	return GoValue(func(e *Evaluator, out *nix.NixValue) error {
		if err := e.validateValue(src); err != nil {
			return fmt.Errorf("eval: failed to validate source value: %w", err)
		}

		if code := nix.CopyValue(e.ctx, out, src.ptr); status.ErrorCode(code) != status.ErrorCodeOK {
			return fmt.Errorf("eval: failed to copy value: %w", status.FromContext(e.ctx))
		}

		return nil
	})
}

func closeValues(values []*Value) {
	for _, value := range values {
		_ = value.Close()
	}
}
