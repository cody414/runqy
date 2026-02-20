package api

import (
	"testing"

	queueworker "github.com/Publikey/runqy/queues"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func boolPtr(b bool) *bool { return &b }

func TestValidateFieldsValidTypes(t *testing.T) {
	fields := []queueworker.FieldSchema{
		{Name: "name", Type: []string{"string"}, Required: boolPtr(true)},
		{Name: "count", Type: []string{"int"}, Required: boolPtr(true)},
		{Name: "ratio", Type: []string{"float"}, Required: boolPtr(false)},
		{Name: "active", Type: []string{"bool"}, Required: boolPtr(true)},
	}

	data := map[string]interface{}{
		"name":   "hello",
		"count":  float64(42), // JSON numbers are float64
		"ratio":  3.14,
		"active": true,
	}

	result, err := validateFields(data, fields)
	require.NoError(t, err)
	assert.Equal(t, "hello", result["name"])
	assert.Equal(t, int64(42), result["count"]) // float64 → int64 conversion
	assert.Equal(t, 3.14, result["ratio"])
	assert.Equal(t, true, result["active"])
}

func TestValidateFieldsMissingRequired(t *testing.T) {
	fields := []queueworker.FieldSchema{
		{Name: "required_field", Type: []string{"string"}, Required: boolPtr(true)},
	}

	data := map[string]interface{}{}

	_, err := validateFields(data, fields)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "required_field is required")
}

func TestValidateFieldsWrongType(t *testing.T) {
	fields := []queueworker.FieldSchema{
		{Name: "name", Type: []string{"string"}, Required: boolPtr(true)},
	}

	data := map[string]interface{}{
		"name": float64(123), // number instead of string
	}

	_, err := validateFields(data, fields)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "name")
	assert.Contains(t, err.Error(), "number")
	assert.Contains(t, err.Error(), "string")
}

func TestValidateFieldsOptionalWithDefault(t *testing.T) {
	fields := []queueworker.FieldSchema{
		{Name: "color", Type: []string{"string"}, Required: boolPtr(false), Default: "blue"},
	}

	data := map[string]interface{}{}

	result, err := validateFields(data, fields)
	require.NoError(t, err)
	assert.Equal(t, "blue", result["color"])
}

func TestValidateFieldsPassthroughExtraFields(t *testing.T) {
	fields := []queueworker.FieldSchema{
		{Name: "name", Type: []string{"string"}, Required: boolPtr(true)},
	}

	data := map[string]interface{}{
		"name":  "hello",
		"extra": "should be kept",
	}

	result, err := validateFields(data, fields)
	require.NoError(t, err)
	assert.Equal(t, "hello", result["name"])
	assert.Equal(t, "should be kept", result["extra"])
}

func TestValidateFieldsMultipleTypes(t *testing.T) {
	fields := []queueworker.FieldSchema{
		{Name: "value", Type: []string{"string", "int"}, Required: boolPtr(true)},
	}

	// String should pass
	data := map[string]interface{}{"value": "hello"}
	_, err := validateFields(data, fields)
	assert.NoError(t, err)

	// Integer should pass
	data = map[string]interface{}{"value": float64(42)}
	_, err = validateFields(data, fields)
	assert.NoError(t, err)

	// Bool should fail
	data = map[string]interface{}{"value": true}
	_, err = validateFields(data, fields)
	assert.Error(t, err)
}

func TestValidateFieldsArrayAndObject(t *testing.T) {
	fields := []queueworker.FieldSchema{
		{Name: "items", Type: []string{"array"}, Required: boolPtr(true)},
		{Name: "config", Type: []string{"object"}, Required: boolPtr(true)},
	}

	data := map[string]interface{}{
		"items":  []interface{}{"a", "b"},
		"config": map[string]interface{}{"key": "val"},
	}

	result, err := validateFields(data, fields)
	require.NoError(t, err)
	assert.IsType(t, []interface{}{}, result["items"])
	assert.IsType(t, map[string]interface{}{}, result["config"])
}

func TestDescribeType(t *testing.T) {
	assert.Equal(t, "string", describeType("hello"))
	assert.Equal(t, "bool", describeType(true))
	assert.Equal(t, "number", describeType(float64(3.14)))
	assert.Equal(t, "array", describeType([]interface{}{1, 2}))
	assert.Equal(t, "object", describeType(map[string]interface{}{"a": 1}))
	assert.Equal(t, "<nil>", describeType(nil))
}

func TestCheckAllowedType(t *testing.T) {
	assert.True(t, checkAllowedType("hello", []string{"string"}))
	assert.True(t, checkAllowedType(true, []string{"bool"}))
	assert.True(t, checkAllowedType(float64(42), []string{"int"}))
	assert.True(t, checkAllowedType(float64(3.14), []string{"float"}))
	assert.True(t, checkAllowedType([]interface{}{}, []string{"array"}))
	assert.True(t, checkAllowedType(map[string]interface{}{}, []string{"object"}))

	assert.False(t, checkAllowedType("hello", []string{"int"}))
	assert.False(t, checkAllowedType(true, []string{"string"}))
	assert.False(t, checkAllowedType(float64(42), []string{"string"}))
}

func TestContains(t *testing.T) {
	assert.True(t, contains([]string{"a", "b", "c"}, "b"))
	assert.False(t, contains([]string{"a", "b", "c"}, "d"))
	assert.False(t, contains([]string{}, "a"))
}
