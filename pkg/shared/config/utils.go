package config

import (
	"reflect"
)

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
