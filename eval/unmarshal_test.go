package eval_test

import (
	"errors"
	"strings"
	"testing"

	"github.com/sund3RRR/gonix"
	"github.com/sund3RRR/gonix/eval"
)

type unmarshalPackage struct {
	Name       string              `nix:"name" validate:"optional"`
	PName      string              `nix:"pname" validate:"required"`
	Version    string              `nix:"version" validate:"required"`
	System     string              `nix:"system" validate:"optional"`
	DrvPath    string              `nix:"drvPath" validate:"required"`
	OutPath    string              `nix:"outPath" validate:"required"`
	OutputName string              `nix:"outputName" validate:"optional"`
	Outputs    []unmarshalOutput   `nix:"outputs" validate:"required"`
	Meta       unmarshalMetadata   `nix:"meta" validate:"optional"`
	Skipped    string              `nix:"-"`
	Extra      unmarshalExtraField `validate:"optional"`
}

type unmarshalOutput struct {
	Name string `nix:"name" validate:"required"`
	Path string `nix:"path" validate:"required"`
}

type unmarshalMetadata struct {
	Description string `nix:"description" validate:"optional"`
	License     string `nix:"license" validate:"optional"`
	Homepage    string `nix:"homepage" validate:"optional"`
}

type unmarshalExtraField struct {
	Flag bool `nix:"flag" validate:"optional"`
}

func TestEvaluatorUnmarshalPackage(t *testing.T) {
	_, e := newTestEvaluator(t)

	value, err := e.NewValue(eval.Attrs(map[string]eval.GoValue{
		"name":       eval.String("hello-2.12.1"),
		"pname":      eval.String("hello"),
		"version":    eval.String("2.12.1"),
		"system":     eval.String("x86_64-linux"),
		"drvPath":    eval.String("/nix/store/demo-hello.drv"),
		"outPath":    eval.String("/nix/store/demo-hello"),
		"outputName": eval.String("out"),
		"outputs": eval.List(
			eval.Attrs(map[string]eval.GoValue{
				"name": eval.String("out"),
				"path": eval.String("/nix/store/demo-hello"),
			}),
			eval.Attrs(map[string]eval.GoValue{
				"name": eval.String("dev"),
				"path": eval.String("/nix/store/demo-hello-dev"),
			}),
		),
		"meta": eval.Attrs(map[string]eval.GoValue{
			"description": eval.String("friendly greeting"),
			"license":     eval.String("GPL-3.0-or-later"),
			"homepage":    eval.String("https://example.invalid/hello"),
			"unknown":     eval.String("ignored"),
		}),
		"unknown": eval.String("ignored"),
	}))
	if err != nil {
		t.Fatalf("Evaluator.NewValue(package) error = %v", err)
	}

	var got unmarshalPackage
	got.Skipped = "kept"
	if err := e.Unmarshal(value, &got); err != nil {
		t.Fatalf("Evaluator.Unmarshal(package) error = %v", err)
	}

	if got.PName != "hello" || got.Version != "2.12.1" || got.OutputName != "out" {
		t.Fatalf("package scalar fields = %#v", got)
	}
	if len(got.Outputs) != 2 {
		t.Fatalf("Outputs length = %d, want 2", len(got.Outputs))
	}
	if got.Outputs[1].Name != "dev" || got.Outputs[1].Path != "/nix/store/demo-hello-dev" {
		t.Fatalf("Outputs[1] = %#v", got.Outputs[1])
	}
	if got.Meta.Description != "friendly greeting" || got.Meta.License != "GPL-3.0-or-later" {
		t.Fatalf("Meta = %#v", got.Meta)
	}
	if got.Skipped != "kept" {
		t.Fatalf("Skipped = %q, want kept", got.Skipped)
	}
}

func TestEvaluatorUnmarshalOptionalAndNullPointers(t *testing.T) {
	_, e := newTestEvaluator(t)

	type target struct {
		Name    string  `nix:"name" validate:"optional"`
		Alias   *string `nix:"alias" validate:"optional"`
		Missing string  `nix:"missing" validate:"optional"`
	}

	value, err := e.NewValue(eval.Attrs(map[string]eval.GoValue{
		"name":  eval.String("demo"),
		"alias": eval.Null(),
	}))
	if err != nil {
		t.Fatalf("Evaluator.NewValue(attrs) error = %v", err)
	}

	got := target{Missing: "unchanged"}
	if err := e.Unmarshal(value, &got); err != nil {
		t.Fatalf("Evaluator.Unmarshal(optional) error = %v", err)
	}
	if got.Name != "demo" {
		t.Fatalf("Name = %q, want demo", got.Name)
	}
	if got.Alias != nil {
		t.Fatalf("Alias = %q, want nil", *got.Alias)
	}
	if got.Missing != "unchanged" {
		t.Fatalf("Missing = %q, want unchanged", got.Missing)
	}
}

func TestEvaluatorUnmarshalRequiredMissing(t *testing.T) {
	_, e := newTestEvaluator(t)

	type target struct {
		Name string `nix:"name" validate:"required"`
	}

	value, err := e.NewValue(eval.Attrs(map[string]eval.GoValue{}))
	if err != nil {
		t.Fatalf("Evaluator.NewValue(attrs) error = %v", err)
	}

	var got target
	err = e.Unmarshal(value, &got)
	var missing *eval.MissingAttrError
	if !errors.As(err, &missing) {
		t.Fatalf("Evaluator.Unmarshal() error = %v, want MissingAttrError", err)
	}
	if missing.Attr != "name" || missing.Path != "$.name" {
		t.Fatalf("MissingAttrError = %#v, want attr name at $.name", missing)
	}
}

func TestEvaluatorUnmarshalTypeErrors(t *testing.T) {
	_, e := newTestEvaluator(t)

	type target struct {
		Name    string `nix:"name" validate:"required"`
		Outputs []struct {
			Name string `nix:"name" validate:"required"`
		} `nix:"outputs" validate:"required"`
	}

	value, err := e.NewValue(eval.Attrs(map[string]eval.GoValue{
		"name": eval.Int(7),
		"outputs": eval.List(eval.Attrs(map[string]eval.GoValue{
			"name": eval.String("out"),
		})),
	}))
	if err != nil {
		t.Fatalf("Evaluator.NewValue(attrs) error = %v", err)
	}

	var got target
	err = e.Unmarshal(value, &got)
	var typeErr *eval.UnmarshalTypeError
	if !errors.As(err, &typeErr) {
		t.Fatalf("Evaluator.Unmarshal() error = %v, want UnmarshalTypeError", err)
	}
	if typeErr.Path != "$.name" {
		t.Fatalf("UnmarshalTypeError.Path = %q, want $.name", typeErr.Path)
	}
	if !strings.Contains(err.Error(), "$.name") {
		t.Fatalf("error = %q, want path", err.Error())
	}
}

func TestEvaluatorUnmarshalInvalidTargets(t *testing.T) {
	_, e := newTestEvaluator(t)

	value, err := e.NewValue(eval.String("demo"))
	if err != nil {
		t.Fatalf("Evaluator.NewValue(string) error = %v", err)
	}

	var got string
	tests := []struct {
		name string
		out  any
	}{
		{name: "nil", out: nil},
		{name: "non_pointer", out: got},
		{name: "nil_pointer", out: (*string)(nil)},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := e.Unmarshal(value, tt.out)
			var invalid *eval.InvalidUnmarshalError
			if !errors.As(err, &invalid) {
				t.Fatalf("Evaluator.Unmarshal() error = %v, want InvalidUnmarshalError", err)
			}
		})
	}
}

func TestEvaluatorUnmarshalNumericAndArray(t *testing.T) {
	_, e := newTestEvaluator(t)

	type target struct {
		Ints   [2]int  `nix:"ints" validate:"required"`
		Count  uint8   `nix:"count" validate:"required"`
		Ratio  float32 `nix:"ratio" validate:"required"`
		Active bool    `nix:"active" validate:"required"`
	}

	value, err := e.NewValue(eval.Attrs(map[string]eval.GoValue{
		"ints":   eval.List(eval.Int(1), eval.Int(2)),
		"count":  eval.Int(9),
		"ratio":  eval.Float(1.5),
		"active": eval.Bool(true),
	}))
	if err != nil {
		t.Fatalf("Evaluator.NewValue(attrs) error = %v", err)
	}

	var got target
	if err := e.Unmarshal(value, &got); err != nil {
		t.Fatalf("Evaluator.Unmarshal(numeric) error = %v", err)
	}
	if got.Ints != [2]int{1, 2} || got.Count != 9 || got.Ratio != 1.5 || !got.Active {
		t.Fatalf("decoded target = %#v", got)
	}
}

func TestEvaluatorUnmarshalMap(t *testing.T) {
	_, e := newTestEvaluator(t)

	type output struct {
		Path string `nix:"path" validate:"required"`
	}
	type target struct {
		Outputs map[string]output `nix:"outputs" validate:"required"`
	}

	value, err := e.NewValue(eval.Attrs(map[string]eval.GoValue{
		"outputs": eval.Attrs(map[string]eval.GoValue{
			"dev": eval.Attrs(map[string]eval.GoValue{
				"path": eval.String("/nix/store/demo-dev"),
			}),
			"out": eval.Attrs(map[string]eval.GoValue{
				"path": eval.String("/nix/store/demo"),
			}),
		}),
	}))
	if err != nil {
		t.Fatalf("Evaluator.NewValue(attrs) error = %v", err)
	}

	var got target
	if err := e.Unmarshal(value, &got); err != nil {
		t.Fatalf("Evaluator.Unmarshal(map) error = %v", err)
	}
	if got.Outputs["out"].Path != "/nix/store/demo" || got.Outputs["dev"].Path != "/nix/store/demo-dev" {
		t.Fatalf("Outputs = %#v", got.Outputs)
	}
}

func TestEvaluatorUnmarshalMapRequiredMissing(t *testing.T) {
	_, e := newTestEvaluator(t)

	type output struct {
		Path string `nix:"path" validate:"required"`
	}
	type target struct {
		Outputs map[string]output `nix:"outputs" validate:"required"`
	}

	value, err := e.NewValue(eval.Attrs(map[string]eval.GoValue{
		"outputs": eval.Attrs(map[string]eval.GoValue{
			"out": eval.Attrs(map[string]eval.GoValue{}),
		}),
	}))
	if err != nil {
		t.Fatalf("Evaluator.NewValue(attrs) error = %v", err)
	}

	var got target
	err = e.Unmarshal(value, &got)
	var missing *eval.MissingAttrError
	if !errors.As(err, &missing) {
		t.Fatalf("Evaluator.Unmarshal(map missing) error = %v, want MissingAttrError", err)
	}
	if missing.Path != "$.outputs.out.path" {
		t.Fatalf("MissingAttrError.Path = %q, want $.outputs.out.path", missing.Path)
	}
}

func TestEvaluatorUnmarshalMapNonStringKey(t *testing.T) {
	_, e := newTestEvaluator(t)

	value, err := e.NewValue(eval.Attrs(map[string]eval.GoValue{}))
	if err != nil {
		t.Fatalf("Evaluator.NewValue(attrs) error = %v", err)
	}

	var got map[int]string
	err = e.Unmarshal(value, &got)
	var unsupported *eval.UnsupportedTypeError
	if !errors.As(err, &unsupported) {
		t.Fatalf("Evaluator.Unmarshal(map non-string key) error = %v, want UnsupportedTypeError", err)
	}
}

func TestEvaluatorUnmarshalLifecycleAndOriginValidation(t *testing.T) {
	r, e := newTestEvaluator(t)

	value, err := e.NewValue(eval.String("demo"))
	if err != nil {
		t.Fatalf("Evaluator.NewValue(string) error = %v", err)
	}
	if err := value.Close(); err != nil {
		t.Fatalf("Value.Close() error = %v", err)
	}
	var got string
	err = e.Unmarshal(value, &got)
	if err == nil {
		t.Fatal("Evaluator.Unmarshal(closed value) error = nil, want error")
	}

	s, err := r.OpenStore("dummy://")
	if err != nil {
		t.Fatalf("Runtime.OpenStore(second) error = %v", err)
	}
	e1, err := r.NewEvaluator(s)
	if err != nil {
		t.Fatalf("Runtime.NewEvaluator(e1) error = %v", err)
	}
	e2, err := r.NewEvaluator(s)
	if err != nil {
		t.Fatalf("Runtime.NewEvaluator(e2) error = %v", err)
	}
	foreign, err := e1.NewValue(eval.String("foreign"))
	if err != nil {
		t.Fatalf("Evaluator.NewValue(foreign) error = %v", err)
	}
	if err := e2.Unmarshal(foreign, &got); err == nil {
		t.Fatal("Evaluator.Unmarshal(foreign) error = nil, want error")
	}

	if err := e2.Close(); err != nil {
		t.Fatalf("Evaluator.Close() error = %v", err)
	}
	err = e2.Unmarshal(foreign, &got)
	if !errors.Is(err, gonix.ErrClosed) {
		t.Fatalf("Evaluator.Unmarshal(closed evaluator) error = %v, want ErrClosed", err)
	}
}
