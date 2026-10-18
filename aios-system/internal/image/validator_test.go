package image

import (
	"errors"
	"testing"
)

func TestValidator_PrimitiveTypes(t *testing.T) {
	v, err := NewStructuredOutputValidator(map[string]any{"type": "string"})
	if err != nil {
		t.Fatal(err)
	}
	if err := v.Validate("hi"); err != nil {
		t.Fatalf("string: %v", err)
	}
	if err := v.Validate(42); !errors.Is(err, ErrValidationFailure) {
		t.Fatalf("want failure, got %v", err)
	}
}

func TestValidator_Object_Required(t *testing.T) {
	schema := map[string]any{
		"type":     "object",
		"required": []any{"name", "age"},
		"properties": map[string]any{
			"name": map[string]any{"type": "string"},
			"age":  map[string]any{"type": "number"},
		},
	}
	v, _ := NewStructuredOutputValidator(schema)
	if err := v.Validate(map[string]any{"name": "a", "age": 1}); err != nil {
		t.Fatalf("good: %v", err)
	}
	if err := v.Validate(map[string]any{"name": "a"}); err == nil {
		t.Fatal("expected error for missing age")
	}
}

func TestValidator_AdditionalProperties(t *testing.T) {
	schema := map[string]any{
		"type": "object",
		"properties": map[string]any{
			"a": map[string]any{"type": "string"},
		},
		"additionalProperties": false,
	}
	v, _ := NewStructuredOutputValidator(schema)
	if err := v.Validate(map[string]any{"a": "x"}); err != nil {
		t.Fatalf("good: %v", err)
	}
	if err := v.Validate(map[string]any{"a": "x", "b": "y"}); err == nil {
		t.Fatal("expected error for extra field")
	}
}

func TestValidator_Array(t *testing.T) {
	schema := map[string]any{
		"type":     "array",
		"minItems": 1,
		"maxItems": 3,
		"items":    map[string]any{"type": "string"},
	}
	v, _ := NewStructuredOutputValidator(schema)
	if err := v.Validate([]any{"a", "b"}); err != nil {
		t.Fatalf("good: %v", err)
	}
	if err := v.Validate([]any{}); err == nil {
		t.Fatal("min items")
	}
	if err := v.Validate([]any{"a", "b", "c", "d"}); err == nil {
		t.Fatal("max items")
	}
	if err := v.Validate([]any{"a", 1}); err == nil {
		t.Fatal("item type")
	}
}

func TestValidator_String(t *testing.T) {
	schema := map[string]any{
		"type":      "string",
		"minLength": 2,
		"maxLength": 4,
	}
	v, _ := NewStructuredOutputValidator(schema)
	if err := v.Validate("ab"); err != nil {
		t.Fatalf("good: %v", err)
	}
	if err := v.Validate("a"); err == nil {
		t.Fatal("min length")
	}
	if err := v.Validate("abcde"); err == nil {
		t.Fatal("max length")
	}
}

func TestValidator_Number(t *testing.T) {
	schema := map[string]any{
		"type":    "number",
		"minimum": 1,
		"maximum": 9,
	}
	v, _ := NewStructuredOutputValidator(schema)
	if err := v.Validate(5.0); err != nil {
		t.Fatalf("good: %v", err)
	}
	if err := v.Validate(0); err == nil {
		t.Fatal("min")
	}
	if err := v.Validate(10); err == nil {
		t.Fatal("max")
	}
}

func TestValidator_Enum(t *testing.T) {
	schema := map[string]any{
		"type": "string",
		"enum": []any{"red", "green", "blue"},
	}
	v, _ := NewStructuredOutputValidator(schema)
	if err := v.Validate("red"); err != nil {
		t.Fatalf("good: %v", err)
	}
	if err := v.Validate("yellow"); err == nil {
		t.Fatal("not in enum")
	}
}

func TestValidator_Nested(t *testing.T) {
	schema := map[string]any{
		"type": "object",
		"properties": map[string]any{
			"user": map[string]any{
				"type": "object",
				"properties": map[string]any{
					"age": map[string]any{"type": "number", "minimum": 0},
				},
			},
		},
	}
	v, _ := NewStructuredOutputValidator(schema)
	if err := v.Validate(map[string]any{"user": map[string]any{"age": 30}}); err != nil {
		t.Fatalf("good: %v", err)
	}
	if err := v.Validate(map[string]any{"user": map[string]any{"age": -1}}); err == nil {
		t.Fatal("nested violation")
	}
}

func TestValidator_StringResponse(t *testing.T) {
	v, _ := NewStructuredOutputValidator(map[string]any{"type": "object"})
	if err := v.Validate(`{"a":1}`); err != nil {
		t.Fatalf("good: %v", err)
	}
	if err := v.Validate("not json"); err == nil {
		t.Fatal("bad json")
	}
	if err := v.Validate(""); err == nil {
		t.Fatal("empty")
	}
}

func TestValidator_StructResponse(t *testing.T) {
	type S struct {
		Name string `json:"name"`
		Age  int    `json:"age"`
	}
	v, _ := NewStructuredOutputValidator(map[string]any{
		"type":     "object",
		"required": []any{"name"},
		"properties": map[string]any{
			"name": map[string]any{"type": "string"},
			"age":  map[string]any{"type": "number"},
		},
	})
	if err := v.Validate(S{Name: "a", Age: 1}); err != nil {
		t.Fatalf("good: %v", err)
	}
	// When the struct omits Name, the JSON form is
	// {"name":"","age":1}; the required check sees the
	// key as present. We verify that the property type
	// check still applies by using a missing key in a map.
	if err := v.Validate(map[string]any{"age": 1}); err == nil {
		t.Fatal("missing name in map")
	}
}

func TestValidator_NilSchema(t *testing.T) {
	_, err := NewStructuredOutputValidator(nil)
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestValidator_BadSchema(t *testing.T) {
	_, err := NewStructuredOutputValidator(42)
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestValidator_SchemaSource_NotASource(t *testing.T) {
	type S struct{ X int }
	_, err := NewStructuredOutputValidator(S{X: 1})
	if err == nil {
		t.Fatal("expected error: struct without Schema() is not a valid schema")
	}
}

func TestValidator_NonObjectResponse(t *testing.T) {
	v, _ := NewStructuredOutputValidator(map[string]any{"type": "object"})
	if err := v.Validate([]any{1, 2, 3}); err == nil {
		t.Fatal("array is not an object")
	}
}

func TestValidator_IntegerType(t *testing.T) {
	v, _ := NewStructuredOutputValidator(map[string]any{"type": "integer"})
	if err := v.Validate(5.0); err != nil {
		t.Fatalf("good: %v", err)
	}
	if err := v.Validate("x"); err == nil {
		t.Fatal("string is not integer")
	}
}

func TestValidator_BooleanAndNull(t *testing.T) {
	v, _ := NewStructuredOutputValidator(map[string]any{"type": "boolean"})
	if err := v.Validate(true); err != nil {
		t.Fatalf("good: %v", err)
	}
	if err := v.Validate(1); err == nil {
		t.Fatal("int is not bool")
	}
	v2, _ := NewStructuredOutputValidator(map[string]any{"type": "null"})
	if err := v2.Validate(nil); err != nil {
		t.Fatalf("good: %v", err)
	}
	if err := v2.Validate(0); err == nil {
		t.Fatal("0 is not null")
	}
}

func TestValidator_CoerceJSONString(t *testing.T) {
	v, _ := NewStructuredOutputValidator(map[string]any{"type": "object"})
	if err := v.Validate("{\"a\":1}\n"); err != nil {
		t.Fatalf("good: %v", err)
	}
	if err := v.Validate("```json\n{\"a\":1}\n```"); err != nil {
		t.Fatalf("fenced: %v", err)
	}
}

func TestValidator_IntegerAsFloat(t *testing.T) {
	v, _ := NewStructuredOutputValidator(map[string]any{"type": "integer"})
	if err := v.Validate(3.14); err != nil {
		t.Fatalf("3.14 is a number: %v", err)
	}
}

// schemaSourceImpl implements SchemaSource.
type schemaSourceImpl struct {
	s any
}

func (s schemaSourceImpl) Schema() any { return s.s }

func TestValidator_SchemaSourceInterface(t *testing.T) {
	schema := map[string]any{
		"type":     "object",
		"required": []any{"x"},
	}
	v, err := NewStructuredOutputValidator(schemaSourceImpl{s: schema})
	if err != nil {
		t.Fatal(err)
	}
	if err := v.Validate(map[string]any{"x": 1}); err != nil {
		t.Fatalf("good: %v", err)
	}
}

func TestValidator_PointerSchemaSource(t *testing.T) {
	v, err := NewStructuredOutputValidator(&schemaSourceImpl{s: map[string]any{"type": "object"}})
	if err != nil {
		t.Fatal(err)
	}
	if err := v.Validate(map[string]any{"a": 1}); err != nil {
		t.Fatalf("good: %v", err)
	}
}

func TestValidator_PointerSchemaSource_NotASource(t *testing.T) {
	type S struct{ A int }
	_, err := NewStructuredOutputValidator(&S{})
	if err == nil {
		t.Fatal("expected error: pointer-to-struct without Schema() is not a valid schema")
	}
}

func TestValidator_PropertiesMapNotMap(t *testing.T) {
	// The "properties" value is a list rather than a map.
	// The validator should not crash.
	v, _ := NewStructuredOutputValidator(map[string]any{
		"type":       "object",
		"properties": []any{"a"},
	})
	if err := v.Validate(map[string]any{"a": 1}); err != nil {
		t.Fatalf("good: %v", err)
	}
}

func TestValidator_SchemaNotMap(t *testing.T) {
	// A subschema is a list, not a map. The validator
	// reports a clear error.
	v, _ := NewStructuredOutputValidator(map[string]any{
		"type": "object",
		"properties": map[string]any{
			"a": []any{"x"},
		},
	})
	if err := v.Validate(map[string]any{"a": 1}); err == nil {
		t.Fatal("expected error for non-map subschema")
	}
}

func TestValidator_EnumNotList(t *testing.T) {
	v, _ := NewStructuredOutputValidator(map[string]any{
		"type": "string",
		"enum": "not a list",
	})
	if err := v.Validate("x"); err == nil {
		t.Fatal("expected error for non-list enum")
	}
}

func TestCoerceValue_Types(t *testing.T) {
	out, err := coerceValue(map[string]any{"a": int64(5), "b": float32(1.0)})
	if err != nil {
		t.Fatal(err)
	}
	m := out.(map[string]any)
	if m["a"] != 5.0 {
		t.Fatalf("a = %v (%T)", m["a"], m["a"])
	}
	if m["b"] != 1.0 {
		t.Fatalf("b = %v (%T)", m["b"], m["b"])
	}
}

func TestCoerceValue_Unsupported(t *testing.T) {
	_, err := coerceValue(make(chan int))
	if err == nil {
		t.Fatal("expected error for chan")
	}
}

func TestCoerceValue_Slice(t *testing.T) {
	out, err := coerceValue([]any{int64(1), int64(2)})
	if err != nil {
		t.Fatal(err)
	}
	sl := out.([]any)
	if sl[0] != 1.0 || sl[1] != 2.0 {
		t.Fatalf("slice = %v", sl)
	}
}

func TestCoerceValue_StringSlice(t *testing.T) {
	out, err := coerceValue([]string{"a", "b"})
	if err != nil {
		t.Fatal(err)
	}
	sl := out.([]any)
	if sl[0] != "a" || sl[1] != "b" {
		t.Fatalf("slice = %v", sl)
	}
}

func TestCoerceValue_MapOfString(t *testing.T) {
	type M map[string]int
	out, err := coerceValue(M{"a": 1})
	if err != nil {
		t.Fatal(err)
	}
	m := out.(map[string]any)
	if m["a"] != 1.0 {
		t.Fatalf("a = %v", m["a"])
	}
}

func TestIndexByte(t *testing.T) {
	if indexByte("abc", 'b') != 1 {
		t.Fatal()
	}
	if indexByte("abc", 'z') != -1 {
		t.Fatal()
	}
	if indexByte("", 'a') != -1 {
		t.Fatal()
	}
}

func TestValidateType_Object(t *testing.T) {
	if err := validateType(map[string]any{"a": 1}, "object", ""); err != nil {
		t.Fatalf("good: %v", err)
	}
	if err := validateType([]any{1}, "object", ""); err == nil {
		t.Fatal("array is not object")
	}
}

func TestValidateType_Array(t *testing.T) {
	if err := validateType([]any{1}, "array", ""); err != nil {
		t.Fatalf("good: %v", err)
	}
	if err := validateType(map[string]any{}, "array", ""); err == nil {
		t.Fatal("object is not array")
	}
}

func TestValidateType_Number(t *testing.T) {
	if err := validateType(1.0, "number", ""); err != nil {
		t.Fatalf("good: %v", err)
	}
	if err := validateType("x", "number", ""); err == nil {
		t.Fatal("string is not number")
	}
}

func TestValidateType_UnknownType(t *testing.T) {
	if err := validateType(1.0, "unknown-type", ""); err != nil {
		t.Fatalf("unknown type is permissive: %v", err)
	}
}

func TestValidateString_Defaults(t *testing.T) {
	if err := validateString("hi", map[string]any{}, ""); err != nil {
		t.Fatalf("empty schema accepts: %v", err)
	}
}

func TestValidateNumber_Defaults(t *testing.T) {
	if err := validateNumber(100, map[string]any{}, ""); err != nil {
		t.Fatalf("empty schema accepts: %v", err)
	}
}

func TestValidateArray_Defaults(t *testing.T) {
	if err := validateArray([]any{1, 2, 3}, map[string]any{}, ""); err != nil {
		t.Fatalf("empty schema accepts: %v", err)
	}
}

func TestStringList_NotList(t *testing.T) {
	if got := stringList("not a list"); got != nil {
		t.Fatalf("got %v", got)
	}
}

func TestStringList_Mixed(t *testing.T) {
	if got := stringList([]any{"a", 1, "b"}); len(got) != 2 {
		t.Fatalf("got %v", got)
	}
}

func TestKeySet_NotMap(t *testing.T) {
	if got := keySet("not a map"); len(got) != 0 {
		t.Fatalf("got %v", got)
	}
}

func TestJoinPath(t *testing.T) {
	if joinPath("", "x") != "x" {
		t.Fatal("empty prefix")
	}
	if joinPath("a", "b") != "a.b" {
		t.Fatal("dot")
	}
}
