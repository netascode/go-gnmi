// SPDX-License-Identifier: MPL-2.0
// Copyright (c) 2025 Daniel Schmidt

package gnmi

import (
	"bytes"
	"log"
	"strings"
	"testing"
	"time"
)

// TestUsernameOption tests the Username functional option
func TestUsernameOption(t *testing.T) {
	client := &Client{}
	opt := Username("admin")
	opt(client)

	if client.username != "admin" {
		t.Errorf("Username() set username to %q, want %q", client.username, "admin")
	}
}

// TestPasswordOption tests the Password functional option
func TestPasswordOption(t *testing.T) {
	client := &Client{}
	opt := Password("secret123")
	opt(client)

	if client.password != "secret123" {
		t.Errorf("Password() set password to %q, want %q", client.password, "secret123")
	}
}

// TestTLSCertOption tests the TLSCert functional option
func TestTLSCertOption(t *testing.T) {
	client := &Client{}
	opt := TLSCert("/path/to/cert.pem")
	opt(client)

	if client.tlsCert != "/path/to/cert.pem" {
		t.Errorf("TLSCert() set tlsCert to %q, want %q", client.tlsCert, "/path/to/cert.pem")
	}
}

// TestTLSKeyOption tests the TLSKey functional option
func TestTLSKeyOption(t *testing.T) {
	client := &Client{}
	opt := TLSKey("/path/to/key.pem")
	opt(client)

	if client.tlsKey != "/path/to/key.pem" {
		t.Errorf("TLSKey() set tlsKey to %q, want %q", client.tlsKey, "/path/to/key.pem")
	}
}

// TestTLSCAOption tests the TLSCA functional option
func TestTLSCAOption(t *testing.T) {
	client := &Client{}
	opt := TLSCA("/path/to/ca.pem")
	opt(client)

	if client.tlsCA != "/path/to/ca.pem" {
		t.Errorf("TLSCA() set tlsCA to %q, want %q", client.tlsCA, "/path/to/ca.pem")
	}
}

// TestPortOption tests the Port functional option
func TestPortOption(t *testing.T) {
	tests := []struct {
		name string
		port int
		want int
	}{
		{
			name: "default port",
			port: 57400,
			want: 57400,
		},
		{
			name: "custom port",
			port: 8443,
			want: 8443,
		},
		{
			name: "low port",
			port: 22,
			want: 22,
		},
		{
			name: "high port",
			port: 65535,
			want: 65535,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := &Client{}
			opt := Port(tt.port)
			opt(client)

			if client.Port != tt.want {
				t.Errorf("Port() set Port to %d, want %d", client.Port, tt.want)
			}
		})
	}
}

// TestTLSOption tests the TLS functional option
func TestTLSOption(t *testing.T) {
	tests := []struct {
		name    string
		enabled bool
	}{
		{
			name:    "TLS enabled",
			enabled: true,
		},
		{
			name:    "TLS disabled",
			enabled: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := &Client{}
			opt := TLS(tt.enabled)
			opt(client)

			if client.UseTLS != tt.enabled {
				t.Errorf("TLS() set UseTLS to %v, want %v", client.UseTLS, tt.enabled)
			}
		})
	}
}

// TestVerifyCertificateOption tests the VerifyCertificate functional option
func TestVerifyCertificateOption(t *testing.T) {
	tests := []struct {
		name   string
		verify bool
	}{
		{
			name:   "certificate verification enabled",
			verify: true,
		},
		{
			name:   "certificate verification disabled",
			verify: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := &Client{}
			opt := VerifyCertificate(tt.verify)
			opt(client)

			if client.VerifyCertificate != tt.verify {
				t.Errorf("VerifyCertificate() set VerifyCertificate to %v, want %v",
					client.VerifyCertificate, tt.verify)
			}
			// Note: InsecureSkipVerify is set by NewClient() after applying options,
			// so we don't test it here (see TestNewClientValidation for full flow)
		})
	}
}

// TestConnectTimeoutOption tests the ConnectTimeout functional option
func TestConnectTimeoutOption(t *testing.T) {
	tests := []struct {
		name    string
		timeout time.Duration
	}{
		{
			name:    "default connect timeout",
			timeout: 30 * time.Second,
		},
		{
			name:    "short connect timeout",
			timeout: 5 * time.Second,
		},
		{
			name:    "long connect timeout",
			timeout: 2 * time.Minute,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := &Client{}
			opt := ConnectTimeout(tt.timeout)
			opt(client)

			if client.ConnectTimeout != tt.timeout {
				t.Errorf("ConnectTimeout() set ConnectTimeout to %v, want %v",
					client.ConnectTimeout, tt.timeout)
			}
		})
	}
}

// TestOperationTimeoutOption tests the OperationTimeout functional option
func TestOperationTimeoutOption(t *testing.T) {
	tests := []struct {
		name    string
		timeout time.Duration
	}{
		{
			name:    "default operation timeout",
			timeout: 60 * time.Second,
		},
		{
			name:    "short operation timeout",
			timeout: 15 * time.Second,
		},
		{
			name:    "long operation timeout",
			timeout: 5 * time.Minute,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := &Client{}
			opt := OperationTimeout(tt.timeout)
			opt(client)

			if client.OperationTimeout != tt.timeout {
				t.Errorf("OperationTimeout() set OperationTimeout to %v, want %v",
					client.OperationTimeout, tt.timeout)
			}
		})
	}
}

// TestMaxRetriesOption tests the MaxRetries functional option
func TestMaxRetriesOption(t *testing.T) {
	tests := []struct {
		name    string
		retries int
	}{
		{
			name:    "default retries",
			retries: 3,
		},
		{
			name:    "no retries",
			retries: 0,
		},
		{
			name:    "many retries",
			retries: 10,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := &Client{}
			opt := MaxRetries(tt.retries)
			opt(client)

			if client.MaxRetries != tt.retries {
				t.Errorf("MaxRetries() set MaxRetries to %d, want %d",
					client.MaxRetries, tt.retries)
			}
		})
	}
}

// TestBackoffMinDelayOption tests the BackoffMinDelay functional option
func TestBackoffMinDelayOption(t *testing.T) {
	tests := []struct {
		name  string
		delay time.Duration
	}{
		{
			name:  "default min delay",
			delay: 1 * time.Second,
		},
		{
			name:  "short min delay",
			delay: 100 * time.Millisecond,
		},
		{
			name:  "long min delay",
			delay: 10 * time.Second,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := &Client{}
			opt := BackoffMinDelay(tt.delay)
			opt(client)

			if client.BackoffMinDelay != tt.delay {
				t.Errorf("BackoffMinDelay() set BackoffMinDelay to %v, want %v",
					client.BackoffMinDelay, tt.delay)
			}
		})
	}
}

// TestBackoffMaxDelayOption tests the BackoffMaxDelay functional option
func TestBackoffMaxDelayOption(t *testing.T) {
	tests := []struct {
		name  string
		delay time.Duration
	}{
		{
			name:  "default max delay",
			delay: 60 * time.Second,
		},
		{
			name:  "short max delay",
			delay: 10 * time.Second,
		},
		{
			name:  "long max delay",
			delay: 5 * time.Minute,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := &Client{}
			opt := BackoffMaxDelay(tt.delay)
			opt(client)

			if client.BackoffMaxDelay != tt.delay {
				t.Errorf("BackoffMaxDelay() set BackoffMaxDelay to %v, want %v",
					client.BackoffMaxDelay, tt.delay)
			}
		})
	}
}

// TestBackoffDelayFactorOption tests the BackoffDelayFactor functional option
func TestBackoffDelayFactorOption(t *testing.T) {
	tests := []struct {
		name   string
		factor float64
	}{
		{
			name:   "default factor",
			factor: 2.0,
		},
		{
			name:   "linear backoff",
			factor: 1.0,
		},
		{
			name:   "aggressive backoff",
			factor: 3.0,
		},
		{
			name:   "fractional factor",
			factor: 1.5,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := &Client{}
			opt := BackoffDelayFactor(tt.factor)
			opt(client)

			if client.BackoffDelayFactor != tt.factor {
				t.Errorf("BackoffDelayFactor() set BackoffDelayFactor to %v, want %v",
					client.BackoffDelayFactor, tt.factor)
			}
		})
	}
}

// TestWithLoggerOption tests the WithLogger functional option
func TestWithLoggerOption(t *testing.T) {
	customLogger := &DefaultLogger{level: LogLevelDebug}
	client := &Client{}
	opt := WithLogger(customLogger)
	opt(client)

	if client.logger != customLogger {
		t.Error("WithLogger() did not set custom logger")
	}
}

// TestWithPrettyPrintLogsOption tests the WithPrettyPrintLogs functional option
func TestWithPrettyPrintLogsOption(t *testing.T) {
	tests := []struct {
		name    string
		enabled bool
	}{
		{
			name:    "pretty print enabled",
			enabled: true,
		},
		{
			name:    "pretty print disabled",
			enabled: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := &Client{}
			opt := WithPrettyPrintLogs(tt.enabled)
			opt(client)

			if client.prettyPrintLogs != tt.enabled {
				t.Errorf("WithPrettyPrintLogs() set prettyPrintLogs to %v, want %v",
					client.prettyPrintLogs, tt.enabled)
			}
		})
	}
}

// TestTimeoutRequestModifier tests the Timeout request modifier
func TestTimeoutRequestModifier(t *testing.T) {
	tests := []struct {
		name    string
		timeout time.Duration
	}{
		{
			name:    "custom timeout",
			timeout: 30 * time.Second,
		},
		{
			name:    "short timeout",
			timeout: 5 * time.Second,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := &Req{}
			mod := Timeout(tt.timeout)
			mod(req)

			if req.Timeout != tt.timeout {
				t.Errorf("Timeout() set Timeout to %v, want %v", req.Timeout, tt.timeout)
			}
		})
	}
}

// TestEncodingRequestModifier tests the Encoding request modifier
func TestEncodingRequestModifier(t *testing.T) {
	tests := []struct {
		name     string
		encoding string
	}{
		{
			name:     "json encoding",
			encoding: EncodingJSON,
		},
		{
			name:     "json_ietf encoding",
			encoding: EncodingJSONIETF,
		},
		{
			name:     "proto encoding",
			encoding: EncodingProto,
		},
		{
			name:     "ascii encoding",
			encoding: EncodingASCII,
		},
		{
			name:     "bytes encoding",
			encoding: EncodingBytes,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := &Req{}
			mod := GetEncoding(tt.encoding)
			mod(req)

			if req.Encoding != tt.encoding {
				t.Errorf("GetEncoding() set Encoding to %q, want %q", req.Encoding, tt.encoding)
			}
		})
	}
}

// TestOptionsCombination tests combining multiple functional options
func TestOptionsCombination(t *testing.T) {
	client := &Client{
		Port:               DefaultPort,
		ConnectTimeout:     DefaultConnectTimeout,
		OperationTimeout:   DefaultOperationTimeout,
		MaxRetries:         DefaultMaxRetries,
		BackoffMinDelay:    DefaultBackoffMinDelay,
		BackoffMaxDelay:    DefaultBackoffMaxDelay,
		BackoffDelayFactor: DefaultBackoffDelayFactor,
	}

	// Apply multiple options
	Username("admin")(client)
	Password("secret")(client)
	Port(8443)(client)
	TLS(true)(client)
	VerifyCertificate(false)(client)
	MaxRetries(5)(client)
	OperationTimeout(120 * time.Second)(client)

	// Verify all options applied
	if client.username != "admin" {
		t.Errorf("username = %q, want %q", client.username, "admin")
	}
	if client.password != "secret" {
		t.Errorf("password = %q, want %q", client.password, "secret")
	}
	if client.Port != 8443 {
		t.Errorf("Port = %d, want %d", client.Port, 8443)
	}
	if !client.UseTLS {
		t.Error("UseTLS = false, want true")
	}
	if client.VerifyCertificate {
		t.Error("VerifyCertificate = true, want false")
	}
	// Note: InsecureSkipVerify is set by NewClient() after applying options
	if client.MaxRetries != 5 {
		t.Errorf("MaxRetries = %d, want %d", client.MaxRetries, 5)
	}
	if client.OperationTimeout != 120*time.Second {
		t.Errorf("OperationTimeout = %v, want %v", client.OperationTimeout, 120*time.Second)
	}
}

// TestSecurityWarnings tests security-related warnings
func TestSecurityWarnings(t *testing.T) {
	tests := []struct {
		name              string
		options           []func(*Client)
		expectWarnings    []string
		notExpectWarnings []string
	}{
		{
			name: "InsecureSkipVerify warning",
			options: []func(*Client){
				Username("admin"),
				Password("test"),
				VerifyCertificate(false),
			},
			expectWarnings: []string{
				"InsecureSkipVerify enabled",
				"Man-in-the-Middle attacks possible",
				"Use only in testing environments",
			},
		},
		{
			name: "TLS disabled warning",
			options: []func(*Client){
				Username("admin"),
				Password("test"),
				TLS(false),
			},
			expectWarnings: []string{
				"TLS disabled",
				"connection is not encrypted",
				"Enable TLS for production use",
			},
		},
		{
			name: "No credentials warning",
			options: []func(*Client){
				TLS(true),
				VerifyCertificate(true),
			},
			expectWarnings: []string{
				"No credentials configured",
				"device may reject connection",
			},
		},
		{
			name: "Secure configuration (no warnings)",
			options: []func(*Client){
				Username("admin"),
				Password("test"),
				TLS(true),
				VerifyCertificate(true),
			},
			notExpectWarnings: []string{
				"InsecureSkipVerify",
				"TLS disabled",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Capture log output
			var buf bytes.Buffer
			log.SetOutput(&buf)
			t.Cleanup(func() { log.SetOutput(nil) })

			logger := NewDefaultLogger(LogLevelWarn)
			opts := append(tt.options, WithLogger(logger))

			// Create client (will fail to connect, but we only care about warnings)
			_, _ = NewClient("192.168.1.1:57400", opts...) //nolint:errcheck // Error intentionally ignored in test

			output := buf.String()

			// Check expected warnings
			for _, warning := range tt.expectWarnings {
				if !strings.Contains(output, warning) {
					t.Errorf("expected warning containing %q but got:\n%s", warning, output)
				}
			}

			// Check unexpected warnings
			for _, warning := range tt.notExpectWarnings {
				if strings.Contains(output, warning) {
					t.Errorf("unexpected warning containing %q in output:\n%s", warning, output)
				}
			}
		})
	}
}
