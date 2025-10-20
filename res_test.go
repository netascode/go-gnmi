// SPDX-License-Identifier: MPL-2.0
// Copyright (c) 2025 Daniel Schmidt

package gnmi

import (
	"encoding/json"
	"testing"

	"github.com/openconfig/gnmi/proto/gnmi"
	"google.golang.org/protobuf/types/known/anypb"
)

func TestGetRes_GetValue(t *testing.T) {
	tests := []struct {
		name          string
		notifications []*gnmi.Notification
		path          string
		want          interface{}
		wantType      string // "string", "int", "bool"
	}{
		{
			name: "get timestamp value",
			notifications: []*gnmi.Notification{
				{
					Timestamp: 1234567890,
					Update: []*gnmi.Update{
						{
							Path: &gnmi.Path{
								Elem: []*gnmi.PathElem{
									{Name: "interfaces"},
									{Name: "interface", Key: map[string]string{"name": "Gi0/0/0/0"}},
									{Name: "config"},
								},
							},
							Val: &gnmi.TypedValue{
								Value: &gnmi.TypedValue_JsonIetfVal{
									JsonIetfVal: []byte(`{"name": "Gi0/0/0/0", "description": "WAN Interface"}`),
								},
							},
						},
					},
				},
			},
			path:     "notification.0.timestamp",
			want:     int64(1234567890),
			wantType: "int",
		},
		{
			name: "get path element name",
			notifications: []*gnmi.Notification{
				{
					Timestamp: 1234567890,
					Update: []*gnmi.Update{
						{
							Path: &gnmi.Path{
								Elem: []*gnmi.PathElem{
									{Name: "interfaces"},
								},
							},
							Val: &gnmi.TypedValue{
								Value: &gnmi.TypedValue_JsonIetfVal{
									JsonIetfVal: []byte(`{"name": "Gi0/0/0/0"}`),
								},
							},
						},
					},
				},
			},
			path:     "notification.0.update.0.path.elem.0.name",
			want:     "interfaces",
			wantType: "string",
		},
		{
			name:          "empty notifications",
			notifications: nil,
			path:          "notification.0.timestamp",
			want:          "",
			wantType:      "string",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := GetRes{
				Notifications: tt.notifications,
				Timestamp:     1234567890,
				OK:            true,
			}

			result := r.GetValue(tt.path)

			switch tt.wantType {
			case "string":
				got := result.String()
				if got != tt.want {
					t.Errorf("GetValue() = %v, want %v", got, tt.want)
				}
			case "int":
				got := result.Int()
				if got != tt.want {
					t.Errorf("GetValue() = %v, want %v", got, tt.want)
				}
			case "bool":
				got := result.Bool()
				if got != tt.want {
					t.Errorf("GetValue() = %v, want %v", got, tt.want)
				}
			}
		})
	}
}

func TestGetRes_JSON(t *testing.T) {
	tests := []struct {
		name          string
		notifications []*gnmi.Notification
		wantContains  []string
	}{
		{
			name: "valid notification",
			notifications: []*gnmi.Notification{
				{
					Timestamp: 1234567890,
					Update: []*gnmi.Update{
						{
							Path: &gnmi.Path{
								Elem: []*gnmi.PathElem{
									{Name: "interfaces"},
								},
							},
							Val: &gnmi.TypedValue{
								Value: &gnmi.TypedValue_JsonIetfVal{
									JsonIetfVal: []byte(`{"name": "Gi0/0/0/0"}`),
								},
							},
						},
					},
				},
			},
			wantContains: []string{"notification", "timestamp", "ok"},
		},
		{
			name:          "nil notifications",
			notifications: nil,
			wantContains:  nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := GetRes{
				Notifications: tt.notifications,
				Timestamp:     1234567890,
				OK:            true,
			}

			got := r.JSON()

			if tt.notifications == nil {
				if got != "" {
					t.Errorf("JSON() with nil notifications = %v, want empty string", got)
				}
				return
			}

			if got == "" {
				t.Errorf("JSON() returned empty string, want non-empty")
				return
			}

			// Verify it's valid JSON
			var result map[string]interface{}
			if err := json.Unmarshal([]byte(got), &result); err != nil {
				t.Errorf("JSON() returned invalid JSON: %v", err)
				return
			}

			// Check for expected fields
			for _, want := range tt.wantContains {
				if _, ok := result[want]; !ok {
					t.Errorf("JSON() missing expected field %q in result: %v", want, got)
				}
			}
		})
	}
}

func TestSetRes_GetValue(t *testing.T) {
	tests := []struct {
		name     string
		response *gnmi.SetResponse
		path     string
		want     interface{}
		wantType string // "string", "int"
	}{
		{
			name: "get operation type",
			response: &gnmi.SetResponse{
				Response: []*gnmi.UpdateResult{
					{
						Op: gnmi.UpdateResult_UPDATE,
						Path: &gnmi.Path{
							Elem: []*gnmi.PathElem{
								{Name: "interfaces"},
							},
						},
					},
				},
				Timestamp: 1234567890,
			},
			path:     "response.response.0.op",
			want:     int64(gnmi.UpdateResult_UPDATE),
			wantType: "int",
		},
		{
			name:     "empty response",
			response: nil,
			path:     "response.timestamp",
			want:     "",
			wantType: "string",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := SetRes{
				Response:  tt.response,
				Timestamp: 1234567890,
				OK:        true,
			}

			result := r.GetValue(tt.path)

			switch tt.wantType {
			case "string":
				got := result.String()
				if got != tt.want {
					t.Errorf("GetValue() = %v, want %v", got, tt.want)
				}
			case "int":
				got := result.Int()
				if got != tt.want {
					t.Errorf("GetValue() = %v, want %v", got, tt.want)
				}
			}
		})
	}
}

func TestSetRes_JSON(t *testing.T) {
	tests := []struct {
		name         string
		response     *gnmi.SetResponse
		wantContains []string
	}{
		{
			name: "valid response",
			response: &gnmi.SetResponse{
				Response: []*gnmi.UpdateResult{
					{
						Op: gnmi.UpdateResult_UPDATE,
						Path: &gnmi.Path{
							Elem: []*gnmi.PathElem{
								{Name: "interfaces"},
							},
						},
					},
				},
				Timestamp: 1234567890,
			},
			wantContains: []string{"response", "timestamp", "ok"},
		},
		{
			name:         "nil response",
			response:     nil,
			wantContains: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := SetRes{
				Response:  tt.response,
				Timestamp: 1234567890,
				OK:        true,
			}

			got := r.JSON()

			if tt.response == nil {
				if got != "" {
					t.Errorf("JSON() with nil response = %v, want empty string", got)
				}
				return
			}

			if got == "" {
				t.Errorf("JSON() returned empty string, want non-empty")
				return
			}

			// Verify it's valid JSON
			var result map[string]interface{}
			if err := json.Unmarshal([]byte(got), &result); err != nil {
				t.Errorf("JSON() returned invalid JSON: %v", err)
				return
			}

			// Check for expected fields
			for _, want := range tt.wantContains {
				if _, ok := result[want]; !ok {
					t.Errorf("JSON() missing expected field %q in result: %v", want, got)
				}
			}
		})
	}
}

// TestGetRes_GetValue_Nested tests nested JSON value extraction
func TestGetRes_GetValue_Nested(t *testing.T) {
	// Create a notification with nested JSON data
	notification := &gnmi.Notification{
		Timestamp: 1234567890,
		Update: []*gnmi.Update{
			{
				Path: &gnmi.Path{
					Elem: []*gnmi.PathElem{
						{Name: "interfaces"},
						{Name: "interface", Key: map[string]string{"name": "Gi0/0/0/0"}},
						{Name: "state"},
					},
				},
				Val: &gnmi.TypedValue{
					Value: &gnmi.TypedValue_JsonIetfVal{
						JsonIetfVal: []byte(`{
							"name": "Gi0/0/0/0",
							"mtu": 9000,
							"enabled": true,
							"counters": {
								"in-octets": 1234567890,
								"out-octets": 987654321
							}
						}`),
					},
				},
			},
		},
	}

	r := GetRes{
		Notifications: []*gnmi.Notification{notification},
		Timestamp:     1234567890,
		OK:            true,
	}

	// Test accessing timestamp
	timestamp := r.GetValue("notification.0.timestamp")
	if timestamp.Int() != 1234567890 {
		t.Errorf("Failed to get timestamp, got %v, want 1234567890", timestamp.Int())
	}

	// Test accessing path elements
	pathName := r.GetValue("notification.0.update.0.path.elem.0.name")
	if pathName.String() != "interfaces" {
		t.Errorf("Failed to get path name, got %v, want interfaces", pathName.String())
	}
}

// Example test demonstrating the usage pattern
func ExampleGetRes_GetValue() {
	// This example shows how to use GetValue with gjson paths
	notification := &gnmi.Notification{
		Timestamp: 1234567890,
		Update: []*gnmi.Update{
			{
				Path: &gnmi.Path{
					Elem: []*gnmi.PathElem{
						{Name: "system"},
						{Name: "config"},
					},
				},
				Val: &gnmi.TypedValue{
					Value: &gnmi.TypedValue_JsonIetfVal{
						JsonIetfVal: []byte(`{"hostname": "router1"}`),
					},
				},
			},
		},
	}

	res := GetRes{
		Notifications: []*gnmi.Notification{notification},
		Timestamp:     1234567890,
		OK:            true,
	}

	// Get the JSON value
	jsonValue := res.GetValue("notification.0.update.0.val.json_ietf_val").String()
	_ = jsonValue // Use the value
}

// Suppress unused import warning
var _ = anypb.Any{}
