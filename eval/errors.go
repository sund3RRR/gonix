package eval

import (
	"fmt"
	"reflect"
)

// ValueTypeError describes a Value getter called for the wrong Nix type.
type ValueTypeError struct {
	// Actual is the value's runtime Nix type.
	Actual ValueType

	// Expected is the Nix type required by the getter.
	Expected ValueType
}

func (e *ValueTypeError) Error() string {
	return fmt.Sprintf("eval: cannot read %s value as %s", e.Actual, e.Expected)
}

// InvalidUnmarshalError describes an invalid target passed to Evaluator.Unmarshal.
type InvalidUnmarshalError struct {
	Type reflect.Type
}

func (e *InvalidUnmarshalError) Error() string {
	if e.Type == nil {
		return "eval: Unmarshal(nil)"
	}
	if e.Type.Kind() != reflect.Pointer {
		return fmt.Sprintf("eval: Unmarshal(non-pointer %s)", e.Type.String())
	}
	return fmt.Sprintf("eval: Unmarshal(nil %s)", e.Type.String())
}

// UnmarshalTypeError describes a Nix value that cannot be decoded into a Go type.
type UnmarshalTypeError struct {
	Value string
	Type  reflect.Type
	Path  string
}

func (e *UnmarshalTypeError) Error() string {
	if e.Path == "" {
		return fmt.Sprintf("eval: cannot unmarshal %s into Go value of type %s", e.Value, e.Type.String())
	}
	return fmt.Sprintf("eval: cannot unmarshal %s into Go value of type %s at %s", e.Value, e.Type.String(), e.Path)
}

// MissingAttrError describes a required Nix attribute that was absent.
type MissingAttrError struct {
	Attr string
	Path string
}

func (e *MissingAttrError) Error() string {
	if e.Path == "" {
		return fmt.Sprintf("eval: missing required attr %q", e.Attr)
	}
	return fmt.Sprintf("eval: missing required attr %q at %s", e.Attr, e.Path)
}

// UnsupportedTypeError describes a Go type that an Evaluator cannot convert
// to or from a Nix value.
type UnsupportedTypeError struct {
	Type reflect.Type
	Path string
}

func (e *UnsupportedTypeError) Error() string {
	if e.Path == "" {
		return fmt.Sprintf("eval: unsupported Go type %s", e.Type.String())
	}
	return fmt.Sprintf("eval: unsupported Go type %s at %s", e.Type.String(), e.Path)
}
