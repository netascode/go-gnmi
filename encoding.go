// SPDX-License-Identifier: MPL-2.0
// Copyright (c) 2025 Daniel Schmidt

package gnmi

import "fmt"

// Encoding constants for gNMI operations
const (
	// EncodingJSON uses standard JSON encoding
	EncodingJSON = "json"

	// EncodingJSONIETF uses JSON encoding with IETF conventions (default)
	// This is the recommended encoding for most use cases
	EncodingJSONIETF = "json_ietf"

	// EncodingProto uses Protocol Buffer encoding
	EncodingProto = "proto"

	// EncodingASCII uses ASCII encoding
	EncodingASCII = "ascii"

	// EncodingBytes uses raw byte encoding
	EncodingBytes = "bytes"
)

// ValidEncodings contains the list of valid encoding values
var ValidEncodings = []string{
	EncodingJSON,
	EncodingJSONIETF,
	EncodingProto,
	EncodingASCII,
	EncodingBytes,
}

// ValidateEncoding checks if the encoding is valid
//
// Returns an error if the encoding is not one of the supported values.
//
// Example:
//
//	if err := gnmi.ValidateEncoding("json_ietf"); err != nil {
//	    log.Fatal(err)
//	}
func ValidateEncoding(enc string) error {
	for _, valid := range ValidEncodings {
		if enc == valid {
			return nil
		}
	}
	return fmt.Errorf("invalid encoding: %s (valid values: json, json_ietf, proto, ascii, bytes)", enc)
}
