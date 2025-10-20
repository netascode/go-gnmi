// SPDX-License-Identifier: MPL-2.0
// Copyright (c) 2025 Daniel Schmidt

package gnmi

import (
	"fmt"

	"google.golang.org/grpc/codes"
)

// GnmiError represents a structured gNMI error with operation context
type GnmiError struct {
	// Operation name that failed
	Operation string

	// Errors from gNMI error details
	Errors []ErrorModel

	// Human-readable error message
	Message string

	// InternalMsg contains detailed error information for internal logging
	InternalMsg string

	// Number of retry attempts made
	Retries int

	// IsTransient indicates if the error is transient and was retried
	IsTransient bool
}

// Error implements the error interface
func (e *GnmiError) Error() string {
	if e.Retries > 0 {
		return fmt.Sprintf("gnmi: %s failed: %s (retries: %d)", e.Operation, e.Message, e.Retries)
	}
	return fmt.Sprintf("gnmi: %s failed: %s", e.Operation, e.Message)
}

// DetailedError returns the full error message including internal details
//
// This should only be used in secure logging contexts where sensitive information
// disclosure is acceptable (e.g., server-side logs, debug output).
//
// Example:
//
//	if err != nil {
//	    if gnmiErr, ok := err.(*GnmiError); ok {
//	        log.Debug(gnmiErr.DetailedError()) // internal logging
//	        return gnmiErr.Error() // client-facing error
//	    }
//	}
func (e *GnmiError) DetailedError() string {
	if e.InternalMsg == "" {
		return e.Error()
	}
	if e.Retries > 0 {
		return fmt.Sprintf("gnmi: %s failed: %s (internal: %s, retries: %d)",
			e.Operation, e.Message, e.InternalMsg, e.Retries)
	}
	return fmt.Sprintf("gnmi: %s failed: %s (internal: %s)",
		e.Operation, e.Message, e.InternalMsg)
}

// ErrorModel represents a gNMI error with gRPC status code
type ErrorModel struct {
	// Code is the gRPC status code
	Code uint32

	// Message is the error message
	Message string

	// Details contains additional error information
	Details string
}

// TransientError defines patterns for detecting transient errors that should be retried
type TransientError struct {
	// Code is the gRPC status code to match
	Code uint32
}

// TransientErrors defines the list of gRPC status codes that should trigger automatic retry
//
// These errors are typically caused by temporary conditions such as:
//   - Service unavailable (server temporarily down, overloaded)
//   - Resource exhausted (rate limiting, quota exceeded)
//   - Deadline exceeded (timeout, slow network)
//   - Aborted (transaction aborted, try again)
//
// NOTE: codes.Internal is intentionally excluded from this list. While some Internal
// errors may be transient, codes.Internal is a catch-all error code that includes many
// permanent failures (bugs, invalid state, etc.). Blindly retrying Internal errors can
// mask real problems and waste resources. If specific Internal errors are known to be
// transient, they should be detected and handled explicitly rather than retrying all
// Internal errors.
//
// Based on gRPC status codes from google.golang.org/grpc/codes
var TransientErrors = []TransientError{
	// Service temporarily unavailable
	{Code: uint32(codes.Unavailable)},

	// Rate limiting or quota exceeded
	{Code: uint32(codes.ResourceExhausted)},

	// Timeout or deadline exceeded
	{Code: uint32(codes.DeadlineExceeded)},

	// Transaction aborted, may succeed on retry
	{Code: uint32(codes.Aborted)},
}
