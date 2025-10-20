# gNMI Paths Guide

This guide covers gNMI path specification, syntax, and best practices.

## Table of Contents

- [Path Syntax](#path-syntax)
- [Path Elements](#path-elements)
- [Wildcards](#wildcards)
- [Best Practices](#best-practices)

## Path Syntax

gNMI paths follow a hierarchical structure starting with `/`:

```go
// Simple path
"/interfaces"

// Nested path
"/interfaces/interface/config"

// Path with key (list entry selection)
"/interfaces/interface[name=GigabitEthernet0/0/0/0]/state"

// Complex nested path
"/interfaces/interface[name=Gi0/0/0/0]/subinterfaces/subinterface[index=0]/config"
```

## Path Elements

### Container Paths

Paths to YANG containers:

```go
paths := []string{
    "/system",
    "/interfaces",
    "/network-instances",
}
```

### Leaf Paths

Paths to specific configuration leaves:

```go
paths := []string{
    "/system/config/hostname",
    "/interfaces/interface[name=Gi0/0/0/0]/config/description",
    "/interfaces/interface[name=Gi0/0/0/0]/config/mtu",
}
```

### List Entry Paths

Paths using keys to select list entries:

```go
// Single key
"/interfaces/interface[name=GigabitEthernet0/0/0/0]"

// Multiple keys
"/network-instances/network-instance[name=default]/protocols/protocol[identifier=BGP][name=65000]"
```

### Wildcard Paths

Use `*` to match any value:

```go
// All interfaces
"/interfaces/interface[name=*]/state"

// All BGP neighbors
"/network-instances/network-instance[name=default]/protocols/protocol[identifier=BGP]/bgp/neighbors/neighbor[neighbor-address=*]/state"
```

## Wildcards

### Single Wildcard

Match any single element:

```go
// Get state from all interfaces
paths := []string{
    "/interfaces/interface[name=*]/state",
}
```

### Partial Wildcards

Wildcards can be partial matches:

```go
// All GigabitEthernet interfaces
"/interfaces/interface[name=GigabitEthernet*]/state"

// All interfaces on module 0
"/interfaces/interface[name=GigabitEthernet0/0/0/*]/state"
```

### Multiple Wildcards

Multiple wildcards in a path:

```go
// All subinterfaces on all interfaces
"/interfaces/interface[name=*]/subinterfaces/subinterface[index=*]/state"
```

## Best Practices

### Use Specific Paths

When possible, use specific paths instead of wildcards:

```go
// ✅ GOOD - Specific path
"/interfaces/interface[name=GigabitEthernet0/0/0/0]/state"

// ❌ BAD - Unnecessary wildcard
"/interfaces/interface[name=*]/state"  // If you only need one interface
```

### Batch Related Paths

Group related paths in a single Get:

```go
// ✅ GOOD - Single request
paths := []string{
    "/interfaces/interface[name=Gi0/0/0/0]/config",
    "/interfaces/interface[name=Gi0/0/0/0]/state",
    "/interfaces/interface[name=Gi0/0/0/0]/statistics",
}
res, err := client.Get(ctx, paths)

// ❌ BAD - Multiple requests
client.Get(ctx, []string{"/interfaces/interface[name=Gi0/0/0/0]/config"})
client.Get(ctx, []string{"/interfaces/interface[name=Gi0/0/0/0]/state"})
client.Get(ctx, []string{"/interfaces/interface[name=Gi0/0/0/0]/statistics"})
```

### Path Validation

Paths are validated by go-gnmi:

```go
// ❌ Invalid - no leading slash
paths := []string{"interfaces"}  // Error: must start with '/'

// ❌ Invalid - path traversal
paths := []string{"/system/../secret"}  // Error: traversal pattern detected

// ❌ Invalid - too long
paths := []string{"/" + strings.Repeat("a", 2000)}  // Error: exceeds maximum length

// ✅ Valid
paths := []string{"/interfaces/interface[name=Gi0/0/0/0]/state"}
```

### Escaping Special Characters

Special characters in keys should be properly formatted:

```go
// Interface names with slashes
"/interfaces/interface[name=GigabitEthernet0/0/0/0]/state"

// Interface names with special characters
"/interfaces/interface[name=Gi0/0/0/0.100]/state"
```

### State vs Config

Distinguish between config and state paths:

```go
// Configuration (read-write)
"/interfaces/interface[name=Gi0/0/0/0]/config"

// Operational state (read-only)
"/interfaces/interface[name=Gi0/0/0/0]/state"

// Both config and state
paths := []string{
    "/interfaces/interface[name=Gi0/0/0/0]/config",
    "/interfaces/interface[name=Gi0/0/0/0]/state",
}
```

## Common Path Patterns

### Interface Operations

```go
// Single interface state
"/interfaces/interface[name=GigabitEthernet0/0/0/0]/state"

// All interfaces state
"/interfaces/interface[name=*]/state"

// Interface configuration
"/interfaces/interface[name=GigabitEthernet0/0/0/0]/config"

// Interface statistics
"/interfaces/interface[name=GigabitEthernet0/0/0/0]/state/counters"
```

### System Operations

```go
// Hostname
"/system/config/hostname"

// System time
"/system/state/current-datetime"

// System information
"/system/state"
```

### Network Instance Operations

```go
// Default VRF
"/network-instances/network-instance[name=default]"

// BGP config
"/network-instances/network-instance[name=default]/protocols/protocol[identifier=BGP]/bgp"

// BGP neighbor state
"/network-instances/network-instance[name=default]/protocols/protocol[identifier=BGP]/bgp/neighbors/neighbor[neighbor-address=192.168.1.1]/state"
```

## See Also

- [Operations Guide](operations.md) - Get, Set, and Capabilities operations
- [Quick Start](quickstart.md) - Getting started with go-gnmi
- [OpenConfig Models](https://github.com/openconfig/public) - YANG model reference
