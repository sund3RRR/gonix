package eval

import (
	"fmt"
	"math"
	"reflect"
)

func goValueFromAny(value any, path string) (GoValue, error) {
	if value == nil {
		return Null(), nil
	}

	return goValueFromReflect(reflect.ValueOf(value), path)
}

func goValueFromReflect(value reflect.Value, path string) (GoValue, error) {
	for value.Kind() == reflect.Interface {
		if value.IsNil() {
			return Null(), nil
		}
		value = value.Elem()
	}

	switch value.Kind() {
	case reflect.Bool:
		return Bool(value.Bool()), nil
	case reflect.String:
		return String(value.String()), nil
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return Int(value.Int()), nil
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
		unsigned := value.Uint()
		if unsigned > math.MaxInt64 {
			return nil, fmt.Errorf("eval: unsigned integer %d overflows Nix integer at %s", unsigned, path)
		}
		return Int(int64(unsigned)), nil
	case reflect.Float32, reflect.Float64:
		return Float(value.Float()), nil
	case reflect.Slice:
		items := make([]GoValue, value.Len())
		for i := range value.Len() {
			item, err := goValueFromReflect(value.Index(i), indexPath(path, uint32(i)))
			if err != nil {
				return nil, err
			}
			items[i] = item
		}
		return List(items...), nil
	case reflect.Map:
		if value.Type().Key().Kind() != reflect.String {
			return nil, &UnsupportedTypeError{Type: value.Type(), Path: path}
		}

		attrs := make(map[string]GoValue, value.Len())
		iter := value.MapRange()
		for iter.Next() {
			name := iter.Key().String()
			item, err := goValueFromReflect(iter.Value(), attrPath(path, name))
			if err != nil {
				return nil, err
			}
			attrs[name] = item
		}
		return Attrs(attrs), nil
	default:
		return nil, &UnsupportedTypeError{Type: value.Type(), Path: path}
	}
}
