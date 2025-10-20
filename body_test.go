// SPDX-License-Identifier: MPL-2.0
// Copyright (c) 2025 Daniel Schmidt

package gnmi

import (
	"strings"
	"testing"

	"github.com/tidwall/gjson"
)

// TestBodySet tests basic Set operation
func TestBodySet(t *testing.T) {
	tests := []struct {
		name     string
		path     string
		value    interface{}
		wantJSON string
	}{
		{
			name:     "set string value",
			path:     "name",
			value:    "eth0",
			wantJSON: `{"name":"eth0"}`,
		},
		{
			name:     "set boolean value",
			path:     "enabled",
			value:    true,
			wantJSON: `{"enabled":true}`,
		},
		{
			name:     "set integer value",
			path:     "mtu",
			value:    1500,
			wantJSON: `{"mtu":1500}`,
		},
		{
			name:     "set nested value",
			path:     "config.hostname",
			value:    "router1",
			wantJSON: `{"config":{"hostname":"router1"}}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			body := Body{}.Set(tt.path, tt.value)
			json, err := body.String()
			if err != nil {
				t.Fatalf("Expected no error, got: %v", err)
			}
			if json != tt.wantJSON {
				t.Errorf("Expected JSON %s, got %s", tt.wantJSON, json)
			}
		})
	}
}

// TestBodySetChaining tests method chaining
func TestBodySetChaining(t *testing.T) {
	body := Body{}.
		Set("name", "eth0").
		Set("enabled", true).
		Set("mtu", 1500)

	json, err := body.String()
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	// Verify all values are present
	if !strings.Contains(json, `"name":"eth0"`) {
		t.Errorf("Expected JSON to contain name field")
	}
	if !strings.Contains(json, `"enabled":true`) {
		t.Errorf("Expected JSON to contain enabled field")
	}
	if !strings.Contains(json, `"mtu":1500`) {
		t.Errorf("Expected JSON to contain mtu field")
	}
}

// TestBodyDelete tests Delete operation
func TestBodyDelete(t *testing.T) {
	// Build initial body
	body := Body{}.
		Set("name", "eth0").
		Set("description", "temp").
		Set("enabled", true).
		Delete("description")

	json, err := body.String()
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	// Verify description was deleted
	if strings.Contains(json, "description") {
		t.Errorf("Expected description to be deleted, got: %s", json)
	}

	// Verify other fields remain
	if !strings.Contains(json, `"name":"eth0"`) {
		t.Errorf("Expected name field to remain")
	}
	if !strings.Contains(json, `"enabled":true`) {
		t.Errorf("Expected enabled field to remain")
	}
}

// TestBodyErrorPropagation tests that first error is captured and subsequent operations are no-ops
func TestBodyErrorPropagation(t *testing.T) {
	// Test that first error is captured and subsequent operations are no-ops
	body := Body{}.
		Set("valid.path", "value1").
		Set("", "invalid-empty-path"). // This should error
		Set("another.path", "value2")  // This should be a no-op

	_, err := body.String()
	if err == nil {
		t.Fatal("Expected error from empty path, got nil")
	}
	if !strings.Contains(err.Error(), "Set") {
		t.Errorf("Expected error message to contain 'Set', got: %v", err)
	}

	// Verify only the valid path before error is set
	json, _ := body.String() //nolint:errcheck // Error intentionally ignored in test
	if !strings.Contains(json, "value1") {
		t.Errorf("Expected JSON to contain value1 (set before error)")
	}
	if strings.Contains(json, "value2") {
		t.Errorf("Expected JSON to NOT contain value2 (set after error)")
	}
}

// TestBodyErr tests the Err() method
func TestBodyErr(t *testing.T) {
	// Test successful case
	body1 := Body{}.Set("name", "value")
	if err := body1.Err(); err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	// Test error case
	body2 := Body{}.Set("", "value") // Invalid path
	if err := body2.Err(); err == nil {
		t.Fatal("Expected error, got nil")
	}
}

// TestBodyRes tests the Res() method
func TestBodyRes(t *testing.T) {
	// Test successful case
	body1 := Body{}.Set("name", "eth0")
	res := body1.Res()
	if res == "" {
		t.Fatal("Expected non-empty result")
	}

	// Verify result can be queried with gjson
	name := gjson.Get(res, "name").String()
	if name != "eth0" {
		t.Errorf("Expected name 'eth0', got '%s'", name)
	}

	// Test error case - should return empty string
	body2 := Body{}.Set("", "value") // Invalid path
	res2 := body2.Res()
	if res2 != "" {
		t.Fatalf("Expected empty string on error, got: %s", res2)
	}
}

// TestBodyString tests String() returns error
func TestBodyString(t *testing.T) {
	// Test successful case
	body := Body{}.Set("config.name", "router1")
	json, err := body.String()
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}
	if !strings.Contains(json, "router1") {
		t.Errorf("Expected JSON to contain 'router1', got: %s", json)
	}

	// Test error case
	body2 := Body{}.Set("", "value")
	_, err2 := body2.String()
	if err2 == nil {
		t.Fatal("Expected error, got nil")
	}
}

// TestBodyBytes tests Bytes() returns error
func TestBodyBytes(t *testing.T) {
	// Test successful case
	body := Body{}.Set("name", "value")
	bytes, err := body.Bytes()
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}
	if len(bytes) == 0 {
		t.Fatal("Expected non-empty bytes")
	}

	// Verify bytes contain expected value
	if !strings.Contains(string(bytes), "value") {
		t.Errorf("Expected bytes to contain 'value', got: %s", string(bytes))
	}

	// Test error case
	body2 := Body{}.Set("", "value")
	bytes2, err2 := body2.Bytes()
	if err2 == nil {
		t.Fatal("Expected error, got nil")
	}
	if bytes2 != nil {
		t.Fatalf("Expected nil bytes on error, got: %v", bytes2)
	}
}

// TestBodyChainingWithErrors tests that error short-circuits further operations
func TestBodyChainingWithErrors(t *testing.T) {
	// Test that error short-circuits further operations
	body := Body{}.
		Set("valid", "value1").
		Set("", "triggers-error").
		Set("should-be-skipped", "value2").
		Delete("should-also-be-skipped")

	json, err := body.String()
	if err == nil {
		t.Fatal("Expected error from invalid path")
	}

	// JSON should not contain the values added after error
	if strings.Contains(json, "should-be-skipped") {
		t.Errorf("Operations after error should be no-ops, but JSON contains skipped value: %s", json)
	}
	if strings.Contains(json, "value2") {
		t.Errorf("Operations after error should be no-ops, but JSON contains value2: %s", json)
	}

	// Should contain value set before error
	if !strings.Contains(json, "value1") {
		t.Errorf("Expected JSON to contain value1 (set before error)")
	}
}

// TestBodyDeleteError tests Delete error handling
func TestBodyDeleteError(t *testing.T) {
	// sjson.Delete doesn't typically error on valid paths
	// Test that delete preserves error state
	body := Body{}.
		Set("", "triggers-error"). // Set error first
		Delete("some.path")        // This should be no-op

	_, err := body.String()
	if err == nil {
		t.Fatal("Expected error to be preserved")
	}
	if !strings.Contains(err.Error(), "Set") {
		t.Errorf("Expected original Set error, got: %v", err)
	}
}

// TestBodyImmutability tests that Body operations are immutable
func TestBodyImmutability(t *testing.T) {
	body1 := Body{}.Set("name", "value1")

	// Create new body from body1
	body2 := body1.Set("name", "value2")

	json1, err1 := body1.String()
	if err1 != nil {
		t.Fatalf("Expected no error for body1, got: %v", err1)
	}

	json2, err2 := body2.String()
	if err2 != nil {
		t.Fatalf("Expected no error for body2, got: %v", err2)
	}

	// Verify body1 is unchanged
	if !strings.Contains(json1, "value1") {
		t.Errorf("Expected body1 to contain value1, got: %s", json1)
	}

	// Verify body2 has new value
	if !strings.Contains(json2, "value2") {
		t.Errorf("Expected body2 to contain value2, got: %s", json2)
	}
}

// TestBodyComplexJSON tests building complex nested structures
func TestBodyComplexJSON(t *testing.T) {
	body := Body{}.
		Set("config.name", "GigabitEthernet0/0/0/0").
		Set("config.description", "WAN Interface").
		Set("config.enabled", true).
		Set("config.mtu", 9000).
		Set("state.counters.in-pkts", 12345).
		Set("state.counters.out-pkts", 67890)

	json, err := body.String()
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	// Verify structure with gjson
	if gjson.Get(json, "config.name").String() != "GigabitEthernet0/0/0/0" {
		t.Errorf("Expected config.name to be 'GigabitEthernet0/0/0/0'")
	}
	if gjson.Get(json, "config.enabled").Bool() != true {
		t.Errorf("Expected config.enabled to be true")
	}
	if gjson.Get(json, "config.mtu").Int() != 9000 {
		t.Errorf("Expected config.mtu to be 9000")
	}
	if gjson.Get(json, "state.counters.in-pkts").Int() != 12345 {
		t.Errorf("Expected state.counters.in-pkts to be 12345")
	}
}

// TestBodyEmptyBody tests behavior with empty body
func TestBodyEmptyBody(t *testing.T) {
	body := Body{}

	json, err := body.String()
	if err != nil {
		t.Fatalf("Expected no error for empty body, got: %v", err)
	}

	if json != "" {
		t.Errorf("Expected empty string for empty body, got: %s", json)
	}
}

// BenchmarkBodySet benchmarks the Set operation
func BenchmarkBodySet(b *testing.B) {
	b.Run("single set", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			_ = Body{}.Set("name", "value")
		}
	})

	b.Run("nested set", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			_ = Body{}.Set("config.interface.name", "GigabitEthernet0/0/0/0")
		}
	})

	b.Run("multiple sets", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			_ = Body{}.
				Set("name", "value1").
				Set("description", "value2").
				Set("enabled", true).
				Set("mtu", 9000)
		}
	})
}

// BenchmarkBodyDelete benchmarks the Delete operation
func BenchmarkBodyDelete(b *testing.B) {
	b.Run("single delete", func(b *testing.B) {
		body := Body{}.Set("name", "value").Set("other", "data")
		b.ResetTimer()
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			_ = body.Delete("name")
		}
	})

	b.Run("nested delete", func(b *testing.B) {
		body := Body{}.
			Set("config.interface.name", "test").
			Set("config.interface.enabled", true)
		b.ResetTimer()
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			_ = body.Delete("config.interface.name")
		}
	})
}

// BenchmarkBodyString benchmarks the String operation
func BenchmarkBodyString(b *testing.B) {
	b.Run("empty body", func(b *testing.B) {
		body := Body{}
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			_, _ = body.String() //nolint:errcheck // Error intentionally ignored in test
		}
	})

	b.Run("simple body", func(b *testing.B) {
		body := Body{}.
			Set("name", "test").
			Set("enabled", true)
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			_, _ = body.String() //nolint:errcheck // Error intentionally ignored in test
		}
	})

	b.Run("complex nested body", func(b *testing.B) {
		body := Body{}.
			Set("config.name", "GigabitEthernet0/0/0/0").
			Set("config.description", "WAN Interface").
			Set("config.enabled", true).
			Set("config.mtu", 9000).
			Set("state.counters.in-pkts", 12345).
			Set("state.counters.out-pkts", 67890)
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			_, _ = body.String() //nolint:errcheck // Error intentionally ignored in test
		}
	})
}

// BenchmarkBodyBytes benchmarks the Bytes operation
func BenchmarkBodyBytes(b *testing.B) {
	body := Body{}.
		Set("config.name", "test").
		Set("config.enabled", true).
		Set("config.mtu", 1500)

	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_, _ = body.Bytes() //nolint:errcheck // Error intentionally ignored in test
	}
}

// BenchmarkBodyBuildAndSerialize benchmarks the full build + serialize flow
func BenchmarkBodyBuildAndSerialize(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		body := Body{}.
			Set("config.name", "GigabitEthernet0/0/0/0").
			Set("config.description", "WAN Interface").
			Set("config.enabled", true).
			Set("config.mtu", 9000)
		_, _ = body.String() //nolint:errcheck // Error intentionally ignored in test
	}
}
