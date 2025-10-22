//nolint:errcheck,gosec // Example code prioritizes readability over error handling

// Package main demonstrates gNMI capability discovery.
//
// This example shows:
//   - Discovering device capabilities with Capabilities()
//   - Checking gNMI version support
//   - Checking supported encodings
//   - Checking supported models
//   - Using HasCapability() to adapt behavior
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
	"time"

	"github.com/netascode/go-gnmi"
)

func main() {
	// Load credentials from environment
	target := getEnv("GNMI_TARGET", "192.168.1.1:57400")
	username := getEnv("GNMI_USERNAME", "admin")
	password := getEnv("GNMI_PASSWORD", "secret")

	// Create client
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

	// Example 1: Basic capability discovery
	fmt.Println("\n=== Capability Discovery ===")
	discoverCapabilities(ctx, client)

	// Example 2: Check for specific capabilities
	fmt.Println("\n=== Capability Checking ===")
	checkSpecificCapabilities(client)

	// Example 3: Adapt behavior based on capabilities
	fmt.Println("\n=== Adaptive Behavior ===")
	adaptiveBehavior(ctx, client)

	fmt.Println("\n=== Examples Complete ===")
}

// discoverCapabilities retrieves and displays all device capabilities
func discoverCapabilities(ctx context.Context, client *gnmi.Client) {
	capRes, err := client.Capabilities(ctx)
	if err != nil {
		fmt.Printf("Capabilities request failed: %v\n", err)
		return
	}

	if !capRes.OK {
		fmt.Println("Capabilities request returned errors:")
		for _, e := range capRes.Errors {
			fmt.Printf("  Error [%d]: %s\n", e.Code, e.Message)
		}
		return
	}

	// Display gNMI version
	fmt.Printf("gNMI Version: %s\n", capRes.Version)

	// Display supported encodings
	fmt.Println("\nSupported Encodings:")
	if len(capRes.Capabilities) > 0 {
		for _, cap := range capRes.Capabilities {
			fmt.Printf("  - %s\n", cap)
		}
	} else {
		fmt.Println("  (none reported)")
	}

	// Display supported models
	fmt.Println("\nSupported Models:")
	if len(capRes.Models) > 0 {
		for i, model := range capRes.Models {
			if i >= 10 {
				fmt.Printf("  ... and %d more models\n", len(capRes.Models)-10)
				break
			}
			fmt.Printf("  - %s (org: %s, version: %s)\n",
				model.GetName(),
				model.GetOrganization(),
				model.GetVersion())
		}
	} else {
		fmt.Println("  (none reported)")
	}
}

// checkSpecificCapabilities demonstrates checking for specific capabilities
func checkSpecificCapabilities(client *gnmi.Client) {
	// Common capability checks
	capabilities := []string{
		"json",
		"json_ietf",
		"proto",
		"ascii",
		"bytes",
	}

	fmt.Println("Checking for common encoding capabilities:")
	for _, cap := range capabilities {
		if client.HasCapability(cap) {
			fmt.Printf("  ✓ %s supported\n", cap)
		} else {
			fmt.Printf("  ✗ %s not supported\n", cap)
		}
	}
}

// adaptiveBehavior demonstrates adapting client behavior based on capabilities
func adaptiveBehavior(ctx context.Context, client *gnmi.Client) {
	// Determine the best encoding to use
	var encoding string
	switch {
	case client.HasCapability("json_ietf"):
		encoding = "json_ietf"
		fmt.Println("Using json_ietf encoding (preferred)")
	case client.HasCapability("json"):
		encoding = "json"
		fmt.Println("Using json encoding (fallback)")
	case client.HasCapability("proto"):
		encoding = "proto"
		fmt.Println("Using proto encoding (fallback)")
	default:
		fmt.Println("ERROR: No supported encoding found")
		fmt.Println("Device does not support json, json_ietf, or proto encodings")
		fmt.Println("Cannot proceed with Set operations")
		return
	}

	// Use the selected encoding for operations
	fmt.Printf("\nPerforming Get operation with %s encoding...\n", encoding)
	paths := []string{"/system/config/hostname"}

	res, err := client.Get(ctx, paths, gnmi.GetEncoding(encoding))
	if err != nil {
		fmt.Printf("Get with %s encoding failed: %v\n", encoding, err)
		return
	}

	if res.OK {
		fmt.Printf("Get with %s encoding succeeded\n", encoding)
		fmt.Printf("Received %d notifications\n", len(res.Notifications))
	}

	// Example: Check for specific model support
	fmt.Println("\nChecking for OpenConfig model support...")
	ctx2, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	capRes, err := client.Capabilities(ctx2)
	if err != nil {
		fmt.Printf("Failed to get capabilities: %v\n", err)
		return
	}

	hasOpenConfig := false
	for _, model := range capRes.Models {
		if model.GetOrganization() == "openconfig" {
			hasOpenConfig = true
			break
		}
	}

	if hasOpenConfig {
		fmt.Println("  ✓ Device supports OpenConfig models")
		fmt.Println("  Can use OpenConfig paths for configuration")
	} else {
		fmt.Println("  ✗ Device does not report OpenConfig models")
		fmt.Println("  Use vendor-specific paths for configuration")
	}
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
