package errors

import (
	"errors"
	"testing"
)

func TestCodeString(t *testing.T) {
	tests := []struct {
		code    Code
		want    string
	}{
		{CodeNone, "NONE"},
		{CodeInvalidInput, "INVALID_INPUT"},
		{CodeNotFound, "NOT_FOUND"},
		{CodeAlreadyExists, "ALREADY_EXISTS"},
		{CodeTrackingDisabled, "TRACKING_DISABLED"},
		{CodeNotImplemented, "NOT_IMPLEMENTED"},
		{CodeInternal, "INTERNAL"},
		{CodeQueryError, "QUERY_ERROR"},
		{CodeConnectionError, "CONNECTION_ERROR"},
		{Code(999), "UNKNOWN"},
	}

	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			if got := tt.code.String(); got != tt.want {
				t.Errorf("Code.String() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestGoreError(t *testing.T) {
	err := &GoreError{Code: CodeInvalidInput, Message: "invalid input"}
	if err.Error() != "[INVALID_INPUT] invalid input" {
		t.Fatalf("unexpected error string: %s", err.Error())
	}
}

func TestGoreErrorWithCause(t *testing.T) {
	cause := errors.New("original error")
	err := &GoreError{Code: CodeQueryError, Message: "query failed", cause: cause}

	if err.Error() != "[QUERY_ERROR] query failed: original error" {
		t.Fatalf("unexpected error string: %s", err.Error())
	}

	if err.Unwrap() != cause {
		t.Fatal("Unwrap should return cause")
	}
}

func TestGoreErrorWithStack(t *testing.T) {
	err := &GoreError{Code: CodeInternal, Message: "internal error"}
	err = err.WithStack("frame1")
	err = err.WithStack("frame2")

	stack := err.Stack()
	if len(stack) != 2 {
		t.Fatalf("expected 2 stack frames, got %d", len(stack))
	}
	if stack[0] != "frame1" {
		t.Fatalf("expected frame1, got %s", stack[0])
	}
}

func TestNew(t *testing.T) {
	err := New(CodeInvalidInput, "test error")
	if err.Code != CodeInvalidInput {
		t.Fatalf("expected CodeInvalidInput, got %v", err.Code)
	}
	if err.Message != "test error" {
		t.Fatalf("expected 'test error', got %s", err.Message)
	}
}

func TestInvalidInput(t *testing.T) {
	err := InvalidInput("field is required")
	if err.Code != CodeInvalidInput {
		t.Fatalf("expected CodeInvalidInput, got %v", err.Code)
	}
	if err.Message != "field is required" {
		t.Fatalf("unexpected message: %s", err.Message)
	}
}

func TestNotFound(t *testing.T) {
	err := NotFound("user")
	if err.Code != CodeNotFound {
		t.Fatalf("expected CodeNotFound, got %v", err.Code)
	}
	if err.Message != "user not found" {
		t.Fatalf("unexpected message: %s", err.Message)
	}
}

func TestAlreadyExists(t *testing.T) {
	err := AlreadyExists("order")
	if err.Code != CodeAlreadyExists {
		t.Fatalf("expected CodeAlreadyExists, got %v", err.Code)
	}
	if err.Message != "order already exists" {
		t.Fatalf("unexpected message: %s", err.Message)
	}
}

func TestTrackingDisabled(t *testing.T) {
	err := TrackingDisabled()
	if err.Code != CodeTrackingDisabled {
		t.Fatalf("expected CodeTrackingDisabled, got %v", err.Code)
	}
}

func TestNotImplemented(t *testing.T) {
	err := NotImplemented()
	if err.Code != CodeNotImplemented {
		t.Fatalf("expected CodeNotImplemented, got %v", err.Code)
	}
}

func TestInternal(t *testing.T) {
	err := Internal("something went wrong")
	if err.Code != CodeInternal {
		t.Fatalf("expected CodeInternal, got %v", err.Code)
	}
}

func TestQueryError(t *testing.T) {
	cause := errors.New("connection timeout")
	err := QueryError("failed to execute query", cause)
	if err.Code != CodeQueryError {
		t.Fatalf("expected CodeQueryError, got %v", err.Code)
	}
	if err.cause != cause {
		t.Fatal("cause should be set")
	}
}

func TestConnectionError(t *testing.T) {
	cause := errors.New("refused connection")
	err := ConnectionError("cannot connect to db", cause)
	if err.Code != CodeConnectionError {
		t.Fatalf("expected CodeConnectionError, got %v", err.Code)
	}
	if err.cause != cause {
		t.Fatal("cause should be set")
	}
}

func TestWrap(t *testing.T) {
	cause := errors.New("original")
	err := Wrap(cause, CodeInternal, "wrapped error")
	if err.Code != CodeInternal {
		t.Fatalf("expected CodeInternal, got %v", err.Code)
	}
	if err.Message != "wrapped error" {
		t.Fatalf("unexpected message: %s", err.Message)
	}
	if err.Unwrap() != cause {
		t.Fatal("Unwrap should return cause")
	}
}

func TestIs(t *testing.T) {
	// Create a wrapped error chain
	cause := ErrEntityNotFound
	err := Wrap(cause, CodeNotFound, "user not found")
	if !Is(err, ErrEntityNotFound) {
		t.Fatal("expected ErrEntityNotFound to match")
	}
}

func TestAs(t *testing.T) {
	err := &GoreError{Code: CodeNotFound, Message: "user not found"}
	var target *GoreError
	if !As(err, &target) {
		t.Fatal("expected As to succeed")
	}
	if target.Code != CodeNotFound {
		t.Fatalf("expected CodeNotFound, got %v", target.Code)
	}
}

func TestSentinelErrors(t *testing.T) {
	if ErrNotImplemented.Error() != "feature not implemented" {
		t.Fatal("unexpected ErrNotImplemented message")
	}
	if ErrTrackingDisabled.Error() != "change tracking is disabled" {
		t.Fatal("unexpected ErrTrackingDisabled message")
	}
	if ErrEntityNotFound.Error() != "entity not found" {
		t.Fatal("unexpected ErrEntityNotFound message")
	}
	if ErrEntityExists.Error() != "entity already exists" {
		t.Fatal("unexpected ErrEntityExists message")
	}
	if ErrNilEntity.Error() != "entity cannot be nil" {
		t.Fatal("unexpected ErrNilEntity message")
	}
	if ErrInvalidEntity.Error() != "entity must be a non-nil pointer to struct" {
		t.Fatal("unexpected ErrInvalidEntity message")
	}
	if ErrQueryBuildFailed.Error() != "failed to build query" {
		t.Fatal("unexpected ErrQueryBuildFailed message")
	}
	if ErrConnectionFailed.Error() != "database connection failed" {
		t.Fatal("unexpected ErrConnectionFailed message")
	}
}
