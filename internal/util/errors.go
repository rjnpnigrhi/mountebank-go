package util

import (
	"encoding/json"
	"fmt"
	"net/http"
)

// ErrorType represents different types of errors in mountebank
type ErrorType string

const (
	// ValidationError represents validation failures ('bad data')
	ValidationError ErrorType = "bad data"
	// InjectionError represents JavaScript injection failures
	InjectionError ErrorType = "invalid injection"
	// ProtocolError represents protocol-specific errors
	ProtocolError ErrorType = "cannot start server"
	// MissingResourceError represents missing resource errors
	MissingResourceError ErrorType = "no such resource"
	// InsufficientAccessError represents authorization errors
	InsufficientAccessError ErrorType = "insufficient access"
	// InvalidJSONError represents JSON parsing errors
	InvalidJSONError ErrorType = "invalid JSON"
)

// MountebankError represents a mountebank-specific error
type MountebankError struct {
	Code    ErrorType   `json:"code"`
	Message string      `json:"message"`
	Source  interface{} `json:"source,omitempty"`
	Details interface{} `json:"details,omitempty"`
}

// Error implements the error interface
func (e *MountebankError) Error() string {
	return fmt.Sprintf("%s: %s", e.Code, e.Message)
}

// NewValidationError creates a new validation error
func NewValidationError(message string, source interface{}) *MountebankError {
	return &MountebankError{
		Code:    ValidationError,
		Message: message,
		Source:  source,
	}
}

// NewInjectionError creates a new injection error
func NewInjectionError(message string, source interface{}, details interface{}) *MountebankError {
	return &MountebankError{
		Code:    InjectionError,
		Message: message,
		Source:  source,
		Details: details,
	}
}

// NewProtocolError creates a new protocol error
func NewProtocolError(message string, source interface{}, details interface{}) *MountebankError {
	return &MountebankError{
		Code:    ProtocolError,
		Message: message,
		Source:  source,
		Details: details,
	}
}

// NewMissingResourceError creates a new missing resource error
func NewMissingResourceError(message string, source interface{}) *MountebankError {
	return &MountebankError{
		Code:    MissingResourceError,
		Message: message,
		Source:  source,
	}
}

// NewInsufficientAccessError creates a new insufficient access error
func NewInsufficientAccessError(message string) *MountebankError {
	return &MountebankError{
		Code:    InsufficientAccessError,
		Message: message,
	}
}

// NewInvalidJSONError creates a new invalid JSON error
func NewInvalidJSONError(message string) *MountebankError {
	return &MountebankError{
		Code:    InvalidJSONError,
		Message: message,
	}
}

// WriteError writes a formatted error response structure to the writer
func WriteError(w http.ResponseWriter, err error, code int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)

	var mbErrors []MountebankError

	if mbErr, ok := err.(*MountebankError); ok {
		mbErrors = []MountebankError{*mbErr}
	} else {
		// Wrap generic error
		mbErrors = []MountebankError{
			{
				Code:    ValidationError, // Default to bad data for unknown errors? Or maybe a generic types
				Message: err.Error(),
			},
		}
	}

	response := map[string]interface{}{
		"errors": mbErrors,
	}

	json.NewEncoder(w).Encode(response)
}
