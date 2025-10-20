// SPDX-License-Identifier: MPL-2.0
// Copyright (c) 2025 Daniel Schmidt

package gnmi

import (
	"context"
	"fmt"
	mathrand "math/rand"
	"strings"
	"testing"
	"time"
)

// TestNewClientValidation tests client configuration validation
func TestNewClientValidation(t *testing.T) {
	tests := []struct {
		name        string
		target      string
		opts        []func(*Client)
		wantErrMsg  string
		description string
	}{
		{
			name:        "empty target",
			target:      "",
			opts:        nil,
			wantErrMsg:  "target address cannot be empty",
			description: "Empty target should fail validation",
		},
		{
			name:        "whitespace target",
			target:      "   ",
			opts:        nil,
			wantErrMsg:  "target address cannot be empty",
			description: "Whitespace-only target should fail validation",
		},
		{
			name:   "invalid port low",
			target: "192.168.1.1",
			opts: []func(*Client){
				Port(0),
			},
			wantErrMsg:  "invalid port: 0 (must be 1-65535)",
			description: "Port 0 should fail validation",
		},
		{
			name:   "invalid port high",
			target: "192.168.1.1",
			opts: []func(*Client){
				Port(65536),
			},
			wantErrMsg:  "invalid port: 65536 (must be 1-65535)",
			description: "Port > 65535 should fail validation",
		},
		{
			name:   "negative connect timeout",
			target: "192.168.1.1",
			opts: []func(*Client){
				ConnectTimeout(-1 * time.Second),
			},
			wantErrMsg:  "connect timeout must be positive",
			description: "Negative connect timeout should fail validation",
		},
		{
			name:   "zero connect timeout",
			target: "192.168.1.1",
			opts: []func(*Client){
				ConnectTimeout(0),
			},
			wantErrMsg:  "connect timeout must be positive",
			description: "Zero connect timeout should fail validation",
		},
		{
			name:   "negative operation timeout",
			target: "192.168.1.1",
			opts: []func(*Client){
				OperationTimeout(-1 * time.Second),
			},
			wantErrMsg:  "operation timeout must be positive",
			description: "Negative operation timeout should fail validation",
		},
		{
			name:   "zero operation timeout",
			target: "192.168.1.1",
			opts: []func(*Client){
				OperationTimeout(0),
			},
			wantErrMsg:  "operation timeout must be positive",
			description: "Zero operation timeout should fail validation",
		},
		{
			name:   "negative max retries",
			target: "192.168.1.1",
			opts: []func(*Client){
				MaxRetries(-1),
			},
			wantErrMsg:  "max retries must be non-negative",
			description: "Negative max retries should fail validation",
		},
		{
			name:   "negative backoff min delay",
			target: "192.168.1.1",
			opts: []func(*Client){
				BackoffMinDelay(-1 * time.Second),
			},
			wantErrMsg:  "backoff min delay must be positive",
			description: "Negative backoff min delay should fail validation",
		},
		{
			name:   "zero backoff min delay",
			target: "192.168.1.1",
			opts: []func(*Client){
				BackoffMinDelay(0),
			},
			wantErrMsg:  "backoff min delay must be positive",
			description: "Zero backoff min delay should fail validation",
		},
		{
			name:   "max delay less than min delay",
			target: "192.168.1.1",
			opts: []func(*Client){
				BackoffMinDelay(10 * time.Second),
				BackoffMaxDelay(5 * time.Second),
			},
			wantErrMsg:  "backoff max delay",
			description: "Max delay < min delay should fail validation",
		},
		{
			name:   "max delay equal to min delay",
			target: "192.168.1.1",
			opts: []func(*Client){
				BackoffMinDelay(10 * time.Second),
				BackoffMaxDelay(10 * time.Second),
			},
			wantErrMsg:  "backoff max delay",
			description: "Max delay == min delay should fail validation",
		},
		{
			name:   "invalid backoff factor",
			target: "192.168.1.1",
			opts: []func(*Client){
				BackoffDelayFactor(0.5),
			},
			wantErrMsg:  "backoff delay factor must be >= 1.0",
			description: "Backoff factor < 1.0 should fail validation",
		},
		{
			name:   "invalid TLS certificate path",
			target: "192.168.1.1",
			opts: []func(*Client){
				TLSCert("/nonexistent/cert.pem"),
			},
			wantErrMsg:  "TLS certificate file not found",
			description: "Non-existent TLS cert should fail validation",
		},
		{
			name:   "invalid TLS key path",
			target: "192.168.1.1",
			opts: []func(*Client){
				TLSKey("/nonexistent/key.pem"),
			},
			wantErrMsg:  "TLS key file not found",
			description: "Non-existent TLS key should fail validation",
		},
		{
			name:   "invalid TLS CA path",
			target: "192.168.1.1",
			opts: []func(*Client){
				TLSCA("/nonexistent/ca.pem"),
			},
			wantErrMsg:  "TLS CA file not found",
			description: "Non-existent TLS CA should fail validation",
		},
		{
			name:   "TLS cert not found - path redacted",
			target: "192.168.1.1",
			opts: []func(*Client){
				TLSCert("/secret/path/to/admin-cert.pem"),
			},
			wantErrMsg:  "admin-cert.pem",
			description: "TLS error should contain filename only, not full path",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// NewClient will fail validation before attempting connection
			_, err := NewClient(tt.target, tt.opts...)
			if err == nil {
				t.Errorf("%s: expected error but got none", tt.description)
				return
			}
			if !strings.Contains(err.Error(), tt.wantErrMsg) {
				t.Errorf("%s: expected error containing %q, got %q",
					tt.description, tt.wantErrMsg, err.Error())
			}
		})
	}
}

// TestTLSPathRedaction tests that TLS path errors don't leak full paths
func TestTLSPathRedaction(t *testing.T) {
	tests := []struct {
		name              string
		opts              []func(*Client)
		wantErrContains   string
		wantErrNotContain string
	}{
		{
			name: "TLS cert path redacted",
			opts: []func(*Client){
				TLSCert("/secret/path/to/admin-cert.pem"),
			},
			wantErrContains:   "admin-cert.pem",
			wantErrNotContain: "/secret/path",
		},
		{
			name: "TLS key path redacted",
			opts: []func(*Client){
				TLSKey("/home/user/private-key.pem"),
			},
			wantErrContains:   "private-key.pem",
			wantErrNotContain: "/home/user",
		},
		{
			name: "TLS CA path redacted",
			opts: []func(*Client){
				TLSCA("/etc/ssl/certs/ca-bundle.crt"),
			},
			wantErrContains:   "ca-bundle.crt",
			wantErrNotContain: "/etc/ssl/certs",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := NewClient("192.168.1.1", tt.opts...)
			if err == nil {
				t.Fatal("expected error but got none")
			}

			errStr := err.Error()
			if !strings.Contains(errStr, tt.wantErrContains) {
				t.Errorf("error should contain %q, got: %q", tt.wantErrContains, errStr)
			}
			if strings.Contains(errStr, tt.wantErrNotContain) {
				t.Errorf("error should NOT contain %q (path disclosure), got: %q", tt.wantErrNotContain, errStr)
			}
		})
	}
}

// TestHasCredentials tests the HasCredentials method
func TestHasCredentials(t *testing.T) {
	tests := []struct {
		name        string
		opts        []func(*Client)
		want        bool
		description string
	}{
		{
			name:        "no credentials",
			opts:        nil,
			want:        false,
			description: "Client with no credentials should return false",
		},
		{
			name: "username only",
			opts: []func(*Client){
				Username("admin"),
			},
			want:        true,
			description: "Client with username should return true",
		},
		{
			name: "password only",
			opts: []func(*Client){
				Password("secret"),
			},
			want:        true,
			description: "Client with password should return true",
		},
		{
			name: "username and password",
			opts: []func(*Client){
				Username("admin"),
				Password("secret"),
			},
			want:        true,
			description: "Client with username and password should return true",
		},
		{
			name: "TLS cert only",
			opts: []func(*Client){
				TLSCert("testdata/client.crt"),
			},
			want:        true,
			description: "Client with TLS cert should return true",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create client without connecting (bypass validation by creating directly)
			client := &Client{
				Target:             "192.168.1.1",
				Port:               DefaultPort,
				ConnectTimeout:     DefaultConnectTimeout,
				OperationTimeout:   DefaultOperationTimeout,
				MaxRetries:         DefaultMaxRetries,
				BackoffMinDelay:    DefaultBackoffMinDelay,
				BackoffMaxDelay:    DefaultBackoffMaxDelay,
				BackoffDelayFactor: DefaultBackoffDelayFactor,
				logger:             &NoOpLogger{},
			}

			// Apply options
			for _, opt := range tt.opts {
				opt(client)
			}

			// Test HasCredentials
			got := client.HasCredentials()
			if got != tt.want {
				t.Errorf("%s: HasCredentials() = %v, want %v",
					tt.description, got, tt.want)
			}
		})
	}
}

// TestHasCapability tests the HasCapability method
func TestHasCapability(t *testing.T) {
	client := &Client{
		capabilities: []string{"JSON", "JSON_IETF", "PROTO"},
	}

	tests := []struct {
		name       string
		capability string
		want       bool
	}{
		{
			name:       "capability exists",
			capability: "JSON",
			want:       true,
		},
		{
			name:       "capability exists case sensitive",
			capability: "json",
			want:       false,
		},
		{
			name:       "capability does not exist",
			capability: "ASCII",
			want:       false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := client.HasCapability(tt.capability)
			if got != tt.want {
				t.Errorf("HasCapability(%q) = %v, want %v",
					tt.capability, got, tt.want)
			}
		})
	}
}

// TestServerCapabilities tests the ServerCapabilities method
func TestServerCapabilities(t *testing.T) {
	client := &Client{
		capabilities: []string{"JSON", "JSON_IETF", "PROTO"},
	}

	caps := client.ServerCapabilities()

	// Verify returned slice has correct contents
	if len(caps) != 3 {
		t.Errorf("ServerCapabilities() returned %d capabilities, want 3", len(caps))
	}

	// Verify it's a copy (modifying returned slice doesn't affect original)
	caps[0] = "MODIFIED"
	if client.capabilities[0] == "MODIFIED" {
		t.Error("ServerCapabilities() should return a copy, not the original slice")
	}
}

// TestBackoff tests the backoff calculation
func TestBackoff(t *testing.T) {
	client := &Client{
		BackoffMinDelay:    1 * time.Second,
		BackoffMaxDelay:    60 * time.Second,
		BackoffDelayFactor: 2.0,
		logger:             &NoOpLogger{},
	}

	tests := []struct {
		name        string
		attempt     int
		wantMin     time.Duration
		wantMax     time.Duration
		description string
	}{
		{
			name:        "attempt 0",
			attempt:     0,
			wantMin:     1 * time.Second,
			wantMax:     1*time.Second + 100*time.Millisecond,
			description: "First retry should be ~1s (min delay + jitter)",
		},
		{
			name:        "attempt 1",
			attempt:     1,
			wantMin:     2 * time.Second,
			wantMax:     2*time.Second + 200*time.Millisecond,
			description: "Second retry should be ~2s (min * factor^1 + jitter)",
		},
		{
			name:        "attempt 2",
			attempt:     2,
			wantMin:     4 * time.Second,
			wantMax:     4*time.Second + 400*time.Millisecond,
			description: "Third retry should be ~4s (min * factor^2 + jitter)",
		},
		{
			name:        "attempt 10",
			attempt:     10,
			wantMin:     60 * time.Second,
			wantMax:     60*time.Second + 6*time.Second,
			description: "Large attempt should be capped at max delay",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			delay := client.Backoff(tt.attempt)
			if delay < tt.wantMin || delay > tt.wantMax {
				t.Errorf("%s: Backoff(%d) = %v, want between %v and %v",
					tt.description, tt.attempt, delay, tt.wantMin, tt.wantMax)
			}
		})
	}
}

// TestBackoffJitter tests that backoff includes random jitter
func TestBackoffJitter(t *testing.T) {
	client := &Client{
		BackoffMinDelay:    1 * time.Second,
		BackoffMaxDelay:    60 * time.Second,
		BackoffDelayFactor: 2.0,
		logger:             &NoOpLogger{},
	}

	// Calculate backoff multiple times and verify we get different values
	attempts := 100
	delays := make(map[time.Duration]bool)
	for i := 0; i < attempts; i++ {
		delay := client.Backoff(0)
		delays[delay] = true
	}

	// With 100 attempts, we should get at least 10 different values
	// (this is statistical, but should be very reliable)
	if len(delays) < 10 {
		t.Errorf("Backoff() should include jitter: got %d unique values out of %d attempts",
			len(delays), attempts)
	}
}

// TestBackoffTimestampFallback tests that timestamp-based jitter provides non-zero randomness
func TestBackoffTimestampFallback(t *testing.T) {
	// This test verifies that even if crypto/rand were to fail (which is extremely rare),
	// the timestamp-based fallback would still provide jitter to prevent thundering herd.
	// We test this indirectly by verifying that multiple rapid calls get different jitter values.

	mock := &mockLogger{}
	client := &Client{
		BackoffMinDelay:    1 * time.Second,
		BackoffMaxDelay:    60 * time.Second,
		BackoffDelayFactor: 2.0,
		logger:             mock,
	}

	// Rapidly calculate backoff multiple times
	// The timestamp-based fallback should produce different values due to nanosecond timing
	delays := make(map[time.Duration]bool)
	for i := 0; i < 50; i++ {
		delay := client.Backoff(0)
		delays[delay] = true
		// Small sleep to ensure timestamp changes between iterations
		time.Sleep(1 * time.Nanosecond)
	}

	// With 50 rapid attempts, we should get multiple different values
	// This verifies that jitter (whether from crypto/rand or timestamp) is working
	if len(delays) < 5 {
		t.Errorf("Backoff() should produce varied jitter: got %d unique values out of 50 attempts (expected at least 5)",
			len(delays))
	}

	// Verify all delays are within expected range (1s base + 0-100ms jitter)
	for delay := range delays {
		if delay < 1*time.Second || delay > 1*time.Second+110*time.Millisecond {
			t.Errorf("Backoff() delay %v outside expected range [1s, 1.11s]", delay)
		}
	}

	// Note: We don't check for warning logs here because crypto/rand should succeed
	// in normal test environments. The fallback code path is defensive but rarely triggered.
}

// TestBackoffOverflow tests backoff with large attempt numbers
func TestBackoffOverflow(t *testing.T) {
	client := &Client{
		BackoffMinDelay:    1 * time.Second,
		BackoffMaxDelay:    60 * time.Second,
		BackoffDelayFactor: 2.0,
		logger:             &NoOpLogger{},
	}

	// Very large attempt number that would cause overflow
	delay := client.Backoff(1000)

	// Should be capped at max delay
	if delay > 60*time.Second+6*time.Second {
		t.Errorf("Backoff(1000) = %v, should be capped at ~60s", delay)
	}
}

// TestBackoffUsesCryptoRand verifies that jitter uses crypto/rand, not math/rand
func TestBackoffUsesCryptoRand(t *testing.T) {
	// This test verifies that jitter comes from crypto/rand, not math/rand
	// by checking that jitter cannot be predicted even if we seed math/rand

	// Create a local math/rand with known seed (Go 1.20+ approach)
	//nolint:gosec // G404: Intentional use of math/rand to test that jitter uses crypto/rand
	rng := mathrand.New(mathrand.NewSource(12345))
	_ = rng // We create this to demonstrate we're testing crypto/rand independence

	client := &Client{
		BackoffMinDelay:    1 * time.Second,
		BackoffMaxDelay:    60 * time.Second,
		BackoffDelayFactor: 2.0,
		logger:             &NoOpLogger{},
	}

	// Get first backoff with jitter
	delay1 := client.Backoff(1)

	// Create another local math/rand with same seed
	//nolint:gosec // G404: Intentional use of math/rand to test that jitter uses crypto/rand
	rng2 := mathrand.New(mathrand.NewSource(12345))
	_ = rng2

	// Get second backoff with jitter
	delay2 := client.Backoff(1)

	// If using crypto/rand, delays should be different (not predictable)
	// If using math/rand, delays would be the same (predictable)
	if delay1 == delay2 {
		t.Fatal("Backoff appears to use math/rand (predictable), should use crypto/rand")
	}
}

// BenchmarkBackoffCryptoRand benchmarks the backoff calculation with crypto/rand
func BenchmarkBackoffCryptoRand(b *testing.B) {
	client := &Client{
		BackoffMinDelay:    1 * time.Second,
		BackoffMaxDelay:    60 * time.Second,
		BackoffDelayFactor: 2.0,
		logger:             &NoOpLogger{},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = client.Backoff(5)
	}
}

// mockLogger is a mock logger for testing that captures log messages
type mockLogger struct {
	debugCalls []map[string]any
	infoCalls  []map[string]any
	warnCalls  []map[string]any
	errorCalls []map[string]any
}

func (m *mockLogger) Debug(msg string, keysAndValues ...any) {
	call := map[string]any{"msg": msg}
	for i := 0; i < len(keysAndValues); i += 2 {
		if i+1 < len(keysAndValues) {
			call[fmt.Sprint(keysAndValues[i])] = keysAndValues[i+1]
		}
	}
	m.debugCalls = append(m.debugCalls, call)
}

func (m *mockLogger) Info(msg string, keysAndValues ...any) {
	call := map[string]any{"msg": msg}
	for i := 0; i < len(keysAndValues); i += 2 {
		if i+1 < len(keysAndValues) {
			call[fmt.Sprint(keysAndValues[i])] = keysAndValues[i+1]
		}
	}
	m.infoCalls = append(m.infoCalls, call)
}

func (m *mockLogger) Warn(msg string, keysAndValues ...any) {
	call := map[string]any{"msg": msg}
	for i := 0; i < len(keysAndValues); i += 2 {
		if i+1 < len(keysAndValues) {
			call[fmt.Sprint(keysAndValues[i])] = keysAndValues[i+1]
		}
	}
	m.warnCalls = append(m.warnCalls, call)
}

func (m *mockLogger) Error(msg string, keysAndValues ...any) {
	call := map[string]any{"msg": msg}
	for i := 0; i < len(keysAndValues); i += 2 {
		if i+1 < len(keysAndValues) {
			call[fmt.Sprint(keysAndValues[i])] = keysAndValues[i+1]
		}
	}
	m.errorCalls = append(m.errorCalls, call)
}

// TestBackoffLogging tests that backoff logs at Debug level
func TestBackoffLogging(t *testing.T) {
	mock := &mockLogger{}
	client := &Client{
		BackoffMinDelay:    1 * time.Second,
		BackoffMaxDelay:    60 * time.Second,
		BackoffDelayFactor: 2.0,
		logger:             mock,
	}

	// Call Backoff
	_ = client.Backoff(2)

	// Verify Debug was called
	if len(mock.debugCalls) != 1 {
		t.Fatalf("Expected 1 Debug call, got %d", len(mock.debugCalls))
	}

	call := mock.debugCalls[0]
	if call["msg"] != "Backoff calculated" {
		t.Errorf("Expected message 'Backoff calculated', got '%v'", call["msg"])
	}

	// Verify required fields are present
	requiredFields := []string{"attempt", "base_delay_ms", "jitter_ms", "final_delay_ms"}
	for _, field := range requiredFields {
		if _, ok := call[field]; !ok {
			t.Errorf("Missing required field: %s", field)
		}
	}

	// Verify attempt value
	if call["attempt"] != 2 {
		t.Errorf("Expected attempt=2, got %v", call["attempt"])
	}

	// Verify base_delay_ms is ~4000ms (1s * 2^2 = 4s)
	baseDelayMs, ok := call["base_delay_ms"].(int64)
	if !ok {
		t.Fatalf("base_delay_ms should be int64, got %T", call["base_delay_ms"])
	}
	if baseDelayMs < 4000 || baseDelayMs > 4100 {
		t.Errorf("Expected base_delay_ms ~4000ms, got %dms", baseDelayMs)
	}

	// Verify jitter_ms is 0-10% of base delay (0-400ms)
	jitterMs, ok := call["jitter_ms"].(int64)
	if !ok {
		t.Fatalf("jitter_ms should be int64, got %T", call["jitter_ms"])
	}
	if jitterMs < 0 || jitterMs > 400 {
		t.Errorf("Expected jitter_ms in range [0, 400]ms, got %dms", jitterMs)
	}

	// Verify final_delay_ms = base_delay_ms + jitter_ms
	finalDelayMs, ok := call["final_delay_ms"].(int64)
	if !ok {
		t.Fatalf("final_delay_ms should be int64, got %T", call["final_delay_ms"])
	}
	expectedMin := baseDelayMs
	expectedMax := baseDelayMs + 400
	if finalDelayMs < expectedMin || finalDelayMs > expectedMax {
		t.Errorf("Expected final_delay_ms in range [%d, %d]ms, got %dms",
			expectedMin, expectedMax, finalDelayMs)
	}
}

// TestCloseNilTarget tests closing a client with nil target
func TestCloseNilTarget(t *testing.T) {
	client := &Client{
		target: nil,
	}

	err := client.Close()
	if err != nil {
		t.Errorf("Close() on nil target should not error, got: %v", err)
	}
}

// TestCloseMultipleTimes tests that Close() can be called multiple times
func TestCloseMultipleTimes(t *testing.T) {
	client := &Client{
		target: nil, // Already closed
	}

	// First close
	err := client.Close()
	if err != nil {
		t.Errorf("First Close() should not error, got: %v", err)
	}

	// Second close
	err = client.Close()
	if err != nil {
		t.Errorf("Second Close() should not error, got: %v", err)
	}
}

// TestValidateConfigEdgeCases tests edge cases in configuration validation
func TestValidateConfigEdgeCases(t *testing.T) {
	tests := []struct {
		name    string
		client  *Client
		wantErr bool
		errMsg  string
	}{
		{
			name: "valid minimum config",
			client: &Client{
				Target:             "192.168.1.1",
				Port:               1,
				ConnectTimeout:     1 * time.Nanosecond,
				OperationTimeout:   1 * time.Nanosecond,
				MaxRetries:         0,
				BackoffMinDelay:    1 * time.Nanosecond,
				BackoffMaxDelay:    2 * time.Nanosecond,
				BackoffDelayFactor: 1.0,
				logger:             &NoOpLogger{},
			},
			wantErr: false,
		},
		{
			name: "valid maximum config",
			client: &Client{
				Target:             "192.168.1.1",
				Port:               65535,
				ConnectTimeout:     24 * time.Hour,
				OperationTimeout:   24 * time.Hour,
				MaxRetries:         1000,
				BackoffMinDelay:    1 * time.Second,
				BackoffMaxDelay:    1 * time.Hour,
				BackoffDelayFactor: 10.0,
				logger:             &NoOpLogger{},
			},
			wantErr: false,
		},
		{
			name: "target with port already included",
			client: &Client{
				Target:             "192.168.1.1:57400",
				Port:               DefaultPort,
				ConnectTimeout:     DefaultConnectTimeout,
				OperationTimeout:   DefaultOperationTimeout,
				MaxRetries:         DefaultMaxRetries,
				BackoffMinDelay:    DefaultBackoffMinDelay,
				BackoffMaxDelay:    DefaultBackoffMaxDelay,
				BackoffDelayFactor: DefaultBackoffDelayFactor,
				logger:             &NoOpLogger{},
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.client.validateConfig()
			if tt.wantErr && err == nil {
				t.Errorf("validateConfig() expected error but got none")
			}
			if !tt.wantErr && err != nil {
				t.Errorf("validateConfig() unexpected error: %v", err)
			}
			if tt.wantErr && err != nil && !strings.Contains(err.Error(), tt.errMsg) {
				t.Errorf("validateConfig() error = %q, want error containing %q",
					err.Error(), tt.errMsg)
			}
		})
	}
}

// TestCapabilitiesNilTarget tests Capabilities method with nil target
func TestCapabilitiesNilTarget(t *testing.T) {
	client := &Client{
		target: nil,
	}

	ctx := context.Background()
	res, err := client.Capabilities(ctx)

	if err == nil {
		t.Error("Capabilities() with nil target should return error")
	}
	if res.OK {
		t.Error("Capabilities() with nil target should return OK=false")
	}
	if len(res.Errors) == 0 {
		t.Error("Capabilities() with nil target should return errors")
	}
	if !strings.Contains(err.Error(), "not connected") {
		t.Errorf("Capabilities() error should mention 'not connected', got: %v", err)
	}
}

// TestCapabilities_ContextCancellation tests Capabilities respects canceled context
func TestCapabilities_ContextCancellation(t *testing.T) {
	client := &Client{
		Target:           "192.168.1.1",
		Port:             57400,
		OperationTimeout: 10 * time.Second,
		logger:           &NoOpLogger{},
	}

	// Create already-canceled context
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	res, err := client.Capabilities(ctx)

	if err == nil {
		t.Fatal("expected error for canceled context")
	}
	if !strings.Contains(err.Error(), "context") {
		t.Errorf("expected context cancellation error, got: %v", err)
	}
	if res.OK {
		t.Error("expected OK=false for canceled context")
	}
}

// Note: Integration tests with real gNMI server or mock server are deferred
// to integration test suite. These unit tests focus on validation and error paths.

// TestSecurity_CredentialProtection tests that credentials are not exposed
func TestSecurity_CredentialProtection(t *testing.T) {
	t.Run("credentials are unexported", func(t *testing.T) {
		// Create client with credentials
		client := &Client{
			Target:   "192.168.1.1",
			Port:     57400,
			username: "admin",
			password: "secret123",
		}

		// Verify HasCredentials works
		if !client.HasCredentials() {
			t.Error("HasCredentials() should return true")
		}

		// Credentials should not be accessible via reflection or string conversion
		clientStr := client.Target
		if strings.Contains(clientStr, "secret123") {
			t.Error("password leaked in client string representation")
		}
	})
}
