package template

import (
	"fmt"
	htmltemplate "html/template"
	"reflect"
)

// BaseFuncMap returns the default functions injected into every compiled
// template set. The registry may merge ExtraFuncs on top.
func BaseFuncMap() htmltemplate.FuncMap {
	return htmltemplate.FuncMap{
		"raw":     rawFn,
		"dict":    dictFn,
		"list":    listFn,
		"default": defaultFn,
	}
}

// rawFn marks content as trusted HTML, disabling auto-escaping.
// SECURITY: only pass pre-sanitized / trusted content. User-controlled
// input must never flow through raw without sanitization — it enables XSS.
func rawFn(v any) htmltemplate.HTML {
	if v == nil {
		return ""
	}
	if h, ok := v.(htmltemplate.HTML); ok {
		return h
	}
	//nolint:gosec // G203: raw() deliberately opts out of auto-escape for trusted HTML only
	return htmltemplate.HTML(fmt.Sprint(v))
}

// dictFn builds a map from alternating key/value pairs.
func dictFn(pairs ...any) (map[string]any, error) {
	if len(pairs)%2 != 0 {
		return nil, fmt.Errorf("dict: odd number of arguments")
	}
	out := make(map[string]any, len(pairs)/2)
	for i := 0; i+1 < len(pairs); i += 2 {
		key, ok := pairs[i].(string)
		if !ok {
			return nil, fmt.Errorf("dict: key %d is not a string", i)
		}
		out[key] = pairs[i+1]
	}
	return out, nil
}

// listFn collects arguments into a slice.
func listFn(items ...any) []any {
	if items == nil {
		return []any{}
	}
	out := make([]any, len(items))
	copy(out, items)
	return out
}

// defaultFn returns fallback when value is empty/zero, otherwise value.
// Usage in templates: {{ default "Guest" .User.Name }}
func defaultFn(fallback, value any) any {
	if isEmptyValue(value) {
		return fallback
	}
	return value
}

// isEmptyValue reports whether v is nil or a "blank" value (empty string,
// false, numeric zero, empty slice/map/array).
func isEmptyValue(v any) bool {
	if v == nil {
		return true
	}
	rv := reflect.ValueOf(v)
	switch rv.Kind() {
	case reflect.Invalid:
		return true
	case reflect.String, reflect.Array, reflect.Slice, reflect.Map, reflect.Chan:
		return rv.Len() == 0
	case reflect.Bool:
		return !rv.Bool()
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return rv.Int() == 0
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
		return rv.Uint() == 0
	case reflect.Float32, reflect.Float64:
		return rv.Float() == 0
	case reflect.Complex64, reflect.Complex128:
		return rv.Complex() == 0
	case reflect.Interface, reflect.Pointer:
		if rv.IsNil() {
			return true
		}
		return isEmptyValue(rv.Elem().Interface())
	case reflect.Func:
		return rv.IsNil()
	default:
		return false
	}
}
