//nolint:errcheck,gosec // Example code prioritizes readability over error handling
// SPDX-License-Identifier: MPL-2.0
// Copyright (c) 2025 Daniel Schmidt

// Package main demonstrates logging configuration in go-gnmi.
//
// This example shows:
//   - Default behavior (no logging)
//   - Configuring log levels (Debug, Info, Warn, Error, None)
//   - Using WithLogger() and WithPrettyPrintLogs() options
//   - Automatic sensitive data redaction
//
// Usage:
//
//	export GNMI_TARGET=192.168.1.1:57400
//	export GNMI_USERNAME=admin
//	export GNMI_PASSWORD=secret
//	go run main.go
package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/netascode/go-gnmi"
)

func main() {
	target := getEnv("GNMI_TARGET", "192.168.1.1:57400")
	username := getEnv("GNMI_USERNAME", "admin")
	password := getEnv("GNMI_PASSWORD", "secret")

	// Example 1: Default behavior (No logging)
	fmt.Println("=== Example 1: Default Behavior (No Logging) ===")
	client1, err := gnmi.NewClient(
		target,
		gnmi.Username(username),
		gnmi.Password(password),
		gnmi.TLS(true),
		gnmi.VerifyCertificate(false), // WARNING: Disables TLS verification - TESTING ONLY
	)
	if err != nil {
		log.Printf("Failed to connect (no logging): %v", err)
	} else {
		fmt.Println("Connected successfully (logs are disabled by default)")
		client1.Close() //nolint:errcheck // Error intentionally ignored in example
	}

	// Example 2: Enable logging at Info level
	fmt.Println("\n=== Example 2: Info Level Logging ===")
	logger := gnmi.NewDefaultLogger(gnmi.LogLevelInfo)
	client2, err := gnmi.NewClient(
		target,
		gnmi.Username(username),
		gnmi.Password(password),
		gnmi.TLS(true),
		gnmi.VerifyCertificate(false), // WARNING: Disables TLS verification - TESTING ONLY
		gnmi.WithLogger(logger),
	)
	if err != nil {
		log.Printf("Failed to connect (info logging): %v", err)
	} else {
		fmt.Println("Connected - check logs above for connection info")
		defer client2.Close() //nolint:errcheck // Error intentionally ignored in example

		// Perform a simple operation
		ctx := context.Background()
		paths := []string{"/system/config"}
		_, err := client2.Get(ctx, paths)
		if err != nil {
			log.Printf("Get failed: %v", err)
		}
	}

	// Example 3: Enable debug logging with pretty printing disabled
	fmt.Println("\n=== Example 3: Debug Level Logging (No Pretty Print) ===")
	debugLogger := gnmi.NewDefaultLogger(gnmi.LogLevelDebug)
	client3, err := gnmi.NewClient(
		target,
		gnmi.Username(username),
		gnmi.Password(password),
		gnmi.TLS(true),
		gnmi.VerifyCertificate(false), // WARNING: Disables TLS verification - TESTING ONLY
		gnmi.WithLogger(debugLogger),
		gnmi.WithPrettyPrintLogs(false), // Disable pretty printing for performance
	)
	if err != nil {
		log.Printf("Failed to connect (debug logging): %v", err)
	} else {
		fmt.Println("Connected - check logs above for detailed debug info")
		defer client3.Close() //nolint:errcheck // Error intentionally ignored in example

		// Perform operations to see detailed logging
		ctx := context.Background()
		paths := []string{"/interfaces/interface[name=GigabitEthernet0/0/0/0]/config"}
		_, err := client3.Get(ctx, paths)
		if err != nil {
			log.Printf("Get with paths failed: %v", err)
		}
	}

	// Example 4: Different log levels
	fmt.Println("\n=== Example 4: Log Level Comparison ===")
	logLevels := []struct {
		name  string
		level gnmi.LogLevel
	}{
		{"Debug (most verbose)", gnmi.LogLevelDebug},
		{"Info", gnmi.LogLevelInfo},
		{"Warn", gnmi.LogLevelWarn},
		{"Error (least verbose)", gnmi.LogLevelError},
		{"None (logging disabled)", gnmi.LogLevelNone},
	}

	for _, ll := range logLevels {
		fmt.Printf("\nLog Level: %s\n", ll.name)
		logger := gnmi.NewDefaultLogger(ll.level)

		// Demonstrate different log levels
		ctx := context.Background()
		logger.Debug(ctx, "This is a debug message", "key", "value")
		logger.Info(ctx, "This is an info message", "target", "192.168.1.1:57400")
		logger.Warn(ctx, "This is a warning message", "attempt", 1)
		logger.Error(ctx, "This is an error message", "error", "something went wrong")
	}

	// Example 5: Sensitive data redaction
	fmt.Println("\n=== Example 5: Sensitive Data Redaction ===")
	fmt.Println("Demonstrating automatic redaction of sensitive data in logs...")

	// Create client with debug logging to show redaction in action
	redactionLogger := gnmi.NewDefaultLogger(gnmi.LogLevelDebug)
	client5, err := gnmi.NewClient(
		target,
		gnmi.Username(username),
		gnmi.Password(password),
		gnmi.TLS(true),
		gnmi.VerifyCertificate(false), // WARNING: Disables TLS verification - TESTING ONLY
		gnmi.WithLogger(redactionLogger),
	)
	if err != nil {
		log.Printf("Failed to connect (redaction example): %v", err)
	} else {
		defer client5.Close() //nolint:errcheck // Error intentionally ignored in example

		// Build configuration with sensitive data in various formats
		// Note: This will likely fail on a real device, but demonstrates redaction
		body := gnmi.Body{}.
			Set("config.hostname", "SecureRouter").
			Set("config.snmp.community", "secret-community-string").
			Set("config.users.user.name", "admin").
			Set("config.users.user.password", "super-secret-password-123").
			Set("config.api.key", "sk-1234567890abcdef").
			Set("config.credentials.secret", "my-secret-token")

		value, err := body.String()
		if err != nil {
			fmt.Printf("Body builder failed: %v\n", err)
			return
		}

		fmt.Println("\nOriginal config contains sensitive data:")
		fmt.Println("  - SNMP community: secret-community-string")
		fmt.Println("  - User password: super-secret-password-123")
		fmt.Println("  - API key: sk-1234567890abcdef")
		fmt.Println("  - Secret token: my-secret-token")

		fmt.Println("\nAttempting Set with sensitive data...")
		fmt.Println("Look at the DEBUG logs above - sensitive values are replaced with [REDACTED]")

		// This operation will likely fail (invalid config for most devices),
		// but the logging demonstrates redaction
		ctx := context.Background()
		ops := []gnmi.SetOperation{
			gnmi.Update("/system/config", value, "json_ietf"),
		}
		_, err = client5.Set(ctx, ops)
		if err != nil {
			fmt.Printf("\nSet failed as expected: %v\n", err)
			fmt.Println("But notice in the DEBUG logs - all sensitive data was redacted!")
		}

		fmt.Println("\nRedaction patterns automatically protect:")
		fmt.Println("  ✓ JSON fields: \"password\": \"value\" → \"password\": \"[REDACTED]\"")
		fmt.Println("  ✓ JSON fields: \"secret\": \"value\" → \"secret\": \"[REDACTED]\"")
		fmt.Println("  ✓ JSON fields: \"key\": \"value\" → \"key\": \"[REDACTED]\"")
		fmt.Println("  ✓ JSON fields: \"community\": \"value\" → \"community\": \"[REDACTED]\"")
		fmt.Println("  ✓ JSON fields: \"token\": \"value\" → \"token\": \"[REDACTED]\"")
		fmt.Println("  ✓ JSON fields: \"auth\": \"value\" → \"auth\": \"[REDACTED]\"")
	}

	// Example 6: Custom logger implementation
	fmt.Println("\n=== Example 6: Custom Logger Implementation ===")
	customLogger := &CustomLogger{prefix: "[CUSTOM]"}
	client6, err := gnmi.NewClient(
		target,
		gnmi.Username(username),
		gnmi.Password(password),
		gnmi.TLS(true),
		gnmi.VerifyCertificate(false), // WARNING: Disables TLS verification - TESTING ONLY
		gnmi.WithLogger(customLogger),
	)
	if err != nil {
		log.Printf("Failed to connect (custom logger): %v", err)
	} else {
		fmt.Println("Connected with custom logger")
		defer client6.Close() //nolint:errcheck // Error intentionally ignored in example

		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		paths := []string{"/system/config/hostname"}
		_, err := client6.Get(ctx, paths)
		if err != nil {
			fmt.Printf("Get failed: %v\n", err)
		}
	}

	// Example 7: Context-aware logging (trace correlation)
	fmt.Println("\n=== Example 7: Context-Aware Logging (Trace Correlation) ===")
	fmt.Println("Demonstrating how to implement context-aware logging for distributed tracing...")

	contextAwareLogger := &ContextAwareLogger{prefix: "[TRACE]"}
	client7, err := gnmi.NewClient(
		target,
		gnmi.Username(username),
		gnmi.Password(password),
		gnmi.TLS(true),
		gnmi.VerifyCertificate(false), // WARNING: Disables TLS verification - TESTING ONLY
		gnmi.WithLogger(contextAwareLogger),
	)
	if err != nil {
		log.Printf("Failed to connect (context-aware logger): %v", err)
	} else {
		fmt.Println("Connected with context-aware logger")
		defer client7.Close() //nolint:errcheck // Error intentionally ignored in example

		// Create context with trace ID (simulating distributed tracing)
		type contextKey string
		traceIDKey := contextKey("trace_id")
		requestIDKey := contextKey("request_id")

		ctx := context.WithValue(context.Background(), traceIDKey, "trace-abc-123")
		ctx = context.WithValue(ctx, requestIDKey, "req-xyz-789")

		// Add deadline to demonstrate deadline extraction
		ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
		defer cancel()

		fmt.Println("\nContext contains:")
		fmt.Println("  - trace_id: trace-abc-123")
		fmt.Println("  - request_id: req-xyz-789")
		fmt.Println("  - deadline: 10 seconds from now")

		fmt.Println("\nPerforming Get operation - notice logs include trace correlation data...")
		paths := []string{"/system/config/hostname"}
		_, err := client7.Get(ctx, paths)
		if err != nil {
			fmt.Printf("Get failed: %v\n", err)
		}

		fmt.Println("\nKey takeaways:")
		fmt.Println("  ✓ Context-aware loggers can extract trace IDs for correlation")
		fmt.Println("  ✓ Request IDs enable tracking operations across services")
		fmt.Println("  ✓ Deadline information helps debug timeout issues")
		fmt.Println("  ✓ Custom metadata can be propagated via context values")
	}

	fmt.Println("\n=== Examples Complete ===")
}

// CustomLogger is a custom implementation of the Logger interface
//
// SECURITY WARNING: This is a simplified example for demonstration purposes.
// Production custom loggers should implement:
//   - Log value sanitization to prevent log injection attacks
//   - Sensitive field redaction (password, secret, key, token, auth)
//   - Message size limits to prevent resource exhaustion
//
// See DefaultLogger in logger.go for a production-ready reference implementation.
type CustomLogger struct {
	prefix string
}

func (l *CustomLogger) Debug(ctx context.Context, msg string, keysAndValues ...interface{}) {
	l.log("DEBUG", msg, keysAndValues...)
}

func (l *CustomLogger) Info(ctx context.Context, msg string, keysAndValues ...interface{}) {
	l.log("INFO", msg, keysAndValues...)
}

func (l *CustomLogger) Warn(ctx context.Context, msg string, keysAndValues ...interface{}) {
	l.log("WARN", msg, keysAndValues...)
}

func (l *CustomLogger) Error(ctx context.Context, msg string, keysAndValues ...interface{}) {
	l.log("ERROR", msg, keysAndValues...)
}

func (l *CustomLogger) log(level, msg string, keysAndValues ...interface{}) {
	fmt.Printf("%s [%s] %s", l.prefix, level, msg)
	for i := 0; i < len(keysAndValues); i += 2 {
		if i+1 < len(keysAndValues) {
			fmt.Printf(" %v=%v", keysAndValues[i], keysAndValues[i+1])
		}
	}
	fmt.Println()
}

// ContextAwareLogger demonstrates how to extract trace correlation data from context
//
// This logger shows best practices for context-aware logging:
//   - Extract trace IDs, request IDs, and other correlation data from context
//   - Check context deadline to log timeout information
//   - Propagate context to downstream logging frameworks (slog, zap, etc.)
//
// SECURITY WARNING: This is a simplified example for demonstration purposes.
// Production custom loggers should implement:
//   - Log value sanitization to prevent log injection attacks
//   - Sensitive field redaction (password, secret, key, token, auth)
//   - Message size limits to prevent resource exhaustion
//
// See DefaultLogger in logger.go for a production-ready reference implementation.
type ContextAwareLogger struct {
	prefix string
}

func (l *ContextAwareLogger) Debug(ctx context.Context, msg string, keysAndValues ...interface{}) {
	l.logWithContext(ctx, "DEBUG", msg, keysAndValues...)
}

func (l *ContextAwareLogger) Info(ctx context.Context, msg string, keysAndValues ...interface{}) {
	l.logWithContext(ctx, "INFO", msg, keysAndValues...)
}

func (l *ContextAwareLogger) Warn(ctx context.Context, msg string, keysAndValues ...interface{}) {
	l.logWithContext(ctx, "WARN", msg, keysAndValues...)
}

func (l *ContextAwareLogger) Error(ctx context.Context, msg string, keysAndValues ...interface{}) {
	l.logWithContext(ctx, "ERROR", msg, keysAndValues...)
}

func (l *ContextAwareLogger) logWithContext(ctx context.Context, level, msg string, keysAndValues ...interface{}) {
	// Extract trace correlation data from context
	type contextKey string
	var extractedValues []interface{}

	// Extract trace ID if present
	if traceID := ctx.Value(contextKey("trace_id")); traceID != nil {
		extractedValues = append(extractedValues, "trace_id", traceID)
	}

	// Extract request ID if present
	if requestID := ctx.Value(contextKey("request_id")); requestID != nil {
		extractedValues = append(extractedValues, "request_id", requestID)
	}

	// Extract deadline information if present
	if deadline, ok := ctx.Deadline(); ok {
		remaining := time.Until(deadline)
		extractedValues = append(extractedValues, "deadline_remaining", remaining.String())
	}

	// Combine extracted context values with provided key-values
	allValues := append(extractedValues, keysAndValues...)

	// Log with all information
	fmt.Printf("%s [%s] %s", l.prefix, level, msg)
	for i := 0; i < len(allValues); i += 2 {
		if i+1 < len(allValues) {
			fmt.Printf(" %v=%v", allValues[i], allValues[i+1])
		}
	}
	fmt.Println()
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
