package config

import (
	"reflect"
	"strings"
)

// GetBoolValue retrieves a boolean value from a nested struct based on a dot-separated path.
// It returns the provided defaultValue if the specified field is not explicitly set or is nil.
func GetBoolValue(config interface{}, fieldPath string, defaultValue bool) bool {
	if config == nil {
		return defaultValue
	}

	fields := strings.Split(fieldPath, ".")
	value := reflect.ValueOf(config)

	for _, field := range fields {
		if value.Kind() == reflect.Ptr {
			value = value.Elem()
		}

		value = value.FieldByName(field)
		if !value.IsValid() {
			return defaultValue
		}
	}

	// Check if the field is a pointer to a bool and is not nil
	if value.Kind() == reflect.Ptr && !value.IsNil() {
		return value.Elem().Bool()
	} else if value.Kind() == reflect.Bool {
		// Handle non-pointer bool directly
		return value.Bool()
	}

	return defaultValue
}

// SetThen returns the first value if it is set; otherwise, it returns the default value.
func SetThen[T any](value T, defaultValue T) T {
	if reflect.ValueOf(value).IsZero() {
		return defaultValue
	}
	return value
}

// SetThenPtr returns the dereferenced value if the pointer is not nil; otherwise, it returns the default value.
func SetThenPtr[T any](value *T, defaultValue T) T {
	if value == nil {
		return defaultValue
	}
	return *value
}
