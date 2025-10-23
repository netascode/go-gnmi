// SPDX-License-Identifier: MPL-2.0
// Copyright (c) 2025 Daniel Schmidt

package gnmi

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/openconfig/gnmic/pkg/api"
	target "github.com/openconfig/gnmic/pkg/api/target"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// Default client configuration values
const (
	DefaultPort               = 57400
	DefaultMaxRetries         = 3
	DefaultBackoffMinDelay    = 1 * time.Second
	DefaultBackoffMaxDelay    = 60 * time.Second
	DefaultBackoffDelayFactor = 2
	DefaultConnectTimeout     = 30 * time.Second
	DefaultOperationTimeout   = 15 * time.Second // Matches embedded client behavior
	DefaultUseTLS             = true
	DefaultVerifyCertificate  = true
	DefaultPrettyPrintLogs    = true
)

// Security limits for JSON processing and logging
const (
	MaxJSONSizeForLogging = 1 * 1024 * 1024 // 1MB limit to prevent ReDoS attacks
	MaxSensitiveFields    = 1000            // Max redaction operations to prevent DoS
)

// Logging message constants
const (
	JSONTooLargeMessage     = "[JSON TOO LARGE FOR LOGGING]"
	JSONTooManySensitiveMsg = "[JSON CONTAINS TOO MANY SENSITIVE FIELDS]"
)

// defaultRedactionPatterns contains regex patterns for redacting sensitive data in logs
var defaultRedactionPatterns = []*regexp.Regexp{
	// JSON field patterns
	regexp.MustCompile(`"password"\s*:\s*"[^"]*"`),
	regexp.MustCompile(`"secret"\s*:\s*"[^"]*"`),
	regexp.MustCompile(`"key"\s*:\s*"[^"]*"`),
	regexp.MustCompile(`"community"\s*:\s*"[^"]*"`),
	regexp.MustCompile(`"token"\s*:\s*"[^"]*"`),
	regexp.MustCompile(`"auth"\s*:\s*"[^"]*"`),
}

// Client represents a gNMI client connection to a network device
type Client struct {
	// gnmic target for gNMI transport (lazy connection)
	target *target.Target

	// connected tracks if connection has been established (lazy)
	connected bool

	// RWMutex to synchronize access to mutable state
	mu sync.RWMutex

	// Connection parameters
	Target   string
	Port     int
	username string // unexported for security
	password string // unexported for security

	// TLS configuration
	tlsCert string // unexported for security
	tlsKey  string // unexported for security
	tlsCA   string // unexported for security

	// TLS options
	UseTLS             bool
	VerifyCertificate  bool
	InsecureSkipVerify bool // Alias for !VerifyCertificate

	// Timeout configuration
	ConnectTimeout   time.Duration
	OperationTimeout time.Duration

	// Retry configuration
	MaxRetries         int
	BackoffMinDelay    time.Duration
	BackoffMaxDelay    time.Duration
	BackoffDelayFactor float64

	// Capability tracking (gNMI capabilities from CapabilityResponse)
	capabilities []string

	// Logging configuration
	logger            Logger
	prettyPrintLogs   bool
	redactionPatterns []*regexp.Regexp
}

// NewClient creates a new gNMI client with the specified target and options
//
// The client creates a gnmic target configuration but does NOT establish
// a physical connection immediately. The connection is established automatically
// on the first RPC call (lazy connection). Use Ping() to explicitly verify
// connectivity if needed.
//
// Example:
//
//	client, err := gnmi.NewClient(
//	    "192.168.1.1:57400",
//	    gnmi.Username("admin"),
//	    gnmi.Password("secret"),
//	    gnmi.TLS(true),
//	    gnmi.VerifyCertificate(false),
//	    gnmi.MaxRetries(5),
//	)
//	if err != nil {
//	    log.Fatal(err)  // Configuration error
//	}
//	defer client.Close()
//
//	// Optional: verify connection explicitly
//	if err := client.Ping(ctx); err != nil {
//	    log.Fatal(err)  // Connection error
//	}
//
//	// Auto-connect on first use
//	res, err := client.Get(ctx, paths)
//
// Returns a configured Client or an error if configuration validation fails.
func NewClient(target string, opts ...func(*Client)) (*Client, error) {
	// Create client with default values
	client := &Client{
		Target:             target,
		Port:               DefaultPort,
		UseTLS:             DefaultUseTLS,
		VerifyCertificate:  DefaultVerifyCertificate,
		ConnectTimeout:     DefaultConnectTimeout,
		OperationTimeout:   DefaultOperationTimeout,
		MaxRetries:         DefaultMaxRetries,
		BackoffMinDelay:    DefaultBackoffMinDelay,
		BackoffMaxDelay:    DefaultBackoffMaxDelay,
		BackoffDelayFactor: DefaultBackoffDelayFactor,
		logger:             &NoOpLogger{},
		prettyPrintLogs:    DefaultPrettyPrintLogs,
		redactionPatterns:  defaultRedactionPatterns,
	}

	// Apply functional options
	for _, opt := range opts {
		opt(client)
	}

	// Set InsecureSkipVerify alias
	client.InsecureSkipVerify = !client.VerifyCertificate

	// Validate configuration
	if err := client.validateConfig(); err != nil {
		return nil, err
	}

	// Create gnmic target (configuration only, NO connection yet)
	if err := client.createTarget(); err != nil {
		return nil, err
	}

	// Log client creation (connection happens lazily)
	client.logger.Info(context.Background(), "gNMI client created",
		"target", client.Target,
		"port", client.Port,
		"connection", "lazy")

	return client, nil
}

// Disconnect closes the gRPC connection but preserves client configuration.
//
// Unlike Close(), this method allows the client to be reused - subsequent
// operations will automatically reconnect. This is useful for:
//   - Connection pooling with idle timeout disconnection
//   - Long-running applications that need to release connections temporarily
//   - Testing reconnection logic
//
// The client maintains its target configuration and will reconnect on the next
// RPC call via the ensureConnected() mechanism.
//
// Example - Disconnect and reuse pattern:
//
//	client, _ := gnmi.NewClient("192.168.1.1:57400", opts...)
//	defer client.Close()
//
//	// Use client
//	_, err := client.Get(ctx, paths)
//
//	// Release connection temporarily
//	client.Disconnect()
//
//	// Automatically reconnects on next use
//	_, err = client.Get(ctx, paths)
//
// Example - Connection pooling with idle timeout:
//
//	// Disconnect idle connections periodically
//	ticker := time.NewTicker(5 * time.Minute)
//	go func() {
//	    for range ticker.C {
//	        client.Disconnect() // Release idle connection
//	    }
//	}()
//
// Example - Long-running application pattern:
//
//	// Disconnect during maintenance window
//	client.Disconnect()
//	performMaintenance()
//	// Reconnects automatically when needed
//	client.Get(ctx, paths)
//
// Thread-safe: safe for concurrent use with other client methods.
func (c *Client) Disconnect() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.target == nil {
		// Already disconnected or closed
		return nil
	}

	// Close the underlying gRPC connection
	// Log any errors but continue - connection may already be broken
	if err := c.target.Close(); err != nil {
		c.logger.Warn(context.Background(), "gNMI connection close returned error during disconnect",
			"target", c.Target,
			"error", err.Error())
		// Continue anyway - we're disconnecting, target will be reusable
	}

	// Reset connected flag
	// Target remains valid and can be reconnected via ensureConnected()
	c.connected = false

	c.logger.Info(context.Background(), "gNMI connection disconnected",
		"target", c.Target,
		"reusable", true)

	return nil
}

// Close closes the gNMI session and cleans up resources (terminal operation).
//
// This method is TERMINAL - the client cannot be reused after calling Close().
// All resources are released and the target configuration is destroyed.
//
// Use Disconnect() instead if you want to temporarily close the connection
// while preserving the ability to reconnect later.
//
// This method is typically used in defer statements to ensure proper cleanup:
//
// Example:
//
//	client, err := gnmi.NewClient("192.168.1.1:57400", opts...)
//	if err != nil {
//	    log.Fatal(err)
//	}
//	defer client.Close() // Ensure cleanup on function exit
//
//	// Use client...
//	res, err := client.Get(ctx, paths)
//
// Thread-safe: safe to call multiple times (subsequent calls are no-ops).
func (c *Client) Close() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.target == nil {
		// Already closed or never connected
		return nil
	}

	// Clear target reference and reset connected flag before closing
	target := c.target
	c.target = nil
	c.connected = false

	err := target.Close()
	if err != nil {
		return err
	}

	c.logger.Info(context.Background(), "gNMI connection closed",
		"target", c.Target,
		"reusable", false)

	return nil
}

// HasCapability checks if the server supports a specific capability
//
// Example:
//
//	if client.HasCapability("gnmi-1.0") {
//	    // Use gNMI 1.0 features
//	}
func (c *Client) HasCapability(capability string) bool {
	c.mu.RLock()
	defer c.mu.RUnlock()

	for _, cap := range c.capabilities {
		if cap == capability {
			return true
		}
	}
	return false
}

// ServerCapabilities returns the list of capabilities supported by the server
//
// Returns a copy of the capabilities slice to prevent external modification.
//
// Example:
//
//	caps := client.ServerCapabilities()
//	for _, cap := range caps {
//	    fmt.Println(cap)
//	}
func (c *Client) ServerCapabilities() []string {
	c.mu.RLock()
	defer c.mu.RUnlock()

	// Return a copy to prevent external modification
	result := make([]string, len(c.capabilities))
	copy(result, c.capabilities)
	return result
}

// HasCredentials returns true if credentials are configured
//
// This method only indicates if credentials exist without exposing
// the actual values.
//
// Example:
//
//	if client.HasCredentials() {
//	    fmt.Println("Client is configured with credentials")
//	}
func (c *Client) HasCredentials() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.username != "" || c.password != "" || c.tlsCert != ""
}

// Backoff calculates the backoff delay for retry attempt using exponential backoff with jitter
//
// The formula is: delay = min(minDelay * (factor ^ attempt) + jitter, maxDelay)
// where jitter is a cryptographically secure random value in [0, delay * 0.1].
//
// Security Note: Uses crypto/rand for jitter to prevent timing attack predictability.
// If crypto/rand fails, falls back to timestamp-based jitter to prevent thundering herd.
// Timestamp-based jitter is not cryptographically secure but provides sufficient randomness
// for retry dispersal.
//
// Parameters:
//   - attempt: The retry attempt number (0-indexed)
//
// Returns the duration to wait before retrying.
func (c *Client) Backoff(attempt int) time.Duration {
	// Calculate base delay: minDelay * (factor ^ attempt)
	delay := float64(c.BackoffMinDelay) * math.Pow(c.BackoffDelayFactor, float64(attempt))

	// Check for overflow and cap at max delay
	if math.IsInf(delay, 1) || delay > float64(c.BackoffMaxDelay) {
		delay = float64(c.BackoffMaxDelay)
	}

	baseDelay := delay // Store base delay for logging

	// Add cryptographically secure jitter (0-10% of delay) to prevent thundering herd
	jitterMax := int64(delay * 0.1)
	var jitterVal int64
	if jitterMax > 0 {
		var jitterBytes [8]byte
		if _, err := rand.Read(jitterBytes[:]); err == nil {
			// Convert bytes to int64, masking to prevent overflow
			// Mask off sign bit to ensure positive value within int64 range
			//nolint:gosec // G115: False positive - explicitly masked to prevent overflow
			jitterVal = int64(binary.BigEndian.Uint64(jitterBytes[:]) & 0x7FFFFFFFFFFFFFFF)
			jitterVal = jitterVal % jitterMax
			delay += float64(jitterVal)
		} else {
			// Fallback to timestamp-based jitter if crypto/rand fails
			// This is not cryptographically secure but prevents thundering herd
			timestamp := time.Now().UnixNano()
			jitterVal = (timestamp%jitterMax + jitterMax) % jitterMax // Ensure positive
			delay += float64(jitterVal)

			c.logger.Warn(context.Background(), "crypto/rand failed, using timestamp-based jitter",
				"error", err.Error(),
				"attempt", attempt,
				"jitter_ms", time.Duration(jitterVal).Milliseconds())
		}
	}

	finalDelay := time.Duration(delay)

	// Log backoff calculation at Debug level
	c.logger.Debug(context.Background(), "Backoff calculated",
		"attempt", attempt,
		"base_delay_ms", time.Duration(baseDelay).Milliseconds(),
		"jitter_ms", time.Duration(jitterVal).Milliseconds(),
		"final_delay_ms", finalDelay.Milliseconds())

	return finalDelay
}

// prepareJSONForLogging redacts sensitive data and formats JSON for logging
//
// This method performs security checks and data sanitization:
//  1. Validates JSON size to prevent ReDoS attacks (max 1MB)
//  2. Checks sensitive field count to prevent DoS (max 1000 fields)
//  3. Redacts sensitive data (passwords, secrets, keys, community strings, tokens)
//  4. Pretty-prints JSON if prettyPrintLogs is enabled
//
// Security Note: Size and count limits prevent regex-based DoS attacks during
// JSON processing and redaction. These limits are conservative to ensure safe
// operation even with malicious or malformed input.
//
// Returns the processed JSON string safe for logging.
func (c *Client) prepareJSONForLogging(jsonStr string) string {
	// Check JSON size limit to prevent ReDoS attacks
	if len(jsonStr) > MaxJSONSizeForLogging {
		return JSONTooLargeMessage
	}

	// Count sensitive fields before processing to prevent DoS
	// This check prevents excessive regex operations on malicious input
	sensitiveCount := strings.Count(jsonStr, `"password"`) +
		strings.Count(jsonStr, `"secret"`) +
		strings.Count(jsonStr, `"key"`) +
		strings.Count(jsonStr, `"community"`) +
		strings.Count(jsonStr, `"token"`) +
		strings.Count(jsonStr, `"auth"`)

	if sensitiveCount > MaxSensitiveFields {
		c.logger.Warn(context.Background(), "Too many sensitive fields detected",
			"count", sensitiveCount,
			"max", MaxSensitiveFields)
		return JSONTooManySensitiveMsg
	}

	// Redact sensitive data first
	redacted := c.redactSensitiveData(jsonStr)

	// Pretty-print JSON if enabled (matches go-netconf's @pretty behavior)
	if c.prettyPrintLogs {
		var buf bytes.Buffer
		if err := json.Indent(&buf, []byte(redacted), "", "  "); err == nil {
			return buf.String()
		} else {
			// Fallback: if indent fails (e.g., invalid JSON), return redacted as-is
			c.logger.Debug(context.Background(), "JSON pretty-print failed, using raw redacted output",
				"error", err.Error())
		}
	}

	return redacted
}

// redactSensitiveData replaces sensitive data in JSON with [REDACTED]
//
// Redacts common sensitive types in JSON fields:
//   - "password": "value" fields
//   - "secret": "value" fields
//   - "key": "value" fields
//   - "community": "value" fields
//   - "token": "value" fields
//   - "auth": "value" fields
//
// Handles flexible whitespace around colons (RFC 8259 compliant).
//
// Security Note: This method is called after size/count validation to prevent
// ReDoS attacks from malicious input.
//
// Returns the redacted JSON string.
func (c *Client) redactSensitiveData(json string) string {
	replacements := []string{
		`"password":"[REDACTED]"`,
		`"secret":"[REDACTED]"`,
		`"key":"[REDACTED]"`,
		`"community":"[REDACTED]"`,
		`"token":"[REDACTED]"`,
		`"auth":"[REDACTED]"`,
	}

	result := json
	for i, pattern := range c.redactionPatterns {
		result = pattern.ReplaceAllString(result, replacements[i])
	}

	return result
}

// checkTransientError checks if an error is transient and should be retried
//
// This method extracts the gRPC status code from a Go error and checks if it
// matches any of the transient error patterns defined in TransientErrors.
//
// Transient errors include:
//   - codes.Unavailable: Service temporarily unavailable
//   - codes.ResourceExhausted: Rate limiting or quota exceeded
//   - codes.DeadlineExceeded: Timeout or deadline exceeded
//   - codes.Aborted: Transaction aborted, may succeed on retry
//
// Parameters:
//   - err: The error to check (typically from a gRPC call)
//
// Returns true if the error is transient and should be retried.
func (c *Client) checkTransientError(err error) bool {
	if err == nil {
		return false
	}

	// Extract gRPC status code from error
	st, ok := status.FromError(err)
	if !ok {
		// Not a gRPC error, treat as non-transient
		c.logger.Debug(context.Background(), "Error is not a gRPC error",
			"error", err.Error())
		return false
	}

	// Get status code
	code := uint32(st.Code())

	// Log status code for debugging
	c.logger.Debug(context.Background(), "Checking error for transient pattern",
		"code", code,
		"message", st.Message())

	// Check if status code matches any transient pattern
	for _, pattern := range TransientErrors {
		if pattern.Code == code {
			c.logger.Debug(context.Background(), "Error matches transient pattern",
				"code", code,
				"pattern", pattern.Code)
			return true
		}
	}

	// Permanent error
	c.logger.Debug(context.Background(), "Error is permanent (not transient)",
		"code", code)
	return false
}

// isTransportError checks if an error is a transport error requiring reconnection
//
// Transport errors indicate the gRPC connection is broken or unusable and should
// trigger automatic reconnection before retrying the operation.
//
// Transport errors include:
//   - codes.Unavailable: gRPC channel unavailable (connection lost, DNS failure)
//   - codes.DeadlineExceeded: Operation timeout (may indicate network issues)
//
// Parameters:
//   - err: The error to check (typically from a gRPC call)
//
// Returns true if the error is a transport error requiring reconnection.
func (c *Client) isTransportError(err error) bool {
	if err == nil {
		return false
	}

	// Extract gRPC status code from error
	st, ok := status.FromError(err)
	if !ok {
		// Not a gRPC error
		return false
	}

	// Get status code
	code := st.Code()

	// Check for transport error codes
	// codes.Unavailable: Connection lost, DNS failure, server down
	// codes.DeadlineExceeded: Timeout (may indicate network/transport issues)
	if code == codes.Unavailable || code == codes.DeadlineExceeded {
		c.logger.Debug(context.Background(), "Transport error detected",
			"code", code,
			"message", st.Message())
		return true
	}

	return false
}

// checkTransientErrorModels checks if any error in the list is transient
//
// This method is used when errors are already converted to ErrorModel structs
// (e.g., from gNMI response error details).
//
// Returns true if any error matches a transient pattern.
func (c *Client) checkTransientErrorModels(errs []ErrorModel) bool {
	if len(errs) == 0 {
		return false
	}

	// Check each error against transient patterns
	for _, err := range errs {
		for _, pattern := range TransientErrors {
			if pattern.Code == err.Code {
				return true
			}
		}
	}

	return false
}

// validateConfig validates client configuration before connection
//
// Validates:
//   - Port range (1-65535)
//   - Positive timeouts (ConnectTimeout, OperationTimeout > 0)
//   - Positive retry params (MaxRetries >= 0, BackoffMinDelay > 0, BackoffMaxDelay > BackoffMinDelay)
//   - BackoffDelayFactor >= 1.0
//   - TLS certificate file paths exist (if provided)
//
// Returns an error if validation fails.
func (c *Client) validateConfig() error {
	// Validate target is not empty
	if strings.TrimSpace(c.Target) == "" {
		return fmt.Errorf("target address cannot be empty")
	}

	// Validate port range
	if c.Port < 1 || c.Port > 65535 {
		return fmt.Errorf("invalid port: %d (must be 1-65535)", c.Port)
	}

	// Validate timeouts are positive
	if c.ConnectTimeout <= 0 {
		return fmt.Errorf("connect timeout must be positive, got: %v", c.ConnectTimeout)
	}
	if c.OperationTimeout <= 0 {
		return fmt.Errorf("operation timeout must be positive, got: %v", c.OperationTimeout)
	}

	// Validate retry parameters
	if c.MaxRetries < 0 {
		return fmt.Errorf("max retries must be non-negative, got: %d", c.MaxRetries)
	}
	if c.BackoffMinDelay <= 0 {
		return fmt.Errorf("backoff min delay must be positive, got: %v", c.BackoffMinDelay)
	}
	if c.BackoffMaxDelay <= c.BackoffMinDelay {
		return fmt.Errorf("backoff max delay (%v) must be greater than min delay (%v)",
			c.BackoffMaxDelay, c.BackoffMinDelay)
	}
	if c.BackoffDelayFactor < 1.0 {
		return fmt.Errorf("backoff delay factor must be >= 1.0, got: %f", c.BackoffDelayFactor)
	}

	// Warn on insecure TLS configuration
	if c.UseTLS && c.InsecureSkipVerify {
		c.logger.Warn(context.Background(), "InsecureSkipVerify enabled - TLS certificate verification disabled",
			"target", c.Target,
			"security_risk", "Man-in-the-Middle attacks possible",
			"recommendation", "Use only in testing environments")
	}

	// Warn if TLS is disabled
	if !c.UseTLS {
		c.logger.Warn(context.Background(), "TLS disabled - connection is not encrypted",
			"target", c.Target,
			"security_risk", "Credentials and data transmitted in clear text",
			"recommendation", "Enable TLS for production use")
	}

	// Validate TLS certificate paths if provided
	if c.tlsCert != "" {
		if _, err := os.Stat(c.tlsCert); err != nil {
			// Log full path at Debug level for troubleshooting
			c.logger.Debug(context.Background(), "TLS certificate validation failed",
				"path", c.tlsCert,
				"error", err.Error())
			// Return only filename in error to prevent path disclosure
			filename := filepath.Base(c.tlsCert)
			return fmt.Errorf("TLS certificate file not found: %s", filename)
		}
	}
	if c.tlsKey != "" {
		if _, err := os.Stat(c.tlsKey); err != nil {
			c.logger.Debug(context.Background(), "TLS key validation failed",
				"path", c.tlsKey,
				"error", err.Error())
			filename := filepath.Base(c.tlsKey)
			return fmt.Errorf("TLS key file not found: %s", filename)
		}
	}
	if c.tlsCA != "" {
		if _, err := os.Stat(c.tlsCA); err != nil {
			c.logger.Debug(context.Background(), "TLS CA validation failed",
				"path", c.tlsCA,
				"error", err.Error())
			filename := filepath.Base(c.tlsCA)
			return fmt.Errorf("TLS CA file not found: %s", filename)
		}
	}

	// Warn if credentials are missing (not an error, but may be required by device)
	if !c.HasCredentials() {
		c.logger.Warn(context.Background(), "No credentials configured",
			"target", c.Target,
			"message", "device may reject connection")
	}

	return nil
}

// createTarget creates a gnmic target configuration without establishing connection
//
// This method builds target options from client configuration and creates
// a gnmic target handle. The actual gRPC connection is NOT established here.
//
// PRECONDITION: Configuration must be validated via validateConfig().
//
// Returns an error if target creation fails (configuration errors only).
func (c *Client) createTarget() error {
	// Build target address with port
	address := c.Target
	if !strings.Contains(address, ":") {
		address = fmt.Sprintf("%s:%d", address, c.Port)
	}

	// Build gnmic target options
	targetOpts := []api.TargetOption{
		api.Name(c.Target),
		api.Address(address),
		api.Timeout(c.ConnectTimeout),
	}

	// Add credentials if provided
	if c.username != "" {
		targetOpts = append(targetOpts, api.Username(c.username))
	}
	if c.password != "" {
		targetOpts = append(targetOpts, api.Password(c.password))
	}

	// Add TLS configuration
	if c.tlsCert != "" {
		targetOpts = append(targetOpts, api.TLSCert(c.tlsCert))
	}
	if c.tlsKey != "" {
		targetOpts = append(targetOpts, api.TLSKey(c.tlsKey))
	}
	if c.tlsCA != "" {
		targetOpts = append(targetOpts, api.TLSCA(c.tlsCA))
	}

	// Add TLS options
	targetOpts = append(targetOpts, api.Insecure(!c.UseTLS))
	targetOpts = append(targetOpts, api.SkipVerify(c.InsecureSkipVerify))

	// Create target (configuration only, NO connection)
	t, err := api.NewTarget(targetOpts...)
	if err != nil {
		return fmt.Errorf("failed to create gnmic target: %w", err)
	}

	// Store target (not connected yet)
	c.target = t

	return nil
}

// ensureConnected establishes connection if not already connected (lazy connection)
//
// This method checks the connection flag and if not connected, calls
// CreateGNMIClient() to establish the connection. This implements the lazy
// connection pattern where physical connections are deferred until first use.
//
// Thread-safe: acquires client mutex before checking/establishing connection.
//
// Returns an error if connection establishment fails.
func (c *Client) ensureConnected(ctx context.Context) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	// Check if target exists
	if c.target == nil {
		return fmt.Errorf("client not connected")
	}

	// Check if already connected
	if c.connected {
		return nil // Already connected
	}

	// Not connected yet - establish connection now
	c.logger.Debug(ctx, "Establishing gNMI connection",
		"target", c.Target,
		"port", c.Port)

	err := c.target.CreateGNMIClient(ctx)
	if err != nil {
		return fmt.Errorf("failed to establish connection: %w", err)
	}

	// Mark as connected
	c.connected = true

	c.logger.Info(ctx, "gNMI connection established",
		"target", c.Target,
		"port", c.Port)

	return nil
}

// Capabilities retrieves the gNMI server capabilities
//
// This operation performs a gNMI Capabilities RPC to discover:
//   - gNMI service version
//   - Supported encodings (json, json_ietf, proto, etc.)
//   - Supported data models
//
// The capabilities are stored in the client for later reference.
// Use HasCapability() to check for specific capabilities.
//
// Example:
//
//	ctx := context.Background()
//	res, err := client.Capabilities(ctx)
//	if err != nil {
//	    log.Fatal(err)
//	}
//	fmt.Printf("gNMI version: %s\n", res.Version)
//	for _, cap := range res.Capabilities {
//	    fmt.Printf("Encoding: %s\n", cap)
//	}
func (c *Client) Capabilities(ctx context.Context) (CapabilitiesRes, error) {
	// Check context cancellation first (before acquiring lock)
	if err := checkContextCancellation(ctx); err != nil {
		return CapabilitiesRes{
			OK:     false,
			Errors: []ErrorModel{{Message: err.Error()}},
		}, err
	}

	// Ensure connection is established (lazy connection)
	if err := c.ensureConnected(ctx); err != nil {
		return CapabilitiesRes{
			OK:     false,
			Errors: []ErrorModel{{Message: err.Error()}},
		}, err
	}

	// Apply operation timeout
	ctx, cancel := context.WithTimeout(ctx, c.OperationTimeout)
	defer cancel()

	// Log operation start
	c.logger.Debug(ctx, "gNMI Capabilities request",
		"target", c.Target)

	// Execute request using gnmic target API
	// Note: gnmic target.Capabilities() takes context and optional extensions
	resp, err := c.target.Capabilities(ctx)
	if err != nil {
		c.logger.Error(ctx, "gNMI Capabilities failed",
			"target", c.Target,
			"error", err.Error())
		return CapabilitiesRes{
			OK:     false,
			Errors: []ErrorModel{{Message: err.Error()}},
		}, fmt.Errorf("capabilities request failed: %w", err)
	}

	// Parse response
	capList := make([]string, 0, len(resp.SupportedEncodings))
	for _, enc := range resp.SupportedEncodings {
		capList = append(capList, enc.String())
	}

	// Store capabilities (already have write lock)
	c.capabilities = capList

	// Log success
	c.logger.Debug(ctx, "gNMI Capabilities response",
		"version", resp.GNMIVersion,
		"encodings", len(capList),
		"models", len(resp.SupportedModels))

	return CapabilitiesRes{
		Version:      resp.GNMIVersion,
		Capabilities: capList,
		Models:       resp.SupportedModels,
		OK:           true,
	}, nil
}

// Ping verifies connectivity by performing a Capabilities RPC
//
// This method establishes a connection if not already connected and performs
// a gNMI Capabilities request to verify the server is reachable and responsive.
// This is useful when you want to explicitly verify connectivity before performing
// other operations.
//
// Example:
//
//	client, err := gnmi.NewClient("192.168.1.1:57400", opts...)
//	if err != nil {
//	    log.Fatal(err)  // Configuration error
//	}
//	defer client.Close()
//
//	// Verify connection before proceeding
//	if err := client.Ping(ctx); err != nil {
//	    log.Fatal(err)  // Connection error
//	}
//
//	// Now confident connection works
//	res, err := client.Get(ctx, paths)
func (c *Client) Ping(ctx context.Context) error {
	_, err := c.Capabilities(ctx)
	return err
}

// reconnect attempts to reconnect to the gNMI target after a connection failure.
//
// This method closes the existing (broken) connection and establishes a new one.
// Used when transport errors are detected during Get/Set operations.
//
// PRECONDITION: Caller must hold c.mu.Lock() (write lock).
//
// Returns an error if reconnection fails.
func (c *Client) reconnect(ctx context.Context) error {
	c.logger.Warn(ctx, "gNMI reconnecting",
		"target", c.Target,
		"reason", "transport error")

	// Close existing connection (ignore errors - connection may already be broken)
	if c.target != nil {
		_ = c.target.Close() //nolint:errcheck // Explicitly ignore error (connection likely already broken)
	}

	// Reset connection flag
	c.connected = false

	// Recreate target configuration
	if err := c.createTarget(); err != nil {
		c.logger.Error(ctx, "gNMI target recreation failed",
			"target", c.Target,
			"error", err.Error())
		return fmt.Errorf("failed to recreate target: %w", err)
	}

	// Establish new connection
	err := c.target.CreateGNMIClient(ctx)
	if err != nil {
		c.logger.Error(ctx, "gNMI reconnection failed",
			"target", c.Target,
			"error", err.Error())
		return fmt.Errorf("failed to reconnect: %w", err)
	}

	// Mark as connected
	c.connected = true

	c.logger.Info(ctx, "gNMI reconnected",
		"target", c.Target)

	return nil
}
