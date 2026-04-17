// Package errors provides standardized error types and sentinel errors for gore.
package errors

import (
	"errors"
	"fmt"
)

// =============================================================================
// Error Codes
// =============================================================================

type Code int

const (
	CodeNone Code = iota
	CodeInvalidInput
	CodeNotFound
	CodeAlreadyExists
	CodeTrackingDisabled
	CodeNotImplemented
	CodeInternal
	CodeQueryError
	CodeConnectionError
)

// String returns the error code name.
func (c Code) String() string {
	switch c {
	case CodeNone:
		return "NONE"
	case CodeInvalidInput:
		return "INVALID_INPUT"
	case CodeNotFound:
		return "NOT_FOUND"
	case CodeAlreadyExists:
		return "ALREADY_EXISTS"
	case CodeTrackingDisabled:
		return "TRACKING_DISABLED"
	case CodeNotImplemented:
		return "NOT_IMPLEMENTED"
	case CodeInternal:
		return "INTERNAL"
	case CodeQueryError:
		return "QUERY_ERROR"
	case CodeConnectionError:
		return "CONNECTION_ERROR"
	default:
		return "UNKNOWN"
	}
}

// =============================================================================
// Sentinel Errors
// =============================================================================

var (
	ErrNotImplemented    = errors.New("feature not implemented")
	ErrTrackingDisabled  = errors.New("change tracking is disabled")
	ErrEntityNotFound    = errors.New("entity not found")
	ErrEntityExists      = errors.New("entity already exists")
	ErrNilEntity         = errors.New("entity cannot be nil")
	ErrInvalidEntity     = errors.New("entity must be a non-nil pointer to struct")
	ErrQueryBuildFailed  = errors.New("failed to build query")
	ErrConnectionFailed = errors.New("database connection failed")
)

// =============================================================================
// GoreError
// =============================================================================

type GoreError struct {
	Code    Code
	Message string
	cause   error
	stack   []string
}

func (e *GoreError) Error() string {
	if e.cause != nil {
		return fmt.Sprintf("[%s] %s: %v", e.Code, e.Message, e.cause)
	}
	return fmt.Sprintf("[%s] %s", e.Code, e.Message)
}

func (e *GoreError) Unwrap() error {
	return e.cause
}

// WithCause wraps an existing error.
func (e *GoreError) WithCause(err error) *GoreError {
	e.cause = err
	return e
}

// WithStack adds a stack frame.
func (e *GoreError) WithStack(frame string) *GoreError {
	e.stack = append(e.stack, frame)
	return e
}

// Stack returns the error stack.
func (e *GoreError) Stack() []string {
	return e.stack
}

// =============================================================================
// Error Factories
// =============================================================================

func New(code Code, message string) *GoreError {
	return &GoreError{Code: code, Message: message}
}

func InvalidInput(message string) *GoreError {
	return &GoreError{Code: CodeInvalidInput, Message: message}
}

func NotFound(entity string) *GoreError {
	return &GoreError{Code: CodeNotFound, Message: fmt.Sprintf("%s not found", entity)}
}

func AlreadyExists(entity string) *GoreError {
	return &GoreError{Code: CodeAlreadyExists, Message: fmt.Sprintf("%s already exists", entity)}
}

func TrackingDisabled() *GoreError {
	return &GoreError{Code: CodeTrackingDisabled, Message: "change tracking is disabled"}
}

func NotImplemented() *GoreError {
	return &GoreError{Code: CodeNotImplemented, Message: "feature not implemented"}
}

func Internal(message string) *GoreError {
	return &GoreError{Code: CodeInternal, Message: message}
}

func QueryError(message string, err error) *GoreError {
	return &GoreError{Code: CodeQueryError, Message: message, cause: err}
}

func ConnectionError(message string, err error) *GoreError {
	return &GoreError{Code: CodeConnectionError, Message: message, cause: err}
}

// Wrap wraps a standard error with a GoreError.
func Wrap(err error, code Code, message string) *GoreError {
	return &GoreError{Code: code, Message: message, cause: err}
}

// =============================================================================
// Is helpers
// =============================================================================

func Is(err, target error) bool {
	return errors.Is(err, target)
}

func As(err error, target any) bool {
	return errors.As(err, target)
}
