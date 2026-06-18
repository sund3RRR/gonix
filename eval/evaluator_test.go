package eval_test

import (
	"errors"
	"math"
	"sort"
	"strings"
	"testing"

	"github.com/sund3RRR/gonix"
	"github.com/sund3RRR/gonix/eval"
	"github.com/sund3RRR/gonix/nixcontext"
	"github.com/sund3RRR/gonix/store"
)

func newTestEvaluator(t *testing.T) (*nixcontext.Context, *eval.Evaluator) {
	t.Helper()

	ctx, err := nixcontext.New(nixcontext.Config{})
	if err != nil {
		t.Fatalf("nixcontext.New() error = %v", err)
	}

	s, err := store.New(ctx, "dummy://")
	if err != nil {
		_ = ctx.Close()
		t.Fatalf("store.New() error = %v", err)
	}

	e, err := eval.New(ctx, s)
	if err != nil {
		_ = s.Close()
		_ = ctx.Close()
		t.Fatalf("eval.New() error = %v", err)
	}
	t.Cleanup(func() {
		if err := e.Close(); err != nil {
			t.Fatalf("Evaluator.Close() error = %v", err)
		}
		if err := s.Close(); err != nil {
			t.Fatalf("Store.Close() error = %v", err)
		}
		if err := ctx.Close(); err != nil {
			t.Fatalf("Context.Close() error = %v", err)
		}
	})

	return ctx, e
}

func requireClosedError(t *testing.T, err error) {
	t.Helper()

	if err == nil {
		t.Fatal("error = nil, want gonix.ErrClosed")
	}
	if !errors.Is(err, gonix.ErrClosed) {
		t.Fatalf("error = %v, want errors.Is(..., gonix.ErrClosed)", err)
	}
}

func closeValueAtCleanup(t *testing.T, value *eval.Value) {
	t.Helper()

	t.Cleanup(func() {
		if err := value.Close(); err != nil {
			t.Errorf("Value.Close() error = %v", err)
		}
	})
}

func TestEvaluatorEvalStringAndPrimitiveGetters(t *testing.T) {
	_, e := newTestEvaluator(t)

	tests := []struct {
		name   string
		expr   string
		assert func(t *testing.T, v *eval.Value)
	}{
		{
			name: "int",
			expr: "1 + 2",
			assert: func(t *testing.T, v *eval.Value) {
				t.Helper()
				got, err := v.Int()
				if err != nil {
					t.Fatalf("Value.Int() error = %v", err)
				}
				if got != 3 {
					t.Fatalf("Value.Int() = %d, want 3", got)
				}
			},
		},
		{
			name: "float",
			expr: "1.25",
			assert: func(t *testing.T, v *eval.Value) {
				t.Helper()
				got, err := v.Float()
				if err != nil {
					t.Fatalf("Value.Float() error = %v", err)
				}
				if math.Abs(got-1.25) > 0.000001 {
					t.Fatalf("Value.Float() = %f, want 1.25", got)
				}
			},
		},
		{
			name: "bool_false",
			expr: "false",
			assert: func(t *testing.T, v *eval.Value) {
				t.Helper()
				got, err := v.Bool()
				if err != nil {
					t.Fatalf("Value.Bool() error = %v", err)
				}
				if got {
					t.Fatal("Value.Bool() = true, want false")
				}
			},
		},
		{
			name: "string",
			expr: `"hello"`,
			assert: func(t *testing.T, v *eval.Value) {
				t.Helper()
				got, err := v.String()
				if err != nil {
					t.Fatalf("Value.String() error = %v", err)
				}
				if got != "hello" {
					t.Fatalf("Value.String() = %q, want hello", got)
				}
			},
		},
		{
			name: "null",
			expr: "null",
			assert: func(t *testing.T, v *eval.Value) {
				t.Helper()
				typ, err := v.Type()
				if err != nil {
					t.Fatalf("Value.Type() error = %v", err)
				}
				if typ != eval.ValueTypeNull {
					t.Fatalf("Value.Type() = %v, want ValueTypeNull", typ)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			v, err := e.EvalString(tt.expr, ".")
			if err != nil {
				t.Fatalf("Evaluator.EvalString() error = %v", err)
			}
			closeValueAtCleanup(t, v)
			if err := e.Force(v); err != nil {
				t.Fatalf("Evaluator.Force() error = %v", err)
			}
			if name, err := v.TypeName(); err != nil || strings.TrimSpace(name) == "" {
				t.Fatalf("Value.TypeName() = %q, %v; want non-empty name", name, err)
			}
			tt.assert(t, v)
		})
	}
}

func TestEvaluatorInvalidExpression(t *testing.T) {
	_, e := newTestEvaluator(t)

	got, err := e.EvalString("let =", ".")
	if err == nil {
		t.Fatal("Evaluator.EvalString(invalid) error = nil, want error")
	}
	if got != nil {
		t.Fatalf("Evaluator.EvalString(invalid) = %v, want nil", got)
	}
}

func TestValueGettersRejectWrongType(t *testing.T) {
	_, e := newTestEvaluator(t)

	value, err := e.EvalString("42", ".")
	if err != nil {
		t.Fatalf("Evaluator.EvalString() error = %v", err)
	}
	closeValueAtCleanup(t, value)
	if err := e.Force(value); err != nil {
		t.Fatalf("Evaluator.Force() error = %v", err)
	}

	tests := []struct {
		name     string
		expected eval.ValueType
		call     func() error
	}{
		{name: "bool", expected: eval.ValueTypeBool, call: func() error { _, err := value.Bool(); return err }},
		{name: "float", expected: eval.ValueTypeFloat, call: func() error { _, err := value.Float(); return err }},
		{name: "string", expected: eval.ValueTypeString, call: func() error { _, err := value.String(); return err }},
		{name: "path", expected: eval.ValueTypePath, call: func() error { _, err := value.PathString(); return err }},
		{name: "list", expected: eval.ValueTypeList, call: func() error { _, err := value.ListLen(); return err }},
		{name: "attrs", expected: eval.ValueTypeAttrs, call: func() error { _, err := value.AttrLen(); return err }},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.call()
			var typeErr *eval.ValueTypeError
			if !errors.As(err, &typeErr) {
				t.Fatalf("getter error = %v, want ValueTypeError", err)
			}
			if typeErr.Actual != eval.ValueTypeInt || typeErr.Expected != tt.expected {
				t.Fatalf("ValueTypeError = %#v, want actual int and expected %s", typeErr, tt.expected)
			}
		})
	}
}

func TestEvaluatorForceDeepCallAndCallMulti(t *testing.T) {
	_, e := newTestEvaluator(t)

	list, err := e.EvalString("[ (1 + 1) (2 + 2) ]", ".")
	if err != nil {
		t.Fatalf("Evaluator.EvalString(list) error = %v", err)
	}
	closeValueAtCleanup(t, list)
	if err := e.ForceDeep(list); err != nil {
		t.Fatalf("Evaluator.ForceDeep() error = %v", err)
	}
	if got, err := list.ListLen(); err != nil || got != 2 {
		t.Fatalf("Value.ListLen() = %d, %v; want 2, nil", got, err)
	}

	fn, err := e.EvalString("x: y: x + y", ".")
	if err != nil {
		t.Fatalf("Evaluator.EvalString(fn) error = %v", err)
	}
	closeValueAtCleanup(t, fn)
	two, err := e.NewValue(eval.Int(2))
	if err != nil {
		t.Fatalf("Evaluator.NewValue(2) error = %v", err)
	}
	closeValueAtCleanup(t, two)
	three, err := e.NewValue(eval.Int(3))
	if err != nil {
		t.Fatalf("Evaluator.NewValue(3) error = %v", err)
	}
	closeValueAtCleanup(t, three)

	partial, err := e.Call(fn, two)
	if err != nil {
		t.Fatalf("Evaluator.Call() error = %v", err)
	}
	closeValueAtCleanup(t, partial)
	sum, err := e.Call(partial, three)
	if err != nil {
		t.Fatalf("Evaluator.Call(partial) error = %v", err)
	}
	closeValueAtCleanup(t, sum)
	if err := e.Force(sum); err != nil {
		t.Fatalf("Evaluator.Force(sum) error = %v", err)
	}
	if got, err := sum.Int(); err != nil || got != 5 {
		t.Fatalf("Value.Int(sum) = %d, %v; want 5, nil", got, err)
	}

	multi, err := e.CallMulti(fn, two, three)
	if err != nil {
		t.Fatalf("Evaluator.CallMulti() error = %v", err)
	}
	closeValueAtCleanup(t, multi)
	if err := e.Force(multi); err != nil {
		t.Fatalf("Evaluator.Force(multi) error = %v", err)
	}
	if got, err := multi.Int(); err != nil || got != 5 {
		t.Fatalf("Value.Int(multi) = %d, %v; want 5, nil", got, err)
	}
}

func TestEvaluatorListAndAttrs(t *testing.T) {
	_, e := newTestEvaluator(t)

	list, err := e.EvalString("[ 1 2 ]", ".")
	if err != nil {
		t.Fatalf("Evaluator.EvalString(list) error = %v", err)
	}
	closeValueAtCleanup(t, list)
	if err := e.Force(list); err != nil {
		t.Fatalf("Evaluator.Force(list) error = %v", err)
	}
	second, err := e.Index(list, 1)
	if err != nil {
		t.Fatalf("Evaluator.Index() error = %v", err)
	}
	closeValueAtCleanup(t, second)
	if got, err := second.Int(); err != nil || got != 2 {
		t.Fatalf("Value.Int(second) = %d, %v; want 2, nil", got, err)
	}

	attrs, err := e.EvalString(`{ a = 1; b = "x"; }`, ".")
	if err != nil {
		t.Fatalf("Evaluator.EvalString(attrs) error = %v", err)
	}
	closeValueAtCleanup(t, attrs)
	if err := e.Force(attrs); err != nil {
		t.Fatalf("Evaluator.Force(attrs) error = %v", err)
	}
	if got, err := attrs.AttrLen(); err != nil || got != 2 {
		t.Fatalf("Value.AttrLen() = %d, %v; want 2, nil", got, err)
	}
	hasA, err := e.HasAttr(attrs, "a")
	if err != nil {
		t.Fatalf("Evaluator.HasAttr(a) error = %v", err)
	}
	if !hasA {
		t.Fatal("Evaluator.HasAttr(a) = false, want true")
	}
	hasMissing, err := e.HasAttr(attrs, "missing")
	if err != nil {
		t.Fatalf("Evaluator.HasAttr(missing) error = %v", err)
	}
	if hasMissing {
		t.Fatal("Evaluator.HasAttr(missing) = true, want false")
	}

	attrA, err := e.Attr(attrs, "a")
	if err != nil {
		t.Fatalf("Evaluator.Attr(a) error = %v", err)
	}
	closeValueAtCleanup(t, attrA)
	if got, err := attrA.Int(); err != nil || got != 1 {
		t.Fatalf("Value.Int(attrA) = %d, %v; want 1, nil", got, err)
	}

	names := []string{}
	for i := uint32(0); i < 2; i++ {
		name, err := e.AttrName(attrs, i)
		if err != nil {
			t.Fatalf("Evaluator.AttrName(%d) error = %v", i, err)
		}
		names = append(names, name)
		value, err := e.AttrByIndex(attrs, i)
		if err != nil {
			t.Fatalf("Evaluator.AttrByIndex(%d) error = %v", i, err)
		}
		closeValueAtCleanup(t, value)
	}
	sort.Strings(names)
	if names[0] != "a" || names[1] != "b" {
		t.Fatalf("attribute names = %v, want [a b]", names)
	}
}

func TestEvaluatorNewValueBuilders(t *testing.T) {
	_, e := newTestEvaluator(t)

	path, err := e.NewValue(eval.PathString("/nix/store"))
	if err != nil {
		t.Fatalf("Evaluator.NewValue(PathString) error = %v", err)
	}
	closeValueAtCleanup(t, path)
	if got, err := path.PathString(); err != nil || got != "/nix/store" {
		t.Fatalf("Value.PathString() = %q, %v; want /nix/store, nil", got, err)
	}

	list, err := e.NewValue(eval.List(eval.Int(7), eval.String("seven"), eval.Bool(true), eval.Null()))
	if err != nil {
		t.Fatalf("Evaluator.NewValue(List) error = %v", err)
	}
	closeValueAtCleanup(t, list)
	if got, err := list.ListLen(); err != nil || got != 4 {
		t.Fatalf("Value.ListLen() = %d, %v; want 4, nil", got, err)
	}

	attrs, err := e.NewValue(eval.Attrs(map[string]eval.GoValue{
		"answer": eval.Int(42),
		"name":   eval.String("gonix"),
	}))
	if err != nil {
		t.Fatalf("Evaluator.NewValue(Attrs) error = %v", err)
	}
	closeValueAtCleanup(t, attrs)
	answer, err := e.Attr(attrs, "answer")
	if err != nil {
		t.Fatalf("Evaluator.Attr(answer) error = %v", err)
	}
	closeValueAtCleanup(t, answer)
	if got, err := answer.Int(); err != nil || got != 42 {
		t.Fatalf("Value.Int(answer) = %d, %v; want 42, nil", got, err)
	}

	copied, err := e.NewValue(eval.Copy(answer))
	if err != nil {
		t.Fatalf("Evaluator.NewValue(Copy) error = %v", err)
	}
	closeValueAtCleanup(t, copied)
	if got, err := copied.Int(); err != nil || got != 42 {
		t.Fatalf("Value.Int(copied) = %d, %v; want 42, nil", got, err)
	}

	fn, err := e.EvalString("x: x + 1", ".")
	if err != nil {
		t.Fatalf("Evaluator.EvalString(fn) error = %v", err)
	}
	closeValueAtCleanup(t, fn)
	applied, err := e.NewValue(eval.Apply(fn, copied))
	if err != nil {
		t.Fatalf("Evaluator.NewValue(Apply) error = %v", err)
	}
	closeValueAtCleanup(t, applied)
	if err := e.Force(applied); err != nil {
		t.Fatalf("Evaluator.Force(applied) error = %v", err)
	}
	if got, err := applied.Int(); err != nil || got != 43 {
		t.Fatalf("Value.Int(applied) = %d, %v; want 43, nil", got, err)
	}
}

func TestEvaluatorRealiseString(t *testing.T) {
	_, e := newTestEvaluator(t)

	value, err := e.EvalString(`"plain"`, ".")
	if err != nil {
		t.Fatalf("Evaluator.EvalString() error = %v", err)
	}
	closeValueAtCleanup(t, value)
	if err := e.Force(value); err != nil {
		t.Fatalf("Evaluator.Force() error = %v", err)
	}

	realised, err := e.RealiseString(value)
	if err != nil {
		t.Fatalf("Evaluator.RealiseString() error = %v", err)
	}
	if realised.Value != "plain" {
		t.Fatalf("RealisedString.Value = %q, want plain", realised.Value)
	}
	if len(realised.Paths) != 0 {
		t.Fatalf("RealisedString.Paths length = %d, want 0", len(realised.Paths))
	}
	if err := realised.Close(); err != nil {
		t.Fatalf("RealisedString.Close() error = %v", err)
	}
}

func TestEvaluatorLifecycleAndOriginValidation(t *testing.T) {
	ctx, e := newTestEvaluator(t)

	value, err := e.NewValue(eval.Int(1))
	if err != nil {
		t.Fatalf("Evaluator.NewValue() error = %v", err)
	}
	if err := value.Close(); err != nil {
		t.Fatalf("Value.Close() error = %v", err)
	}
	if err := value.Close(); err != nil {
		t.Fatalf("second Value.Close() error = %v", err)
	}
	_, err = value.Int()
	requireClosedError(t, err)

	second, err := e.NewValue(eval.Int(2))
	if err != nil {
		t.Fatalf("Evaluator.NewValue(second) error = %v", err)
	}
	closeValueAtCleanup(t, second)
	if err := e.Close(); err != nil {
		t.Fatalf("Evaluator.Close() error = %v", err)
	}
	if err := e.Close(); err != nil {
		t.Fatalf("second Evaluator.Close() error = %v", err)
	}
	_, err = second.Int()
	requireClosedError(t, err)
	_, err = second.Borrow()
	requireClosedError(t, err)

	_, err = e.EvalString("1", ".")
	requireClosedError(t, err)

	s, err := store.New(ctx, "dummy://")
	if err != nil {
		t.Fatalf("store.New(second) error = %v", err)
	}
	t.Cleanup(func() { _ = s.Close() })
	e1, err := eval.New(ctx, s)
	if err != nil {
		t.Fatalf("eval.New(e1) error = %v", err)
	}
	t.Cleanup(func() { _ = e1.Close() })
	e2, err := eval.New(ctx, s)
	if err != nil {
		t.Fatalf("eval.New(e2) error = %v", err)
	}
	t.Cleanup(func() { _ = e2.Close() })
	foreign, err := e1.NewValue(eval.Int(3))
	if err != nil {
		t.Fatalf("Evaluator.NewValue(foreign) error = %v", err)
	}
	closeValueAtCleanup(t, foreign)
	if err := e2.Force(foreign); err == nil {
		t.Fatal("Evaluator.Force(foreign) error = nil, want error")
	}
}

func TestContextEvaluatorLifecycle(t *testing.T) {
	ctx, err := nixcontext.New(nixcontext.Config{})
	if err != nil {
		t.Fatalf("nixcontext.New() error = %v", err)
	}

	s, err := store.New(ctx, "dummy://")
	if err != nil {
		t.Fatalf("store.New() error = %v", err)
	}
	e, err := eval.New(ctx, s, eval.WithLookupPath("nixpkgs=/no-such-path"))
	if err != nil {
		t.Fatalf("eval.New() error = %v", err)
	}
	v, err := e.NewValue(eval.Int(9))
	if err != nil {
		t.Fatalf("Evaluator.NewValue() error = %v", err)
	}

	if err := e.Close(); err != nil {
		t.Fatalf("Evaluator.Close() error = %v", err)
	}
	_, err = v.Int()
	requireClosedError(t, err)
	if err := v.Close(); err != nil {
		t.Fatalf("Value.Close() after Evaluator.Close() error = %v", err)
	}
	if err := s.Close(); err != nil {
		t.Fatalf("Store.Close() error = %v", err)
	}
	if err := ctx.Close(); err != nil {
		t.Fatalf("Context.Close() error = %v", err)
	}
	_, err = eval.New(ctx, s)
	requireClosedError(t, err)
}
