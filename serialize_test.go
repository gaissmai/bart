// Copyright (c) 2025 Karl Gaissmaier
// SPDX-License-Identifier: MIT

package bart

import (
	"encoding/json"
	"net/netip"
	"strings"
	"testing"
)

func TestTableMarshalText(t *testing.T) {
	tests := []struct {
		name         string
		expectedData map[netip.Prefix]string
	}{
		{
			name:         "empty",
			expectedData: map[netip.Prefix]string{},
		},
		{
			name: "with_data",
			expectedData: map[netip.Prefix]string{
				netip.MustParsePrefix("192.168.1.0/24"): "test1",
				netip.MustParsePrefix("10.0.0.0/8"):     "test2",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			table := &Table[string]{}

			// Insert test data
			for prefix, value := range tt.expectedData {
				table.Insert(prefix, value)
			}

			data, err := table.MarshalText()
			if err != nil {
				t.Errorf("MarshalText failed: %v", err)
			}

			if len(tt.expectedData) > 0 && len(data) == 0 {
				t.Error("Expected non-empty marshaled text")
			}

			// Check that all expected values appear in marshaled text
			text := string(data)
			for _, value := range tt.expectedData {
				if !strings.Contains(text, value) {
					t.Errorf("Marshaled text doesn't contain expected value: %s", value)
				}
			}
		})
	}
}

func TestTableMarshalJSON(t *testing.T) {
	tests := []struct {
		name         string
		expectedData map[netip.Prefix]any
	}{
		{
			name:         "empty",
			expectedData: map[netip.Prefix]any{},
		},
		{
			name: "string_values",
			expectedData: map[netip.Prefix]any{
				netip.MustParsePrefix("192.168.1.0/24"): "net1",
				netip.MustParsePrefix("10.0.0.0/8"):     "net2",
			},
		},
		{
			name: "mixed_values",
			expectedData: map[netip.Prefix]any{
				netip.MustParsePrefix("192.168.1.0/24"): "string",
				netip.MustParsePrefix("10.0.0.0/8"):     42,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			table := &Table[any]{}

			// Insert test data
			for prefix, value := range tt.expectedData {
				table.Insert(prefix, value)
			}

			jsonData, err := json.Marshal(table)
			if err != nil {
				t.Errorf("JSON marshaling failed: %v", err)
			}

			if len(jsonData) == 0 {
				t.Error("Expected valid JSON")
			}

			// Should be valid JSON
			var result interface{}
			if err := json.Unmarshal(jsonData, &result); err != nil {
				t.Errorf("Invalid JSON produced: %v", err)
			}
		})
	}
}

func TestTableDumpList4(t *testing.T) {
	tests := []struct {
		name         string
		expectedData map[netip.Prefix]string
		expectItems  int
	}{
		{
			name:         "empty",
			expectedData: map[netip.Prefix]string{},
			expectItems:  0,
		},
		{
			name: "single_ipv4",
			expectedData: map[netip.Prefix]string{
				netip.MustParsePrefix("192.168.1.0/24"): "lan",
			},
			expectItems: 1,
		},
		{
			name: "multiple_ipv4",
			expectedData: map[netip.Prefix]string{
				netip.MustParsePrefix("192.168.1.0/24"): "lan",
				netip.MustParsePrefix("10.0.0.0/8"):     "private",
			},
			expectItems: 2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			table := &Table[string]{}

			// Insert test data
			for prefix, value := range tt.expectedData {
				table.Insert(prefix, value)
			}

			dumpList := table.DumpList4()

			// Count total nodes in the tree (including nested)
			totalNodes := countDumpListNodes(dumpList)
			if totalNodes != tt.expectItems {
				t.Errorf("DumpList4() total nodes (%d) does not match expected (%d)", totalNodes, tt.expectItems)
			}

			// Verify all nodes are IPv4
			verifyAllIPv4Nodes(t, dumpList)
		})
	}
}

func TestTableDumpList6(t *testing.T) {
	tests := []struct {
		name         string
		expectedData map[netip.Prefix]string
		expectItems  int
	}{
		{
			name:         "empty",
			expectedData: map[netip.Prefix]string{},
			expectItems:  0,
		},
		{
			name: "single_ipv6",
			expectedData: map[netip.Prefix]string{
				netip.MustParsePrefix("2001:db8::/32"): "doc",
			},
			expectItems: 1,
		},
		{
			name: "multiple_ipv6",
			expectedData: map[netip.Prefix]string{
				netip.MustParsePrefix("2001:db8::/32"): "doc",
				netip.MustParsePrefix("fe80::/10"):     "link-local",
			},
			expectItems: 2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			table := &Table[string]{}

			// Insert test data
			for prefix, value := range tt.expectedData {
				table.Insert(prefix, value)
			}

			dumpList := table.DumpList6()

			// Count total nodes in the tree (including nested)
			totalNodes := countDumpListNodes(dumpList)
			if totalNodes != tt.expectItems {
				t.Errorf("DumpList6() total nodes (%d) does not match expected (%d)", totalNodes, tt.expectItems)
			}

			// Verify all nodes are IPv6
			verifyAllIPv6Nodes(t, dumpList)
		})
	}
}

func TestFastMarshalText(t *testing.T) {
	tests := []struct {
		name         string
		expectedData map[netip.Prefix]string
	}{
		{
			name:         "empty",
			expectedData: map[netip.Prefix]string{},
		},
		{
			name: "with_data",
			expectedData: map[netip.Prefix]string{
				netip.MustParsePrefix("192.168.1.0/24"): "ipv4",
				netip.MustParsePrefix("2001:db8::/32"):  "ipv6",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fast := &Fast[string]{}

			// Insert test data
			for prefix, value := range tt.expectedData {
				fast.Insert(prefix, value)
			}

			data, err := fast.MarshalText()
			if err != nil {
				t.Errorf("MarshalText failed: %v", err)
			}

			if len(tt.expectedData) > 0 && len(data) == 0 {
				t.Error("Expected non-empty marshaled text")
			}

			// Check that all expected values appear in marshaled text
			text := string(data)
			for _, value := range tt.expectedData {
				if !strings.Contains(text, value) {
					t.Errorf("Marshaled text doesn't contain expected value: %s", value)
				}
			}
		})
	}
}

func TestFastMarshalJSON(t *testing.T) {
	tests := []struct {
		name         string
		expectedData map[netip.Prefix]any
	}{
		{
			name:         "empty",
			expectedData: map[netip.Prefix]any{},
		},
		{
			name: "ipv4_and_ipv6",
			expectedData: map[netip.Prefix]any{
				netip.MustParsePrefix("192.168.1.0/24"): "ipv4",
				netip.MustParsePrefix("2001:db8::/32"):  "ipv6",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fast := &Fast[any]{}

			// Insert test data
			for prefix, value := range tt.expectedData {
				fast.Insert(prefix, value)
			}

			jsonData, err := json.Marshal(fast)
			if err != nil {
				t.Errorf("JSON marshaling failed: %v", err)
			}

			if len(jsonData) == 0 {
				t.Error("Expected valid JSON")
			}

			// Should be valid JSON
			var result interface{}
			if err := json.Unmarshal(jsonData, &result); err != nil {
				t.Errorf("Invalid JSON produced: %v", err)
			}
		})
	}
}

func TestFastDumpList4(t *testing.T) {
	fast := &Fast[string]{}

	// Insert test data
	prefixes := []netip.Prefix{
		netip.MustParsePrefix("192.168.1.0/24"),
		netip.MustParsePrefix("10.0.0.0/8"),
	}
	values := []string{"lan", "private"}

	for i, prefix := range prefixes {
		fast.Insert(prefix, values[i])
	}

	dumpList := fast.DumpList4()

	// Count total IPv4 nodes
	totalNodes := countDumpListNodes(dumpList)
	if totalNodes != 2 {
		t.Errorf("DumpList4() total nodes (%d) does not match expected (2)", totalNodes)
	}

	// Verify all nodes are IPv4
	verifyAllIPv4Nodes(t, dumpList)
}

func TestFastDumpList6(t *testing.T) {
	fast := &Fast[string]{}

	// Insert test data
	prefixes := []netip.Prefix{
		netip.MustParsePrefix("2001:db8::/32"),
		netip.MustParsePrefix("fe80::/10"),
	}
	values := []string{"doc", "link-local"}

	for i, prefix := range prefixes {
		fast.Insert(prefix, values[i])
	}

	dumpList := fast.DumpList6()

	// Count total IPv6 nodes
	totalNodes := countDumpListNodes(dumpList)
	if totalNodes != 2 {
		t.Errorf("DumpList6() total nodes (%d) does not match expected (2)", totalNodes)
	}

	// Verify all nodes are IPv6
	verifyAllIPv6Nodes(t, dumpList)
}

func TestLiteMarshalText(t *testing.T) {
	expectedPrefix := netip.MustParsePrefix("192.168.1.0/24")
	lite := &Lite{}

	// Add test data
	lite.Insert(expectedPrefix)

	data, err := lite.MarshalText()
	if err != nil {
		t.Errorf("MarshalText failed: %v", err)
	}

	if len(data) == 0 {
		t.Error("Expected non-empty marshaled text")
	}

	// Should contain the expected prefix
	text := string(data)
	if !strings.Contains(text, expectedPrefix.String()) {
		t.Errorf("Marshaled text doesn't contain expected prefix: %s", expectedPrefix)
	}
}

// Nil tests for robustness - NOTE: Lite does NOT have nil-receiver safety
func TestNilTableSerialization(t *testing.T) {
	var table *Table[string] = nil

	// MarshalText() should not panic
	data, err := table.MarshalText()
	if err != nil {
		t.Errorf("Nil Table MarshalText() should not error: %v", err)
	}
	if len(data) != 0 {
		t.Errorf("Nil Table MarshalText() should return empty data, got %d bytes", len(data))
	}

	// DumpList4() should not panic
	dumpList4 := table.DumpList4()
	if len(dumpList4) != 0 {
		t.Errorf("Nil Table DumpList4() should return empty slice, got %d items", len(dumpList4))
	}

	// DumpList6() should not panic
	dumpList6 := table.DumpList6()
	if len(dumpList6) != 0 {
		t.Errorf("Nil Table DumpList6() should return empty slice, got %d items", len(dumpList6))
	}
}

func TestNilFastSerialization(t *testing.T) {
	var fast *Fast[string] = nil

	// MarshalText() should not panic
	data, err := fast.MarshalText()
	if err != nil {
		t.Errorf("Nil Fast MarshalText() should not error: %v", err)
	}
	if len(data) != 0 {
		t.Errorf("Nil Fast MarshalText() should return empty data, got %d bytes", len(data))
	}

	// DumpList4() should not panic
	dumpList4 := fast.DumpList4()
	if len(dumpList4) != 0 {
		t.Errorf("Nil Fast DumpList4() should return empty slice, got %d items", len(dumpList4))
	}

	// DumpList6() should not panic
	dumpList6 := fast.DumpList6()
	if len(dumpList6) != 0 {
		t.Errorf("Nil Fast DumpList6() should return empty slice, got %d items", len(dumpList6))
	}
}

// Helper functions
func countDumpListNodes[V any](nodes []DumpListNode[V]) int {
	count := len(nodes)
	for _, node := range nodes {
		count += countDumpListNodes(node.Subnets)
	}
	return count
}

func verifyAllIPv4Nodes[V any](t *testing.T, nodes []DumpListNode[V]) {
	for i, node := range nodes {
		if !node.CIDR.Addr().Is4() {
			t.Errorf("Node %d is not IPv4 prefix: %v", i, node.CIDR)
		}
		// Recursively check subnets
		verifyAllIPv4Nodes(t, node.Subnets)
	}
}

func verifyAllIPv6Nodes[V any](t *testing.T, nodes []DumpListNode[V]) {
	for i, node := range nodes {
		if !node.CIDR.Addr().Is6() {
			t.Errorf("Node %d is not IPv6 prefix: %v", i, node.CIDR)
		}
		// Recursively check subnets
		verifyAllIPv6Nodes(t, node.Subnets)
	}
}
