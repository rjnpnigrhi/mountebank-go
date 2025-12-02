package util

import "fmt"

// ErrorType represents different types of errors in mountebank
type ErrorType string

const (
	// ValidationError represents validation failures
	ValidationError ErrorType = "validation error"
	// InjectionError represents JavaScript injection failures
	InjectionError ErrorType = "injection error"
	// ProtocolError represents protocol-specific errors
	ProtocolError ErrorType = "protocol error"
	// MissingResourceError represents missing resource errors
	MissingResourceError ErrorType = "missing resource error"
	// InsufficientAccessError represents authorization errors
	InsufficientAccessError ErrorType = "insufficient access error"
)

// MountebankError represents a mountebank-specific error
type MountebankError struct {
	Type    ErrorType
	Message string
	Source  interface{}
	Details interface{}
}

// Error implements the error interface
func (e *MountebankError) Error() string {
	return fmt.Sprintf("%s: %s", e.Type, e.Message)
}

// NewValidationError creates a new validation error
func NewValidationError(message string, source interface{}) *MountebankError {
	return &MountebankError{
		Type:    ValidationError,
		Message: message,
		Source:  source,
	}
}

// NewInjectionError creates a new injection error
func NewInjectionError(message string, source interface{}, details interface{}) *MountebankError {
	return &MountebankError{
		Type:    InjectionError,
		Message: message,
		Source:  source,
		Details: details,
	}
}

// NewProtocolError creates a new protocol error
func NewProtocolError(message string, source interface{}, details interface{}) *MountebankError {
	return &MountebankError{
		Type:    ProtocolError,
		Message: message,
		Source:  source,
		Details: details,
	}
}

// NewMissingResourceError creates a new missing resource error
func NewMissingResourceError(message string, source interface{}) *MountebankError {
	return &MountebankError{
		Type:    MissingResourceError,
		Message: message,
		Source:  source,
	}
}

// NewInsufficientAccessError creates a new insufficient access error
func NewInsufficientAccessError(message string) *MountebankError {
	return &MountebankError{
		Type:    InsufficientAccessError,
		Message: message,
	}
}
