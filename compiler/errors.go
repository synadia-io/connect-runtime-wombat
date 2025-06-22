package compiler

import (
	"errors"
	"fmt"
)

// Common error variables for compiler package
var (
	// ErrInvalidConsumerType indicates that an invalid number of consumer types were specified
	ErrInvalidConsumerType = errors.New("exactly one consumer type (core, stream, kv) must be defined")

	// ErrInvalidProducerType indicates that an invalid number of producer types were specified
	ErrInvalidProducerType = errors.New("exactly one producer type (core, stream, kv) must be defined")

	// ErrNoProducerType indicates that no producer type was specified
	ErrNoProducerType = errors.New("at least one producer type (core, stream, kv) must be defined")
)

// CompilationError represents an error that occurred during compilation
type CompilationError struct {
	Phase   string // Phase of compilation (e.g., "consumer", "producer", "transformer")
	Step    string // Specific step that failed
	Message string // Human-readable error message
	Err     error  // Underlying error
}

// Error implements the error interface
func (e *CompilationError) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("compilation failed in %s phase at %s: %s: %v", e.Phase, e.Step, e.Message, e.Err)
	}
	return fmt.Sprintf("compilation failed in %s phase at %s: %s", e.Phase, e.Step, e.Message)
}

// Unwrap returns the underlying error
func (e *CompilationError) Unwrap() error {
	return e.Err
}

// Is checks if the error matches the target error
func (e *CompilationError) Is(target error) bool {
	return errors.Is(e.Err, target)
}

// NewCompilationError creates a new compilation error
func NewCompilationError(phase, step, message string, err error) *CompilationError {
	return &CompilationError{
		Phase:   phase,
		Step:    step,
		Message: message,
		Err:     err,
	}
}

// ValidationError represents an error that occurred during validation
type ValidationError struct {
	Component string // Component being validated
	Field     string // Field that failed validation
	Value     any    // The invalid value
	Message   string // Human-readable error message
}

// Error implements the error interface
func (e *ValidationError) Error() string {
	if e.Field != "" && e.Value != nil {
		return fmt.Sprintf("validation failed for %s: field '%s' with value '%v' is invalid: %s",
			e.Component, e.Field, e.Value, e.Message)
	}
	return fmt.Sprintf("validation failed for %s: %s", e.Component, e.Message)
}

// NewValidationError creates a new validation error
func NewValidationError(component, field string, value any, message string) *ValidationError {
	return &ValidationError{
		Component: component,
		Field:     field,
		Value:     value,
		Message:   message,
	}
}
