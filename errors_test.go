// SPDX-License-Identifier: MPL-2.0
// Copyright (c) 2025 Daniel Schmidt

package gnmi

import (
	"errors"
	"testing"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// TestGnmiError_Error tests the Error() method of GnmiError
func TestGnmiError_Error(t *testing.T) {
	tests := []struct {
		name     string
		err      GnmiError
		expected string
	}{
		{
			name: "error without retries",
			err: GnmiError{
				Operation: "Get",
				Message:   "connection failed",
				Retries:   0,
			},
			expected: "gnmi: Get failed: connection failed",
		},
		{
			name: "error with retries",
			err: GnmiError{
				Operation: "Set",
				Message:   "timeout exceeded",
				Retries:   3,
			},
			expected: "gnmi: Set failed: timeout exceeded (retries: 3)",
		},
		{
			name: "error with single retry",
			err: GnmiError{
				Operation: "Capabilities",
				Message:   "unavailable",
				Retries:   1,
			},
			expected: "gnmi: Capabilities failed: unavailable (retries: 1)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.err.Error()
			if got != tt.expected {
				t.Errorf("Error() = %q, want %q", got, tt.expected)
			}
		})
	}
}

// TestGnmiError_DetailedError tests the DetailedError() method
func TestGnmiError_DetailedError(t *testing.T) {
	tests := []struct {
		name     string
		err      GnmiError
		expected string
	}{
		{
			name: "error without internal message or retries",
			err: GnmiError{
				Operation:   "Get",
				Message:     "connection failed",
				InternalMsg: "",
				Retries:     0,
			},
			expected: "gnmi: Get failed: connection failed",
		},
		{
			name: "error with internal message, no retries",
			err: GnmiError{
				Operation:   "Get",
				Message:     "connection failed",
				InternalMsg: "dial tcp 192.168.1.1:57400: connect: connection refused",
				Retries:     0,
			},
			expected: "gnmi: Get failed: connection failed (internal: dial tcp 192.168.1.1:57400: connect: connection refused)",
		},
		{
			name: "error with internal message and retries",
			err: GnmiError{
				Operation:   "Set",
				Message:     "timeout exceeded",
				InternalMsg: "context deadline exceeded after 15s",
				Retries:     3,
			},
			expected: "gnmi: Set failed: timeout exceeded (internal: context deadline exceeded after 15s, retries: 3)",
		},
		{
			name: "error with retries but no internal message",
			err: GnmiError{
				Operation:   "Capabilities",
				Message:     "unavailable",
				InternalMsg: "",
				Retries:     2,
			},
			expected: "gnmi: Capabilities failed: unavailable (retries: 2)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.err.DetailedError()
			if got != tt.expected {
				t.Errorf("DetailedError() = %q, want %q", got, tt.expected)
			}
		})
	}
}

// TestCheckTransientError tests transient error detection from gRPC errors
func TestCheckTransientError(t *testing.T) {
	// Create a test client with logger
	client := &Client{
		logger: &NoOpLogger{},
	}

	tests := []struct {
		name       string
		err        error
		wantResult bool
	}{
		{
			name:       "nil error",
			err:        nil,
			wantResult: false,
		},
		{
			name:       "non-gRPC error",
			err:        errors.New("regular error"),
			wantResult: false,
		},
		{
			name:       "Unavailable (transient)",
			err:        status.Error(codes.Unavailable, "service unavailable"),
			wantResult: true,
		},
		{
			name:       "ResourceExhausted (transient)",
			err:        status.Error(codes.ResourceExhausted, "rate limit exceeded"),
			wantResult: true,
		},
		{
			name:       "DeadlineExceeded (transient)",
			err:        status.Error(codes.DeadlineExceeded, "timeout"),
			wantResult: true,
		},
		{
			name:       "Aborted (transient)",
			err:        status.Error(codes.Aborted, "transaction aborted"),
			wantResult: true,
		},
		{
			name:       "InvalidArgument (permanent)",
			err:        status.Error(codes.InvalidArgument, "invalid path"),
			wantResult: false,
		},
		{
			name:       "NotFound (permanent)",
			err:        status.Error(codes.NotFound, "path not found"),
			wantResult: false,
		},
		{
			name:       "PermissionDenied (permanent)",
			err:        status.Error(codes.PermissionDenied, "access denied"),
			wantResult: false,
		},
		{
			name:       "Unauthenticated (permanent)",
			err:        status.Error(codes.Unauthenticated, "authentication failed"),
			wantResult: false,
		},
		{
			name:       "Internal (permanent - intentionally excluded)",
			err:        status.Error(codes.Internal, "internal server error"),
			wantResult: false,
		},
		{
			name:       "OK (not an error)",
			err:        status.Error(codes.OK, "success"),
			wantResult: false,
		},
		{
			name:       "Canceled (permanent)",
			err:        status.Error(codes.Canceled, "operation canceled"),
			wantResult: false,
		},
		{
			name:       "Unknown (permanent)",
			err:        status.Error(codes.Unknown, "unknown error"),
			wantResult: false,
		},
		{
			name:       "AlreadyExists (permanent)",
			err:        status.Error(codes.AlreadyExists, "resource exists"),
			wantResult: false,
		},
		{
			name:       "FailedPrecondition (permanent)",
			err:        status.Error(codes.FailedPrecondition, "precondition failed"),
			wantResult: false,
		},
		{
			name:       "OutOfRange (permanent)",
			err:        status.Error(codes.OutOfRange, "value out of range"),
			wantResult: false,
		},
		{
			name:       "Unimplemented (permanent)",
			err:        status.Error(codes.Unimplemented, "not implemented"),
			wantResult: false,
		},
		{
			name:       "DataLoss (permanent)",
			err:        status.Error(codes.DataLoss, "data corruption"),
			wantResult: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := client.checkTransientError(tt.err)
			if got != tt.wantResult {
				t.Errorf("checkTransientError() = %v, want %v", got, tt.wantResult)
			}
		})
	}
}

// TestCheckTransientErrorModels tests transient error detection from ErrorModel list
func TestCheckTransientErrorModels(t *testing.T) {
	// Create a test client
	client := &Client{
		logger: &NoOpLogger{},
	}

	tests := []struct {
		name       string
		errors     []ErrorModel
		wantResult bool
	}{
		{
			name:       "empty error list",
			errors:     []ErrorModel{},
			wantResult: false,
		},
		{
			name: "single transient error (Unavailable)",
			errors: []ErrorModel{
				{Code: uint32(codes.Unavailable), Message: "service unavailable"},
			},
			wantResult: true,
		},
		{
			name: "single transient error (ResourceExhausted)",
			errors: []ErrorModel{
				{Code: uint32(codes.ResourceExhausted), Message: "rate limit"},
			},
			wantResult: true,
		},
		{
			name: "single transient error (DeadlineExceeded)",
			errors: []ErrorModel{
				{Code: uint32(codes.DeadlineExceeded), Message: "timeout"},
			},
			wantResult: true,
		},
		{
			name: "single transient error (Aborted)",
			errors: []ErrorModel{
				{Code: uint32(codes.Aborted), Message: "aborted"},
			},
			wantResult: true,
		},
		{
			name: "single permanent error (InvalidArgument)",
			errors: []ErrorModel{
				{Code: uint32(codes.InvalidArgument), Message: "invalid path"},
			},
			wantResult: false,
		},
		{
			name: "single permanent error (NotFound)",
			errors: []ErrorModel{
				{Code: uint32(codes.NotFound), Message: "not found"},
			},
			wantResult: false,
		},
		{
			name: "mixed errors with one transient",
			errors: []ErrorModel{
				{Code: uint32(codes.InvalidArgument), Message: "invalid path"},
				{Code: uint32(codes.Unavailable), Message: "service unavailable"},
			},
			wantResult: true,
		},
		{
			name: "multiple permanent errors",
			errors: []ErrorModel{
				{Code: uint32(codes.InvalidArgument), Message: "invalid path"},
				{Code: uint32(codes.NotFound), Message: "not found"},
				{Code: uint32(codes.PermissionDenied), Message: "access denied"},
			},
			wantResult: false,
		},
		{
			name: "multiple transient errors",
			errors: []ErrorModel{
				{Code: uint32(codes.Unavailable), Message: "service unavailable"},
				{Code: uint32(codes.DeadlineExceeded), Message: "timeout"},
			},
			wantResult: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := client.checkTransientErrorModels(tt.errors)
			if got != tt.wantResult {
				t.Errorf("checkTransientErrorModels() = %v, want %v", got, tt.wantResult)
			}
		})
	}
}

// TestTransientErrors_Coverage tests that TransientErrors list contains expected codes
func TestTransientErrors_Coverage(t *testing.T) {
	expectedCodes := map[uint32]string{
		uint32(codes.Unavailable):       "Unavailable",
		uint32(codes.ResourceExhausted): "ResourceExhausted",
		uint32(codes.DeadlineExceeded):  "DeadlineExceeded",
		uint32(codes.Aborted):           "Aborted",
	}

	// Verify TransientErrors list contains all expected codes
	foundCodes := make(map[uint32]bool)
	for _, pattern := range TransientErrors {
		foundCodes[pattern.Code] = true
	}

	// Check all expected codes are present
	for code, name := range expectedCodes {
		if !foundCodes[code] {
			t.Errorf("TransientErrors missing expected code: %s (%d)", name, code)
		}
	}

	// Verify no extra codes (should be exactly 4)
	if len(TransientErrors) != len(expectedCodes) {
		t.Errorf("TransientErrors has %d codes, expected %d", len(TransientErrors), len(expectedCodes))
	}
}

// TestGnmiError_IsTransientField tests the IsTransient field behavior
func TestGnmiError_IsTransientField(t *testing.T) {
	tests := []struct {
		name        string
		err         GnmiError
		expectField bool
	}{
		{
			name: "transient error marked",
			err: GnmiError{
				Operation:   "Get",
				Message:     "service unavailable",
				IsTransient: true,
				Retries:     3,
			},
			expectField: true,
		},
		{
			name: "permanent error not marked",
			err: GnmiError{
				Operation:   "Set",
				Message:     "invalid argument",
				IsTransient: false,
				Retries:     0,
			},
			expectField: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.err.IsTransient != tt.expectField {
				t.Errorf("IsTransient = %v, want %v", tt.err.IsTransient, tt.expectField)
			}
		})
	}
}

// TestErrorModel tests the ErrorModel struct
func TestErrorModel(t *testing.T) {
	model := ErrorModel{
		Code:    uint32(codes.NotFound),
		Message: "path not found",
		Details: "additional context",
	}

	if model.Code != uint32(codes.NotFound) {
		t.Errorf("Code = %d, want %d", model.Code, uint32(codes.NotFound))
	}
	if model.Message != "path not found" {
		t.Errorf("Message = %q, want %q", model.Message, "path not found")
	}
	if model.Details != "additional context" {
		t.Errorf("Details = %q, want %q", model.Details, "additional context")
	}
}

// BenchmarkCheckTransientError benchmarks transient error detection
func BenchmarkCheckTransientError(b *testing.B) {
	client := &Client{
		logger: &NoOpLogger{},
	}

	testCases := []struct {
		name string
		err  error
	}{
		{"Unavailable", status.Error(codes.Unavailable, "unavailable")},
		{"InvalidArgument", status.Error(codes.InvalidArgument, "invalid")},
		{"NonGRPC", errors.New("regular error")},
		{"Nil", nil},
	}

	for _, tc := range testCases {
		b.Run(tc.name, func(b *testing.B) {
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				_ = client.checkTransientError(tc.err)
			}
		})
	}
}

// BenchmarkCheckTransientErrorModels benchmarks ErrorModel-based transient detection
func BenchmarkCheckTransientErrorModels(b *testing.B) {
	client := &Client{
		logger: &NoOpLogger{},
	}

	testCases := []struct {
		name   string
		errors []ErrorModel
	}{
		{
			"SingleTransient",
			[]ErrorModel{{Code: uint32(codes.Unavailable), Message: "unavailable"}},
		},
		{
			"SinglePermanent",
			[]ErrorModel{{Code: uint32(codes.InvalidArgument), Message: "invalid"}},
		},
		{
			"Mixed",
			[]ErrorModel{
				{Code: uint32(codes.InvalidArgument), Message: "invalid"},
				{Code: uint32(codes.Unavailable), Message: "unavailable"},
			},
		},
		{
			"Empty",
			[]ErrorModel{},
		},
	}

	for _, tc := range testCases {
		b.Run(tc.name, func(b *testing.B) {
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				_ = client.checkTransientErrorModels(tc.errors)
			}
		})
	}
}
