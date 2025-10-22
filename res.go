// SPDX-License-Identifier: MPL-2.0
// Copyright (c) 2025 Daniel Schmidt

package gnmi

import (
	"encoding/json"

	"github.com/openconfig/gnmi/proto/gnmi"
	"github.com/tidwall/gjson"
)

// GetRes represents a gNMI Get response
type GetRes struct {
	// Notifications contains the gNMI notification messages
	Notifications []*gnmi.Notification

	// Timestamp is the response timestamp (nanoseconds since Unix epoch)
	Timestamp int64

	// OK indicates if the operation succeeded
	OK bool

	// Errors contains any error information
	Errors []ErrorModel
}

// GetValue retrieves a value from the response notifications using a gjson path.
// The path follows gjson syntax for querying JSON structures.
//
// Example paths:
//   - "notification.0.timestamp" - Get notification timestamp
//   - "notification.0.update.0.path.elem.0.name" - Get path element name
//   - "notification.0.update.0.val.Value.JsonIetfVal" - Get JSON IETF value (base64 encoded)
//
// Note: The JSON structure uses protobuf JSON marshaling conventions where
// field names are lowercase and TypedValue.Value is nested with capitalized names.
//
// Returns gjson.Result which can be converted to specific types:
//   - result.String() for string values
//   - result.Int() for integer values
//   - result.Bool() for boolean values
//   - result.Array() for array values
//
// Example:
//
//	res, err := client.Get(ctx, []string{"/interfaces/interface[name=Gi0/0/0/0]/state"})
//	if err != nil {
//	    log.Fatal(err)
//	}
//	timestamp := res.GetValue("notification.0.timestamp").Int()
//	pathName := res.GetValue("notification.0.update.0.path.elem.0.name").String()
func (r GetRes) GetValue(path string) gjson.Result {
	jsonStr := r.JSON()
	if jsonStr == "" {
		return gjson.Result{}
	}
	return gjson.Get(jsonStr, path)
}

// JSON returns the response notifications as a formatted JSON string.
// This is useful for debugging, logging, or custom parsing.
// Returns an empty string if marshaling fails.
//
// Example:
//
//	res, err := client.Get(ctx, []string{"/system/config"})
//	if err != nil {
//	    log.Fatal(err)
//	}
//	fmt.Println(res.JSON()) // Print full response as JSON
func (r GetRes) JSON() string {
	if r.Notifications == nil {
		return ""
	}

	// Create a wrapper structure for better JSON output
	wrapper := struct {
		Notification []*gnmi.Notification `json:"notification"`
		Timestamp    int64                `json:"timestamp"`
		OK           bool                 `json:"ok"`
	}{
		Notification: r.Notifications,
		Timestamp:    r.Timestamp,
		OK:           r.OK,
	}

	data, err := json.Marshal(wrapper)
	if err != nil {
		return ""
	}
	return string(data)
}

// SetRes represents a gNMI Set response
type SetRes struct {
	// Response is the gNMI SetResponse
	Response *gnmi.SetResponse

	// Timestamp is the response timestamp (nanoseconds since Unix epoch)
	Timestamp int64

	// OK indicates if the operation succeeded
	OK bool

	// Errors contains any error information
	Errors []ErrorModel
}

// GetValue retrieves a value from the SetResponse using a gjson path.
// The path follows gjson syntax for querying JSON structures.
//
// Example paths:
//   - "response.0.op" - Get operation type (UPDATE, REPLACE, DELETE)
//   - "response.0.path" - Get path string
//   - "timestamp" - Get response timestamp
//
// Returns gjson.Result which can be converted to specific types:
//   - result.String() for string values
//   - result.Int() for integer values
//
// Example:
//
//	ops := []gnmi.SetOperation{
//	    gnmi.Update("/interfaces/interface[name=Gi0/0/0/0]/config/mtu", `{"mtu": 9000}`),
//	}
//	res, err := client.Set(ctx, ops)
//	if err != nil {
//	    log.Fatal(err)
//	}
//	op := res.GetValue("response.0.op").String()
func (r SetRes) GetValue(path string) gjson.Result {
	jsonStr := r.JSON()
	if jsonStr == "" {
		return gjson.Result{}
	}
	return gjson.Get(jsonStr, path)
}

// JSON returns the SetResponse as a formatted JSON string.
// This is useful for debugging, logging, or custom parsing.
// Returns an empty string if marshaling fails.
//
// Example:
//
//	ops := []gnmi.SetOperation{
//	    gnmi.Update("/system/config/hostname", `{"hostname": "router1"}`),
//	}
//	res, err := client.Set(ctx, ops)
//	if err != nil {
//	    log.Fatal(err)
//	}
//	fmt.Println(res.JSON()) // Print full response as JSON
func (r SetRes) JSON() string {
	if r.Response == nil {
		return ""
	}

	// Create a wrapper structure for better JSON output
	wrapper := struct {
		Response  *gnmi.SetResponse `json:"response"`
		Timestamp int64             `json:"timestamp"`
		OK        bool              `json:"ok"`
	}{
		Response:  r.Response,
		Timestamp: r.Timestamp,
		OK:        r.OK,
	}

	data, err := json.Marshal(wrapper)
	if err != nil {
		return ""
	}
	return string(data)
}

// CapabilitiesRes represents a gNMI Capabilities response
type CapabilitiesRes struct {
	// Version is the gNMI service version
	Version string

	// Capabilities lists supported capabilities
	Capabilities []string

	// Models contains supported data models
	Models []*gnmi.ModelData

	// OK indicates if the operation succeeded
	OK bool

	// Errors contains any error information
	Errors []ErrorModel
}
