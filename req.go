// SPDX-License-Identifier: MPL-2.0
// Copyright (c) 2025 Daniel Schmidt

package gnmi

import "time"

// Req represents a gNMI request modifier
//
// This struct is used to apply request-specific options via functional modifiers.
// Operation parameters (paths, operations) are passed directly to methods.
//
// Example:
//
//	// Get with custom encoding and timeout
//	res, err := client.Get(ctx, paths,
//	    gnmi.Encoding("proto"),
//	    gnmi.Timeout(30*time.Second))
type Req struct {
	// Encoding specifies the data encoding
	// Valid values: json, json_ietf (default), proto, ascii, bytes
	Encoding string

	// Timeout is the request-specific timeout
	// Overrides client default timeout if set
	Timeout time.Duration
}

// SetOperationType represents the type of Set operation
type SetOperationType string

const (
	// OperationUpdate modifies existing configuration, creating it if it doesn't exist
	OperationUpdate SetOperationType = "update"

	// OperationReplace removes existing configuration before applying new value
	OperationReplace SetOperationType = "replace"

	// OperationDelete removes configuration at the specified path
	OperationDelete SetOperationType = "delete"
)

// SetOperation represents a single gNMI Set operation (Update, Replace, or Delete)
type SetOperation struct {
	// OperationType specifies the operation type (update, replace, delete)
	OperationType SetOperationType

	// Path is the gNMI path
	Path string

	// Value is the JSON value for Update/Replace operations
	// Empty for Delete operations
	Value string

	// Encoding specifies the value encoding
	// Valid values: json, json_ietf (default), proto, ascii, bytes
	Encoding string
}
