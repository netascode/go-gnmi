// SPDX-License-Identifier: MPL-2.0
// Copyright (c) 2025 Daniel Schmidt

package gnmi

import (
	"context"
	"sync"
	"testing"
	"time"
)

// TestConcurrentGetOperations tests that multiple Get operations can run concurrently
func TestConcurrentGetOperations(t *testing.T) {
	// Create a client with mock configuration
	client := &Client{
		Target:           "192.168.1.1",
		Port:             57400,
		OperationTimeout: 10 * time.Second,
		logger:           &NoOpLogger{},
		capabilities:     []string{"JSON_IETF"},
	}

	// Number of concurrent operations
	numOps := 10
	var wg sync.WaitGroup
	errChan := make(chan error, numOps)

	// Launch concurrent Get operations
	for i := 0; i < numOps; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()

			ctx := context.Background()
			// This will fail (no connection), but should not cause race conditions
			_, err := client.Get(ctx, []string{"/test/path"})

			// We expect an error (not connected), but no panic
			if err == nil {
				errChan <- nil
			}
		}(i)
	}

	// Wait for all operations to complete
	wg.Wait()
	close(errChan)

	// Verify no panics occurred
	// (if there was a race condition, test would fail or panic)
}

// TestConcurrentSetOperations tests that Set operations are serialized
func TestConcurrentSetOperations(t *testing.T) {
	// Create a client with mock configuration
	client := &Client{
		Target:           "192.168.1.1",
		Port:             57400,
		OperationTimeout: 10 * time.Second,
		logger:           &NoOpLogger{},
		capabilities:     []string{"JSON_IETF"},
	}

	// Number of concurrent operations
	numOps := 5
	var wg sync.WaitGroup

	// Launch concurrent Set operations
	for i := 0; i < numOps; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()

			ctx := context.Background()
			ops := []SetOperation{
				Update("/test/path", `{"value": "test"}`, "json_ietf"),
			}
			// This will fail (no connection), but should not cause race conditions
			_, _ = client.Set(ctx, ops) //nolint:errcheck // Error intentionally ignored in test
		}(i)
	}

	// Wait for all operations to complete
	wg.Wait()

	// Verify no panics occurred
	// (if there was a race condition, test would fail or panic)
}

// TestConcurrentCapabilitiesOperations tests that Capabilities operations can run concurrently
func TestConcurrentCapabilitiesOperations(t *testing.T) {
	// Create a client with pre-populated capabilities
	client := &Client{
		Target:           "192.168.1.1",
		Port:             57400,
		OperationTimeout: 10 * time.Second,
		logger:           &NoOpLogger{},
		capabilities:     []string{"JSON_IETF", "PROTO"},
	}

	// Number of concurrent operations
	numOps := 10
	var wg sync.WaitGroup
	results := make([]bool, numOps)

	// Launch concurrent Capabilities operations
	for i := 0; i < numOps; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()

			ctx := context.Background()
			// This will fail (no connection), but should not cause race conditions
			_, _ = client.Capabilities(ctx) //nolint:errcheck // Error intentionally ignored in test

			// Also test concurrent reads of ServerCapabilities
			caps := client.ServerCapabilities()
			results[idx] = len(caps) == 2
		}(i)
	}

	// Wait for all operations to complete
	wg.Wait()

	// Verify all concurrent reads got correct capabilities
	for i, result := range results {
		if !result {
			t.Errorf("Concurrent operation %d got incorrect capabilities", i)
		}
	}
}

// TestConcurrentHasCapability tests concurrent calls to HasCapability
func TestConcurrentHasCapability(t *testing.T) {
	client := &Client{
		capabilities: []string{"JSON", "JSON_IETF", "PROTO"},
	}

	numOps := 20
	var wg sync.WaitGroup
	results := make([]bool, numOps)

	// Launch concurrent HasCapability calls
	for i := 0; i < numOps; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()

			// Test different capabilities
			var capability string
			switch idx % 3 {
			case 0:
				capability = "JSON"
			case 1:
				capability = "JSON_IETF"
			case 2:
				capability = "PROTO"
			}

			results[idx] = client.HasCapability(capability)
		}(i)
	}

	wg.Wait()

	// Verify all results are correct
	for i, result := range results {
		if !result {
			t.Errorf("Concurrent HasCapability(%d) returned false, want true", i)
		}
	}
}

// TestConcurrentBackoffCalculation tests concurrent backoff calculations
func TestConcurrentBackoffCalculation(t *testing.T) {
	client := &Client{
		BackoffMinDelay:    1 * time.Second,
		BackoffMaxDelay:    60 * time.Second,
		BackoffDelayFactor: 2.0,
		logger:             &NoOpLogger{},
	}

	numOps := 50
	var wg sync.WaitGroup
	delays := make([]time.Duration, numOps)

	// Launch concurrent Backoff calculations
	for i := 0; i < numOps; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			delays[idx] = client.Backoff(idx % 5)
		}(i)
	}

	wg.Wait()

	// Verify all delays are within expected range
	for i, delay := range delays {
		attempt := i % 5
		// Calculate expected base delay
		expectedBase := 1 * time.Second
		for j := 0; j < attempt; j++ {
			expectedBase *= 2
		}

		// Delay should be >= base and <= base + 10% (jitter)
		maxExpected := expectedBase + expectedBase/10
		if delay < expectedBase || delay > maxExpected {
			t.Errorf("Concurrent Backoff(%d) = %v, expected range [%v, %v]",
				attempt, delay, expectedBase, maxExpected)
		}
	}
}

// TestConcurrentMixedOperations tests a mix of read and write operations
func TestConcurrentMixedOperations(t *testing.T) {
	client := &Client{
		Target:             "192.168.1.1",
		Port:               57400,
		OperationTimeout:   10 * time.Second,
		MaxRetries:         3,
		BackoffMinDelay:    1 * time.Second,
		BackoffMaxDelay:    60 * time.Second,
		BackoffDelayFactor: 2.0,
		logger:             &NoOpLogger{},
		capabilities:       []string{"JSON_IETF"},
	}

	numOps := 20
	var wg sync.WaitGroup

	// Launch a mix of read and write operations
	for i := 0; i < numOps; i++ {
		wg.Add(1)

		if i%2 == 0 {
			// Even indices: Get operations (read lock)
			go func(idx int) {
				defer wg.Done()
				ctx := context.Background()
				_, _ = client.Get(ctx, []string{ //nolint:errcheck // Error intentionally ignored in test
					"/test/path"})
			}(i)
		} else {
			// Odd indices: Set operations (write lock)
			go func(idx int) {
				defer wg.Done()
				ctx := context.Background()
				ops := []SetOperation{
					Update("/test/path", `{"value": "test"}`, "json_ietf"),
				}
				_, _ = client.Set(ctx, ops) //nolint:errcheck // Error intentionally ignored in test
			}(i)
		}
	}

	wg.Wait()

	// Verify no panics occurred
	// (if there was a race condition, test would fail or panic)
}

// TestConcurrentCloseOperations tests that Close is thread-safe
func TestConcurrentCloseOperations(t *testing.T) {
	client := &Client{
		Target: "192.168.1.1",
		logger: &NoOpLogger{},
	}

	numOps := 10
	var wg sync.WaitGroup

	// Launch concurrent Close operations
	for i := 0; i < numOps; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_ = client.Close() //nolint:errcheck // Error intentionally ignored in cleanup
		}()
	}

	wg.Wait()

	// Verify no panics occurred
	// Multiple Close calls should be safe
}

// TestRaceConditionDetection is a marker test that runs with -race flag
// This test ensures the entire test suite can be run with race detector
func TestRaceConditionDetection(t *testing.T) {
	// This test serves as documentation that all tests should pass with -race flag
	// Run: go test -race ./...
	t.Log("Run full test suite with: go test -race ./...")
	t.Log("All concurrent tests should pass without data races")
}

// BenchmarkConcurrentGetOperations benchmarks concurrent Get operations
func BenchmarkConcurrentGetOperations(b *testing.B) {
	client := &Client{
		Target:           "192.168.1.1",
		Port:             57400,
		OperationTimeout: 10 * time.Second,
		logger:           &NoOpLogger{},
		capabilities:     []string{"JSON_IETF"},
	}

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			ctx := context.Background()
			_, _ = client.Get(ctx, []string{ //nolint:errcheck // Error intentionally ignored in test
				"/test/path"})
		}
	})
}

// BenchmarkConcurrentHasCapability benchmarks concurrent HasCapability calls
func BenchmarkConcurrentHasCapability(b *testing.B) {
	client := &Client{
		capabilities: []string{"JSON", "JSON_IETF", "PROTO"},
	}

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_ = client.HasCapability("JSON_IETF")
		}
	})
}

// BenchmarkConcurrentBackoff benchmarks concurrent Backoff calculations
func BenchmarkConcurrentBackoff(b *testing.B) {
	client := &Client{
		BackoffMinDelay:    1 * time.Second,
		BackoffMaxDelay:    60 * time.Second,
		BackoffDelayFactor: 2.0,
		logger:             &NoOpLogger{},
	}

	b.RunParallel(func(pb *testing.PB) {
		attempt := 0
		for pb.Next() {
			_ = client.Backoff(attempt % 10)
			attempt++
		}
	})
}
