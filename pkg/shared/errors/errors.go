package errors

import "fmt"

// Custom error type for not implemented errors
type NotImplementedError struct {
	MethodName string
	PluginName string
}

// Implement the error interface for NotImplementedError
func (e *NotImplementedError) Error() string {
	return fmt.Sprintf("method %s is not implemented for %s", e.MethodName, e.PluginName)
}

// Constructor for NotImplementedError
func NewNotImplementedError(methodName, pluginName string) error {
	return &NotImplementedError{
		MethodName: methodName,
	}
}
