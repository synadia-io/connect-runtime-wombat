package compiler

import (
	"fmt"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	// Compilation error metrics
	compilationErrors = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "connect_runtime_wombat_compilation_errors_total",
			Help: "Total number of compilation errors by type",
		},
		[]string{"error_type", "phase", "step"},
	)

	// Compilation duration metric
	compilationDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "connect_runtime_wombat_compilation_duration_seconds",
			Help:    "Time spent compiling configurations",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"success", "connector_type"},
	)

	// Validation error metrics
	validationErrors = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "connect_runtime_wombat_validation_errors_total",
			Help: "Total number of validation errors by type",
		},
		[]string{"error_type", "component"},
	)

	// Runtime error metrics
	runtimeErrors = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "connect_runtime_wombat_runtime_errors_total",
			Help: "Total number of runtime errors by type",
		},
		[]string{"error_type", "component"},
	)
)

// CompilationError represents an error that occurred during compilation
type CompilationError struct {
	Phase   string
	Step    string
	Message string
	Err     error
}

func (e *CompilationError) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("compilation error in %s/%s: %s: %v", e.Phase, e.Step, e.Message, e.Err)
	}
	return fmt.Sprintf("compilation error in %s/%s: %s", e.Phase, e.Step, e.Message)
}

func (e *CompilationError) Unwrap() error {
	return e.Err
}

// NewCompilationError creates a new CompilationError and records metrics
func NewCompilationError(phase, step, message string, err error) *CompilationError {
	compilationErrors.WithLabelValues("compilation", phase, step).Inc()
	return &CompilationError{
		Phase:   phase,
		Step:    step,
		Message: message,
		Err:     err,
	}
}

// ValidationError represents an error that occurred during validation
type ValidationError struct {
	Component string
	Message   string
	Err       error
}

func (e *ValidationError) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("validation error in %s: %s: %v", e.Component, e.Message, e.Err)
	}
	return fmt.Sprintf("validation error in %s: %s", e.Component, e.Message)
}

func (e *ValidationError) Unwrap() error {
	return e.Err
}

// NewValidationError creates a new ValidationError and records metrics
func NewValidationError(component, message string, err error) *ValidationError {
	validationErrors.WithLabelValues("validation", component).Inc()
	return &ValidationError{
		Component: component,
		Message:   message,
		Err:       err,
	}
}

// RuntimeError represents an error that occurred during runtime
type RuntimeError struct {
	Component string
	Message   string
	Err       error
}

func (e *RuntimeError) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("runtime error in %s: %s: %v", e.Component, e.Message, e.Err)
	}
	return fmt.Sprintf("runtime error in %s: %s", e.Component, e.Message)
}

func (e *RuntimeError) Unwrap() error {
	return e.Err
}

// NewRuntimeError creates a new RuntimeError and records metrics
func NewRuntimeError(component, message string, err error) *RuntimeError {
	runtimeErrors.WithLabelValues("runtime", component).Inc()
	return &RuntimeError{
		Component: component,
		Message:   message,
		Err:       err,
	}
}

// RecordCompilationMetrics records compilation duration and success metrics
func RecordCompilationMetrics(start time.Time, success bool, connectorType string) {
	duration := time.Since(start).Seconds()
	successLabel := "false"
	if success {
		successLabel = "true"
	}
	compilationDuration.WithLabelValues(successLabel, connectorType).Observe(duration)
}
