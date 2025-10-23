// SPDX-License-Identifier: MPL-2.0
// Copyright (c) 2025 Daniel Schmidt

package gnmi

import "time"

// Client configuration options using the functional options pattern

// Username sets the username for gNMI authentication
func Username(username string) func(*Client) {
	return func(c *Client) {
		c.username = username
	}
}

// Password sets the password for gNMI authentication
func Password(password string) func(*Client) {
	return func(c *Client) {
		c.password = password
	}
}

// TLSCert sets the TLS certificate file path for authentication
//
// The certificate will be loaded and validated when the connection is established.
// If the certificate file cannot be read, an error will be returned during connection.
func TLSCert(certPath string) func(*Client) {
	return func(c *Client) {
		c.tlsCert = certPath
	}
}

// TLSKey sets the TLS private key file path for authentication
//
// The key will be loaded and validated when the connection is established.
// If the key file cannot be read, an error will be returned during connection.
func TLSKey(keyPath string) func(*Client) {
	return func(c *Client) {
		c.tlsKey = keyPath
	}
}

// TLSCA sets the TLS CA certificate file path for server verification
//
// The CA certificate will be loaded and validated when the connection is established.
// If the CA file cannot be read, an error will be returned during connection.
func TLSCA(caPath string) func(*Client) {
	return func(c *Client) {
		c.tlsCA = caPath
	}
}

// Port sets the gNMI port (default: 57400)
func Port(port int) func(*Client) {
	return func(c *Client) {
		c.Port = port
	}
}

// TLS enables or disables TLS (default: true)
//
// WARNING: Disabling TLS makes the connection vulnerable to eavesdropping
// and Man-in-the-Middle attacks. Only use this in isolated testing
// environments where security is not a concern.
func TLS(enabled bool) func(*Client) {
	return func(c *Client) {
		c.UseTLS = enabled
	}
}

// VerifyCertificate enables or disables TLS certificate verification (default: true)
//
// WARNING: Disabling certificate verification makes the connection vulnerable
// to Man-in-the-Middle attacks. Only use this in testing environments where
// security is not a concern.
//
// Example:
//
//	client, _ := gnmi.NewClient("192.168.1.1:57400",
//	    gnmi.Username("admin"),
//	    gnmi.Password("secret"),
//	    gnmi.VerifyCertificate(false))  // Insecure, use only for testing
func VerifyCertificate(verify bool) func(*Client) {
	return func(c *Client) {
		c.VerifyCertificate = verify
	}
}

// ConnectTimeout sets the connection timeout (default: 30s)
func ConnectTimeout(duration time.Duration) func(*Client) {
	return func(c *Client) {
		c.ConnectTimeout = duration
	}
}

// OperationTimeout sets the operation timeout (default: 15s)
func OperationTimeout(duration time.Duration) func(*Client) {
	return func(c *Client) {
		c.OperationTimeout = duration
	}
}

// MaxRetries sets the maximum number of retry attempts for transient errors (default: 3)
func MaxRetries(retries int) func(*Client) {
	return func(c *Client) {
		c.MaxRetries = retries
	}
}

// BackoffMinDelay sets the minimum backoff delay (default: 1s)
func BackoffMinDelay(duration time.Duration) func(*Client) {
	return func(c *Client) {
		c.BackoffMinDelay = duration
	}
}

// BackoffMaxDelay sets the maximum backoff delay (default: 60s)
func BackoffMaxDelay(duration time.Duration) func(*Client) {
	return func(c *Client) {
		c.BackoffMaxDelay = duration
	}
}

// BackoffDelayFactor sets the backoff multiplication factor (default: 2.0)
func BackoffDelayFactor(factor float64) func(*Client) {
	return func(c *Client) {
		c.BackoffDelayFactor = factor
	}
}

// WithLogger configures a custom logger for the client
//
// By default, the client uses NoOpLogger which discards all log messages.
// Use this option to enable logging with DefaultLogger or a custom logger.
//
// All JSON content logged at Debug level is automatically redacted to remove
// sensitive data (passwords, secrets, keys, tokens).
//
// Example (DefaultLogger):
//
//	logger := gnmi.NewDefaultLogger(gnmi.LogLevelInfo)
//	client, _ := gnmi.NewClient("192.168.1.1:57400",
//	    gnmi.Username("admin"),
//	    gnmi.Password("secret"),
//	    gnmi.WithLogger(logger))
//
// Example (Custom Logger):
//
//	type SlogAdapter struct {
//	    logger *slog.Logger
//	}
//
//	func (s *SlogAdapter) Debug(ctx context.Context, msg string, keysAndValues ...interface{}) {
//	    s.logger.DebugContext(ctx, msg, keysAndValues...)
//	}
//	// ... implement Info, Warn, Error (all with ctx context.Context as first parameter)
//
//	client, _ := gnmi.NewClient("192.168.1.1:57400",
//	    gnmi.WithLogger(&SlogAdapter{logger: slog.Default()}))
func WithLogger(logger Logger) func(*Client) {
	return func(c *Client) {
		if logger != nil {
			c.logger = logger
		}
	}
}

// WithPrettyPrintLogs enables/disables JSON pretty printing in logs
//
// When enabled, JSON content in debug logs is formatted for better
// readability. When disabled (default), raw JSON is logged without formatting.
//
// This only affects Debug-level log output. Disabling pretty printing
// can improve performance when high-frequency operations are logged.
//
// Default: disabled (false)
//
// Example:
//
//	logger := gnmi.NewDefaultLogger(gnmi.LogLevelDebug)
//	client, _ := gnmi.NewClient("192.168.1.1:57400",
//	    gnmi.Username("admin"),
//	    gnmi.Password("secret"),
//	    gnmi.WithLogger(logger),
//	    gnmi.WithPrettyPrintLogs(true))  // Enable formatting for readability
func WithPrettyPrintLogs(enabled bool) func(*Client) {
	return func(c *Client) {
		c.prettyPrintLogs = enabled
	}
}

// Request modifiers for individual operations

// Timeout returns a request modifier that sets a custom timeout for the operation.
//
// This timeout takes precedence over the context deadline and client's
// OperationTimeout. Use this to set operation-specific timeouts that differ
// from the client's default.
//
// The timeout priority model is:
//  1. Request-specific timeout (this modifier) - highest priority
//  2. Context deadline (if already set) - medium priority
//  3. Client.OperationTimeout - fallback default
//
// Example:
//
//	// Get with 30 second timeout
//	res, err := client.Get(ctx, []string{"/interfaces"},
//	    gnmi.Timeout(30*time.Second))
//
//	// Set with 2 minute timeout for long-running operation
//	res, err := client.Set(ctx, ops,
//	    gnmi.Timeout(2*time.Minute))
func Timeout(duration time.Duration) func(*Req) {
	return func(req *Req) {
		req.Timeout = duration
	}
}

// GetEncoding returns a request modifier that sets the encoding for Get operations.
//
// Valid encodings: json, json_ietf (default), proto, ascii, bytes
//
// This encoding overrides the default encoding (json_ietf) for this specific
// request. Note that the device must support the specified encoding.
//
// Common encodings:
//   - json_ietf: JSON with IETF conventions (recommended, default)
//   - json: Standard JSON encoding
//   - proto: Protocol Buffer encoding (binary)
//   - ascii: ASCII text encoding
//   - bytes: Raw byte encoding
//
// The modifier validates the encoding at request time. If an invalid encoding
// is provided, the operation will fail with an error.
//
// Example:
//
//	// Get with Protocol Buffer encoding
//	res, err := client.Get(ctx, []string{"/system/config"},
//	    gnmi.GetEncoding("proto"))
//
//	// Get with standard JSON
//	res, err := client.Get(ctx, []string{"/interfaces"},
//	    gnmi.GetEncoding("json"))
//
//	// Combined with timeout
//	res, err := client.Get(ctx, []string{"/interfaces"},
//	    gnmi.Timeout(30*time.Second),
//	    gnmi.GetEncoding("json_ietf"))
func GetEncoding(encoding string) func(*Req) {
	return func(req *Req) {
		req.Encoding = encoding
	}
}

// SetEncoding returns a modifier that sets the encoding for individual Set operations.
//
// Valid encodings: json, json_ietf (default), proto, ascii, bytes
//
// This modifier is used with Update() and Replace() operations to specify
// the encoding for individual operations within a Set request. Each operation
// can have a different encoding.
//
// Common encodings:
//   - json_ietf: JSON with IETF conventions (recommended, default)
//   - json: Standard JSON encoding
//   - proto: Protocol Buffer encoding (binary)
//   - ascii: ASCII text encoding
//   - bytes: Raw byte encoding
//
// Example:
//
//	// Set with Protocol Buffer encoding
//	ops := []gnmi.SetOperation{
//	    gnmi.Update("/interfaces/interface[name=Gi0]/config", protoBytes,
//	        gnmi.SetEncoding("proto")),
//	}
//
//	// Set with standard JSON (non-IETF)
//	ops := []gnmi.SetOperation{
//	    gnmi.Replace("/system/config", jsonData,
//	        gnmi.SetEncoding("json")),
//	}
//
//	// Default encoding (json_ietf) - no encoding parameter needed
//	ops := []gnmi.SetOperation{
//	    gnmi.Update("/system/hostname", `{"hostname": "router1"}`),
//	}
func SetEncoding(encoding string) func(*SetOperation) {
	return func(op *SetOperation) {
		if encoding != "" {
			op.Encoding = encoding
		}
	}
}
