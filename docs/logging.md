# Logging Guide

This guide covers logging configuration, log levels, and best practices for observability.

## Table of Contents

- [Logger Interface](#logger-interface)
- [Default Logger](#default-logger)
- [Log Levels](#log-levels)
- [Custom Loggers](#custom-loggers)
- [Security](#security)

## Logger Interface

go-gnmi uses a context-aware logger interface:

```go
type Logger interface {
    Debug(ctx context.Context, msg string, keysAndValues ...interface{})
    Info(ctx context.Context, msg string, keysAndValues ...interface{})
    Warn(ctx context.Context, msg string, keysAndValues ...interface{})
    Error(ctx context.Context, msg string, keysAndValues ...interface{})
}
```

All methods receive a `context.Context` as the first parameter, enabling integration with context-based logging frameworks and distributed tracing.

## Default Logger

### Basic Usage

```go
logger := gnmi.NewDefaultLogger(gnmi.LogLevelInfo)

client, err := gnmi.NewClient(
    "device:57400",
    gnmi.Username("admin"),
    gnmi.Password("secret"),
    gnmi.WithLogger(logger),
)
```

### Pretty Printing

JSON pretty printing is **enabled by default** for better readability in debug logs. You can disable it for performance when high-frequency operations are logged:

```go
// Pretty printing enabled by default
client, err := gnmi.NewClient(
    "device:57400",
    gnmi.Username("admin"),
    gnmi.Password("secret"),
    gnmi.WithLogger(logger),
)

// Disable for performance in high-frequency scenarios
client, err := gnmi.NewClient(
    "device:57400",
    gnmi.Username("admin"),
    gnmi.Password("secret"),
    gnmi.WithLogger(logger),
    gnmi.WithPrettyPrintLogs(false),  // Disable for performance
)
```

## Log Levels

### Available Levels

```go
gnmi.LogLevelDebug  // Most verbose
gnmi.LogLevelInfo   // Default
gnmi.LogLevelWarn   // Warnings only
gnmi.LogLevelError  // Errors only
```

### Debug Level

Logs all operations including request/response details:

```go
logger := gnmi.NewDefaultLogger(gnmi.LogLevelDebug)
// Output:
// DEBUG: Creating new client target=device:57400
// DEBUG: gNMI Get request target=device:57400 paths=2 encoding=json_ietf
// DEBUG: gNMI Get path index=0 path=/interfaces/interface[name=Gi0]/state
// DEBUG: gNMI Get path index=1 path=/system/config/hostname
// DEBUG: gNMI Get response target=device:57400 notifications=2
// DEBUG: gNMI Get notification index=0 timestamp=1234567890 updates=1 deletes=0
```

### Info Level

Logs operation starts and completions:

```go
logger := gnmi.NewDefaultLogger(gnmi.LogLevelInfo)
// Output:
// INFO: Client connected target=device:57400
// INFO: Get operation successful paths=1
```

### Warn Level

Logs warnings and errors:

```go
logger := gnmi.NewDefaultLogger(gnmi.LogLevelWarn)
// Output:
// WARN: Retry attempt=2 operation=Get error=Unavailable
// ERROR: Get operation failed after 3 retries
```

## Custom Loggers

### slog Adapter

```go
import (
    "context"
    "log/slog"
)

type SlogAdapter struct {
    logger *slog.Logger
}

func (s *SlogAdapter) Debug(ctx context.Context, msg string, keysAndValues ...interface{}) {
    s.logger.DebugContext(ctx, msg, keysAndValues...)
}

func (s *SlogAdapter) Info(ctx context.Context, msg string, keysAndValues ...interface{}) {
    s.logger.InfoContext(ctx, msg, keysAndValues...)
}

func (s *SlogAdapter) Warn(ctx context.Context, msg string, keysAndValues ...interface{}) {
    s.logger.WarnContext(ctx, msg, keysAndValues...)
}

func (s *SlogAdapter) Error(ctx context.Context, msg string, keysAndValues ...interface{}) {
    s.logger.ErrorContext(ctx, msg, keysAndValues...)
}

// Usage
client, err := gnmi.NewClient(
    "device:57400",
    gnmi.WithLogger(&SlogAdapter{logger: slog.Default()}),
)
```

### Zap Adapter

```go
import (
    "context"
    "go.uber.org/zap"
)

type ZapAdapter struct {
    logger *zap.SugaredLogger
}

func (z *ZapAdapter) Debug(ctx context.Context, msg string, keysAndValues ...interface{}) {
    z.logger.Debugw(msg, keysAndValues...)
}

func (z *ZapAdapter) Info(ctx context.Context, msg string, keysAndValues ...interface{}) {
    z.logger.Infow(msg, keysAndValues...)
}

func (z *ZapAdapter) Warn(ctx context.Context, msg string, keysAndValues ...interface{}) {
    z.logger.Warnw(msg, keysAndValues...)
}

func (z *ZapAdapter) Error(ctx context.Context, msg string, keysAndValues ...interface{}) {
    z.logger.Errorw(msg, keysAndValues...)
}

// Usage
zapLogger, _ := zap.NewProduction()
client, err := gnmi.NewClient(
    "device:57400",
    gnmi.WithLogger(&ZapAdapter{logger: zapLogger.Sugar()}),
)
```

**Note**: Zap doesn't have built-in context support, so the context is not used in this adapter. For context-aware logging with trace correlation, use slog or other context-aware logging frameworks.

### Context-Aware Logger (Trace Correlation)

Example logger that extracts trace IDs and request IDs from context:

```go
import (
    "context"
    "fmt"
    "time"
)

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

// Usage with trace context
type contextKey string
ctx := context.WithValue(context.Background(), contextKey("trace_id"), "trace-abc-123")
ctx = context.WithValue(ctx, contextKey("request_id"), "req-xyz-789")

client, err := gnmi.NewClient(
    "device:57400",
    gnmi.WithLogger(&ContextAwareLogger{prefix: "[TRACE]"}),
)

// Operations will log with trace_id and request_id
result, err := client.Get(ctx, []string{"/interfaces"})
// Output: [TRACE] [INFO] Get operation successful trace_id=trace-abc-123 request_id=req-xyz-789 paths=1
```

## Context Usage Patterns

### When to use context.Background()

Use `context.Background()` for internal utility methods that don't involve user requests:

- Validation and formatting operations
- Constructor methods (NewClient)
- Configuration processing
- Static operations

### When to propagate user context

Propagate the user's context for operations initiated by API calls:

- gNMI operations (Get, Set, Subscribe, Capabilities)
- Network operations (Connect, retry logic)
- Request/response processing
- Any operation that should be traceable

### Example: Distributed Tracing

```go
// Create context with trace ID
type contextKey string
traceIDKey := contextKey("trace_id")
ctx := context.WithValue(context.Background(), traceIDKey, "trace-abc-123")

// Pass context to operations
result, err := client.Get(ctx, []string{"/system/config"})

// Custom logger extracts trace ID and includes it in all log messages
// enabling correlation across services
```

## Security

### Automatic Redaction

go-gnmi automatically redacts sensitive data from logs:

```go
// Credentials never logged
client, err := gnmi.NewClient(
    "device:57400",
    gnmi.Username("admin"),
    gnmi.Password("super-secret"),  // Never appears in logs
    gnmi.WithLogger(logger),
)

// Sensitive JSON fields redacted
value := `{"password": "secret123", "config": {"enabled": true}}`
// Logged as: {"password": "***REDACTED***", "config": {"enabled": true}}
```

### Redacted Fields

The following fields are automatically redacted:
- password
- secret
- key
- token
- auth
- credential

### No-Op Logger

Disable all logging:

```go
// Default if no logger configured
client, err := gnmi.NewClient(
    "device:57400",
    // No WithLogger() = NoOpLogger used
)

// Or explicitly
client, err := gnmi.NewClient(
    "device:57400",
    gnmi.WithLogger(&gnmi.NoOpLogger{}),
)
```

## See Also

- [Error Handling](error-handling.md) - Error handling strategies
- [Operations Guide](operations.md) - gNMI operations
