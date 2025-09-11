package errors

import (
	"fmt"

	"github.com/scan-io-git/scan-io/pkg/shared"
)

// Custom error type for not implemented errors
type NotImplementedError struct {
	MethodName string
	PluginName string
}

// Implement the error interface for NotImplementedError
func (e *NotImplementedError) Error() string {
	return fmt.Sprintf("method %q is not implemented for %q", e.MethodName, e.PluginName)
}

// Constructor for NotImplementedError
func NewNotImplementedError(methodName, pluginName string) error {
	return &NotImplementedError{
		MethodName: methodName,
		PluginName: pluginName,
	}
}

// CommandError represents an error that occurred during plugin execution, storing relevant results.
type CommandError struct {
	ExitCode    int
	CommonError string
	Result      shared.GenericLaunchesResult
}

// Error implements the error interface, returning the message from the common error.
func (e *CommandError) Error() string {
	return e.CommonError
}

// NewCommandError creates a new CommandError instance, encapsulating args, result, and the error message.
func NewCommandError(args interface{}, result interface{}, err error, code int) *CommandError {
	return &CommandError{
		ExitCode:    code,
		CommonError: err.Error(),
		Result: shared.GenericLaunchesResult{
			Launches: []shared.GenericResult{
				{
					Args:    args,
					Result:  result,
					Status:  "FAILED",
					Message: err.Error(),
				},
			},
		},
	}
}

// NewCommandErrorWithResult creates a new CommandError with a pre-formed GenericLaunchesResult.
func NewCommandErrorWithResult(launches shared.GenericLaunchesResult, err error, code int) *CommandError {
	return &CommandError{
		ExitCode:    code,
		CommonError: err.Error(),
		Result:      launches,
	}
}
