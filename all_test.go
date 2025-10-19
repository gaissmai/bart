package bart

import (
	"fmt"
	"iter"
	"net/netip"
	"testing"
)

// this file contains init functions and helpers for test functions

var mpa = netip.MustParseAddr

var mpp = func(s string) netip.Prefix {
	pfx := netip.MustParsePrefix(s)
	if pfx == pfx.Masked() {
		return pfx
	}
	panic(fmt.Sprintf("%s is not canonicalized as %s", s, pfx.Masked()))
}

// workLoadN to adjust loops for tests with -short
func workLoadN() int {
	if testing.Short() {
		return 100
	}
	return 1_000
}

// #########################################################

// tests for deep copies with Cloner interface
type MyInt int

// implement the Cloner interface
func (i *MyInt) Clone() *MyInt {
	a := *i
	return &a
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

func mustPanic(t *testing.T, name string, fn func()) {
	t.Helper()
	defer func() {
		if recover() == nil {
			t.Fatalf("%s must panic", name)
		}
	}()
	fn()
}

func noPanic(t *testing.T, name string, fn func()) {
	t.Helper()
	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("%s panicked: %v", name, r)
		}
	}()
	fn()
}

func noPanicRangeOverFunc[V any](t *testing.T, name string, fn any) {
	t.Helper()
	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("%s panicked: %v", name, r)
		}
	}()

	switch iterFunc := fn.(type) {
	case func() iter.Seq[netip.Prefix]:
		for range iterFunc() {
		}
	case func() iter.Seq2[netip.Prefix, V]:
		for range iterFunc() {
		}
	case func(netip.Prefix) iter.Seq[netip.Prefix]:
		pfx := mpp("1.2.3.4/32")
		for range iterFunc(pfx) {
		}
	case func(netip.Prefix) iter.Seq2[netip.Prefix, V]:
		pfx := mpp("1.2.3.4/32")
		for range iterFunc(pfx) {
		}
	default:
		t.Fatalf("%s unknown iter function: %T", name, iterFunc)
	}
}
