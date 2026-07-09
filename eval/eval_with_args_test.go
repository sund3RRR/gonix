package eval_test

import (
	"errors"
	"math"
	"strings"
	"testing"

	"github.com/sund3RRR/gonix/eval"
)

func TestEvaluatorEvalWithArgsComposite(t *testing.T) {
	_, e := newTestEvaluator(t)

	type name string
	type number int16

	arg := map[name]any{
		"enabled": true,
		"meta": map[string]string{
			"title": "gonix",
		},
		"nothing": nil,
		"ratio":   float32(1.5),
		"values":  []number{7, 11},
	}
	value, err := e.EvalWithArgs(`args: {
		title = args.meta.title;
		first = builtins.elemAt args.values 0;
		count = builtins.length args.values;
		enabled = args.enabled;
		ratio = args.ratio;
		nothing = args.nothing;
	}`, "<eval-with-args-test>", arg)
	if err != nil {
		t.Fatalf("Evaluator.EvalWithArgs() error = %v", err)
	}

	var got struct {
		Title   string  `nix:"title"`
		First   int     `nix:"first"`
		Count   int     `nix:"count"`
		Enabled bool    `nix:"enabled"`
		Ratio   float64 `nix:"ratio"`
		Nothing *string `nix:"nothing"`
	}
	if err := e.Unmarshal(value, &got); err != nil {
		t.Fatalf("Evaluator.Unmarshal() error = %v", err)
	}
	if got.Title != "gonix" || got.First != 7 || got.Count != 2 || !got.Enabled || got.Ratio != 1.5 || got.Nothing != nil {
		t.Fatalf("Evaluator.EvalWithArgs() result = %#v", got)
	}

	if err := value.Close(); err != nil {
		t.Fatalf("Value.Close() error = %v", err)
	}
	if err := value.Close(); err != nil {
		t.Fatalf("second Value.Close() error = %v", err)
	}
}

func TestEvaluatorEvalWithArgsArbitraryArgument(t *testing.T) {
	_, e := newTestEvaluator(t)

	tests := []struct {
		name string
		expr string
		arg  any
		want int64
	}{
		{name: "signed integer", expr: "x: x + 1", arg: int8(2), want: 3},
		{name: "unsigned integer", expr: "x: x + 1", arg: uint16(3), want: 4},
		{name: "slice", expr: "x: builtins.length x", arg: []string{"a", "b"}, want: 2},
		{name: "nil slice", expr: "x: builtins.length x", arg: []string(nil), want: 0},
		{name: "nil map", expr: "x: builtins.length (builtins.attrNames x)", arg: map[string]int(nil), want: 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			value, err := e.EvalWithArgs(tt.expr, "<eval-with-args-test>", tt.arg)
			if err != nil {
				t.Fatalf("Evaluator.EvalWithArgs() error = %v", err)
			}
			defer value.Close() //nolint:errcheck

			got, err := value.Int()
			if err != nil {
				t.Fatalf("Value.Int() error = %v", err)
			}
			if got != tt.want {
				t.Fatalf("Value.Int() = %d, want %d", got, tt.want)
			}
		})
	}

	null, err := e.EvalWithArgs("x: x", "<eval-with-args-test>", nil)
	if err != nil {
		t.Fatalf("Evaluator.EvalWithArgs(nil) error = %v", err)
	}
	defer null.Close() //nolint:errcheck
	if typ, err := null.Type(); err != nil || typ != eval.ValueTypeNull {
		t.Fatalf("Value.Type() = %v, %v; want null, nil", typ, err)
	}
}

func TestEvaluatorEvalWithArgsErrors(t *testing.T) {
	_, e := newTestEvaluator(t)

	tests := []struct {
		name string
		arg  any
		path string
	}{
		{name: "struct", arg: struct{ Name string }{Name: "gonix"}, path: "$"},
		{name: "array", arg: [1]int{1}, path: "$"},
		{name: "pointer", arg: new(int), path: "$"},
		{name: "non-string map key", arg: map[int]string{1: "one"}, path: "$"},
		{name: "nested unsupported value", arg: map[string][]any{"items": {make(chan int)}}, path: "$.items[0]"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := e.EvalWithArgs("x: x", "<eval-with-args-test>", tt.arg)
			var typeErr *eval.UnsupportedTypeError
			if !errors.As(err, &typeErr) {
				t.Fatalf("Evaluator.EvalWithArgs() error = %v, want UnsupportedTypeError", err)
			}
			if typeErr.Path != tt.path {
				t.Fatalf("UnsupportedTypeError.Path = %q, want %q", typeErr.Path, tt.path)
			}
		})
	}

	_, err := e.EvalWithArgs("x: x", "<eval-with-args-test>", uint64(math.MaxUint64))
	if err == nil || !strings.Contains(err.Error(), "overflows Nix integer at $") {
		t.Fatalf("Evaluator.EvalWithArgs(overflow) error = %v, want overflow at root", err)
	}

	_, err = e.EvalWithArgs("let =", "<eval-with-args-test>", 1)
	if err == nil {
		t.Fatal("Evaluator.EvalWithArgs(invalid expression) error = nil")
	}

	_, err = e.EvalWithArgs("1", "<eval-with-args-test>", 1)
	if err == nil {
		t.Fatal("Evaluator.EvalWithArgs(non-function) error = nil")
	}

	if err := e.Close(); err != nil {
		t.Fatalf("Evaluator.Close() error = %v", err)
	}
	_, err = e.EvalWithArgs("x: x", "<eval-with-args-test>", 1)
	requireClosedError(t, err)
}
