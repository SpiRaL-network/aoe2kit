// Package jsonout is the single JSON boundary for user-facing command output.
// It preserves ordinary encoding/json output and makes non-finite floats
// explicit diagnostic strings instead of failing the entire command.
package jsonout

import (
	"encoding/json"
	"fmt"
	"io"
	"math"
	"os"
	"reflect"
	"strings"
)

// Print writes indented JSON to stdout.
func Print(value any) error { return Encode(os.Stdout, value) }

// Encode writes indented JSON to w. When non-finite floats are found, they are
// rendered as "NaN", "Infinity", or "-Infinity" and the object receives a
// nonfinite_count field. The command still succeeds because the value is data,
// not a transport error.
func Encode(w io.Writer, value any) error {
	sanitized, count, firstPath := sanitize(value)
	if count == 0 {
		enc := json.NewEncoder(w)
		enc.SetIndent("", "  ")
		return enc.Encode(value)
	}
	if object, ok := sanitized.(map[string]any); ok {
		object["nonfinite_count"] = count
	}
	data, err := json.MarshalIndent(sanitized, "", "  ")
	if err != nil {
		return err
	}
	if _, err := w.Write(append(data, '\n')); err != nil {
		return err
	}
	fmt.Fprintf(os.Stderr, "kit: %d non-finite floats replaced (first at %s)\n", count, firstPath)
	return nil
}

// Sanitize converts non-finite float32/float64 values to explicit strings and
// returns the replacement count. It is exported so command/report tests can
// verify the boundary without invoking a subprocess.
func Sanitize(value any) (any, int) {
	sanitized, count, _ := sanitize(value)
	return sanitized, count
}

func sanitize(value any) (any, int, string) {
	count := 0
	firstPath := "value"
	result := sanitizeValue(reflect.ValueOf(value), "value", &count, &firstPath)
	return result, count, firstPath
}

func sanitizeValue(value reflect.Value, path string, count *int, firstPath *string) any {
	if !value.IsValid() {
		return nil
	}
	if value.Kind() == reflect.Interface || value.Kind() == reflect.Pointer {
		if value.IsNil() {
			return nil
		}
		return sanitizeValue(value.Elem(), path, count, firstPath)
	}
	switch value.Kind() {
	case reflect.Float32, reflect.Float64:
		f := value.Float()
		if math.IsNaN(f) {
			*count++
			if *count == 1 {
				*firstPath = path
			}
			return "NaN"
		}
		if math.IsInf(f, 1) {
			*count++
			if *count == 1 {
				*firstPath = path
			}
			return "Infinity"
		}
		if math.IsInf(f, -1) {
			*count++
			if *count == 1 {
				*firstPath = path
			}
			return "-Infinity"
		}
		return value.Interface()
	case reflect.Struct:
		out := map[string]any{}
		typ := value.Type()
		for i := 0; i < value.NumField(); i++ {
			field := typ.Field(i)
			if field.PkgPath != "" {
				continue
			}
			name, omit := jsonFieldName(field)
			if name == "-" || (omit && isEmpty(value.Field(i))) {
				continue
			}
			out[name] = sanitizeValue(value.Field(i), joinPath(path, name), count, firstPath)
		}
		return out
	case reflect.Map:
		if value.IsNil() {
			return nil
		}
		out := map[string]any{}
		iter := value.MapRange()
		for iter.Next() {
			key := fmt.Sprint(iter.Key().Interface())
			out[key] = sanitizeValue(iter.Value(), joinPath(path, key), count, firstPath)
		}
		return out
	case reflect.Slice, reflect.Array:
		out := make([]any, value.Len())
		for i := 0; i < value.Len(); i++ {
			out[i] = sanitizeValue(value.Index(i), fmt.Sprintf("%s[%d]", path, i), count, firstPath)
		}
		return out
	default:
		return value.Interface()
	}
}

func jsonFieldName(field reflect.StructField) (string, bool) {
	tag := field.Tag.Get("json")
	if tag == "" {
		return field.Name, false
	}
	parts := strings.Split(tag, ",")
	if parts[0] == "" {
		return field.Name, len(parts) > 1 && contains(parts[1:], "omitempty")
	}
	return parts[0], contains(parts[1:], "omitempty")
}

func contains(values []string, needle string) bool {
	for _, value := range values {
		if value == needle {
			return true
		}
	}
	return false
}

func isEmpty(value reflect.Value) bool {
	if !value.IsValid() {
		return true
	}
	switch value.Kind() {
	case reflect.Array, reflect.Map, reflect.Slice, reflect.String:
		return value.Len() == 0
	case reflect.Bool:
		return !value.Bool()
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return value.Int() == 0
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
		return value.Uint() == 0
	case reflect.Float32, reflect.Float64:
		return value.Float() == 0
	case reflect.Interface, reflect.Pointer:
		return value.IsNil()
	}
	return false
}

func joinPath(parent, child string) string {
	if parent == "value" {
		return child
	}
	return parent + "." + child
}
