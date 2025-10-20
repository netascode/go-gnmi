# Operations Guide

This guide covers all gNMI operations supported by go-gnmi: Get, Set, and Capabilities.

## Table of Contents

- [Get Operation](#get-operation)
- [Set Operation](#set-operation)
- [Capabilities Operation](#capabilities-operation)
- [Operation Modifiers](#operation-modifiers)
- [Best Practices](#best-practices)

## Get Operation

The Get operation retrieves configuration and state data from the device using gNMI paths.

### Basic Get

Retrieve data from one or more paths:

```go
ctx := context.Background()

paths := []string{
    "/interfaces/interface[name=GigabitEthernet0/0/0/0]/state",
}

res, err := client.Get(ctx, paths)
if err != nil {
    log.Fatal(err)
}

for _, notification := range res.Notifications {
    fmt.Printf("Timestamp: %d\n", notification.Timestamp)
    for _, update := range notification.Update {
        fmt.Printf("Path: %s\n", update.Path)
        fmt.Printf("Value: %v\n", update.Val)
    }
}
```

### Multiple Paths

Get data from multiple paths in a single request:

```go
paths := []string{
    "/interfaces/interface[name=Gi0/0/0/0]/config",
    "/interfaces/interface[name=Gi0/0/0/0]/state",
    "/system/config/hostname",
}

res, err := client.Get(ctx, paths)
```

### Get with Encoding

Specify the data encoding format:

```go
// JSON IETF (default, recommended)
res, err := client.Get(ctx, paths, gnmi.Encoding("json_ietf"))

// Standard JSON
res, err := client.Get(ctx, paths, gnmi.Encoding("json"))

// Protocol Buffer
res, err := client.Get(ctx, paths, gnmi.Encoding("proto"))
```

### Get with Timeout

Set a custom timeout for the operation:

```go
// 30 second timeout
res, err := client.Get(ctx, paths, gnmi.Timeout(30*time.Second))

// Combined with encoding
res, err := client.Get(ctx, paths,
    gnmi.Timeout(30*time.Second),
    gnmi.Encoding("json_ietf"),
)
```

## Set Operation

The Set operation modifies device configuration using Update, Replace, or Delete operations.

### Update Operation

Update adds or modifies configuration without removing existing sibling nodes:

```go
// Build JSON payload
body := gnmi.Body{}.
    Set("description", "WAN Interface").
    Set("enabled", true).
    Set("mtu", 9000)

value, err := body.String()
if err != nil {
    log.Fatal(err)
}

ops := []gnmi.SetOperation{
    gnmi.Update(
        "/interfaces/interface[name=Gi0/0/0/0]/config",
        value,
        "json_ietf",
    ),
}

res, err := client.Set(ctx, ops)
if err != nil {
    log.Fatal(err)
}
```

### Replace Operation

Replace removes existing data at the path before applying the new configuration:

```go
body := gnmi.Body{}.
    Set("config.description", "Replaced Interface").
    Set("config.mtu", 1500)

value, err := body.String()
if err != nil {
    log.Fatal(err)
}

ops := []gnmi.SetOperation{
    gnmi.Replace(
        "/interfaces/interface[name=Gi0/0/0/0]",
        value,
        "json_ietf",
    ),
}

res, err = client.Set(ctx, ops)
if err != nil {
    log.Fatal(err)
}
```

### Delete Operation

Delete removes configuration at the specified path:

```go
ops := []gnmi.SetOperation{
    gnmi.Delete("/interfaces/interface[name=Gi0/0/0/1]/config/description"),
}

res, err := client.Set(ctx, ops)
```

### Composite Set Operations

Combine multiple operations in a single atomic Set request:

```go
// Build JSON payloads
wanConfig, err := gnmi.Body{}.
    Set("description", "WAN").
    Set("enabled", true).String()
if err != nil {
    log.Fatal(err)
}

lanConfig, err := gnmi.Body{}.
    Set("description", "LAN").
    Set("mtu", 1500).String()
if err != nil {
    log.Fatal(err)
}

ops := []gnmi.SetOperation{
    // Update interface 0
    gnmi.Update(
        "/interfaces/interface[name=Gi0/0/0/0]/config",
        wanConfig,
        "json_ietf",
    ),

    // Replace interface 1
    gnmi.Replace(
        "/interfaces/interface[name=Gi0/0/0/1]/config",
        lanConfig,
        "json_ietf",
    ),
    
    // Delete interface 2
    gnmi.Delete("/interfaces/interface[name=Gi0/0/0/2]/config"),
}

res, err := client.Set(ctx, ops)
```

## Capabilities Operation

The Capabilities operation discovers the gNMI version, supported encodings, and YANG models.

### Basic Capabilities

Retrieve server capabilities:

```go
ctx := context.Background()

res, err := client.Capabilities(ctx)
if err != nil {
    log.Fatal(err)
}

fmt.Printf("gNMI Version: %s\n", res.Version)
fmt.Println("\nSupported Encodings:")
for _, cap := range res.Capabilities {
    fmt.Printf("  - %s\n", cap)
}

fmt.Println("\nSupported Models:")
for _, model := range res.Models {
    fmt.Printf("  - %s (version %s)\n", model.Name, model.Version)
}
```

### Checking Capabilities

Use helper methods to check if specific capabilities are supported:

```go
// Check if json_ietf encoding is supported
if client.HasCapability("json_ietf") {
    res, err := client.Get(ctx, paths, gnmi.Encoding("json_ietf"))
}

// Check if proto encoding is supported
if client.HasCapability("proto") {
    res, err := client.Get(ctx, paths, gnmi.Encoding("proto"))
}
```

## Operation Modifiers

Operation modifiers allow you to customize individual requests.

### Timeout Modifier

Override the client's default timeout for a specific operation:

```go
// Get with 60 second timeout
res, err := client.Get(ctx, paths, gnmi.Timeout(60*time.Second))

// Set with 2 minute timeout
res, err := client.Set(ctx, ops, gnmi.Timeout(2*time.Minute))

// Capabilities with custom timeout
res, err := client.Capabilities(ctx, gnmi.Timeout(10*time.Second))
```

### Encoding Modifier

Specify encoding for Get operations:

```go
// Use json_ietf encoding
res, err := client.Get(ctx, paths, gnmi.Encoding("json_ietf"))

// Use proto encoding for better performance
res, err := client.Get(ctx, paths, gnmi.Encoding("proto"))
```

### Combining Modifiers

Multiple modifiers can be combined:

```go
res, err := client.Get(ctx, paths,
    gnmi.Timeout(30*time.Second),
    gnmi.Encoding("json_ietf"),
)
```

## Best Practices

### Path Naming

Always use absolute paths starting with `/`:

```go
// ✅ GOOD
paths := []string{"/interfaces/interface[name=Gi0/0/0/0]/state"}

// ❌ BAD
paths := []string{"interfaces/interface[name=Gi0/0/0/0]/state"}
```

### Encoding Selection

Use `json_ietf` for YANG-modeled data (recommended):

```go
// ✅ GOOD - json_ietf for YANG models
res, err := client.Get(ctx, paths, gnmi.Encoding("json_ietf"))

// Use proto for performance-critical operations
res, err := client.Get(ctx, paths, gnmi.Encoding("proto"))
```

### Atomic Set Operations

Group related configuration changes in a single Set request:

```go
// ✅ GOOD - Atomic
ops := []gnmi.SetOperation{
    gnmi.Update("/interfaces/interface[name=Gi0]/config", value1, "json_ietf"),
    gnmi.Update("/interfaces/interface[name=Gi1]/config", value2, "json_ietf"),
}
res, err := client.Set(ctx, ops)

// ❌ BAD - Non-atomic, inconsistent state if second fails
client.Set(ctx, []gnmi.SetOperation{gnmi.Update("/interfaces/interface[name=Gi0]/config", value1, "json_ietf")})
client.Set(ctx, []gnmi.SetOperation{gnmi.Update("/interfaces/interface[name=Gi1]/config", value2, "json_ietf")})
```

### Error Checking

Always check operation results:

```go
res, err := client.Get(ctx, paths)
if err != nil {
    // Handle error
    return err
}

if !res.OK {
    // Operation failed
    log.Fatal("Operation returned not OK")
}
```

### Resource Cleanup

Always close the client when done:

```go
client, err := gnmi.NewClient(target, options...)
if err != nil {
    log.Fatal(err)
}
defer client.Close()  // Important!
```

### Context Usage

Use context for cancellation and timeouts:

```go
// With timeout
ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
defer cancel()
res, err := client.Get(ctx, paths)

// With cancellation
ctx, cancel := context.WithCancel(context.Background())
defer cancel()

// Cancel from another goroutine if needed
go func() {
    <-stopChan
    cancel()
}()

res, err := client.Get(ctx, paths)
```

## Performance Considerations

### Batch Gets

Retrieve multiple paths in a single Get request:

```go
// ✅ GOOD - Single request
paths := []string{
    "/interfaces/interface[name=Gi0]/state",
    "/interfaces/interface[name=Gi1]/state",
    "/interfaces/interface[name=Gi2]/state",
}
res, err := client.Get(ctx, paths)

// ❌ BAD - Multiple requests
for _, path := range paths {
    res, err := client.Get(ctx, []string{path})
}
```

### Encoding Performance

Use `proto` encoding for large datasets:

```go
// For large responses, proto is more efficient
res, err := client.Get(ctx, paths, gnmi.Encoding("proto"))
```

### Timeout Tuning

Set appropriate timeouts based on operation complexity:

```go
// Simple Get - short timeout
res, err := client.Get(ctx, paths, gnmi.Timeout(10*time.Second))

// Complex Set with validation - longer timeout
res, err := client.Set(ctx, ops, gnmi.Timeout(60*time.Second))
```

## See Also

- [Error Handling](error-handling.md) - Comprehensive error handling strategies
- [gNMI Paths](paths.md) - gNMI path specification and syntax
- [Concurrency](concurrency.md) - Thread-safe concurrent operations
