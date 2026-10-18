package image

import (
	"fmt"
	"reflect"
)

// SchemaSource is the interface a struct implements to
// provide its own JSON Schema. The validator accepts a
// struct that implements this interface and uses the
// returned schema for validation.
type SchemaSource interface {
	Schema() any
}

// StructuredOutputValidator validates a JSON response
// against a JSON Schema (REQ-IMG-SOV-001 to 004). The
// validator is a small, hand-rolled implementation of
// the JSON Schema 2020-12 subset the project needs.
type StructuredOutputValidator struct {
	schema any
}

// NewStructuredOutputValidator returns a validator for
// the given schema. The schema can be a map (parsed
// JSON) or a struct that implements SchemaSource. The
// validator is a pure value; it is safe for concurrent
// use.
func NewStructuredOutputValidator(schema any) (*StructuredOutputValidator, error) {
	resolved, err := resolveSchema(schema)
	if err != nil {
		return nil, err
	}
	return &StructuredOutputValidator{schema: resolved}, nil
}

// resolveSchema returns the schema map. For a struct
// that implements SchemaSource, it calls Schema() and
// recurses on the result. For a map, it returns the
// map as-is. Anything else returns an error.
func resolveSchema(s any) (any, error) {
	switch v := s.(type) {
	case nil:
		return nil, fmt.Errorf("image: validator: schema is nil")
	case map[string]any:
		return v, nil
	case SchemaSource:
		return resolveSchema(v.Schema())
	}
	rv := reflect.ValueOf(s)
	if rv.Kind() == reflect.Pointer {
		rv = rv.Elem()
	}
	if rv.Kind() == reflect.Struct {
		if m, ok := rv.Interface().(SchemaSource); ok {
			return resolveSchema(m.Schema())
		}
	}
	return nil, fmt.Errorf("image: validator: schema must be a map or SchemaSource, got %T", s)
}

// Validate validates the response against the schema. The
// response is parsed as JSON when it is a string; otherwise
// it is taken as a Go value. The function returns nil on
// success and a *ValidationFailure on failure.
func (v *StructuredOutputValidator) Validate(response any) error {
	value := response
	// Parse JSON strings. Strings that are not valid JSON
	// are treated as plain string values.
	if s, ok := response.(string); ok {
		parsed, err := parseJSON(s)
		if err == nil {
			value = parsed
		}
	}
	value, err := coerceValue(value)
	if err != nil {
		return &ValidationFailure{Reason: err.Error()}
	}
	// Coerce again on the result so maps/slices built
	// with int or other numeric types become float64.
	value, _ = coerceValue(value)
	if err := validateValue(value, v.schema, ""); err != nil {
		return &ValidationFailure{Reason: err.Error()}
	}
	return nil
}

// coerceValue turns the response into a map/slice/scalar
// the validator can walk. The function recurses into
// maps and slices so that numeric Go types (int, int64,
// float32) become float64. The function returns the
// value unchanged when the value is already one of the
// scalar JSON types.
func coerceValue(response any) (any, error) {
	switch v := response.(type) {
	case nil:
		return nil, nil
	case string, float64, bool:
		return v, nil
	}
	rv := reflect.ValueOf(response)
	if rv.Kind() == reflect.Pointer {
		rv = rv.Elem()
	}
	switch rv.Kind() {
	case reflect.Map:
		out := make(map[string]any, rv.Len())
		iter := rv.MapRange()
		for iter.Next() {
			k := iter.Key().String()
			ev, _ := coerceValue(iter.Value().Interface())
			out[k] = ev
		}
		return out, nil
	case reflect.Slice, reflect.Array:
		out := make([]any, 0, rv.Len())
		for i := 0; i < rv.Len(); i++ {
			ev, _ := coerceValue(rv.Index(i).Interface())
			out = append(out, ev)
		}
		return out, nil
	case reflect.Struct:
		out := make(map[string]any, rv.NumField())
		rt := rv.Type()
		for i := 0; i < rt.NumField(); i++ {
			f := rt.Field(i)
			if !f.IsExported() {
				continue
			}
			tag := f.Tag.Get("json")
			name := tag
			if comma := indexByte(tag, ','); comma >= 0 {
				name = tag[:comma]
			}
			if name == "" || name == "-" {
				name = f.Name
			}
			ev, _ := coerceValue(rv.Field(i).Interface())
			out[name] = ev
		}
		return out, nil
	case reflect.String:
		return rv.String(), nil
	case reflect.Int, reflect.Int64, reflect.Int32:
		return float64(rv.Int()), nil
	case reflect.Float32, reflect.Float64:
		return rv.Float(), nil
	case reflect.Bool:
		return rv.Bool(), nil
	}
	return nil, fmt.Errorf("unsupported response type %T", response)
}

func indexByte(s string, c byte) int {
	for i := 0; i < len(s); i++ {
		if s[i] == c {
			return i
		}
	}
	return -1
}

// validateValue walks the value and the schema in
// parallel. The path argument is the JSON path of the
// current value, used for error messages.
func validateValue(value any, schema any, path string) error {
	m, ok := schema.(map[string]any)
	if !ok {
		// Subschemas in non-map form are not supported.
		return fmt.Errorf("%s: schema must be a map", path)
	}
	if enumRaw, ok := m["enum"]; ok {
		if !matchesEnum(value, enumRaw) {
			return fmt.Errorf("%s: not in enum", path)
		}
	}
	if typeStr, ok := m["type"].(string); ok {
		if err := validateType(value, typeStr, path); err != nil {
			return err
		}
	}
	switch v := value.(type) {
	case map[string]any:
		if err := validateObject(v, m, path); err != nil {
			return err
		}
	case []any:
		if err := validateArray(v, m, path); err != nil {
			return err
		}
	case string:
		if err := validateString(v, m, path); err != nil {
			return err
		}
	case float64:
		if err := validateNumber(v, m, path); err != nil {
			return err
		}
	}
	return nil
}

// toFloat returns the value as a float64 when the value
// is a Go numeric type. The function returns (n, true)
// when the value is a number; (0, false) otherwise.
func toFloat(v any) (float64, bool) {
	switch n := v.(type) {
	case float64:
		return n, true
	case float32:
		return float64(n), true
	case int:
		return float64(n), true
	case int32:
		return float64(n), true
	case int64:
		return float64(n), true
	}
	return 0, false
}

func validateObject(v map[string]any, schema map[string]any, path string) error {
	for _, req := range stringList(schema["required"]) {
		if _, ok := v[req]; !ok {
			return fmt.Errorf("%s: missing required field %q", path, req)
		}
	}
	if props, ok := schema["properties"].(map[string]any); ok {
		for k, subSchema := range props {
			if sub, ok := v[k]; ok {
				if err := validateValue(sub, subSchema, joinPath(path, k)); err != nil {
					return err
				}
			}
		}
	}
	if addl, ok := schema["additionalProperties"]; ok {
		if ap, ok := addl.(bool); ok && !ap {
			props := keySet(schema["properties"])
			for k := range v {
				if _, allowed := props[k]; !allowed {
					return fmt.Errorf("%s: unexpected field %q", path, k)
				}
			}
		}
	}
	return nil
}

func validateArray(v []any, schema map[string]any, path string) error {
	if minR, ok := schema["minItems"]; ok {
		if n, ok := toFloat(minR); ok && float64(len(v)) < n {
			return fmt.Errorf("%s: array has %d items, min %v", path, len(v), n)
		}
	}
	if maxR, ok := schema["maxItems"]; ok {
		if n, ok := toFloat(maxR); ok && float64(len(v)) > n {
			return fmt.Errorf("%s: array has %d items, max %v", path, len(v), n)
		}
	}
	if items, ok := schema["items"]; ok {
		for i, item := range v {
			if err := validateValue(item, items, fmt.Sprintf("%s[%d]", path, i)); err != nil {
				return err
			}
		}
	}
	return nil
}

func validateString(v string, schema map[string]any, path string) error {
	if mn, ok := schema["minLength"]; ok {
		if n, ok := toFloat(mn); ok && float64(len(v)) < n {
			return fmt.Errorf("%s: length %d < minLength %v", path, len(v), n)
		}
	}
	if mx, ok := schema["maxLength"]; ok {
		if n, ok := toFloat(mx); ok && float64(len(v)) > n {
			return fmt.Errorf("%s: length %d > maxLength %v", path, len(v), n)
		}
	}
	return nil
}

func validateNumber(v float64, schema map[string]any, path string) error {
	if mn, ok := schema["minimum"]; ok {
		if n, ok := toFloat(mn); ok && v < n {
			return fmt.Errorf("%s: %v < minimum %v", path, v, n)
		}
	}
	if mx, ok := schema["maximum"]; ok {
		if n, ok := toFloat(mx); ok && v > n {
			return fmt.Errorf("%s: %v > maximum %v", path, v, n)
		}
	}
	return nil
}

func validateType(value any, typeStr string, path string) error {
	switch typeStr {
	case "object":
		if _, ok := value.(map[string]any); !ok {
			return fmt.Errorf("%s: not an object", path)
		}
	case "array":
		if _, ok := value.([]any); !ok {
			return fmt.Errorf("%s: not an array", path)
		}
	case "string":
		if _, ok := value.(string); !ok {
			return fmt.Errorf("%s: not a string", path)
		}
	case "number":
		if _, ok := value.(float64); !ok {
			return fmt.Errorf("%s: not a number", path)
		}
	case "integer":
		if _, ok := value.(float64); !ok {
			return fmt.Errorf("%s: not an integer", path)
		}
	case "boolean":
		if _, ok := value.(bool); !ok {
			return fmt.Errorf("%s: not a boolean", path)
		}
	case "null":
		if value != nil {
			return fmt.Errorf("%s: not null", path)
		}
	}
	return nil
}

func matchesEnum(value any, enum any) bool {
	list, ok := enum.([]any)
	if !ok {
		return false
	}
	for _, e := range list {
		if reflect.DeepEqual(value, e) {
			return true
		}
	}
	return false
}

func stringList(v any) []string {
	list, ok := v.([]any)
	if !ok {
		return nil
	}
	out := make([]string, 0, len(list))
	for _, item := range list {
		if s, ok := item.(string); ok {
			out = append(out, s)
		}
	}
	return out
}

func keySet(v any) map[string]struct{} {
	m, ok := v.(map[string]any)
	if !ok {
		return map[string]struct{}{}
	}
	out := make(map[string]struct{}, len(m))
	for k := range m {
		out[k] = struct{}{}
	}
	return out
}

func joinPath(prefix, key string) string {
	if prefix == "" {
		return key
	}
	return prefix + "." + key
}
