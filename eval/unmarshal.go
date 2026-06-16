package eval

import (
	"fmt"
	"reflect"
	"strconv"
	"strings"
	"unicode"

	"github.com/sund3RRR/gonix/internal/status"
)

// Unmarshal decodes a Nix value into the Go value pointed to by out.
//
// Struct fields are matched with the nix tag, or with the exact Go field name
// when no nix tag is present. Fields tagged nix:"-" are skipped. A missing
// field tagged validate:"required" returns a MissingAttrError; other missing
// fields are left unchanged. Extra Nix attributes are ignored.
func (e *Evaluator) Unmarshal(v *Value, out any) error {
	if e.state == nil {
		return status.ErrClosed
	}

	if v == nil {
		return &InvalidUnmarshalError{}
	}

	if err := e.validateValue(v); err != nil {
		return fmt.Errorf("eval: failed to validate value: %w", err)
	}

	target := reflect.ValueOf(out)
	if !target.IsValid() {
		return &InvalidUnmarshalError{}
	}
	if target.Kind() != reflect.Pointer || target.IsNil() {
		return &InvalidUnmarshalError{Type: target.Type()}
	}

	return e.unmarshalValue(v, target.Elem(), "$")
}

func (e *Evaluator) unmarshalValue(v *Value, target reflect.Value, path string) error {
	if !target.CanSet() {
		return &UnsupportedTypeError{Type: target.Type(), Path: path}
	}

	if err := e.Force(v); err != nil {
		return fmt.Errorf("eval: failed to force value at %s: %w", path, err)
	}

	typ, err := v.Type()
	if err != nil {
		return fmt.Errorf("eval: failed to get value type at %s: %w", path, err)
	}

	if typ == ValueTypeNull {
		if target.Kind() == reflect.Pointer {
			target.SetZero()
			return nil
		}
		return &UnmarshalTypeError{Value: typ.String(), Type: target.Type(), Path: path}
	}

	if target.Kind() == reflect.Pointer {
		if target.IsNil() {
			target.Set(reflect.New(target.Type().Elem()))
		}
		return e.unmarshalValue(v, target.Elem(), path)
	}

	switch target.Kind() {
	case reflect.Struct:
		return e.unmarshalStruct(v, typ, target, path)
	case reflect.Slice:
		return e.unmarshalSlice(v, typ, target, path)
	case reflect.Array:
		return e.unmarshalArray(v, typ, target, path)
	case reflect.String:
		return e.unmarshalString(v, typ, target, path)
	case reflect.Bool:
		return e.unmarshalBool(v, typ, target, path)
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return e.unmarshalInt(v, typ, target, path)
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
		return e.unmarshalUint(v, typ, target, path)
	case reflect.Float32, reflect.Float64:
		return e.unmarshalFloat(v, typ, target, path)
	default:
		return &UnsupportedTypeError{Type: target.Type(), Path: path}
	}
}

func (e *Evaluator) unmarshalStruct(v *Value, typ ValueType, target reflect.Value, path string) error {
	if typ != ValueTypeAttrs {
		return &UnmarshalTypeError{Value: typ.String(), Type: target.Type(), Path: path}
	}

	targetType := target.Type()
	for i := 0; i < target.NumField(); i++ {
		fieldType := targetType.Field(i)
		if fieldType.PkgPath != "" {
			continue
		}

		attr, skip := fieldAttr(fieldType)
		if skip {
			continue
		}

		has, err := e.HasAttr(v, attr)
		if err != nil {
			return fmt.Errorf("eval: failed to check attr %q at %s: %w", attr, path, err)
		}
		fieldPath := attrPath(path, attr)
		if !has {
			if fieldRequired(fieldType) {
				return &MissingAttrError{Attr: attr, Path: fieldPath}
			}
			continue
		}

		child, err := e.Attr(v, attr)
		if err != nil {
			return fmt.Errorf("eval: failed to get attr %q at %s: %w", attr, path, err)
		}
		err = e.unmarshalValue(child, target.Field(i), fieldPath)
		if closeErr := child.Close(); closeErr != nil && err == nil {
			err = fmt.Errorf("eval: failed to close attr %q at %s: %w", attr, path, closeErr)
		}
		if err != nil {
			return err
		}
	}

	return nil
}

func (e *Evaluator) unmarshalSlice(v *Value, typ ValueType, target reflect.Value, path string) error {
	if typ != ValueTypeList {
		return &UnmarshalTypeError{Value: typ.String(), Type: target.Type(), Path: path}
	}

	length, err := v.ListLen()
	if err != nil {
		return fmt.Errorf("eval: failed to get list size at %s: %w", path, err)
	}

	items := reflect.MakeSlice(target.Type(), int(length), int(length))
	for i := uint32(0); i < length; i++ {
		if err := e.unmarshalIndex(v, items.Index(int(i)), i, indexPath(path, i)); err != nil {
			return err
		}
	}
	target.Set(items)
	return nil
}

func (e *Evaluator) unmarshalArray(v *Value, typ ValueType, target reflect.Value, path string) error {
	if typ != ValueTypeList {
		return &UnmarshalTypeError{Value: typ.String(), Type: target.Type(), Path: path}
	}

	length, err := v.ListLen()
	if err != nil {
		return fmt.Errorf("eval: failed to get list size at %s: %w", path, err)
	}
	if int(length) != target.Len() {
		return &UnmarshalTypeError{Value: fmt.Sprintf("list of length %d", length), Type: target.Type(), Path: path}
	}

	for i := uint32(0); i < length; i++ {
		if err := e.unmarshalIndex(v, target.Index(int(i)), i, indexPath(path, i)); err != nil {
			return err
		}
	}
	return nil
}

func (e *Evaluator) unmarshalIndex(v *Value, target reflect.Value, index uint32, path string) error {
	child, err := e.Index(v, index)
	if err != nil {
		return fmt.Errorf("eval: failed to get list item %d at %s: %w", index, path, err)
	}
	err = e.unmarshalValue(child, target, path)
	if closeErr := child.Close(); closeErr != nil && err == nil {
		err = fmt.Errorf("eval: failed to close list item %d at %s: %w", index, path, closeErr)
	}
	return err
}

func (e *Evaluator) unmarshalString(v *Value, typ ValueType, target reflect.Value, path string) error {
	switch typ {
	case ValueTypeString:
		got, err := v.String()
		if err != nil {
			return fmt.Errorf("eval: failed to get string at %s: %w", path, err)
		}
		target.SetString(got)
		return nil
	case ValueTypePath:
		got, err := v.PathString()
		if err != nil {
			return fmt.Errorf("eval: failed to get path string at %s: %w", path, err)
		}
		target.SetString(got)
		return nil
	default:
		return &UnmarshalTypeError{Value: typ.String(), Type: target.Type(), Path: path}
	}
}

func (e *Evaluator) unmarshalBool(v *Value, typ ValueType, target reflect.Value, path string) error {
	if typ != ValueTypeBool {
		return &UnmarshalTypeError{Value: typ.String(), Type: target.Type(), Path: path}
	}
	got, err := v.Bool()
	if err != nil {
		return fmt.Errorf("eval: failed to get bool at %s: %w", path, err)
	}
	target.SetBool(got)
	return nil
}

func (e *Evaluator) unmarshalInt(v *Value, typ ValueType, target reflect.Value, path string) error {
	if typ != ValueTypeInt {
		return &UnmarshalTypeError{Value: typ.String(), Type: target.Type(), Path: path}
	}
	got, err := v.Int()
	if err != nil {
		return fmt.Errorf("eval: failed to get int at %s: %w", path, err)
	}
	if target.OverflowInt(got) {
		return &UnmarshalTypeError{Value: "integer " + strconv.FormatInt(got, 10), Type: target.Type(), Path: path}
	}
	target.SetInt(got)
	return nil
}

func (e *Evaluator) unmarshalUint(v *Value, typ ValueType, target reflect.Value, path string) error {
	if typ != ValueTypeInt {
		return &UnmarshalTypeError{Value: typ.String(), Type: target.Type(), Path: path}
	}
	got, err := v.Int()
	if err != nil {
		return fmt.Errorf("eval: failed to get int at %s: %w", path, err)
	}
	if got < 0 || target.OverflowUint(uint64(got)) {
		return &UnmarshalTypeError{Value: "integer " + strconv.FormatInt(got, 10), Type: target.Type(), Path: path}
	}
	target.SetUint(uint64(got))
	return nil
}

func (e *Evaluator) unmarshalFloat(v *Value, typ ValueType, target reflect.Value, path string) error {
	if typ != ValueTypeFloat {
		return &UnmarshalTypeError{Value: typ.String(), Type: target.Type(), Path: path}
	}
	got, err := v.Float()
	if err != nil {
		return fmt.Errorf("eval: failed to get float at %s: %w", path, err)
	}
	if target.OverflowFloat(got) {
		return &UnmarshalTypeError{Value: "float " + strconv.FormatFloat(got, 'g', -1, 64), Type: target.Type(), Path: path}
	}
	target.SetFloat(got)
	return nil
}

func fieldAttr(field reflect.StructField) (string, bool) {
	tag := field.Tag.Get("nix")
	if tag == "-" {
		return "", true
	}
	if comma := strings.IndexByte(tag, ','); comma >= 0 {
		tag = tag[:comma]
	}
	if tag == "" {
		return field.Name, false
	}
	return tag, false
}

func fieldRequired(field reflect.StructField) bool {
	for _, item := range strings.Split(field.Tag.Get("validate"), ",") {
		if strings.TrimSpace(item) == "required" {
			return true
		}
	}
	return false
}

func attrPath(parent, attr string) string {
	if isPathIdent(attr) {
		return parent + "." + attr
	}
	return parent + "[" + strconv.Quote(attr) + "]"
}

func indexPath(parent string, index uint32) string {
	return parent + "[" + strconv.FormatUint(uint64(index), 10) + "]"
}

func isPathIdent(value string) bool {
	if value == "" {
		return false
	}
	for i, r := range value {
		if i == 0 {
			if r != '_' && !unicode.IsLetter(r) {
				return false
			}
			continue
		}
		if r != '_' && !unicode.IsLetter(r) && !unicode.IsDigit(r) {
			return false
		}
	}
	return true
}
