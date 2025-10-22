// SPDX-License-Identifier: MPL-2.0
// Copyright (c) 2025 Daniel Schmidt

// Package gnmi provides a simple, fluent API for interacting with network devices
// using the gNMI protocol (gRPC Network Management Interface).
//
// The library provides a high-level client interface that handles connection management,
// JSON manipulation, error handling with automatic retry logic, and thread-safe operations.
//
// # Quick Start
//
// Create a client and perform basic operations:
//
//	client, err := gnmi.NewClient(
//	    "192.168.1.1:57400",
//	    gnmi.Username("admin"),
//	    gnmi.Password("secret"),
//	    gnmi.TLS(true),
//	)
//	if err != nil {
//	    log.Fatal(err)
//	}
//	defer client.Close()
//
//	// Get operation with paths
//	ctx := context.Background()
//	paths := []string{"/interfaces/interface[name=GigabitEthernet0/0/0/0]/state"}
//	res, err := client.Get(ctx, paths)
//	if err != nil {
//	    log.Fatal(err)
//	}
//
//	// Parse response using gjson
//	value := res.GetValue("notification.0.update.0.val").String()
//	fmt.Println("Value:", value)
//
// # JSON Manipulation
//
// Use the Body builder for constructing JSON payloads:
//
//	body := gnmi.Body{}.
//	    Set("config.name", "GigabitEthernet0/0/0/0").
//	    Set("config.description", "WAN Interface").
//	    Set("config.enabled", true).
//	    Set("config.mtu", 9000)
//
//	value, err := body.String()
//	if err != nil {
//	    log.Fatal(err)
//	}
//
//	ops := []gnmi.SetOperation{
//	    gnmi.Update("/interfaces/interface[name=Gi0/0/0/0]/config", value),
//	}
//	res, err = client.Set(ctx, ops)
//
// # Error Handling
//
// The library automatically retries transient errors with exponential backoff:
//
//	client, err := gnmi.NewClient(
//	    "192.168.1.1:57400",
//	    gnmi.Username("admin"),
//	    gnmi.Password("secret"),
//	    gnmi.MaxRetries(5),
//	    gnmi.BackoffMinDelay(1*time.Second),
//	    gnmi.BackoffMaxDelay(60*time.Second),
//	)
//
// # Thread Safety
//
// Read operations (Get, Capabilities) are thread-safe and can be called concurrently.
// Write operations (Set) are synchronized with a mutex.
//
// # Supported Operations
//
//   - Get: Retrieve configuration and state data
//   - Set: Update, replace, or delete configuration
//   - Capabilities: Discover supported encodings and gNMI version
//
// # References
//
//   - gNMI Specification: https://github.com/openconfig/reference/blob/master/rpc/gnmi/gnmi-specification.md
//   - gNMI Protocol: https://github.com/openconfig/gnmi/blob/master/proto/gnmi/gnmi.proto
//   - gjson: https://github.com/tidwall/gjson
//   - sjson: https://github.com/tidwall/sjson
//   - gnmic: https://github.com/openconfig/gnmic
package gnmi
