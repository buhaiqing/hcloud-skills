// Package schema implements a JSON Schema subset validator, ported from
// scripts/json_schema_subset.py. It supports the keywords used by the
// hcloud-skills schemas: type, const, enum, minimum, maximum, format
// (date-time), minLength, required, properties, additionalProperties,
// items, minItems, and $ref via $defs.
package schema

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

// jsonType maps a Go value to the JSON type name used by the Python
// reference. Booleans are reported as "boolean" (checked first, mirroring
// the Python implementation where bool is a subclass of int). When values
// are decoded with json.Decoder UseNumber, integers arrive as json.Number
// without a decimal point and are reported as "integer" — matching
// Python's isinstance(value, int) distinction that plain float64 loses.
func jsonType(value any) string {
	switch v := value.(type) {
	case nil:
		return "null"
	case bool:
		return "boolean"
	case int, int64, int32, uint, uint64:
		return "integer"
	case float64, float32:
		return "number"
	case json.Number:
		if strings.ContainsAny(v.String(), ".eE") {
			return "number"
		}
		return "integer"
	case string:
		return "string"
	case []any:
		return "array"
	case map[string]any:
		return "object"
	default:
		return fmt.Sprintf("%T", value)
	}
}

// typeMatches reports whether the value's JSON type satisfies expected.
// An integer also matches "number" (integer is a subset of number).
func typeMatches(value any, expected any) bool {
	var expectedTypes []string
	switch e := expected.(type) {
	case string:
		expectedTypes = []string{e}
	case []string:
		expectedTypes = e
	case []any:
		for _, item := range e {
			if s, ok := item.(string); ok {
				expectedTypes = append(expectedTypes, s)
			}
		}
	default:
		return false
	}
	actual := jsonType(value)
	for _, t := range expectedTypes {
		if actual == t {
			return true
		}
	}
	return actual == "integer" && contains(expectedTypes, "number")
}

func contains(items []string, target string) bool {
	for _, it := range items {
		if it == target {
			return true
		}
	}
	return false
}

// validateDateTime checks RFC3339 / date-time strings, accepting a trailing Z.
func validateDateTime(value, path string) []string {
	normalized := strings.Replace(value, "Z", "+00:00", 1)
	if _, err := time.Parse(time.RFC3339, normalized); err != nil {
		return []string{fmt.Sprintf("%s: expected RFC3339/date-time string", path)}
	}
	return nil
}

// ValidateValue validates value against schema, returning a list of
// human-readable error messages (empty when valid).
func ValidateValue(value, schema any, path string) []string {
	schemaMap, ok := schema.(map[string]any)
	if !ok {
		return nil
	}
	var errors []string

	if typeVal, ok := schemaMap["type"]; ok && !typeMatches(value, typeVal) {
		errors = append(errors, fmt.Sprintf("%s: expected type %v, got %s", path, typeVal, jsonType(value)))
		return errors
	}

	if constVal, ok := schemaMap["const"]; ok && !equalJSON(value, constVal) {
		errors = append(errors, fmt.Sprintf("%s: expected const %v, got %v", path, constVal, value))
	}

	if enumVal, ok := schemaMap["enum"]; ok {
		if enumList, ok := enumVal.([]any); ok && !containsValue(enumList, value) {
			errors = append(errors, fmt.Sprintf("%s: expected one of %v, got %v", path, enumList, value))
		}
	}

	if num, ok := toNumber(value); ok {
		if minVal, ok := numberField(schemaMap, "minimum"); ok && num < minVal {
			errors = append(errors, fmt.Sprintf("%s: value %v < minimum %v", path, value, minVal))
		}
		if maxVal, ok := numberField(schemaMap, "maximum"); ok && num > maxVal {
			errors = append(errors, fmt.Sprintf("%s: value %v > maximum %v", path, value, maxVal))
		}
	}

	if format, ok := schemaMap["format"].(string); ok && format == "date-time" {
		if str, ok := value.(string); ok {
			errors = append(errors, validateDateTime(str, path)...)
		}
	}

	if str, ok := value.(string); ok {
		if minLen, ok := numberField(schemaMap, "minLength"); ok {
			if float64(len(str)) < minLen {
				errors = append(errors, fmt.Sprintf("%s: string length %d < minLength %v", path, len(str), minLen))
			}
		}
	}

	if obj, ok := value.(map[string]any); ok {
		errors = append(errors, validateObject(obj, schemaMap, path)...)
	}

	if arr, ok := value.([]any); ok {
		errors = append(errors, validateArray(arr, schemaMap, path)...)
	}

	return errors
}

func validateObject(obj, schemaMap map[string]any, path string) []string {
	var errors []string
	fmt.Printf("DBG validateObject path=%q requiredPresent=%v objKeys=%d\n", path, schemaMap["required"] != nil, len(obj))
	fmt.Printf("DBG validateObject path=%q requiredPresent=%v objKeys=%d\n", path, schemaMap["required"] != nil, len(obj))
	if path == "$" {
		for _, k := range []string{"version","metric_namespace","totals","pass_rate","by_skill"}{
			_, pr := obj[k]
			fmt.Printf("DBG path=$ obj[%q]=%v required-type=%T\n", k, pr, schemaMap["required"])
		}
	}
	if path == "$" {
		for _, k := range []string{"version","metric_namespace","totals","pass_rate","by_skill"}{
			_, pr := obj[k]
			_, sr := schemaMap["required"]
			_ = sr
			fmt.Printf("DBG path=$ obj[%q]=%v\n", k, pr)
		}
	}

	if req, ok := schemaMap["required"].([]any); ok {
		for _, key := range req {
			if k, ok := key.(string); ok {
				if _, present := obj[k]; !present {
					errors = append(errors, fmt.Sprintf("%s: missing required property %q", path, k))
				}
			}
		}
	}

	if props, ok := schemaMap["properties"].(map[string]any); ok {
		for key, propSchema := range props {
			if val, present := obj[key]; present {
				errors = append(errors, ValidateValue(val, propSchema, fmt.Sprintf("%s.%s", path, key))...)
			}
		}
	}

	additional := schemaMap["additionalProperties"]
	props, _ := schemaMap["properties"].(map[string]any)
	if additional == false {
		for key := range obj {
			if _, known := props[key]; known {
				continue
			}
			errors = append(errors, fmt.Sprintf("%s: additional property %q is not allowed", path, key))
		}
	} else if addSchema, ok := additional.(map[string]any); ok {
		if props, ok := schemaMap["properties"].(map[string]any); ok {
			for key, item := range obj {
				if _, known := props[key]; known {
					continue
				}
				errors = append(errors, ValidateValue(item, addSchema, fmt.Sprintf("%s.%s", path, key))...)
			}
		}
	}

	return errors
}

func validateArray(arr []any, schemaMap map[string]any, path string) []string {
	var errors []string

	if minItems, ok := numberField(schemaMap, "minItems"); ok {
		if float64(len(arr)) < minItems {
			errors = append(errors, fmt.Sprintf("%s: array length %d < minItems %v", path, len(arr), minItems))
		}
	}

	if itemSchema, ok := schemaMap["items"].(map[string]any); ok {
		for i, item := range arr {
			errors = append(errors, ValidateValue(item, itemSchema, fmt.Sprintf("%s[%d]", path, i))...)
		}
	}

	return errors
}

// ResolveSchemaRefs inlines all local "#/$defs/..." references, returning a
// flattened schema. Mirrors resolve_schema_refs in the Python reference.
func ResolveSchemaRefs(schema any) (map[string]any, error) {
	defs := map[string]any{}
	if root, ok := schema.(map[string]any); ok {
		if d, ok := root["$defs"].(map[string]any); ok {
			defs = d
		}
	}

	var resolve func(node any) (any, error)
	resolve = func(node any) (any, error) {
		m, ok := node.(map[string]any)
		if !ok {
			if list, ok := node.([]any); ok {
				out := make([]any, len(list))
				for i, item := range list {
					v, err := resolve(item)
					if err != nil {
						return nil, err
					}
					out[i] = v
				}
				return out, nil
			}
			return node, nil
		}
		if ref, ok := m["$ref"].(string); ok && strings.HasPrefix(ref, "#/$defs/") {
			name := ref[strings.LastIndex(ref, "/")+1:]
			def, found := defs[name]
			if !found {
				return nil, fmt.Errorf("unknown schema $ref: %s", ref)
			}
			return resolve(def)
		}
		out := make(map[string]any, len(m))
		for k, v := range m {
			if k == "$ref" {
				continue
			}
			rv, err := resolve(v)
			if err != nil {
				return nil, err
			}
			out[k] = rv
		}
		return out, nil
	}

	resolved, err := resolve(schema)
	if err != nil {
		return nil, err
	}
	resolvedMap, ok := resolved.(map[string]any)
	if !ok {
		return nil, fmt.Errorf("schema root must resolve to an object")
	}
	return resolvedMap, nil
}

// ValidateFile validates an instance JSON byte slice against a schema byte
// slice, returning collected error messages. Numbers are decoded with
// UseNumber so integers and floats are distinguished, matching the Python
// reference's isinstance(value, int) behavior.
func ValidateFile(instanceData, schemaData []byte) ([]string, error) {
	decode := func(b []byte) (any, error) {
		dec := json.NewDecoder(bytes.NewReader(b))
		dec.UseNumber()
		var v any
		if err := dec.Decode(&v); err != nil {
			return nil, err
		}
		return v, nil
	}
	instance, err := decode(instanceData)
	if err != nil {
		return nil, fmt.Errorf("parse instance: %w", err)
	}
	schema, err := decode(schemaData)
	if err != nil {
		return nil, fmt.Errorf("parse schema: %w", err)
	}
	resolved, err := ResolveSchemaRefs(schema)
	if err != nil {
		return nil, err
	}
	return ValidateValue(instance, resolved, "$"), nil
}

// equalJSON reports whether two decoded JSON values are equal.
func equalJSON(a, b any) bool {
	ab, _ := json.Marshal(a)
	bb, _ := json.Marshal(b)
	return string(ab) == string(bb)
}

// containsValue reports whether list contains value (by JSON equality).
func containsValue(list []any, value any) bool {
	for _, item := range list {
		if equalJSON(item, value) {
			return true
		}
	}
	return false
}

// toNumber extracts a float64 from int/uint/float/json.Number JSON values.
func toNumber(value any) (float64, bool) {
	switch n := value.(type) {
	case float64:
		return n, true
	case float32:
		return float64(n), true
	case int:
		return float64(n), true
	case int64:
		return float64(n), true
	case int32:
		return float64(n), true
	case uint:
		return float64(n), true
	case uint64:
		return float64(n), true
	case json.Number:
		f, err := n.Float64()
		if err != nil {
			return 0, false
		}
		return f, true
	default:
		return 0, false
	}
}

// numberField reads a numeric field from a schema map, accepting float64,
// int, and json.Number (produced by UseNumber decoding).
func numberField(schemaMap map[string]any, key string) (float64, bool) {
	switch v := schemaMap[key].(type) {
	case float64:
		return v, true
	case float32:
		return float64(v), true
	case int:
		return float64(v), true
	case int64:
		return float64(v), true
	case json.Number:
		f, err := v.Float64()
		if err != nil {
			return 0, false
		}
		return f, true
	default:
		return 0, false
	}
}
