<h1 align="center">go-gnmi</h1>

<p align="center">
<a href="https://godoc.org/github.com/netascode/go-gnmi"><img src="https://img.shields.io/badge/api-reference-blue.svg?style=flat-square" alt="GoDoc"></a>
<a href="https://goreportcard.com/report/github.com/netascode/go-gnmi"><img src="https://goreportcard.com/badge/github.com/netascode/go-gnmi?style=flat-square" alt="Go Report Card"></a>
<a href="https://github.com/netascode/go-gnmi/actions/workflows/ci.yml"><img src="https://img.shields.io/github/actions/workflow/status/netascode/go-gnmi/ci.yml?branch=main&style=flat-square&label=build" alt="CI"></a>
<a href="https://codecov.io/gh/netascode/go-gnmi"><img src="https://codecov.io/gh/netascode/go-gnmi/branch/main/graph/badge.svg?style=flat-square" alt="codecov"></a>
<a href="https://github.com/netascode/go-gnmi/blob/main/LICENSE"><img src="https://img.shields.io/badge/license-MPL--2.0-blue.svg?style=flat-square" alt="License"></a>
</p>

<h3 align="center">A simple, fluent Go client library for interacting with network devices using the gNMI protocol (gRPC Network Management Interface).</h3>

## Features

- **Simple API**: Fluent, chainable API design
- **Lazy Connection**: Non-blocking client initialization with automatic connection on first use
- **JSON Manipulation**: Path-based JSON operations using [gjson](https://github.com/tidwall/gjson) and [sjson](https://github.com/tidwall/sjson)
- **Complete gNMI Support**: Get, Set, and Capabilities operations
- **Robust Transport**: Built on [gnmic](https://github.com/openconfig/gnmic) for reliable gRPC connectivity and gNMI protocol handling
- **Automatic Retry**: Built-in retry logic with exponential backoff for transient errors
- **Thread-Safe**: Concurrent read operations with synchronized write operations
- **Capability Discovery**: Automatic capability negotiation and checking
- **Structured Logging**: Configurable logging with automatic sensitive data redaction
- **TLS Security**: TLS by default with certificate verification

## Installation

```bash
go get github.com/netascode/go-gnmi
```

## Requirements

- Go 1.24 or later
- Network device with gNMI support

## Quick Start

```go
package main

import (
    "context"
    "fmt"
    "log"

    "github.com/netascode/go-gnmi"
)

func main() {
    // Create client
    client, err := gnmi.NewClient(
        "192.168.1.1:57400",
        gnmi.Username("admin"),
        gnmi.Password("secret"),
        gnmi.VerifyCertificate(false), // Use true in production
    )
    if err != nil {
        log.Fatal(err)
    }
    defer client.Close()

    // Get operation
    ctx := context.Background()
    res, err := client.Get(ctx, []string{"/interfaces/interface[name=GigabitEthernet0/0/0/0]/state"})
    if err != nil {
        log.Fatal(err)
    }

    // Parse response using gjson
    ifState := res.GetValue("notification.0.update.0.val").String()
    fmt.Println("Interface state:", ifState)
}
```

## Usage

### Client Creation

The client uses **lazy connection** - `NewClient()` validates configuration but doesn't establish a physical connection. The connection happens automatically on first use:

```go
// Creates client without connecting (validates config only)
client, err := gnmi.NewClient(
    "device.example.com:57400",
    gnmi.Username("admin"),
    gnmi.Password("secret"),
    gnmi.TLS(true),
    gnmi.VerifyCertificate(true),
    gnmi.TLSCA("/path/to/ca.crt"),
    gnmi.MaxRetries(5),
    gnmi.OperationTimeout(120*time.Second),
)
if err != nil {
    log.Fatal(err)  // Configuration error
}
defer client.Close()

// Optional: Verify connection before operations
if err := client.Ping(ctx); err != nil {
    log.Fatal(err)  // Connection error
}
```

### Get Operations

Retrieve configuration and state data:

```go
ctx := context.Background()

// Get with paths
paths := []string{
    "/interfaces/interface[name=GigabitEthernet0/0/0/0]/config",
    "/interfaces/interface[name=GigabitEthernet0/0/0/0]/state",
}

res, err := client.Get(ctx, paths)
if err != nil {
    log.Fatal(err)
}

// Get with encoding
res, err := client.Get(ctx, paths, gnmi.GetEncoding("json_ietf"))
```

### Set Operations

Update, replace, or delete configuration:

```go
ctx := context.Background()

// Build JSON payload using Body builder
value, err := gnmi.Body{}.
    Set("config.description", "WAN Interface").
    Set("config.enabled", true).
    Set("config.mtu", 9000).String()
if err != nil {
    log.Fatal(err)
}

// Create set operations
ops := []gnmi.SetOperation{
    gnmi.Update("/interfaces/interface[name=Gi0/0/0/0]/config", value),
    gnmi.Delete("/interfaces/interface[name=Gi0/0/0/1]/config"),
}

res, err = client.Set(ctx, ops)
if err != nil {
    log.Fatal(err)
}
```

### Error Handling

The library automatically retries transient errors (service unavailable, rate limiting, timeout) with exponential backoff:

```go
client, err := gnmi.NewClient(
    "192.168.1.1:57400",
    gnmi.Username("admin"),
    gnmi.Password("secret"),
    gnmi.MaxRetries(3),                         // Maximum retry attempts
    gnmi.BackoffMinDelay(1*time.Second),        // Minimum 1 second delay
    gnmi.BackoffMaxDelay(60*time.Second),       // Maximum 60 second delay
    gnmi.BackoffDelayFactor(2.0),               // Exponential factor
)

// Detailed error information
res, err := client.Get(ctx, paths)
if err != nil {
    if gnmiErr, ok := err.(*gnmi.GnmiError); ok {
        log.Printf("Operation: %s", gnmiErr.Operation)
        log.Printf("Retries: %d", gnmiErr.Retries)
        log.Printf("Transient: %v", gnmiErr.IsTransient)
        for _, e := range gnmiErr.Errors {
            log.Printf("  Code %d: %s", e.Code, e.Message)
        }
    }
}
```

### Capability Checking

```go
// Check if server supports specific encoding
if client.HasCapability("json_ietf") {
    // Use json_ietf encoding
    res, err := client.Get(ctx, paths, gnmi.GetEncoding("json_ietf"))
}

// Get all server capabilities
caps := client.ServerCapabilities()
for _, cap := range caps {
    fmt.Println(cap)
}
```

### Body Builder

Build JSON payloads with a fluent API:

```go
value, err := gnmi.Body{}.
    Set("config.name", "GigabitEthernet0/0/0/0").
    Set("config.description", "WAN Interface").
    Set("config.enabled", true).
    Set("config.mtu", 9000).
    Set("config.ipv4.address", "192.168.1.1").
    Set("config.ipv4.prefix-length", 24).
    String()
if err != nil {
    log.Fatal(err)
}

ops := []gnmi.SetOperation{
    gnmi.Update("/interfaces/interface[name=Gi0/0/0/0]", value),
}
```

## Supported Operations

| Operation | Description |
|-----------|-------------|
| Get | Retrieve configuration and state data from device |
| Set | Update, replace, or delete configuration (supports atomic operations) |
| Capabilities | Discover supported encodings, models, and gNMI version |

## Security

### TLS Configuration

The library uses TLS by default with certificate verification enabled. For production:

```go
client, err := gnmi.NewClient(
    "device.example.com:57400",
    gnmi.Username("admin"),
    gnmi.Password("secret"),
    gnmi.TLS(true),                      // Enable TLS (default: true)
    gnmi.VerifyCertificate(true),        // Verify certificates (default: true)
    gnmi.TLSCA("/path/to/ca.crt"),       // Specify CA certificate
    gnmi.TLSCert("/path/to/client.crt"), // Client certificate (optional)
    gnmi.TLSKey("/path/to/client.key"),  // Client key (optional)
)
```

**⚠️ WARNING**: Disabling TLS or certificate verification makes connections vulnerable to eavesdropping and Man-in-the-Middle attacks. Only use `VerifyCertificate(false)` in isolated testing environments.

## Documentation

- [GoDoc](https://pkg.go.dev/github.com/netascode/go-gnmi)
- [gNMI Specification](https://github.com/openconfig/reference/blob/master/rpc/gnmi/gnmi-specification.md)
- [gNMI Protocol Buffers](https://github.com/openconfig/gnmi/blob/master/proto/gnmi/gnmi.proto)

## Examples

See the [examples](examples/) directory for library usage examples:

- **basic** - Client creation, Get, Set operations, response parsing
- **concurrent** - Thread-safe concurrent operations
- **capabilities** - Capability discovery and checking
- **logging** - Logger configuration and log levels

## Testing

```bash
# Run tests
make test

# Run tests with coverage
make coverage

# Run linters
make lint

# Run all checks
make verify
```

## Contributing

Contributions are welcome! Please see [CONTRIBUTING.md](CONTRIBUTING.md) for guidelines.

## Acknowledgments

This library is built on top of [gnmic](https://github.com/openconfig/gnmic) for the gNMI transport layer. Gnmic provides robust gRPC connectivity and gNMI protocol handling, allowing go-gnmi to focus on providing a simple, idiomatic Go API.

## License

Mozilla Public License Version 2.0 - see [LICENSE](LICENSE) for details.
