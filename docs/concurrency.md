# Concurrency Guide

This guide covers thread-safe operations and best practices for concurrent gNMI client usage.

## Table of Contents

- [Thread Safety Model](#thread-safety-model)
- [Concurrent Gets](#concurrent-gets)
- [Set Serialization](#set-serialization)
- [Best Practices](#best-practices)

## Thread Safety Model

go-gnmi uses RWMutex for safe concurrent access:

- **Read operations** (Get, Capabilities): Use `RLock` - can run concurrently
- **Write operations** (Set): Use `Lock` - serialized for consistency

## Concurrent Gets

Multiple goroutines can safely call Get concurrently:

```go
var wg sync.WaitGroup
ctx := context.Background()

// Launch 10 concurrent Get operations
for i := 0; i < 10; i++ {
    wg.Add(1)
    go func(id int) {
        defer wg.Done()
        
        paths := []string{
            fmt.Sprintf("/interfaces/interface[name=Gi0/0/0/%d]/state", id),
        }
        
        res, err := client.Get(ctx, paths)
        if err != nil {
            log.Printf("Get %d failed: %v", id, err)
            return
        }
        
        log.Printf("Get %d succeeded: %d notifications", id, len(res.Notifications))
    }(i)
}

wg.Wait()
```

## Set Serialization

Set operations are serialized with a mutex:

```go
var wg sync.WaitGroup
ctx := context.Background()

// Multiple Set operations are serialized automatically
for i := 0; i < 5; i++ {
    wg.Add(1)
    go func(id int) {
        defer wg.Done()

        body := gnmi.Body{}.
            Set("description", fmt.Sprintf("Interface %d", id)).
            Set("enabled", true)

        value, err := body.String()
        if err != nil {
            log.Printf("Body build failed: %v", err)
            return
        }

        ops := []gnmi.SetOperation{
            gnmi.Update(
                fmt.Sprintf("/interfaces/interface[name=Gi0/0/0/%d]/config", id),
                value,
                "json_ietf",
            ),
        }

        res, err := client.Set(ctx, ops)
        if err != nil {
            log.Printf("Set %d failed: %v", id, err)
            return
        }

        log.Printf("Set %d succeeded", id)
    }(i)
}

wg.Wait()
```

## Best Practices

### Read-Heavy Workloads

For workloads with many Gets and few Sets:

```go
// ✅ GOOD - Concurrent Gets
var wg sync.WaitGroup
for _, path := range manyPaths {
    wg.Add(1)
    go func(p string) {
        defer wg.Done()
        res, err := client.Get(ctx, []string{p})
        // Handle result...
    }(path)
}
wg.Wait()
```

### Write-Heavy Workloads

For workloads with many Sets, batch operations:

```go
// ✅ GOOD - Batch multiple changes in one Set
ops := []gnmi.SetOperation{}
for i := 0; i < 10; i++ {
    body := gnmi.Body{}.Set("description", fmt.Sprintf("IF %d", i))
    value, err := body.String()
    if err != nil {
        log.Printf("Body build failed: %v", err)
        continue
    }
    ops = append(ops, gnmi.Update(
        fmt.Sprintf("/interfaces/interface[name=Gi0/0/0/%d]/config", i),
        value,
        "json_ietf",
    ))
}
res, err := client.Set(ctx, ops)

// ❌ BAD - Multiple serialized Sets
for i := 0; i < 10; i++ {
    body := gnmi.Body{}.Set("description", fmt.Sprintf("IF %d", i))
    value, _ := body.String()  // Error ignored for brevity
    ops := []gnmi.SetOperation{gnmi.Update(path, value, "json_ietf")}
    client.Set(ctx, ops)  // Each blocks the next
}
```

### Context Cancellation

Use context for coordinated cancellation:

```go
ctx, cancel := context.WithCancel(context.Background())
defer cancel()

var wg sync.WaitGroup
for i := 0; i < 10; i++ {
    wg.Add(1)
    go func(id int) {
        defer wg.Done()
        
        select {
        case <-ctx.Done():
            log.Printf("Get %d cancelled", id)
            return
        default:
        }
        
        res, err := client.Get(ctx, paths)
        // Handle result...
    }(i)
}

// Cancel all operations
cancel()
wg.Wait()
```

### Error Handling

Each goroutine should handle errors independently:

```go
type Result struct {
    ID    int
    Data  *gnmi.GetRes
    Error error
}

results := make(chan Result, 10)

var wg sync.WaitGroup
for i := 0; i < 10; i++ {
    wg.Add(1)
    go func(id int) {
        defer wg.Done()
        
        res, err := client.Get(ctx, paths)
        results <- Result{ID: id, Data: res, Error: err}
    }(i)
}

go func() {
    wg.Wait()
    close(results)
}()

// Collect results
for result := range results {
    if result.Error != nil {
        log.Printf("Get %d failed: %v", result.ID, result.Error)
    } else {
        log.Printf("Get %d succeeded", result.ID)
    }
}
```

## See Also

- [Operations Guide](operations.md) - gNMI operations
- [Error Handling](error-handling.md) - Error handling strategies
