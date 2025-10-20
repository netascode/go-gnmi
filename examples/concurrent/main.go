//nolint:errcheck,gosec // Example code prioritizes readability over error handling

// Package main demonstrates concurrent operations with go-gnmi.
//
// This example shows:
//   - Thread-safe concurrent read operations (Get)
//   - Serialized write operations (Set)
//   - Proper goroutine synchronization with WaitGroup
//   - Error handling in concurrent operations
//   - Performance timing and measurements
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
	"sync"
	"time"

	"github.com/netascode/go-gnmi"
)

func main() {
	// Load credentials from environment
	target := getEnv("GNMI_TARGET", "192.168.1.1:57400")
	username := getEnv("GNMI_USERNAME", "admin")
	password := getEnv("GNMI_PASSWORD", "secret")

	// Create client (thread-safe for concurrent reads)
	client, err := gnmi.NewClient(
		target,
		gnmi.Username(username),
		gnmi.Password(password),
		gnmi.TLS(true),
		gnmi.VerifyCertificate(false), // WARNING: Disables TLS verification - TESTING ONLY
		gnmi.OperationTimeout(30*time.Second),
	)
	if err != nil {
		log.Fatalf("NewClient failed: %v", err)
	}
	defer client.Close() //nolint:errcheck // Error intentionally ignored in example

	fmt.Printf("Connected to %s\n", target)

	ctx := context.Background()

	// Example 1: Basic concurrent Get operations
	fmt.Println("\n=== Concurrent Get Operations ===")
	concurrentGets(ctx, client)

	// Example 2: Concurrent reads with results collection
	fmt.Println("\n=== Concurrent Gets with Results ===")
	concurrentGetsWithResults(ctx, client)

	// Example 3: Serialized Set operations
	fmt.Println("\n=== Serialized Set Operations ===")
	serializedSets(ctx, client)

	// Example 4: Mixed concurrent Gets and serialized Sets
	fmt.Println("\n=== Mixed Operations (Concurrent Gets + Serialized Sets) ===")
	mixedOperations(ctx, client)

	fmt.Println("\n=== Examples Complete ===")
}

// concurrentGets demonstrates thread-safe concurrent Get operations
func concurrentGets(ctx context.Context, client *gnmi.Client) {
	// Multiple paths to query concurrently
	paths := [][]string{
		{"/interfaces/interface[name=GigabitEthernet0/0/0/0]/config"},
		{"/interfaces/interface[name=GigabitEthernet0/0/0/1]/config"},
		{"/system/config/hostname"},
		{"/system/config/domain-name"},
	}

	start := time.Now()
	var wg sync.WaitGroup

	// Launch concurrent Get operations (these can run in parallel)
	for i, pathList := range paths {
		wg.Add(1)
		go func(index int, p []string) {
			defer wg.Done()

			opStart := time.Now()
			_, err := client.Get(ctx, p)
			elapsed := time.Since(opStart)

			if err != nil {
				fmt.Printf("  Operation %d failed: %v (elapsed: %v)\n", index+1, err, elapsed)
			} else {
				fmt.Printf("  Operation %d completed (elapsed: %v)\n", index+1, elapsed)
			}
		}(i, pathList)
	}

	wg.Wait()
	fmt.Printf("All Get operations completed in %v\n", time.Since(start))
	fmt.Println("Note: Multiple Gets can run concurrently (RLock allows this)")
}

// concurrentGetsWithResults demonstrates collecting results from concurrent operations
func concurrentGetsWithResults(ctx context.Context, client *gnmi.Client) {
	type result struct {
		name  string
		path  string
		count int
		err   error
	}

	results := make(chan result, 4)
	var wg sync.WaitGroup

	queries := []struct {
		name string
		path []string
	}{
		{"Hostname", []string{"/system/config/hostname"}},
		{"Domain Name", []string{"/system/config/domain-name"}},
		{"Interface0 Config", []string{"/interfaces/interface[name=GigabitEthernet0/0/0/0]/config"}},
		{"Interface1 Config", []string{"/interfaces/interface[name=GigabitEthernet0/0/0/1]/config"}},
	}

	for _, query := range queries {
		wg.Add(1)
		go func(q struct {
			name string
			path []string
		}) {
			defer wg.Done()

			res, err := client.Get(ctx, q.path)

			r := result{name: q.name, path: q.path[0], err: err}
			if err == nil {
				r.count = len(res.Notifications)
			}

			results <- r
		}(query)
	}

	wg.Wait()
	close(results)

	// Process results
	fmt.Println("Query results:")
	for res := range results {
		if res.err != nil {
			fmt.Printf("  %s (%s) failed: %v\n", res.name, res.path, res.err)
		} else {
			fmt.Printf("  %s (%s): %d notifications\n", res.name, res.path, res.count)
		}
	}
}

// serializedSets demonstrates that Set operations are serialized (not concurrent)
func serializedSets(ctx context.Context, client *gnmi.Client) {
	// Multiple Set operations to perform
	setOps := []struct {
		name  string
		path  string
		value string
	}{
		{"Description 1", "/interfaces/interface[name=GigabitEthernet0/0/0/0]/config/description", `"Serial Set 1"`},
		{"Description 2", "/interfaces/interface[name=GigabitEthernet0/0/0/1]/config/description", `"Serial Set 2"`},
		{"MTU 1", "/interfaces/interface[name=GigabitEthernet0/0/0/0]/config/mtu", "9000"},
		{"MTU 2", "/interfaces/interface[name=GigabitEthernet0/0/0/1]/config/mtu", "9000"},
	}

	start := time.Now()
	var wg sync.WaitGroup

	// Launch Set operations (these will be serialized by the client's Lock)
	for i, op := range setOps {
		wg.Add(1)
		go func(index int, operation struct {
			name  string
			path  string
			value string
		}) {
			defer wg.Done()

			opStart := time.Now()
			ops := []gnmi.SetOperation{
				gnmi.Update(operation.path, operation.value, "json_ietf"),
			}

			_, err := client.Set(ctx, ops)
			elapsed := time.Since(opStart)

			if err != nil {
				fmt.Printf("  %s failed: %v (elapsed: %v)\n", operation.name, err, elapsed)
			} else {
				fmt.Printf("  %s completed (elapsed: %v)\n", operation.name, elapsed)
			}
		}(i, op)
	}

	wg.Wait()
	fmt.Printf("All Set operations completed in %v\n", time.Since(start))
	fmt.Println("Note: Set operations are serialized (Lock ensures only one at a time)")
}

// mixedOperations demonstrates concurrent Gets and serialized Sets
func mixedOperations(ctx context.Context, client *gnmi.Client) {
	var wg sync.WaitGroup

	// Launch several concurrent Get operations
	fmt.Println("Launching concurrent Get operations...")
	for i := 0; i < 5; i++ {
		wg.Add(1)
		go func(index int) {
			defer wg.Done()

			paths := []string{"/system/config"}
			_, err := client.Get(ctx, paths)
			if err != nil {
				fmt.Printf("  Get %d failed: %v\n", index+1, err)
			} else {
				fmt.Printf("  Get %d completed\n", index+1)
			}
		}(i)
	}

	// Launch a Set operation (will acquire exclusive lock)
	wg.Add(1)
	go func() {
		defer wg.Done()

		fmt.Println("Launching Set operation (will serialize)...")
		ops := []gnmi.SetOperation{
			gnmi.Update("/system/config/hostname", `"ConcurrentTest"`, "json_ietf"),
		}

		_, err := client.Set(ctx, ops)
		if err != nil {
			fmt.Printf("  Set failed: %v\n", err)
		} else {
			fmt.Println("  Set completed")
		}
	}()

	wg.Wait()
	fmt.Println("Mixed operations completed")
	fmt.Println("Note: Gets run concurrently, but Set acquired exclusive lock when needed")
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
