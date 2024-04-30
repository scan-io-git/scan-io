package config

import (
	"reflect"
	"strings"
)

// getBoolValue retrieves a boolean value from a nested struct based on a dot-separated path.
// It returns the provided defaultValue if the specified field is not explicitly set or is nil.
func GetBoolValue(config interface{}, fieldPath string, defaultValue bool) bool {
	if config == nil {
		return defaultValue
	}

	fields := strings.Split(fieldPath, ".")
	val := reflect.ValueOf(config)

	for _, field := range fields {
		if val.Kind() == reflect.Ptr {
			val = val.Elem()
		}

		val = val.FieldByName(field)
		if !val.IsValid() {
			return defaultValue
		}
	}

	// Check if the field is a pointer to a bool and is not nil
	if val.Kind() == reflect.Ptr && !val.IsNil() {
		return val.Elem().Bool()
	} else if val.Kind() == reflect.Bool {
		// Handle non-pointer bool directly
		return val.Bool()
	}

	return defaultValue
}

// setThen provides a utility to select the first value if set, otherwise defaults.
func SetThen[T any](value T, defaultValue T) T {
	if reflect.ValueOf(value).IsZero() {
		return defaultValue
	}
	return value
}
