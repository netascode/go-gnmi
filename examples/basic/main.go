// Package main demonstrates basic go-gnmi API usage.
//
// This example shows:
//   - Client creation with options
//   - Get operations with multiple paths
//   - Set operations (Update, Replace, Delete)
//   - Body builder for JSON payloads
//   - Response parsing with gjson
//
// Usage:
//
//	export GNMI_TARGET=192.168.1.1:57400
//	export GNMI_USERNAME=admin
//	export GNMI_PASSWORD=secret
//	go run main.go
//
//nolint:errcheck,gosec // Example code prioritizes readability over error handling
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
	// Load credentials from environment
	target := getEnv("GNMI_TARGET", "192.168.1.1:57400")
	username := getEnv("GNMI_USERNAME", "admin")
	password := getEnv("GNMI_PASSWORD", "secret")

	// Create client with functional options
	client, err := gnmi.NewClient(
		target,
		gnmi.Username(username),
		gnmi.Password(password),
		gnmi.TLS(true),
		gnmi.VerifyCertificate(false), // For testing only - use true in production
		gnmi.OperationTimeout(30*time.Second),
	)
	if err != nil {
		log.Fatalf("NewClient failed: %v", err)
	}
	defer client.Close() //nolint:errcheck // Error intentionally ignored in example

	fmt.Printf("Connected to %s\n", target)

	ctx := context.Background()

	// Example 1: Get operation with single path
	fmt.Println("\n=== Get Operation (Single Path) ===")
	getSinglePath(ctx, client)

	// Example 2: Get operation with multiple paths
	fmt.Println("\n=== Get Operation (Multiple Paths) ===")
	getMultiplePaths(ctx, client)

	// Example 3: Set operation - Update
	fmt.Println("\n=== Set Operation (Update) ===")
	setUpdate(ctx, client)

	// Example 4: Set operation - Replace
	fmt.Println("\n=== Set Operation (Replace) ===")
	setReplace(ctx, client)

	// Example 5: Set operation - Delete
	fmt.Println("\n=== Set Operation (Delete) ===")
	setDelete(ctx, client)

	// Example 6: Complex Set with Body builder
	fmt.Println("\n=== Complex Set with Body Builder ===")
	setWithBodyBuilder(ctx, client)

	fmt.Println("\n=== Examples Complete ===")
}

// getSinglePath demonstrates a basic Get operation with a single path
func getSinglePath(ctx context.Context, client *gnmi.Client) {
	// Get interface configuration
	paths := []string{
		"/interfaces/interface[name=GigabitEthernet0/0/0/0]/config",
	}

	res, err := client.Get(ctx, paths)
	if err != nil {
		fmt.Printf("Get failed: %v\n", err)
		return
	}

	if res.OK {
		fmt.Println("Get succeeded")
		fmt.Printf("Received %d notifications\n", len(res.Notifications))

		// Parse response with gjson
		if len(res.Notifications) > 0 {
			fmt.Println("Interface configuration retrieved")
			// In a real scenario, you would parse the notification updates here
		}
	}
}

// getMultiplePaths demonstrates Get with multiple paths
func getMultiplePaths(ctx context.Context, client *gnmi.Client) {
	// Get multiple configuration elements
	paths := []string{
		"/interfaces/interface[name=GigabitEthernet0/0/0/0]/config/description",
		"/interfaces/interface[name=GigabitEthernet0/0/0/0]/config/enabled",
		"/interfaces/interface[name=GigabitEthernet0/0/0/0]/config/mtu",
	}

	res, err := client.Get(ctx, paths)
	if err != nil {
		fmt.Printf("Get failed: %v\n", err)
		return
	}

	if res.OK {
		fmt.Printf("Retrieved %d paths successfully\n", len(paths))
		fmt.Printf("Received %d notifications\n", len(res.Notifications))
	}
}

// setUpdate demonstrates Update operation
func setUpdate(ctx context.Context, client *gnmi.Client) {
	// Update interface description
	path := "/interfaces/interface[name=GigabitEthernet0/0/0/0]/config/description"
	value := `"WAN Interface - Updated"`

	ops := []gnmi.SetOperation{
		gnmi.Update(path, value),
	}

	res, err := client.Set(ctx, ops)
	if err != nil {
		fmt.Printf("Set (Update) failed: %v\n", err)
		return
	}

	if res.OK {
		fmt.Println("Update operation succeeded")
		fmt.Printf("Timestamp: %d\n", res.Timestamp)
	}
}

// setReplace demonstrates Replace operation
func setReplace(ctx context.Context, client *gnmi.Client) {
	// Replace interface MTU
	path := "/interfaces/interface[name=GigabitEthernet0/0/0/0]/config/mtu"
	value := "9000"

	ops := []gnmi.SetOperation{
		gnmi.Replace(path, value),
	}

	res, err := client.Set(ctx, ops)
	if err != nil {
		fmt.Printf("Set (Replace) failed: %v\n", err)
		return
	}

	if res.OK {
		fmt.Println("Replace operation succeeded")
		fmt.Printf("Timestamp: %d\n", res.Timestamp)
	}
}

// setDelete demonstrates Delete operation
func setDelete(ctx context.Context, client *gnmi.Client) {
	// Delete interface description
	path := "/interfaces/interface[name=GigabitEthernet0/0/0/0]/config/description"

	ops := []gnmi.SetOperation{
		gnmi.Delete(path),
	}

	res, err := client.Set(ctx, ops)
	if err != nil {
		fmt.Printf("Set (Delete) failed: %v\n", err)
		return
	}

	if res.OK {
		fmt.Println("Delete operation succeeded")
		fmt.Printf("Timestamp: %d\n", res.Timestamp)
	}
}

// setWithBodyBuilder demonstrates using Body builder for complex JSON payloads
func setWithBodyBuilder(ctx context.Context, client *gnmi.Client) {
	// Build complex interface configuration using Body builder
	body := gnmi.Body{}.
		Set("name", "GigabitEthernet0/0/0/0").
		Set("config.description", "Complex Config Example").
		Set("config.enabled", true).
		Set("config.mtu", 9000).
		Set("config.type", "ethernetCsmacd")

	value, err := body.String()
	if err != nil {
		fmt.Printf("Body builder failed: %v\n", err)
		return
	}

	path := "/interfaces/interface[name=GigabitEthernet0/0/0/0]"

	ops := []gnmi.SetOperation{
		gnmi.Update(path, value),
	}

	res, err := client.Set(ctx, ops)
	if err != nil {
		fmt.Printf("Set with Body builder failed: %v\n", err)
		return
	}

	if res.OK {
		fmt.Println("Complex configuration applied successfully")
		fmt.Println("Body builder created JSON payload:")
		fmt.Println(value)
	}
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
