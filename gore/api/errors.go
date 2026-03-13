package api

import "errors"

var (
	// ErrNotImplemented indicates a stub method.
	ErrNotImplemented = errors.New("not implemented")
	// ErrTrackingDisabled indicates tracking is disabled.
	ErrTrackingDisabled = errors.New("change tracking is disabled")
)
