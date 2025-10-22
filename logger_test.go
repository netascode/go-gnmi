// SPDX-License-Identifier: MPL-2.0
// Copyright (c) 2025 Daniel Schmidt

package gnmi

import (
	"bytes"
	"context"
	"log"
	"strings"
	"testing"
)

// TestDefaultLogger_LogLevels verifies log level filtering
func TestDefaultLogger_LogLevels(t *testing.T) {
	tests := []struct {
		name          string
		level         LogLevel
		logFunc       func(*DefaultLogger)
		expectMessage bool
	}{
		{
			name:  "debug level logs debug",
			level: LogLevelDebug,
			logFunc: func(l *DefaultLogger) {
				l.Debug(context.Background(), "test message")
			},
			expectMessage: true,
		},
		{
			name:  "info level filters debug",
			level: LogLevelInfo,
			logFunc: func(l *DefaultLogger) {
				l.Debug(context.Background(), "test message")
			},
			expectMessage: false,
		},
		{
			name:  "info level logs info",
			level: LogLevelInfo,
			logFunc: func(l *DefaultLogger) {
				l.Info(context.Background(), "test message")
			},
			expectMessage: true,
		},
		{
			name:  "warn level filters info",
			level: LogLevelWarn,
			logFunc: func(l *DefaultLogger) {
				l.Info(context.Background(), "test message")
			},
			expectMessage: false,
		},
		{
			name:  "error level filters warn",
			level: LogLevelError,
			logFunc: func(l *DefaultLogger) {
				l.Warn(context.Background(), "test message")
			},
			expectMessage: false,
		},
		{
			name:  "error level logs error",
			level: LogLevelError,
			logFunc: func(l *DefaultLogger) {
				l.Error(context.Background(), "test message")
			},
			expectMessage: true,
		},
		{
			name:  "none level filters all",
			level: LogLevelNone,
			logFunc: func(l *DefaultLogger) {
				l.Error(context.Background(), "test message")
			},
			expectMessage: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Capture log output
			var buf bytes.Buffer
			log.SetOutput(&buf)
			t.Cleanup(func() { log.SetOutput(nil) })

			logger := NewDefaultLogger(tt.level)
			tt.logFunc(logger)

			output := buf.String()
			if tt.expectMessage && output == "" {
				t.Errorf("expected log message but got none")
			}
			if !tt.expectMessage && output != "" {
				t.Errorf("expected no log message but got: %s", output)
			}
		})
	}
}

// TestSanitizeLogValue_ControlCharacters tests control character sanitization
func TestSanitizeLogValue_ControlCharacters(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "newline injection",
			input:    "user\n[ERROR] Fake attack",
			expected: "user [ERROR] Fake attack", // Newlines replaced with spaces
		},
		{
			name:     "carriage return",
			input:    "test\roverwrite",
			expected: "test overwrite",
		},
		{
			name:     "tab injection",
			input:    "value\tinjected",
			expected: "value injected",
		},
		{
			name:     "ANSI escape sequence",
			input:    "text\x1B[31mred\x1B[0m",
			expected: "text.[31mred.[0m", // ESC replaced with dot
		},
		{
			name:     "bell character",
			input:    "beep\x07beep",
			expected: "beep.beep",
		},
		{
			name:     "backspace manipulation",
			input:    "abc\x08\x08\x08def",
			expected: "abc...def",
		},
		{
			name:     "null byte",
			input:    "test\x00null",
			expected: "test.null",
		},
		{
			name:     "form feed",
			input:    "page1\x0Cpage2",
			expected: "page1 page2",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := sanitizeLogValue(tt.input)
			if result != tt.expected {
				t.Errorf("sanitizeLogValue() = %q, want %q", result, tt.expected)
			}
		})
	}
}

// TestSanitizeLogValue_UnicodeAttacks tests Unicode attack prevention
func TestSanitizeLogValue_UnicodeAttacks(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
		desc     string
	}{
		{
			name:     "zero-width space",
			input:    "admin\u200Bhidden",
			expected: "admin..hidden", // Skipped multi-byte chars show as dots
			desc:     "zero-width space should be neutralized",
		},
		{
			name:     "zero-width non-joiner",
			input:    "test\u200Cvalue",
			expected: "test..value", // Skipped multi-byte chars show as dots
			desc:     "zero-width non-joiner should be neutralized",
		},
		{
			name:     "zero-width joiner",
			input:    "test\u200Dvalue",
			expected: "test..value", // Skipped multi-byte chars show as dots
			desc:     "zero-width joiner should be neutralized",
		},
		{
			name:     "right-to-left override",
			input:    "admin\u202Ekcatta",
			expected: "admin ..kcatta", // RTL override replaced with space + dot
			desc:     "RTL override should be neutralized",
		},
		{
			name:     "byte order mark",
			input:    "test\uFEFFvalue",
			expected: "test..value", // BOM neutralized as dots
			desc:     "BOM should be neutralized",
		},
		{
			name:     "normal unicode",
			input:    "こんにちは世界",
			expected: "こんにちは世界",
			desc:     "normal Unicode should be preserved",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := sanitizeLogValue(tt.input)
			if result != tt.expected {
				t.Errorf("sanitizeLogValue() = %q, want %q\nDesc: %s", result, tt.expected, tt.desc)
			}
		})
	}
}

// TestSanitizeLogValue_Truncation tests value truncation for DoS prevention
func TestSanitizeLogValue_Truncation(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		maxLen    int
		wantTrunc bool
	}{
		{
			name:      "short value",
			input:     "short",
			maxLen:    1024,
			wantTrunc: false,
		},
		{
			name:      "exact max length",
			input:     strings.Repeat("a", MaxLogValueLength),
			maxLen:    MaxLogValueLength,
			wantTrunc: false,
		},
		{
			name:      "exceeds max length",
			input:     strings.Repeat("a", MaxLogValueLength+100),
			maxLen:    MaxLogValueLength,
			wantTrunc: true,
		},
		{
			name:      "very large value",
			input:     strings.Repeat("a", 10000),
			maxLen:    MaxLogValueLength,
			wantTrunc: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := sanitizeLogValue(tt.input)

			if tt.wantTrunc {
				if !strings.Contains(result, "...[TRUNCATED]") {
					t.Errorf("expected truncation marker but got: %s", result)
				}
				if len(result) > MaxLogValueLength+20 { // Allow for truncation marker
					t.Errorf("result length %d exceeds expected max", len(result))
				}
			} else {
				if strings.Contains(result, "...[TRUNCATED]") {
					t.Errorf("unexpected truncation for input length %d", len(tt.input))
				}
			}
		})
	}
}

// TestDefaultLogger_KeyValuePairs tests structured logging with key-value pairs
func TestDefaultLogger_KeyValuePairs(t *testing.T) {
	tests := []struct {
		name            string
		keysAndValues   []any
		expectedPairs   []string
		unexpectedPairs []string
	}{
		{
			name:          "even pairs",
			keysAndValues: []any{"key1", "value1", "key2", 123},
			expectedPairs: []string{"key1=value1", "key2=123"},
		},
		{
			name:          "odd pairs (missing value)",
			keysAndValues: []any{"key1", "value1", "key2"},
			expectedPairs: []string{"key1=value1", "key2=<MISSING>"},
		},
		{
			name:            "control characters in values",
			keysAndValues:   []any{"key", "value\ninjection"},
			expectedPairs:   []string{"key=value injection"}, // Newline replaced with space
			unexpectedPairs: []string{"key=value\ninjection"},
		},
		{
			name:          "unicode in keys and values",
			keysAndValues: []any{"キー", "値"},
			expectedPairs: []string{"キー=値"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			log.SetOutput(&buf)
			t.Cleanup(func() { log.SetOutput(nil) })

			logger := NewDefaultLogger(LogLevelDebug)
			logger.Debug(context.Background(), "test", tt.keysAndValues...)

			output := buf.String()

			for _, expected := range tt.expectedPairs {
				if !strings.Contains(output, expected) {
					t.Errorf("expected log to contain %q but got: %s", expected, output)
				}
			}

			for _, unexpected := range tt.unexpectedPairs {
				if strings.Contains(output, unexpected) {
					t.Errorf("expected log NOT to contain %q but got: %s", unexpected, output)
				}
			}
		})
	}
}

// TestNoOpLogger verifies that NoOpLogger methods can be called without panics.
//
// COVERAGE NOTE: NoOpLogger methods show 0% coverage despite being tested.
// This is CORRECT and INTENTIONAL:
//   - NoOpLogger methods are empty stubs by design (no code to execute)
//   - Tests verify methods can be called without panic (behavioral testing)
//   - Coverage tools cannot measure "empty function called successfully"
//   - This is appropriate for a no-op implementation with zero overhead
//
// The 0% coverage does NOT indicate untested code - it indicates empty implementations.
func TestNoOpLogger(t *testing.T) {
	var buf bytes.Buffer
	log.SetOutput(&buf)
	t.Cleanup(func() { log.SetOutput(nil) })

	logger := &NoOpLogger{}

	// All these should produce no output
	logger.Debug(context.Background(), "test")
	logger.Info(context.Background(), "test")
	logger.Warn(context.Background(), "test")
	logger.Error(context.Background(), "test")

	output := buf.String()
	if output != "" {
		t.Errorf("NoOpLogger produced output: %s", output)
	}
}

// TestNoOpLogger_WithKeyValues verifies NoOpLogger with key-value pairs
func TestNoOpLogger_WithKeyValues(t *testing.T) {
	var buf bytes.Buffer
	log.SetOutput(&buf)
	t.Cleanup(func() { log.SetOutput(nil) })

	logger := &NoOpLogger{}

	// All these should produce no output, even with key-value pairs
	logger.Debug(context.Background(), "debug message", "key1", "value1", "key2", 123)
	logger.Info(context.Background(), "info message", "operation", "test", "duration", 500)
	logger.Warn(context.Background(), "warn message", "warning", "something")
	logger.Error(context.Background(), "error message", "error", "failed", "code", 500)

	output := buf.String()
	if output != "" {
		t.Errorf("NoOpLogger produced output with key-values: %s", output)
	}
}

// TestNoOpLogger_NoPanic verifies NoOpLogger is panic-safe under high-volume usage
func TestNoOpLogger_NoPanic(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("NoOpLogger panicked: %v", r)
		}
	}()

	logger := &NoOpLogger{}

	// High-volume test: 1000 calls should complete without panic
	for i := 0; i < 1000; i++ {
		logger.Debug(context.Background(), "test message", "iteration", i, "key", "value")
		logger.Info(context.Background(), "test message", "iteration", i, "key", "value")
		logger.Warn(context.Background(), "test message", "iteration", i, "key", "value")
		logger.Error(context.Background(), "test message", "iteration", i, "key", "value")
	}

	t.Log("NoOpLogger successfully handled 4000 log calls without panic")
}

// TestLogLevel_String tests LogLevel string representation
func TestLogLevel_String(t *testing.T) {
	tests := []struct {
		level    LogLevel
		expected string
	}{
		{LogLevelDebug, "DEBUG"},
		{LogLevelInfo, "INFO"},
		{LogLevelWarn, "WARN"},
		{LogLevelError, "ERROR"},
		{LogLevelNone, "NONE"},
		{LogLevel(99), "UNKNOWN(99)"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			result := tt.level.String()
			if result != tt.expected {
				t.Errorf("String() = %q, want %q", result, tt.expected)
			}
		})
	}
}

// TestSanitizeLogValue_InvalidUTF8 tests invalid UTF-8 handling
func TestSanitizeLogValue_InvalidUTF8(t *testing.T) {
	// Invalid UTF-8 sequences
	tests := []struct {
		name  string
		input string
	}{
		{
			name:  "invalid 2-byte sequence",
			input: "test\xc3\x28data", // Invalid UTF-8
		},
		{
			name:  "truncated 3-byte sequence",
			input: "test\xe2\x82data", // Incomplete UTF-8
		},
		{
			name:  "invalid byte",
			input: "test\xffdata", // Invalid UTF-8 start byte
		},
		{
			name:  "mixed valid and invalid",
			input: "hello\xc3\x28world\xe2\x82end\xff",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Should not hang - if test times out, infinite loop bug exists
			result := sanitizeLogValue(tt.input)

			// Should return something (sanitized version)
			if result == "" {
				t.Error("expected non-empty result")
			}

			// Should not panic
			t.Logf("Sanitized: %q", result)

			// Result should not contain invalid UTF-8
			if !strings.Contains(result, ".") {
				// At least some invalid bytes should be replaced with dots
				t.Logf("Warning: no dots in result, may not have sanitized: %q", result)
			}
		})
	}
}

// TestSanitizeLogValue_SecurityPatterns tests security-sensitive patterns
func TestSanitizeLogValue_SecurityPatterns(t *testing.T) {
	tests := []struct {
		name  string
		input string
		desc  string
	}{
		{
			name:  "log injection attempt",
			input: "user\n[ERROR] System compromised\n[INFO] User logged in",
			desc:  "should prevent fake log entry injection",
		},
		{
			name:  "ANSI terminal manipulation",
			input: "text\x1B[2J\x1B[H",
			desc:  "should prevent terminal screen clearing",
		},
		{
			name:  "combined attack",
			input: "user\n\x1B[31m[ERROR]\x1B[0m Attack\r\nOverwrite",
			desc:  "should prevent multiple attack vectors",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := sanitizeLogValue(tt.input)

			// Should not contain newlines (injection prevention)
			if strings.Contains(result, "\n") {
				t.Errorf("result contains newline (injection risk): %q\nDesc: %s", result, tt.desc)
			}

			// Should not contain carriage returns
			if strings.Contains(result, "\r") {
				t.Errorf("result contains carriage return (injection risk): %q\nDesc: %s", result, tt.desc)
			}

			// Should not contain ESC sequences
			if strings.Contains(result, "\x1B") {
				t.Errorf("result contains ESC character (ANSI injection risk): %q\nDesc: %s", result, tt.desc)
			}
		})
	}
}

// BenchmarkSanitizeLogValue benchmarks log sanitization performance
func BenchmarkSanitizeLogValue(b *testing.B) {
	testCases := []struct {
		name  string
		input string
	}{
		{
			name:  "short clean",
			input: "simple value",
		},
		{
			name:  "long clean",
			input: strings.Repeat("a", 500),
		},
		{
			name:  "with control chars",
			input: "value\nwith\tcontrol\rchars",
		},
		{
			name:  "with unicode",
			input: "こんにちは世界 Hello World",
		},
		{
			name:  "needs truncation",
			input: strings.Repeat("a", MaxLogValueLength+100),
		},
	}

	for _, tc := range testCases {
		b.Run(tc.name, func(b *testing.B) {
			b.ReportAllocs()
			for i := 0; i < b.N; i++ {
				_ = sanitizeLogValue(tc.input)
			}
		})
	}
}

// BenchmarkDefaultLogger benchmarks structured logging performance
func BenchmarkDefaultLogger(b *testing.B) {
	// Discard log output
	log.SetOutput(&bytes.Buffer{})
	b.Cleanup(func() { log.SetOutput(nil) })

	logger := NewDefaultLogger(LogLevelInfo)

	b.Run("simple message", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			logger.Info(context.Background(), "test message")
		}
	})

	b.Run("with key-values", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			logger.Info(context.Background(), "test message", "key1", "value1", "key2", 123)
		}
	})

	b.Run("filtered debug", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			logger.Debug(context.Background(), "test message", "key", "value")
		}
	})
}

// TestSecurity_ReDoSProtection tests ReDoS protection in log redaction
func TestSecurity_ReDoSProtection(t *testing.T) {
	t.Run("size limit exceeded", func(t *testing.T) {
		client := &Client{
			prettyPrintLogs:   true,
			redactionPatterns: defaultRedactionPatterns,
			logger:            &NoOpLogger{},
		}

		// Create JSON larger than MaxJSONSizeForLogging (1MB)
		largeJSON := strings.Repeat(`{"data":"x"}`, 100000) // ~1.2MB

		result := client.prepareJSONForLogging(largeJSON)

		if result != JSONTooLargeMessage {
			t.Errorf("Expected JSONTooLargeMessage, got: %q", result)
		}
	})

	t.Run("sensitive field count limit", func(t *testing.T) {
		client := &Client{
			prettyPrintLogs:   true,
			redactionPatterns: defaultRedactionPatterns,
			logger:            &NoOpLogger{},
		}

		// Create JSON with too many sensitive fields (>1000)
		var builder strings.Builder
		builder.WriteString("{")
		for i := 0; i < 1001; i++ {
			if i > 0 {
				builder.WriteString(",")
			}
			builder.WriteString(`"password":"secret`)
			builder.WriteString(string(rune('0' + (i % 10))))
			builder.WriteString(`"`)
		}
		builder.WriteString("}")

		result := client.prepareJSONForLogging(builder.String())

		if result != JSONTooManySensitiveMsg {
			t.Errorf("Expected JSONTooManySensitiveMsg, got: %q", result)
		}
	})
}

// TestPrepareJSONForLogging_PrettyPrint tests JSON pretty-printing functionality
func TestPrepareJSONForLogging_PrettyPrint(t *testing.T) {
	tests := []struct {
		name           string
		prettyPrint    bool
		input          string
		wantFormatted  bool
		wantRedacted   bool
		checkMultiline bool
		checkIndented  bool
	}{
		{
			name:           "pretty print enabled - simple JSON",
			prettyPrint:    true,
			input:          `{"hostname":"router1","enabled":true}`,
			wantFormatted:  true,
			checkMultiline: true,
			checkIndented:  true,
		},
		{
			name:           "pretty print disabled - simple JSON",
			prettyPrint:    false,
			input:          `{"hostname":"router1","enabled":true}`,
			wantFormatted:  false,
			checkMultiline: false,
		},
		{
			name:           "pretty print enabled with redaction",
			prettyPrint:    true,
			input:          `{"hostname":"router1","password":"secret123"}`,
			wantFormatted:  true,
			wantRedacted:   true,
			checkMultiline: true,
			checkIndented:  true,
		},
		{
			name:          "pretty print disabled with redaction",
			prettyPrint:   false,
			input:         `{"hostname":"router1","password":"secret123"}`,
			wantFormatted: false,
			wantRedacted:  true,
		},
		{
			name:           "pretty print enabled - nested JSON",
			prettyPrint:    true,
			input:          `{"system":{"hostname":"router1","config":{"domain":"example.com"}}}`,
			wantFormatted:  true,
			checkMultiline: true,
			checkIndented:  true,
		},
		{
			name:          "pretty print disabled - nested JSON",
			prettyPrint:   false,
			input:         `{"system":{"hostname":"router1","config":{"domain":"example.com"}}}`,
			wantFormatted: false,
		},
		{
			name:          "invalid JSON - pretty print enabled (fallback)",
			prettyPrint:   true,
			input:         `{invalid json}`,
			wantFormatted: false, // Should fallback to raw
		},
		{
			name:          "invalid JSON - pretty print disabled",
			prettyPrint:   false,
			input:         `{invalid json}`,
			wantFormatted: false,
		},
		{
			name:           "empty JSON object - pretty print enabled",
			prettyPrint:    true,
			input:          `{}`,
			wantFormatted:  true,
			checkMultiline: false, // json.Indent doesn't add newlines for empty objects
		},
		{
			name:           "JSON array - pretty print enabled",
			prettyPrint:    true,
			input:          `[{"id":1,"name":"device1"},{"id":2,"name":"device2"}]`,
			wantFormatted:  true,
			checkMultiline: true,
			checkIndented:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := &Client{
				prettyPrintLogs:   tt.prettyPrint,
				redactionPatterns: defaultRedactionPatterns,
				logger:            &NoOpLogger{},
			}

			result := client.prepareJSONForLogging(tt.input)

			// Check redaction
			if tt.wantRedacted {
				if strings.Contains(result, "secret123") {
					t.Error("Expected password to be redacted, but found 'secret123'")
				}
				if !strings.Contains(result, "[REDACTED]") {
					t.Error("Expected [REDACTED] placeholder, but not found")
				}
			}

			// Check formatting
			if tt.checkMultiline {
				lineCount := strings.Count(result, "\n")
				if tt.wantFormatted && lineCount < 2 {
					t.Errorf("Expected multiline output (pretty-printed), got %d lines: %q", lineCount+1, result)
				}
				if !tt.wantFormatted && lineCount >= 2 {
					t.Errorf("Expected single-line output (not pretty-printed), got %d lines", lineCount+1)
				}
			}

			// Check indentation (2 spaces)
			if tt.checkIndented && tt.wantFormatted {
				if !strings.Contains(result, "  ") {
					t.Error("Expected 2-space indentation in pretty-printed output, but not found")
				}
			}

			// Ensure result is not empty
			if len(result) == 0 {
				t.Error("prepareJSONForLogging returned empty string")
			}
		})
	}
}

// TestPrepareJSONForLogging_RealWorld tests with real-world JSON payloads
func TestPrepareJSONForLogging_RealWorld(t *testing.T) {
	tests := []struct {
		name   string
		pretty bool
		input  string
	}{
		{
			name:   "gNMI GetResponse structure",
			pretty: true,
			input:  `{"notification":[{"timestamp":1234567890,"update":[{"path":"/interfaces/interface[name=eth0]/config/description","val":{"string_val":"WAN Interface"}}]}]}`,
		},
		{
			name:   "gNMI SetRequest with credentials (top-level)",
			pretty: true,
			input:  `{"update":[{"path":"/system/aaa/authentication","password":"mysecret","enabled":true}]}`,
		},
		{
			name:   "Complex nested configuration",
			pretty: false,
			input:  `{"interfaces":{"interface":[{"name":"eth0","config":{"description":"test","enabled":true},"state":{"admin-status":"UP"}}]}}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := &Client{
				prettyPrintLogs:   tt.pretty,
				redactionPatterns: defaultRedactionPatterns,
				logger:            &NoOpLogger{},
			}

			result := client.prepareJSONForLogging(tt.input)

			// Basic sanity checks
			if len(result) == 0 {
				t.Error("prepareJSONForLogging returned empty string")
			}

			// Verify sensitive data is redacted
			if strings.Contains(result, "mysecret") {
				t.Error("Expected password 'mysecret' to be redacted")
			}

			// If pretty-print enabled, check for formatting
			if tt.pretty {
				if !strings.Contains(result, "\n") {
					t.Error("Expected newlines in pretty-printed output")
				}
			}
		})
	}
}

// TestPrepareJSONForLogging_Consistency ensures behavior matches go-netconf patterns
func TestPrepareJSONForLogging_Consistency(t *testing.T) {
	t.Run("redaction happens before formatting", func(t *testing.T) {
		client := &Client{
			prettyPrintLogs:   true,
			redactionPatterns: defaultRedactionPatterns,
			logger:            &NoOpLogger{},
		}

		input := `{"username":"admin","password":"secret123","enabled":true}`
		result := client.prepareJSONForLogging(input)

		// Should be redacted
		if strings.Contains(result, "secret123") {
			t.Error("Password should be redacted before pretty-printing")
		}

		// Should be formatted (multiline)
		if !strings.Contains(result, "\n") {
			t.Error("Output should be pretty-printed")
		}

		// Should contain redacted marker
		if !strings.Contains(result, "[REDACTED]") {
			t.Error("Output should contain [REDACTED] marker")
		}
	})

	t.Run("fallback on malformed JSON", func(t *testing.T) {
		client := &Client{
			prettyPrintLogs:   true,
			redactionPatterns: defaultRedactionPatterns,
			logger:            &NoOpLogger{},
		}

		malformed := `{"incomplete": "json"`
		result := client.prepareJSONForLogging(malformed)

		// Should return the input (after redaction attempt)
		if len(result) == 0 {
			t.Error("Should return fallback output for malformed JSON")
		}

		// Should not panic
		// (test passes if we reach here)
	})
}
