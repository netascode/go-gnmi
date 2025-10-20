# Error Handling Guide

This guide covers error handling strategies, retry logic, and best practices for robust gNMI client applications.

## Table of Contents

- [Error Types](#error-types)
- [Automatic Retry](#automatic-retry)
- [Error Inspection](#error-inspection)
- [Best Practices](#best-practices)

## Error Types

go-gnmi distinguishes between transient and permanent errors.

### Transient Errors

Transient errors are temporary failures that can be resolved by retrying:

- `Unavailable` - Service temporarily unavailable
- `ResourceExhausted` - Rate limiting or resource constraints
- `DeadlineExceeded` - Operation timeout (may succeed on retry)
- `Aborted` - Transaction aborted (can be retried)

These errors trigger automatic retry with exponential backoff.

### Permanent Errors

Permanent errors indicate problems that won't be resolved by retrying:

- `InvalidArgument` - Malformed request (invalid path, encoding, etc.)
- `NotFound` - Requested path doesn't exist
- `PermissionDenied` - Authentication or authorization failure
- `Unimplemented` - Operation not supported by device
- `Internal` - Internal server error (typically not retryable)

These errors fail immediately without retry.

## Automatic Retry

go-gnmi automatically retries transient errors with exponential backoff.

### Default Retry Configuration

```go
client, err := gnmi.NewClient(
    "device:57400",
    gnmi.Username("admin"),
    gnmi.Password("secret"),
    // Default values:
    // MaxRetries: 3 (total 4 attempts: initial + 3 retries)
    // BackoffMinDelay: 1 second
    // BackoffMaxDelay: 60 seconds
    // BackoffDelayFactor: 2.0 (exponential)
)
```

### Custom Retry Configuration

```go
client, err := gnmi.NewClient(
    "device:57400",
    gnmi.Username("admin"),
    gnmi.Password("secret"),
    gnmi.MaxRetries(5),                         // 6 total attempts
    gnmi.BackoffMinDelay(2*time.Second),        // Start at 2s
    gnmi.BackoffMaxDelay(120*time.Second),      // Cap at 2min
    gnmi.BackoffDelayFactor(2.0),               // Double each retry
)
```

### Retry Behavior

Backoff delays increase exponentially with jitter:

```
Attempt 1: No delay (initial)
Attempt 2: 1s + jitter
Attempt 3: 2s + jitter
Attempt 4: 4s + jitter
Attempt 5: 8s + jitter
Attempt 6: 16s + jitter (or max delay)
```

Jitter (0-10% random variation) prevents thundering herd problems.

## Error Inspection

### Basic Error Checking

```go
res, err := client.Get(ctx, paths)
if err != nil {
    log.Printf("Get failed: %v", err)
    return err
}

if !res.OK {
    log.Fatal("Operation returned not OK")
}
```

### Detailed Error Information

```go
res, err := client.Get(ctx, paths)
if err != nil {
    if gnmiErr, ok := err.(*gnmi.GnmiError); ok {
        log.Printf("Operation: %s", gnmiErr.Operation)
        log.Printf("Message: %s", gnmiErr.Message)
        log.Printf("Retries: %d", gnmiErr.Retries)
        log.Printf("Is Transient: %v", gnmiErr.IsTransient)
        
        for i, e := range gnmiErr.Errors {
            log.Printf("Error %d - Code: %d, Message: %s",
                i+1, e.Code, e.Message)
        }
    }
    return err
}
```

### Checking Error Types

```go
res, err := client.Get(ctx, paths)
if err != nil {
    if gnmiErr, ok := err.(*gnmi.GnmiError); ok {
        if gnmiErr.IsTransient {
            // Transient error - already retried
            log.Println("Service temporarily unavailable after retries")
        } else {
            // Permanent error - fix the request
            log.Println("Permanent error - check request validity")
        }
    }
}
```

## Best Practices

### Context Cancellation

Always use contexts for timeout and cancellation:

```go
// ✅ GOOD - With timeout
ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
defer cancel()
res, err := client.Get(ctx, paths)

// ✅ GOOD - With cancellation
ctx, cancel := context.WithCancel(context.Background())
defer cancel()

go func() {
    <-stopChan
    cancel()
}()
res, err := client.Get(ctx, paths)
```

### Retry Strategy

Configure retries based on operation criticality:

```go
// Critical operations - more retries
criticalClient, err := gnmi.NewClient(
    target,
    gnmi.MaxRetries(10),
    gnmi.BackoffMaxDelay(300*time.Second),
)

// Best-effort operations - fewer retries
bestEffortClient, err := gnmi.NewClient(
    target,
    gnmi.MaxRetries(1),
    gnmi.BackoffMaxDelay(10*time.Second),
)
```

### Graceful Degradation

Handle errors gracefully:

```go
res, err := client.Get(ctx, paths)
if err != nil {
    if gnmiErr, ok := err.(*gnmi.GnmiError); ok {
        if gnmiErr.IsTransient {
            // Use cached data or default values
            return cachedData, nil
        }
        // Permanent error - return error
        return nil, err
    }
}
```

### Error Logging

Log errors with sufficient context:

```go
res, err := client.Get(ctx, paths)
if err != nil {
    log.Printf("Get failed for paths %v: %v", paths, err)
    
    if gnmiErr, ok := err.(*gnmi.GnmiError); ok {
        log.Printf("After %d retries, transient=%v",
            gnmiErr.Retries, gnmiErr.IsTransient)
    }
    return err
}
```

## See Also

- [Operations Guide](operations.md) - Get, Set, and Capabilities operations
- [Logging](logging.md) - Configure structured logging
