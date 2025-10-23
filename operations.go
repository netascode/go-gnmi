// SPDX-License-Identifier: MPL-2.0
// Copyright (c) 2025 Daniel Schmidt

package gnmi

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	gnmipb "github.com/openconfig/gnmi/proto/gnmi"
	"github.com/openconfig/gnmic/pkg/api"
	"google.golang.org/grpc/status"
)

// Input validation constants
const (
	// MaxValueSize is the maximum size for a single value in bytes (10MB)
	MaxValueSize = 10 * 1024 * 1024

	// MaxPathLength is the maximum length for a gNMI path (1024 characters)
	MaxPathLength = 1024
)

// Input validation functions

// validatePaths validates a slice of gNMI paths
//
// Checks:
//   - Paths slice is not empty
//   - Each path starts with "/"
//   - Each path length does not exceed MaxPathLength
//   - Each path does not contain malicious patterns (null bytes, path traversal)
//
// Returns an error if any path is invalid with a descriptive message.
func validatePaths(paths []string) error {
	if len(paths) == 0 {
		return fmt.Errorf("paths cannot be empty")
	}

	for i, path := range paths {
		if path == "" {
			return fmt.Errorf("path cannot be empty (at index %d)", i)
		}

		if len(path) > MaxPathLength {
			return fmt.Errorf("path at index %d exceeds maximum length of %d characters: %s", i, MaxPathLength, truncatePath(path))
		}

		// gNMI paths can be in two formats:
		// 1. Absolute: /interfaces/interface[name=eth0]
		// 2. Module-qualified: openconfig-interfaces:/interfaces/interface[name=eth0]
		//    or: Cisco-IOS-XR-um-banner-cfg:/banners/banner[banner-type=login]
		// Check if path is valid (starts with / or is module-qualified)
		if !isValidGNMIPath(path) {
			return fmt.Errorf("path at index %d must start with '/' or be module-qualified (module:path): %s", i, path)
		}

		// Check for malicious patterns
		if err := checkPathSecurity(path); err != nil {
			return fmt.Errorf("path at index %d is invalid: %w", i, err)
		}
	}

	return nil
}

// validateEncoding validates a gNMI encoding string
//
// Valid encodings: json, json_ietf, proto, ascii, bytes
// Empty string is valid (will default to json_ietf)
//
// Returns an error if the encoding is not supported.
func validateEncoding(encoding string) error {
	if encoding == "" {
		// Empty encoding is valid (will default to json_ietf)
		return nil
	}
	return ValidateEncoding(encoding)
}

// validateValue validates a value string for gNMI operations
//
// Checks:
//   - Value size does not exceed MaxValueSize (10MB)
//   - For json/json_ietf encodings: basic JSON syntax validation
//
// Returns an error if the value is invalid with a descriptive message.
func validateValue(value string, encoding string) error {
	valueSize := len(value)
	if valueSize > MaxValueSize {
		return fmt.Errorf("value size exceeds maximum of %d bytes (got %d bytes)", MaxValueSize, valueSize)
	}

	// Validate JSON syntax for json/json_ietf encodings
	if encoding == EncodingJSON || encoding == EncodingJSONIETF || encoding == "" {
		if err := validateJSONSyntax(value); err != nil {
			return fmt.Errorf("invalid JSON syntax: %w", err)
		}
	}

	return nil
}

// validateSetOperations validates a slice of SetOperation structs
//
// Checks:
//   - Operations slice is not empty
//   - Each operation has a valid path
//   - Each operation has a valid encoding
//   - Each operation has a valid value (for Update/Replace operations)
//   - Each operation has a valid operation type
//
// Returns an error if any operation is invalid with a descriptive message.
func validateSetOperations(ops []SetOperation) error {
	if len(ops) == 0 {
		return fmt.Errorf("operations cannot be empty")
	}

	for i, op := range ops {
		// Validate operation type
		if op.OperationType == "" {
			return fmt.Errorf("operation type cannot be empty (at index %d)", i)
		}
		if op.OperationType != OperationUpdate &&
			op.OperationType != OperationReplace &&
			op.OperationType != OperationDelete {
			return fmt.Errorf("operation type invalid: %s (must be 'update', 'replace', or 'delete', at index %d)", op.OperationType, i)
		}

		// Validate path
		if err := validatePaths([]string{op.Path}); err != nil {
			return fmt.Errorf("operation at index %d: %w", i, err)
		}

		// Validate encoding
		encoding := op.Encoding
		if encoding == "" {
			encoding = EncodingJSONIETF
		}
		if err := validateEncoding(encoding); err != nil {
			return fmt.Errorf("operation at index %d: %w", i, err)
		}

		// Validate value for Update/Replace operations
		if op.OperationType == OperationUpdate || op.OperationType == OperationReplace {
			if err := validateValue(op.Value, encoding); err != nil {
				return fmt.Errorf("operation at index %d: %w", i, err)
			}
		}
	}

	return nil
}

// Security validation helpers

// checkPathSecurity checks a path for malicious patterns
//
// Checks for:
//   - Null bytes (path injection)
//   - Path traversal patterns (..)
//
// Returns an error if malicious patterns are detected.
func checkPathSecurity(path string) error {
	// Check for null bytes
	for i := 0; i < len(path); i++ {
		if path[i] == 0 {
			return fmt.Errorf("path contains null byte at position %d", i)
		}
	}

	// Check for path traversal patterns
	// Note: ".." is valid in gNMI paths (e.g., "../config"), but we check for
	// suspicious patterns like "/../" which could indicate path traversal attacks
	if len(path) >= 4 {
		for i := 0; i < len(path)-3; i++ {
			if path[i] == '/' && path[i+1] == '.' && path[i+2] == '.' && path[i+3] == '/' {
				return fmt.Errorf("path contains suspicious traversal pattern '/../' at position %d", i)
			}
		}
	}

	return nil
}

// validateJSONSyntax performs basic JSON syntax validation
//
// This is a lightweight check to catch obvious JSON errors without
// fully parsing the JSON (which would be expensive for large values).
//
// Checks:
//   - Empty values are allowed (for some operations)
//   - JSON must start with '{', '[', '"', or be a JSON literal (true/false/null/number)
//   - Balanced braces/brackets (basic check)
//
// Returns an error if JSON syntax appears invalid.
//
//nolint:gocyclo // Validation logic naturally has high complexity
func validateJSONSyntax(value string) error {
	if value == "" {
		// Empty value is valid for some operations
		return nil
	}

	// Trim whitespace
	trimmed := value
	for len(trimmed) > 0 && (trimmed[0] == ' ' || trimmed[0] == '\t' || trimmed[0] == '\n' || trimmed[0] == '\r') {
		trimmed = trimmed[1:]
	}
	for len(trimmed) > 0 && (trimmed[len(trimmed)-1] == ' ' || trimmed[len(trimmed)-1] == '\t' || trimmed[len(trimmed)-1] == '\n' || trimmed[len(trimmed)-1] == '\r') {
		trimmed = trimmed[:len(trimmed)-1]
	}

	if len(trimmed) == 0 {
		return nil
	}

	// Check first character for valid JSON start
	firstChar := trimmed[0]
	if firstChar != '{' && firstChar != '[' && firstChar != '"' &&
		firstChar != 't' && firstChar != 'f' && firstChar != 'n' &&
		(firstChar < '0' || firstChar > '9') && firstChar != '-' {
		return fmt.Errorf("JSON must start with '{', '[', '\"', or a JSON literal")
	}

	// Basic brace/bracket balancing check
	braceCount := 0
	bracketCount := 0
	inString := false
	escaped := false

	for i := 0; i < len(trimmed); i++ {
		ch := trimmed[i]

		if escaped {
			escaped = false
			continue
		}

		if ch == '\\' {
			escaped = true
			continue
		}

		if ch == '"' {
			inString = !inString
			continue
		}

		if !inString {
			switch ch {
			case '{':
				braceCount++
			case '}':
				braceCount--
				if braceCount < 0 {
					return fmt.Errorf("unbalanced braces: too many '}'")
				}
			case '[':
				bracketCount++
			case ']':
				bracketCount--
				if bracketCount < 0 {
					return fmt.Errorf("unbalanced brackets: too many ']'")
				}
			}
		}
	}

	if braceCount != 0 {
		return fmt.Errorf("unbalanced braces: %d unclosed '{'", braceCount)
	}

	if bracketCount != 0 {
		return fmt.Errorf("unbalanced brackets: %d unclosed '['", bracketCount)
	}

	if inString {
		return fmt.Errorf("unterminated string")
	}

	return nil
}

// truncatePath truncates a path for error messages
//
// Returns the first 100 characters of the path followed by "..." if longer.
func truncatePath(path string) string {
	if len(path) <= 100 {
		return path
	}
	return path[:100] + "..."
}

// Get performs a gNMI Get operation to retrieve data from the device
//
// Get supports querying multiple paths in a single request. The paths parameter
// must be a non-empty slice of gNMI path strings. The encoding can be specified
// via request modifiers, defaulting to json_ietf.
//
// The operation uses RLock for concurrent read access, allowing multiple Get
// operations to run in parallel. Context timeout follows priority:
//  1. Request-specific timeout (via Timeout modifier)
//  2. Context deadline (if already set)
//  3. Client.OperationTimeout (fallback default)
//
// Example:
//
//	ctx := context.Background()
//	paths := []string{
//	    "/interfaces/interface[name=GigabitEthernet0/0/0/0]/state",
//	    "/system/config/hostname",
//	}
//	res, err := client.Get(ctx, paths)
//	if err != nil {
//	    log.Fatal(err)
//	}
//	for _, notif := range res.Notifications {
//	    fmt.Printf("Path: %s\n", notif.Prefix)
//	}
//
// Returns GetRes with notifications, timestamp, OK status, and any errors.
func (c *Client) Get(ctx context.Context, paths []string, mods ...func(*Req)) (GetRes, error) {
	// Validate paths (before acquiring lock)
	if err := validatePaths(paths); err != nil {
		return GetRes{
			OK:     false,
			Errors: []ErrorModel{{Message: err.Error()}},
		}, fmt.Errorf("get: %w", err)
	}

	// Build request with default encoding
	req := &Req{
		Encoding: EncodingJSONIETF,
	}

	// Apply modifiers
	for _, mod := range mods {
		mod(req)
	}

	// Validate encoding (before acquiring lock)
	if err := validateEncoding(req.Encoding); err != nil {
		return GetRes{
			OK:     false,
			Errors: []ErrorModel{{Message: err.Error()}},
		}, fmt.Errorf("get: %w", err)
	}

	// Check context cancellation first (before acquiring lock)
	if err := checkContextCancellation(ctx); err != nil {
		return GetRes{
			OK:     false,
			Errors: []ErrorModel{{Message: err.Error()}},
		}, err
	}

	// Ensure connection is established (lazy connection)
	if err := c.ensureConnected(ctx); err != nil {
		return GetRes{
			OK:     false,
			Errors: []ErrorModel{{Message: err.Error()}},
		}, fmt.Errorf("get: connection failed: %w", err)
	}

	// Acquire lock after validation
	c.mu.RLock()
	defer c.mu.RUnlock()

	// Check target exists
	if c.target == nil {
		return GetRes{
			OK:     false,
			Errors: []ErrorModel{{Message: "client not connected"}},
		}, fmt.Errorf("get: client not connected")
	}

	// Calculate total timeout budget to prevent unbounded accumulation
	// Total timeout = OperationTimeout + sum of actual backoff delays
	// This accurately reflects the maximum time needed for all retry attempts
	totalTimeout := c.calculateTotalTimeout()

	c.logger.Debug(ctx, "applying total timeout budget",
		"totalTimeout", totalTimeout.String(),
		"operationTimeout", c.OperationTimeout.String(),
		"maxRetries", c.MaxRetries,
		"target", c.Target)

	// Apply parent context timeout for total budget
	ctx, parentCancel := context.WithTimeout(ctx, totalTimeout)
	defer parentCancel()

	// NOTE: Full integration tests with mock gnmic target are documented in operations_test.go.
	// Current tests focus on input validation, error handling, and API structure correctness.

	// Build gnmic GetRequest
	gnmicOpts := []api.GNMIOption{
		api.Encoding(req.Encoding),
	}
	for _, path := range paths {
		gnmicOpts = append(gnmicOpts, api.Path(path))
	}

	getReq, err := api.NewGetRequest(gnmicOpts...)
	if err != nil {
		c.logger.Error(ctx, "gNMI Get request creation failed",
			"target", c.Target,
			"error", err.Error())
		return GetRes{
			OK:     false,
			Errors: []ErrorModel{{Message: err.Error()}},
		}, fmt.Errorf("get: failed to create request: %w", err)
	}

	// Log request
	c.logger.Debug(ctx, "gNMI Get request",
		"target", c.Target,
		"paths", len(paths),
		"encoding", req.Encoding)

	// Log each path (at Debug level)
	for i, path := range paths {
		c.logger.Debug(ctx, "gNMI Get path",
			"index", i,
			"path", path)
	}

	// Execute request with retry logic
	var getResp *gnmipb.GetResponse
	var lastErr error

	//nolint:dupl // Get and Set retry logic are similar but have different error handling
	for attempt := 0; attempt <= c.MaxRetries; attempt++ {
		// Check parent context cancellation before attempt
		if err := checkContextCancellation(ctx); err != nil {
			c.logger.Debug(ctx, "get operation canceled",
				"operation", "get",
				"attempt", attempt,
				"error", err.Error())
			return GetRes{
				OK:     false,
				Errors: []ErrorModel{{Message: fmt.Sprintf("context canceled: %s", err.Error())}},
			}, fmt.Errorf("get: %w", err)
		}

		// Create attempt-specific context with timeout
		attemptCtx, attemptCancel := c.createAttemptContext(ctx, req)

		// Execute Get request with attempt context
		resp, err := c.target.Get(attemptCtx, getReq)

		// Clean up attempt context immediately to prevent goroutine leak
		attemptCancel()
		if err == nil {
			// Success
			getResp = resp
			lastErr = nil
			break
		}

		// Store error
		lastErr = err

		// Extract error details for transient checking
		errors := c.extractErrorDetails(err)

		// Check if error is transient and retries remain
		if c.checkTransientErrorModels(errors) && attempt < c.MaxRetries {
			// Check for transport errors requiring reconnection
			if c.isTransportError(lastErr) {
				// Upgrade to write lock for reconnection
				// First, release read lock
				c.mu.RUnlock()
				c.mu.Lock()

				// Attempt to reconnect
				if reconnectErr := c.reconnect(ctx); reconnectErr != nil {
					// Reconnection failed, downgrade to read lock and return error
					c.mu.Unlock()
					c.mu.RLock()
					c.logger.Error(ctx, "gNMI reconnection failed",
						"operation", "get",
						"error", reconnectErr.Error())
					return GetRes{
						OK:     false,
						Errors: []ErrorModel{{Message: fmt.Sprintf("operation failed and reconnection failed: %s", reconnectErr.Error())}},
					}, fmt.Errorf("get: reconnection failed: %w", reconnectErr)
				}

				// Reconnection succeeded, downgrade to read lock and continue retry
				c.mu.Unlock()
				c.mu.RLock()
			}

			backoff := c.Backoff(attempt)
			c.logger.Warn(ctx, "transient error, retrying",
				"operation", "get",
				"attempt", attempt+1,
				"max_retries", c.MaxRetries,
				"backoff", backoff,
				"error", err.Error())

			// Sleep with context cancellation awareness (uses ctx)
			select {
			case <-time.After(backoff):
				// Backoff complete, continue to next attempt
				continue
			case <-ctx.Done():
				// Context canceled during backoff
				c.logger.Debug(ctx, "get operation canceled during backoff",
					"operation", "get",
					"attempt", attempt+1)
				return GetRes{
					OK:     false,
					Errors: []ErrorModel{{Message: fmt.Sprintf("context canceled during backoff: %s", ctx.Err().Error())}},
				}, fmt.Errorf("get: context canceled during backoff: %w", ctx.Err())
			}
		} else {
			// Non-transient error or no retries remaining
			break
		}
	}

	// Check if all retries failed
	if lastErr != nil {
		c.logger.Error(ctx, "gNMI Get failed",
			"target", c.Target,
			"error", lastErr.Error())

		// Extract gRPC error details
		errors := c.extractErrorDetails(lastErr)
		return GetRes{
			OK:     false,
			Errors: errors,
		}, fmt.Errorf("get: request failed: %w", lastErr)
	}

	// Log response
	c.logger.Debug(ctx, "gNMI Get response",
		"target", c.Target,
		"notifications", len(getResp.Notification))

	// Log each notification with redacted JSON values (at Debug level)
	for i, notif := range getResp.Notification {
		// Convert notification to JSON and redact sensitive data
		if notifJSON, err := json.Marshal(notif); err == nil {
			sanitizedNotif := c.prepareJSONForLogging(string(notifJSON))
			c.logger.Debug(ctx, "gNMI Get notification",
				"index", i,
				"timestamp", notif.Timestamp,
				"updates", len(notif.Update),
				"deletes", len(notif.Delete),
				"notification", sanitizedNotif)
		}
	}

	// Parse response
	timestamp := time.Now().UnixNano()
	return GetRes{
		Notifications: getResp.Notification,
		Timestamp:     timestamp,
		OK:            true,
	}, nil
}

// Set performs a gNMI Set operation to configure the device
//
// Set supports multiple update, replace, and delete operations in a single request.
// The ops parameter must be a non-empty slice of SetOperation structs created via
// the Update(), Replace(), or Delete() helper functions.
//
// The operation uses Lock for exclusive write access, serializing all Set operations
// to prevent concurrent writes. Context timeout follows priority:
//  1. Request-specific timeout (via Timeout modifier)
//  2. Context deadline (if already set)
//  3. Client.OperationTimeout (fallback default)
//
// Example:
//
//	ctx := context.Background()
//	ops := []gnmi.SetOperation{
//	    gnmi.Update("/interfaces/interface[name=Gi0/0/0/0]/config/description",
//	        `{"description": "WAN Interface"}`),
//	    gnmi.Replace("/interfaces/interface[name=Gi0/0/0/0]/config/enabled",
//	        `{"enabled": true}`),
//	    gnmi.Delete("/interfaces/interface[name=Gi0/0/0/1]/config"),
//	}
//	res, err := client.Set(ctx, ops)
//	if err != nil {
//	    log.Fatal(err)
//	}
//	fmt.Printf("Set operation successful: %v\n", res.OK)
//
// Returns SetRes with response, timestamp, OK status, and any errors.
func (c *Client) Set(ctx context.Context, ops []SetOperation, mods ...func(*Req)) (SetRes, error) {
	// Validate operations (before acquiring lock for better performance)
	if err := validateSetOperations(ops); err != nil {
		return SetRes{
			OK:     false,
			Errors: []ErrorModel{{Message: err.Error()}},
		}, fmt.Errorf("set: %w", err)
	}

	// Check context cancellation first (before acquiring lock)
	if err := checkContextCancellation(ctx); err != nil {
		return SetRes{
			OK:     false,
			Errors: []ErrorModel{{Message: err.Error()}},
		}, err
	}

	// Ensure connection is established (lazy connection)
	if err := c.ensureConnected(ctx); err != nil {
		return SetRes{
			OK:     false,
			Errors: []ErrorModel{{Message: err.Error()}},
		}, fmt.Errorf("set: connection failed: %w", err)
	}

	// Acquire lock after validation and connection
	c.mu.Lock()
	defer c.mu.Unlock()

	// Build request for modifiers
	req := &Req{}

	// Apply modifiers
	for _, mod := range mods {
		mod(req)
	}

	// Calculate total timeout budget to prevent unbounded accumulation
	// Total timeout = OperationTimeout + sum of actual backoff delays
	// This accurately reflects the maximum time needed for all retry attempts
	totalTimeout := c.calculateTotalTimeout()

	c.logger.Debug(ctx, "applying total timeout budget",
		"totalTimeout", totalTimeout.String(),
		"operationTimeout", c.OperationTimeout.String(),
		"maxRetries", c.MaxRetries,
		"target", c.Target)

	// Apply parent context timeout for total budget
	ctx, parentCancel := context.WithTimeout(ctx, totalTimeout)
	defer parentCancel()

	// NOTE: Full integration tests with mock gnmic target are documented in operations_test.go.
	// Current tests focus on input validation, operation helpers, and error handling correctness.

	// Build gnmic SetRequest based on operation types
	gnmicOpts := []api.GNMIOption{}

	for _, op := range ops {
		encoding := op.Encoding
		if encoding == "" {
			encoding = EncodingJSONIETF
		}

		switch op.OperationType {
		case OperationUpdate:
			gnmicOpts = append(gnmicOpts, api.Update(api.Path(op.Path), api.Value(op.Value, encoding)))
		case OperationReplace:
			gnmicOpts = append(gnmicOpts, api.Replace(api.Path(op.Path), api.Value(op.Value, encoding)))
		case OperationDelete:
			gnmicOpts = append(gnmicOpts, api.Delete(op.Path))
		default:
			// Invalid operation type
			return SetRes{
				OK: false,
				Errors: []ErrorModel{{
					Message: fmt.Sprintf("invalid operation type: %s", op.OperationType),
				}},
			}, fmt.Errorf("set: invalid operation type: %s", op.OperationType)
		}
	}

	setReq, err := api.NewSetRequest(gnmicOpts...)
	if err != nil {
		c.logger.Error(ctx, "gNMI Set request creation failed",
			"target", c.Target,
			"error", err.Error())
		return SetRes{
			OK:     false,
			Errors: []ErrorModel{{Message: err.Error()}},
		}, fmt.Errorf("set: failed to create request: %w", err)
	}

	// Log request with sanitized operation details
	c.logger.Debug(ctx, "gNMI Set request",
		"target", c.Target,
		"operations", len(ops))

	// Log each operation with redacted JSON values (at Debug level)
	for i, op := range ops {
		// Prepare JSON for logging (redacts sensitive data)
		sanitizedValue := c.prepareJSONForLogging(op.Value)

		c.logger.Debug(ctx, "gNMI Set operation",
			"index", i,
			"type", op.OperationType,
			"path", op.Path,
			"encoding", op.Encoding,
			"value", sanitizedValue)
	}

	// Execute request with retry logic
	var setResp *gnmipb.SetResponse
	var lastErr error

	//nolint:dupl // Get and Set retry logic are similar but have different error handling
	for attempt := 0; attempt <= c.MaxRetries; attempt++ {
		// Check parent context cancellation before attempt
		if err := checkContextCancellation(ctx); err != nil {
			c.logger.Debug(ctx, "set operation canceled",
				"operation", "set",
				"attempt", attempt,
				"error", err.Error())
			return SetRes{
				OK:     false,
				Errors: []ErrorModel{{Message: fmt.Sprintf("context canceled: %s", err.Error())}},
			}, fmt.Errorf("set: %w", err)
		}

		// Create attempt-specific context with timeout
		attemptCtx, attemptCancel := c.createAttemptContext(ctx, req)

		// Execute Set request with attempt context
		resp, err := c.target.Set(attemptCtx, setReq)

		// Clean up attempt context immediately to prevent goroutine leak
		attemptCancel()
		if err == nil {
			// Success
			setResp = resp
			lastErr = nil
			break
		}

		// Store error
		lastErr = err

		// Extract error details for transient checking
		errors := c.extractErrorDetails(err)

		// Check if error is transient and retries remain
		if c.checkTransientErrorModels(errors) && attempt < c.MaxRetries {
			// Check for transport errors requiring reconnection
			// Note: Set operation already holds write lock (c.mu.Lock), so no lock upgrade needed
			if c.isTransportError(lastErr) {
				// Attempt to reconnect (already holding write lock)
				if reconnectErr := c.reconnect(ctx); reconnectErr != nil {
					// Reconnection failed, return error
					c.logger.Error(ctx, "gNMI reconnection failed",
						"operation", "set",
						"error", reconnectErr.Error())
					return SetRes{
						OK:     false,
						Errors: []ErrorModel{{Message: fmt.Sprintf("operation failed and reconnection failed: %s", reconnectErr.Error())}},
					}, fmt.Errorf("set: reconnection failed: %w", reconnectErr)
				}
				// Reconnection succeeded, continue to retry
			}

			backoff := c.Backoff(attempt)
			c.logger.Warn(ctx, "transient error, retrying",
				"operation", "set",
				"attempt", attempt+1,
				"max_retries", c.MaxRetries,
				"backoff", backoff,
				"error", err.Error())

			// Sleep with context cancellation awareness (uses ctx)
			select {
			case <-time.After(backoff):
				// Backoff complete, continue to next attempt
				continue
			case <-ctx.Done():
				// Context canceled during backoff
				c.logger.Debug(ctx, "set operation canceled during backoff",
					"operation", "set",
					"attempt", attempt+1)
				return SetRes{
					OK:     false,
					Errors: []ErrorModel{{Message: fmt.Sprintf("context canceled during backoff: %s", ctx.Err().Error())}},
				}, fmt.Errorf("set: context canceled during backoff: %w", ctx.Err())
			}
		} else {
			// Non-transient error or no retries remaining
			break
		}
	}

	// Check if all retries failed
	if lastErr != nil {
		c.logger.Error(ctx, "gNMI Set failed",
			"target", c.Target,
			"error", lastErr.Error())

		// Extract gRPC error details
		errors := c.extractErrorDetails(lastErr)
		return SetRes{
			OK:     false,
			Errors: errors,
		}, fmt.Errorf("set: request failed: %w", lastErr)
	}

	// Log response
	c.logger.Debug(ctx, "gNMI Set response",
		"target", c.Target,
		"results", len(setResp.Response))

	// Parse response
	timestamp := time.Now().UnixNano()
	return SetRes{
		Response:  setResp,
		Timestamp: timestamp,
		OK:        true,
	}, nil
}

// Update creates a SetOperation for updating a path with a value
//
// Update operations modify existing configuration, creating it if it doesn't exist.
// This is the most common Set operation type.
//
// The encoding defaults to json_ietf. Use the SetEncoding() modifier to specify
// a different encoding (json, proto, ascii, bytes).
//
// Parameters:
//   - path: gNMI path string (e.g., "/interfaces/interface[name=Gi0/0/0/0]/config")
//   - value: JSON-encoded value string
//   - opts: optional modifiers (SetEncoding, etc.)
//
// Example:
//
//	// Default encoding (json_ietf)
//	op := gnmi.Update("/system/config/hostname", `{"hostname": "router1"}`)
//
//	// Explicit encoding
//	op := gnmi.Update("/interfaces/interface[name=Gi0]/config", protoBytes,
//	    gnmi.SetEncoding("proto"))
func Update(path, value string, opts ...func(*SetOperation)) SetOperation {
	op := SetOperation{
		OperationType: OperationUpdate,
		Path:          path,
		Value:         value,
		Encoding:      EncodingJSONIETF, // default
	}

	// Apply functional options
	for _, opt := range opts {
		opt(&op)
	}

	return op
}

// Replace creates a SetOperation for replacing a path with a value
//
// Replace operations remove existing configuration at the path before applying
// the new value. Use Replace when you need to ensure no old config remains.
//
// The encoding defaults to json_ietf. Use the SetEncoding() modifier to specify
// a different encoding (json, proto, ascii, bytes).
//
// Parameters:
//   - path: gNMI path string
//   - value: JSON-encoded value string
//   - opts: optional modifiers (SetEncoding, etc.)
//
// Example:
//
//	// Default encoding (json_ietf)
//	op := gnmi.Replace("/interfaces/interface[name=Gi0/0/0/0]/config",
//	    `{"mtu": 9000}`)
//
//	// Explicit encoding
//	op := gnmi.Replace("/system/config", jsonData,
//	    gnmi.SetEncoding("json"))
func Replace(path, value string, opts ...func(*SetOperation)) SetOperation {
	op := SetOperation{
		OperationType: OperationReplace,
		Path:          path,
		Value:         value,
		Encoding:      EncodingJSONIETF, // default
	}

	// Apply functional options
	for _, opt := range opts {
		opt(&op)
	}

	return op
}

// Delete creates a SetOperation for deleting a path
//
// Delete operations remove configuration at the specified path.
//
// Parameters:
//   - path: gNMI path string
//
// Example:
//
//	op := gnmi.Delete("/interfaces/interface[name=Gi0/0/0/1]/config")
func Delete(path string) SetOperation {
	return SetOperation{
		OperationType: OperationDelete,
		Path:          path,
		Value:         "", // Empty value for delete
	}
}

// Internal helper methods

// calculateTotalTimeout calculates the total timeout for all retry attempts
//
// This method calculates the actual total timeout based on exponential backoff
// by summing the backoff delays for all attempts. This provides an accurate
// timeout budget instead of assuming maximum backoff for all retries.
//
// Formula: OperationTimeout + sum(Backoff(0), Backoff(1), ..., Backoff(MaxRetries))
//
// Example:
//
//	OperationTimeout = 15s
//	MaxRetries = 3
//	BackoffMinDelay = 1s
//	BackoffDelayFactor = 2.0
//
//	Backoff delays:
//	  Attempt 0: 1s
//	  Attempt 1: 2s
//	  Attempt 2: 4s
//
//	Total timeout = 15s + 1s + 2s + 4s = 22s
//
// This is much more accurate than the old formula which would calculate:
//
//	15s + (3+1) × 60s = 255s (10x too long!)
//
// Returns the total timeout duration for all retry attempts.
func (c *Client) calculateTotalTimeout() time.Duration {
	totalBackoff := time.Duration(0)
	for attempt := 0; attempt <= c.MaxRetries; attempt++ {
		backoff := c.Backoff(attempt)
		totalBackoff += backoff
	}
	return c.OperationTimeout + totalBackoff
}

// checkContextCancellation checks if context is canceled or deadline exceeded
//
// This is a non-blocking check that immediately returns if the context is canceled
// or deadline has exceeded. Used before retry attempts to avoid wasted work.
//
// Returns context.Canceled if context is canceled, context.DeadlineExceeded if
// deadline exceeded, or nil if context is still valid.
func checkContextCancellation(ctx context.Context) error {
	select {
	case <-ctx.Done():
		return ctx.Err() // context.Canceled or context.DeadlineExceeded
	default:
		return nil
	}
}

// createAttemptContext creates a new context for a single retry attempt with timeout
//
// Timeout priority model:
//  1. Request-specific timeout (req.Timeout > 0) - highest priority
//  2. Existing context deadline (ctx.Deadline() set) - medium priority
//  3. Client default timeout (c.OperationTimeout) - fallback
//
// This model allows:
//   - Per-request timeout overrides: client.Get(ctx, paths, gnmi.Timeout(5*time.Second))
//   - Context deadline propagation: ctx, cancel := context.WithTimeout(parent, 30*time.Second)
//   - Sensible defaults: No timeout specified uses client.OperationTimeout
//
// CRITICAL: Caller MUST call the returned cancel function after operation completes
// to prevent goroutine leaks. Failure to call cancel will leak resources.
//
// Example usage:
//
//	attemptCtx, attemptCancel := c.createAttemptContext(ctx, req)
//	resp, err := c.target.Get(attemptCtx, getReq)
//	attemptCancel() // CRITICAL: Must clean up immediately
//
// Warnings are logged for extreme timeouts:
//   - Very short timeouts (<1s) may not allow operations to complete
//   - Very long timeouts (>5min) may delay error detection
//
// Returns a context with timeout applied and its cancel function.
func (c *Client) createAttemptContext(ctx context.Context, req *Req) (context.Context, context.CancelFunc) {
	// Priority 1: Request-specific timeout (highest)
	if req.Timeout > 0 {
		// Warn about extreme timeouts
		if req.Timeout < time.Second {
			c.logger.Warn(ctx, "request timeout is very short (may not complete)",
				"timeout", req.Timeout.String(),
				"target", c.Target)
		} else if req.Timeout > 5*time.Minute {
			c.logger.Warn(ctx, "request timeout is very long (may delay error detection)",
				"timeout", req.Timeout.String(),
				"target", c.Target)
		}

		c.logger.Debug(ctx, "applying request-specific timeout",
			"timeout", req.Timeout.String(),
			"source", "request",
			"target", c.Target)

		return context.WithTimeout(ctx, req.Timeout)
	}

	// Priority 2: Existing context deadline (medium)
	if deadline, hasDeadline := ctx.Deadline(); hasDeadline {
		// Context already has deadline, respect it
		remaining := time.Until(deadline)
		c.logger.Debug(ctx, "using existing context deadline",
			"remaining", remaining.String(),
			"source", "context",
			"target", c.Target)
		// Return context with cancel to maintain consistent API
		return context.WithCancel(ctx)
	}

	// Priority 3: Client default timeout (fallback)
	c.logger.Debug(ctx, "applying client default timeout",
		"timeout", c.OperationTimeout.String(),
		"source", "client",
		"target", c.Target)

	return context.WithTimeout(ctx, c.OperationTimeout)
}

// extractErrorDetails extracts error details from gRPC errors
//
// Parses gRPC status codes and error messages into ErrorModel structs.
//
// Returns a slice of ErrorModel with code, message, and details.
func (c *Client) extractErrorDetails(err error) []ErrorModel {
	if err == nil {
		return nil
	}

	// Try to extract gRPC status
	if st, ok := status.FromError(err); ok {
		return []ErrorModel{{
			Code:    uint32(st.Code()),
			Message: st.Message(),
			Details: st.String(),
		}}
	}

	// Fallback: return generic error
	return []ErrorModel{{
		Code:    0,
		Message: err.Error(),
		Details: "",
	}}
}

// isValidGNMIPath checks if a path is in valid gNMI format
//
// Valid formats:
//  1. Absolute path: /interfaces/interface[name=eth0]
//  2. Module-qualified path: module-name:/path (e.g., openconfig-interfaces:/interfaces)
//
// Returns true if the path is valid, false otherwise.
func isValidGNMIPath(path string) bool {
	if len(path) == 0 {
		return false
	}

	// Check for absolute path (starts with /)
	if path[0] == '/' {
		return true
	}

	// Check for module-qualified path (module:path format)
	// Must contain : and the part after : must start with /
	colonIdx := strings.IndexByte(path, ':')
	if colonIdx > 0 && colonIdx < len(path)-1 {
		// Has module prefix, check if path part starts with /
		pathPart := path[colonIdx+1:]
		return len(pathPart) > 0 && pathPart[0] == '/'
	}

	return false
}
