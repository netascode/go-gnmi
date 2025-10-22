// SPDX-License-Identifier: MPL-2.0
// Copyright (c) 2025 Daniel Schmidt

package gnmi

import (
	"testing"
	"time"
)

func TestTimeout(t *testing.T) {
	tests := []struct {
		name     string
		duration time.Duration
		want     time.Duration
	}{
		{
			name:     "30 second timeout",
			duration: 30 * time.Second,
			want:     30 * time.Second,
		},
		{
			name:     "2 minute timeout",
			duration: 2 * time.Minute,
			want:     2 * time.Minute,
		},
		{
			name:     "zero timeout",
			duration: 0,
			want:     0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := &Req{}
			modifier := Timeout(tt.duration)
			modifier(req)

			if req.Timeout != tt.want {
				t.Errorf("Timeout() timeout = %v, want %v", req.Timeout, tt.want)
			}
		})
	}
}

func TestGetEncoding(t *testing.T) {
	tests := []struct {
		name     string
		encoding string
		want     string
	}{
		{
			name:     "json_ietf encoding",
			encoding: "json_ietf",
			want:     "json_ietf",
		},
		{
			name:     "json encoding",
			encoding: "json",
			want:     "json",
		},
		{
			name:     "proto encoding",
			encoding: "proto",
			want:     "proto",
		},
		{
			name:     "ascii encoding",
			encoding: "ascii",
			want:     "ascii",
		},
		{
			name:     "bytes encoding",
			encoding: "bytes",
			want:     "bytes",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := &Req{}
			modifier := GetEncoding(tt.encoding)
			modifier(req)

			if req.Encoding != tt.want {
				t.Errorf("GetEncoding() encoding = %v, want %v", req.Encoding, tt.want)
			}
		})
	}
}

func TestMultipleModifiers(t *testing.T) {
	// Test that multiple modifiers can be applied
	req := &Req{}

	// Apply timeout modifier
	timeoutMod := Timeout(30 * time.Second)
	timeoutMod(req)

	// Apply encoding modifier
	encodingMod := GetEncoding("proto")
	encodingMod(req)

	// Verify both modifiers were applied
	if req.Timeout != 30*time.Second {
		t.Errorf("Timeout not applied correctly: got %v, want %v", req.Timeout, 30*time.Second)
	}
	if req.Encoding != "proto" {
		t.Errorf("Encoding not applied correctly: got %v, want %v", req.Encoding, "proto")
	}
}

func TestModifierOverwrite(t *testing.T) {
	// Test that later modifiers overwrite earlier ones (last one wins)
	req := &Req{}

	// Apply first timeout
	timeoutMod1 := Timeout(30 * time.Second)
	timeoutMod1(req)

	// Apply second timeout (should overwrite)
	timeoutMod2 := Timeout(60 * time.Second)
	timeoutMod2(req)

	if req.Timeout != 60*time.Second {
		t.Errorf("Second timeout should overwrite first: got %v, want %v", req.Timeout, 60*time.Second)
	}

	// Same test for encoding
	encodingMod1 := GetEncoding("json")
	encodingMod1(req)

	encodingMod2 := GetEncoding("proto")
	encodingMod2(req)

	if req.Encoding != "proto" {
		t.Errorf("Second encoding should overwrite first: got %v, want %v", req.Encoding, "proto")
	}
}

func TestSetOperation(t *testing.T) {
	tests := []struct {
		name      string
		opType    SetOperationType
		path      string
		value     string
		encoding  string
		wantType  SetOperationType
		wantPath  string
		wantValue string
		wantEnc   string
	}{
		{
			name:      "update operation",
			opType:    OperationUpdate,
			path:      "/interfaces/interface[name=Gi0/0/0/0]/config",
			value:     `{"enabled": true}`,
			encoding:  "json_ietf",
			wantType:  OperationUpdate,
			wantPath:  "/interfaces/interface[name=Gi0/0/0/0]/config",
			wantValue: `{"enabled": true}`,
			wantEnc:   "json_ietf",
		},
		{
			name:      "replace operation",
			opType:    OperationReplace,
			path:      "/system/config",
			value:     `{"hostname": "router1"}`,
			encoding:  "json",
			wantType:  OperationReplace,
			wantPath:  "/system/config",
			wantValue: `{"hostname": "router1"}`,
			wantEnc:   "json",
		},
		{
			name:      "delete operation",
			opType:    OperationDelete,
			path:      "/interfaces/interface[name=Gi0/0/0/1]",
			value:     "",
			encoding:  "",
			wantType:  OperationDelete,
			wantPath:  "/interfaces/interface[name=Gi0/0/0/1]",
			wantValue: "",
			wantEnc:   "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			op := SetOperation{
				OperationType: tt.opType,
				Path:          tt.path,
				Value:         tt.value,
				Encoding:      tt.encoding,
			}

			if op.OperationType != tt.wantType {
				t.Errorf("OperationType = %v, want %v", op.OperationType, tt.wantType)
			}
			if op.Path != tt.wantPath {
				t.Errorf("Path = %v, want %v", op.Path, tt.wantPath)
			}
			if op.Value != tt.wantValue {
				t.Errorf("Value = %v, want %v", op.Value, tt.wantValue)
			}
			if op.Encoding != tt.wantEnc {
				t.Errorf("Encoding = %v, want %v", op.Encoding, tt.wantEnc)
			}
		})
	}
}
