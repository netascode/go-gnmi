// SPDX-License-Identifier: MPL-2.0
// Copyright (c) 2025 Daniel Schmidt

package gnmi

import (
	"fmt"

	"github.com/tidwall/sjson"
)

// Body provides a fluent interface for building JSON configurations
// using sjson for path-based manipulation.
//
// The Body builder tracks errors internally to enable method chaining
// while providing error checking through String() or Err() methods.
//
// Example:
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
type Body struct {
	// str contains the JSON string being built
	str string
	// err tracks the first error encountered during building
	err error
}

// Set sets a value at the specified JSON path and returns a new Body
//
// The path uses dot notation for nested fields (e.g., "config.name").
// The value can be any type that sjson supports (string, number, bool, etc.).
//
// If an error occurs, the error is stored and returned by String() or Err().
// Once an error occurs, all subsequent operations are no-ops that preserve the error.
//
// Example:
//
//	body := gnmi.Body{}.
//	    Set("config.name", "eth0").
//	    Set("config.enabled", true).
//	    Set("config.mtu", 1500)
//	json, err := body.String()
//
// Returns the Body for method chaining.
func (b Body) Set(path string, value any) Body {
	// Short-circuit if already in error state
	if b.err != nil {
		return b
	}

	result, err := sjson.Set(b.str, path, value)
	if err != nil {
		// Store error and return body with error state
		return Body{str: b.str, err: fmt.Errorf("Set(%q): %w", path, err)}
	}
	return Body{str: result, err: nil}
}

// Delete removes a value at the specified JSON path and returns a new Body
//
// The path uses dot notation for nested fields (e.g., "config.description").
//
// If an error occurs, the error is stored and returned by String() or Err().
//
// Example:
//
//	body := gnmi.Body{}.
//	    Set("name", "eth0").
//	    Set("description", "temp").
//	    Delete("description")  // Remove description field
//	json, err := body.String()
//
// Returns the Body for method chaining.
func (b Body) Delete(path string) Body {
	// Short-circuit if already in error state
	if b.err != nil {
		return b
	}

	result, err := sjson.Delete(b.str, path)
	if err != nil {
		return Body{str: b.str, err: fmt.Errorf("Delete(%q): %w", path, err)}
	}
	return Body{str: result, err: nil}
}

// String returns the JSON string representation and any error encountered during building
//
// This method returns both the JSON string and any error that occurred during the building process.
// If an error occurred during any Set/Delete operation, the error will be returned here.
//
// Example:
//
//	body := gnmi.Body{}.Set("config.hostname", "router1")
//	json, err := body.String()
//	if err != nil {
//	    log.Fatal(err)
//	}
func (b Body) String() (string, error) {
	return b.str, b.err
}

// Err returns any error that occurred during the building process
//
// This method allows checking for errors without retrieving the string value.
//
// Example:
//
//	body := gnmi.Body{}.Set("config.hostname", "router1")
//	if err := body.Err(); err != nil {
//	    log.Fatal(err)
//	}
func (b Body) Err() error {
	return b.err
}

// Res returns the JSON string for further processing with gjson
//
// This allows you to query the built JSON using gjson's Get function.
// If an error occurred during building, this returns an empty string.
// Use Err() or String() to check for errors.
//
// Example:
//
//	body := gnmi.Body{}.Set("config.hostname", "router1")
//	if body.Err() == nil {
//	    json := body.Res()
//	    hostname := gjson.Get(json, "config.hostname").String()
//	}
//
// Returns the JSON string that can be queried with gjson.Get.
func (b Body) Res() string {
	// Return empty string if there's an error
	// (caller should check Err() first)
	if b.err != nil {
		return ""
	}
	return b.str
}

// Bytes returns the JSON byte slice representation and any error encountered during building
//
// This is useful when you need []byte instead of string for efficiency.
//
// Example:
//
//	body := gnmi.Body{}.Set("name", "eth0")
//	jsonBytes, err := body.Bytes()
//	if err != nil {
//	    log.Fatal(err)
//	}
func (b Body) Bytes() ([]byte, error) {
	if b.err != nil {
		return nil, b.err
	}
	return []byte(b.str), nil
}
