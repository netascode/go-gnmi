# Quick Start Guide

This guide will help you get started with go-gnmi in minutes.

## Installation

Install go-gnmi using `go get`:

```bash
go get github.com/netascode/go-gnmi
```

## Requirements

- Go 1.24 or later
- Network device with gNMI support
- gRPC/gNMI access to the device (port 57400 by default)
- Device must have gNMI enabled and configured

## Your First gNMI Connection

Let's create a simple program that connects to a gNMI device and retrieves its capabilities:

```go
package main

import (
    "context"
    "fmt"
    "log"

    "github.com/netascode/go-gnmi"
)

func main() {
    // Create a gNMI client (lazy connection - doesn't connect yet)
    client, err := gnmi.NewClient(
        "192.168.1.1:57400",  // Device hostname:port
        gnmi.Username("admin"),
        gnmi.Password("secret"),
        gnmi.TLS(true),  // Enable TLS (default)
        gnmi.VerifyCertificate(false),  // For testing only
    )
    if err != nil {
        log.Fatalf("Client creation failed: %v", err)  // Configuration error
    }
    defer client.Close()

    ctx := context.Background()

    // Optional: Verify connection explicitly before proceeding
    // if err := client.Ping(ctx); err != nil {
    //     log.Fatalf("Connection failed: %v", err)
    // }

    // Get server capabilities (connection established automatically on first use)
    capRes, err := client.Capabilities(ctx)
    if err != nil {
        log.Fatalf("Capabilities failed: %v", err)
    }

    // Print gNMI version and supported encodings
    fmt.Printf("gNMI Version: %s\n", capRes.Version)
    fmt.Println("\nSupported Encodings:")
    for _, cap := range capRes.Capabilities {
        fmt.Printf("  - %s\n", cap)
    }
}
```

## Basic Get Operation

Retrieve configuration or state data using gNMI paths:

```go
package main

import (
    "context"
    "fmt"
    "log"

    "github.com/netascode/go-gnmi"
)

func main() {
    // Connect to device
    client, err := gnmi.NewClient(
        "192.168.1.1:57400",
        gnmi.Username("admin"),
        gnmi.Password("secret"),
        gnmi.TLS(true),
        gnmi.VerifyCertificate(false),
    )
    if err != nil {
        log.Fatal(err)
    }
    defer client.Close()

    ctx := context.Background()

    // Get interface state using gNMI path
    paths := []string{
        "/interfaces/interface[name=GigabitEthernet0/0/0/0]/state",
    }

    res, err := client.Get(ctx, paths)
    if err != nil {
        log.Fatalf("Get failed: %v", err)
    }

    // Check if operation succeeded
    if !res.OK {
        log.Fatal("Operation did not return OK")
    }

    // Print response
    fmt.Printf("Retrieved %d notifications\n", len(res.Notifications))
    for _, notification := range res.Notifications {
        for _, update := range notification.Update {
            fmt.Printf("Path: %s\n", update.Path)
            fmt.Printf("Value: %v\n", update.Val)
        }
    }
}
```

## Basic Set Operation

Update device configuration using gNMI Set:

```go
package main

import (
    "context"
    "fmt"
    "log"

    "github.com/netascode/go-gnmi"
)

func main() {
    // Connect to device
    client, err := gnmi.NewClient(
        "192.168.1.1:57400",
        gnmi.Username("admin"),
        gnmi.Password("secret"),
        gnmi.TLS(true),
        gnmi.VerifyCertificate(false),
    )
    if err != nil {
        log.Fatal(err)
    }
    defer client.Close()

    ctx := context.Background()

    // Build JSON configuration using Body builder
    body := gnmi.Body{}.
        Set("description", "WAN Interface").
        Set("enabled", true).
        Set("mtu", 9000)

    value, err := body.String()
    if err != nil {
        log.Fatalf("Body build failed: %v", err)
    }

    // Create Set operation
    ops := []gnmi.SetOperation{
        gnmi.Update(
            "/interfaces/interface[name=GigabitEthernet0/0/0/0]/config",
            value,
            "json_ietf",  // Encoding
        ),
    }

    res, err = client.Set(ctx, ops)
    if err != nil {
        log.Fatalf("Set failed: %v", err)
    }

    if !res.OK {
        log.Fatal("Set operation did not return OK")
    }

    fmt.Println("Configuration updated successfully!")
}
```

## Using the Body Builder

The Body builder provides a fluent API for constructing JSON payloads:

```go
// Simple configuration
body := gnmi.Body{}.
    Set("config.description", "LAN Interface").
    Set("config.enabled", true)

value, err := body.String()
if err != nil {
    log.Fatal(err)
}

// Complex nested configuration
body = gnmi.Body{}.
    Set("config.name", "GigabitEthernet0/0/0/0").
    Set("config.type", "ethernetCsmacd").
    Set("config.enabled", true).
    Set("config.mtu", 9000).
    Set("config.ipv4.address", "192.168.1.1").
    Set("config.ipv4.prefix-length", 24).
    Set("config.ipv4.enabled", true)

value, err = body.String()
if err != nil {
    log.Fatal(err)
}

// Use in Set operation
ops := []gnmi.SetOperation{
    gnmi.Update("/interfaces/interface[name=Gi0/0/0/0]", value),
}
```

## Working with Encodings

gNMI supports multiple data encodings. The most common are:

```go
// JSON IETF (default, recommended)
res, err := client.Get(ctx, paths, gnmi.GetEncoding("json_ietf"))

// Standard JSON
res, err := client.Get(ctx, paths, gnmi.GetEncoding("json"))

// Protocol Buffer (binary)
res, err := client.Get(ctx, paths, gnmi.GetEncoding("proto"))
```

The library defaults to `json_ietf` which follows IETF JSON encoding conventions for YANG models.

## Error Handling

go-gnmi provides detailed error information:

```go
res, err := client.Get(ctx, paths)
if err != nil {
    // Type assertion for detailed error info
    if gnmiErr, ok := err.(*gnmi.GnmiError); ok {
        fmt.Printf("Operation: %s\n", gnmiErr.Operation)
        fmt.Printf("Retries: %d\n", gnmiErr.Retries)
        fmt.Printf("Transient: %v\n", gnmiErr.IsTransient)

        for _, e := range gnmiErr.Errors {
            fmt.Printf("gRPC Code %d: %s\n", e.Code, e.Message)
        }
    }
    return err
}
```

Transient errors (network issues, service unavailable) are automatically retried with exponential backoff.

## Connection Behavior

### Lazy Connection

The client uses **lazy connection** pattern - `NewClient()` does NOT establish a physical connection immediately:

```go
// NewClient returns immediately (validates config only, no connection)
client, err := gnmi.NewClient(target, opts...)
if err != nil {
    log.Fatal(err)  // Configuration error, not connection error
}
defer client.Close()

// Connection happens automatically on first RPC call
res, err := client.Get(ctx, paths)  // Connects here if needed
```

### Explicit Connection Verification

Use `Ping()` to verify connectivity before performing operations:

```go
client, err := gnmi.NewClient(target, opts...)
if err != nil {
    log.Fatal(err)  // Configuration error
}
defer client.Close()

// Verify connection works before proceeding
if err := client.Ping(ctx); err != nil {
    log.Fatal(err)  // Connection error - device unreachable
}

// Now confident the connection works
res, err := client.Get(ctx, paths)
```

When to use `Ping()`:
- **Testing/debugging** - Verify device is reachable
- **Critical operations** - Ensure connectivity before important work
- **Health checks** - Verify connection in monitoring systems

For most use cases, `Ping()` is optional - operations will fail gracefully if connection can't be established.

## Next Steps

- [Operations Guide](operations.md) - Detailed coverage of Get, Set, and Capabilities operations
- [Error Handling](error-handling.md) - Comprehensive error handling and retry strategies
- [Logging](logging.md) - Configure structured logging
- [Concurrency](concurrency.md) - Thread-safe operations and best practices
- [gNMI Paths](paths.md) - Working with gNMI path specifications

## Common Issues

### Connection Refused

If you get "connection refused" errors:

1. Verify gNMI is enabled on the device
2. Check the port (default 57400)
3. Verify firewall rules allow gRPC connections
4. Ensure TLS is properly configured

### Certificate Verification Errors

For production, always use proper TLS certificates. For testing:

```go
client, err := gnmi.NewClient(
    "device:57400",
    gnmi.Username("admin"),
    gnmi.Password("secret"),
    gnmi.VerifyCertificate(false),  // Only for testing!
)
```

### Authentication Failures

Ensure credentials are correct and the user has appropriate permissions for gNMI operations.

## Examples

Complete working examples are available in the [examples/](../examples/) directory:

- `examples/basic/` - Basic Get and Set operations
- `examples/concurrent/` - Thread-safe concurrent operations
- `examples/capabilities/` - Capability discovery
- `examples/logging/` - Logging configuration

## Additional Resources

- [gNMI Specification](https://github.com/openconfig/reference/blob/master/rpc/gnmi/gnmi-specification.md)
- [gNMI Protocol](https://github.com/openconfig/gnmi/blob/master/proto/gnmi/gnmi.proto)
- [OpenConfig Models](https://github.com/openconfig/public)
