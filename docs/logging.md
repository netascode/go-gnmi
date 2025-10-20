# Logging Guide

This guide covers logging configuration, log levels, and best practices for observability.

## Table of Contents

- [Logger Interface](#logger-interface)
- [Default Logger](#default-logger)
- [Log Levels](#log-levels)
- [Custom Loggers](#custom-loggers)
- [Security](#security)

## Logger Interface

go-gnmi uses a simple logger interface:

```go
type Logger interface {
    Debug(msg string, keysAndValues ...interface{})
    Info(msg string, keysAndValues ...interface{})
    Warn(msg string, keysAndValues ...interface{})
    Error(msg string, keysAndValues ...interface{})
}
```

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

### With Pretty Printing

Enable JSON pretty printing for better readability:

```go
client, err := gnmi.NewClient(
    "device:57400",
    gnmi.Username("admin"),
    gnmi.Password("secret"),
    gnmi.WithLogger(logger),
    gnmi.WithPrettyPrintLogs(true),
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
// DEBUG: Get request paths=[/interfaces/interface[name=Gi0]/state]
// DEBUG: Get response OK=true notifications=1
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
import "log/slog"

type SlogAdapter struct {
    logger *slog.Logger
}

func (s *SlogAdapter) Debug(msg string, keysAndValues ...interface{}) {
    s.logger.Debug(msg, keysAndValues...)
}

func (s *SlogAdapter) Info(msg string, keysAndValues ...interface{}) {
    s.logger.Info(msg, keysAndValues...)
}

func (s *SlogAdapter) Warn(msg string, keysAndValues ...interface{}) {
    s.logger.Warn(msg, keysAndValues...)
}

func (s *SlogAdapter) Error(msg string, keysAndValues ...interface{}) {
    s.logger.Error(msg, keysAndValues...)
}

// Usage
client, err := gnmi.NewClient(
    "device:57400",
    gnmi.WithLogger(&SlogAdapter{logger: slog.Default()}),
)
```

### Zap Adapter

```go
import "go.uber.org/zap"

type ZapAdapter struct {
    logger *zap.SugaredLogger
}

func (z *ZapAdapter) Debug(msg string, keysAndValues ...interface{}) {
    z.logger.Debugw(msg, keysAndValues...)
}

func (z *ZapAdapter) Info(msg string, keysAndValues ...interface{}) {
    z.logger.Infow(msg, keysAndValues...)
}

func (z *ZapAdapter) Warn(msg string, keysAndValues ...interface{}) {
    z.logger.Warnw(msg, keysAndValues...)
}

func (z *ZapAdapter) Error(msg string, keysAndValues ...interface{}) {
    z.logger.Errorw(msg, keysAndValues...)
}

// Usage
zapLogger, _ := zap.NewProduction()
client, err := gnmi.NewClient(
    "device:57400",
    gnmi.WithLogger(&ZapAdapter{logger: zapLogger.Sugar()}),
)
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
