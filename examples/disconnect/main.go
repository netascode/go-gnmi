// SPDX-License-Identifier: MPL-2.0
// Copyright (c) 2025 Daniel Schmidt

// Package main demonstrates the use of Disconnect() for connection pooling
// and temporary connection release patterns in go-gnmi.
//
// This example shows:
//  1. Basic disconnect and reconnect pattern
//  2. Connection pooling with idle timeout
//  3. Long-running application with maintenance windows
//  4. Difference between Disconnect() (reusable) and Close() (terminal)
package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/netascode/go-gnmi"
)

func main() {
	// Get connection details from environment
	target := os.Getenv("GNMI_TARGET")
	if target == "" {
		target = "192.168.1.1:57400"
	}

	username := os.Getenv("GNMI_USERNAME")
	if username == "" {
		username = "admin"
	}

	password := os.Getenv("GNMI_PASSWORD")
	if password == "" {
		log.Fatal("GNMI_PASSWORD environment variable required")
	}

	// Example 1: Basic Disconnect and Reconnect
	fmt.Println("\n=== Example 1: Basic Disconnect and Reconnect ===")
	basicDisconnectReconnect(target, username, password)

	// Example 2: Connection Pooling with Idle Timeout
	fmt.Println("\n=== Example 2: Connection Pooling with Idle Timeout ===")
	connectionPoolingWithIdleTimeout(target, username, password)

	// Example 3: Long-running Application with Maintenance Windows
	fmt.Println("\n=== Example 3: Long-running Application with Maintenance Windows ===")
	longRunningAppWithMaintenance(target, username, password)

	// Example 4: Difference between Disconnect() and Close()
	fmt.Println("\n=== Example 4: Disconnect vs Close ===")
	disconnectVsClose(target, username, password)
}

// basicDisconnectReconnect demonstrates basic disconnect and automatic reconnect
func basicDisconnectReconnect(target, username, password string) {
	client, err := gnmi.NewClient(
		target,
		gnmi.Username(username),
		gnmi.Password(password),
		gnmi.VerifyCertificate(false), // Skip cert verification for demo
	)
	if err != nil {
		log.Fatalf("Failed to create client: %v", err)
	}
	defer client.Close()

	ctx := context.Background()

	// First operation - establishes connection
	fmt.Println("1. Performing first Get operation (establishes connection)...")
	_, err = client.Get(ctx, []string{"/system/state/hostname"})
	if err != nil {
		fmt.Printf("   Get failed: %v\n", err)
	} else {
		fmt.Println("   Get successful")
	}

	// Disconnect - closes connection but preserves config
	fmt.Println("2. Disconnecting (releases connection)...")
	if err := client.Disconnect(); err != nil {
		log.Fatalf("Disconnect failed: %v", err)
	}
	fmt.Println("   Disconnected successfully")

	// Second operation - automatically reconnects
	fmt.Println("3. Performing second Get operation (auto-reconnects)...")
	_, err = client.Get(ctx, []string{"/system/state/hostname"})
	if err != nil {
		fmt.Printf("   Get failed: %v\n", err)
	} else {
		fmt.Println("   Get successful (automatically reconnected)")
	}
}

// connectionPoolingWithIdleTimeout demonstrates connection pooling with periodic idle timeout
func connectionPoolingWithIdleTimeout(target, username, password string) {
	client, err := gnmi.NewClient(
		target,
		gnmi.Username(username),
		gnmi.Password(password),
		gnmi.VerifyCertificate(false),
	)
	if err != nil {
		log.Fatalf("Failed to create client: %v", err)
	}
	defer client.Close()

	ctx := context.Background()

	// Simulate connection pool manager that disconnects idle connections
	idleTimeout := 2 * time.Second
	lastActivity := time.Now()

	// Background goroutine to disconnect idle connections
	stopMonitor := make(chan struct{})
	defer close(stopMonitor)

	go func() {
		ticker := time.NewTicker(1 * time.Second)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				if time.Since(lastActivity) > idleTimeout {
					fmt.Printf("   [Pool Manager] Connection idle for %v, disconnecting...\n", idleTimeout)
					if err := client.Disconnect(); err != nil {
						fmt.Printf("   [Pool Manager] Disconnect failed: %v\n", err)
					} else {
						fmt.Println("   [Pool Manager] Disconnected idle connection")
					}
					// Reset to prevent repeated disconnects
					lastActivity = time.Now()
				}
			case <-stopMonitor:
				return
			}
		}
	}()

	// Simulate periodic operations with idle periods
	for i := 0; i < 3; i++ {
		fmt.Printf("%d. Performing operation...\n", i+1)
		_, err = client.Get(ctx, []string{"/system/state/hostname"})
		lastActivity = time.Now()

		if err != nil {
			fmt.Printf("   Operation failed: %v\n", err)
		} else {
			fmt.Println("   Operation successful")
		}

		if i < 2 {
			// Idle period
			fmt.Printf("   Idle for %v...\n", idleTimeout+500*time.Millisecond)
			time.Sleep(idleTimeout + 500*time.Millisecond)
		}
	}
}

// longRunningAppWithMaintenance demonstrates disconnect during maintenance windows
func longRunningAppWithMaintenance(target, username, password string) {
	client, err := gnmi.NewClient(
		target,
		gnmi.Username(username),
		gnmi.Password(password),
		gnmi.VerifyCertificate(false),
	)
	if err != nil {
		log.Fatalf("Failed to create client: %v", err)
	}
	defer client.Close()

	ctx := context.Background()

	// Normal operation
	fmt.Println("1. Normal operation period...")
	for i := 0; i < 3; i++ {
		_, err = client.Get(ctx, []string{"/system/state/hostname"})
		if err != nil {
			fmt.Printf("   Operation %d failed: %v\n", i+1, err)
		} else {
			fmt.Printf("   Operation %d successful\n", i+1)
		}
		time.Sleep(100 * time.Millisecond)
	}

	// Maintenance window - disconnect
	fmt.Println("2. Entering maintenance window (disconnecting)...")
	if err := client.Disconnect(); err != nil {
		log.Fatalf("Disconnect failed: %v", err)
	}
	fmt.Println("   Disconnected for maintenance")

	// Simulate maintenance work
	fmt.Println("   Performing maintenance...")
	time.Sleep(1 * time.Second)

	// Resume normal operation - auto-reconnect
	fmt.Println("3. Maintenance complete, resuming operations...")
	_, err = client.Get(ctx, []string{"/system/state/hostname"})
	if err != nil {
		fmt.Printf("   First operation after maintenance failed: %v\n", err)
	} else {
		fmt.Println("   First operation after maintenance successful (auto-reconnected)")
	}
}

// disconnectVsClose demonstrates the key difference between Disconnect() and Close()
func disconnectVsClose(target, username, password string) {
	// Test Disconnect() - reusable
	fmt.Println("Testing Disconnect() - reusable:")
	client1, err := gnmi.NewClient(
		target,
		gnmi.Username(username),
		gnmi.Password(password),
		gnmi.VerifyCertificate(false),
	)
	if err != nil {
		log.Fatalf("Failed to create client: %v", err)
	}

	ctx := context.Background()

	// First operation
	fmt.Println("1. First operation...")
	_, err = client1.Get(ctx, []string{"/system/state/hostname"})
	if err != nil {
		fmt.Printf("   Failed: %v\n", err)
	} else {
		fmt.Println("   Success")
	}

	// Disconnect
	fmt.Println("2. Disconnecting...")
	if err := client1.Disconnect(); err != nil {
		log.Fatalf("Disconnect failed: %v", err)
	}
	fmt.Println("   Disconnected")

	// Try to use again - should auto-reconnect
	fmt.Println("3. Second operation after Disconnect...")
	_, err = client1.Get(ctx, []string{"/system/state/hostname"})
	if err != nil {
		fmt.Printf("   Failed: %v\n", err)
	} else {
		fmt.Println("   Success (auto-reconnected)")
	}

	// Clean up
	client1.Close()

	// Test Close() - terminal
	fmt.Println("\nTesting Close() - terminal:")
	client2, err := gnmi.NewClient(
		target,
		gnmi.Username(username),
		gnmi.Password(password),
		gnmi.VerifyCertificate(false),
	)
	if err != nil {
		log.Fatalf("Failed to create client: %v", err)
	}

	// First operation
	fmt.Println("1. First operation...")
	_, err = client2.Get(ctx, []string{"/system/state/hostname"})
	if err != nil {
		fmt.Printf("   Failed: %v\n", err)
	} else {
		fmt.Println("   Success")
	}

	// Close (terminal)
	fmt.Println("2. Closing...")
	if err := client2.Close(); err != nil {
		log.Fatalf("Close failed: %v", err)
	}
	fmt.Println("   Closed")

	// Try to use again - should fail
	fmt.Println("3. Second operation after Close...")
	_, err = client2.Get(ctx, []string{"/system/state/hostname"})
	if err != nil {
		fmt.Printf("   Failed as expected: %v\n", err)
	} else {
		fmt.Println("   Unexpected success!")
	}
}
