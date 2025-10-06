// Copyright (c) 2025 Karl Gaissmaier
// SPDX-License-Identifier: MIT

package bart

import (
	"net/netip"
	"testing"

	"github.com/gaissmai/bart/internal/nodes"
)

// ---- Test helper types ----

// routeEntry represents a realistic routing table entry that needs deep cloning
type routeEntry struct {
	nextHop    netip.Addr
	exitIF     string
	attributes map[string]int
}

// Clone implements Cloner[*routeEntry] for deep cloning of routing entries
func (r *routeEntry) Clone() *routeEntry {
	if r == nil {
		return nil
	}

	clone := &routeEntry{
		nextHop:    r.nextHop,
		exitIF:     r.exitIF,
		attributes: make(map[string]int, len(r.attributes)),
	}

	// Deep clone the attributes map
	for k, v := range r.attributes {
		clone.attributes[k] = v
	}

	return clone
}

// routeEntryNonCloner is the same struct but without Clone method for testing non-cloner behavior
type routeEntryNonCloner struct {
	nextHop    netip.Addr
	exitIF     string
	attributes map[string]int
}

// ---- cloneFnFactory / cloneVal / copyVal ----

func TestCloneFnFactory_WithCloner(t *testing.T) {
	t.Parallel()
	fn := cloneFnFactory[*routeEntry]()
	if fn == nil {
		t.Fatalf("expected non-nil clone func when V implements Cloner[V]")
	}

	in := &routeEntry{
		nextHop:    netip.MustParseAddr("10.0.0.1"),
		exitIF:     "eth0",
		attributes: map[string]int{"metric": 100, "preference": 10},
	}

	out := fn(in)
	expected := in.Clone()
	if out.nextHop != expected.nextHop || out.exitIF != expected.exitIF {
		t.Fatalf("expected cloned route with nextHop=%v exitIF=%s, got nextHop=%v exitIF=%s",
			expected.nextHop, expected.exitIF, out.nextHop, out.exitIF)
	}
	if out.attributes["metric"] != expected.attributes["metric"] {
		t.Fatalf("expected cloned attributes, got metric=%d", out.attributes["metric"])
	}
}

func TestCloneFnFactory_WithoutCloner(t *testing.T) {
	t.Parallel()
	fn := cloneFnFactory[*routeEntryNonCloner]()
	if fn != nil {
		t.Fatalf("expected nil clone func when V does not implement Cloner[V]")
	}
}

func TestCloneVal_WithCloner(t *testing.T) {
	t.Parallel()
	in := &routeEntry{
		nextHop:    netip.MustParseAddr("192.168.1.1"),
		exitIF:     "wlan0",
		attributes: map[string]int{"cost": 50, "bandwidth": 1000},
	}

	got := cloneVal[*routeEntry](in)
	want := in.Clone()
	if got.nextHop != want.nextHop || got.exitIF != want.exitIF {
		t.Fatalf("expected cloned route, got different values")
	}

	// Verify independence - modify clone shouldn't affect original
	got.attributes["cost"] = 999
	if in.attributes["cost"] != 50 {
		t.Fatalf("modifying clone affected original")
	}
}

func TestCloneVal_WithoutCloner(t *testing.T) {
	t.Parallel()
	in := &routeEntryNonCloner{
		nextHop: netip.MustParseAddr("172.16.0.1"),
		exitIF:  "eth1",
	}

	got := cloneVal[*routeEntryNonCloner](in)
	if got != in {
		t.Fatalf("expected passthrough for non-cloner, got different instance")
	}
}

func TestCopyVal_Passthrough(t *testing.T) {
	t.Parallel()
	in := &routeEntry{
		nextHop: netip.MustParseAddr("203.0.113.1"),
		exitIF:  "tun0",
	}

	if got := copyVal[*routeEntry](in); got != in {
		t.Fatalf("copyVal should return input; want same instance")
	}
}

// ---- leafNode.cloneLeaf / fringeNode.cloneFringe ----

func TestCloneLeaf_NilCloneFn(t *testing.T) {
	t.Parallel()
	prefix := netip.MustParsePrefix("192.0.2.0/24")
	route := &routeEntry{
		nextHop:    netip.MustParseAddr("192.0.2.1"),
		exitIF:     "eth0",
		attributes: map[string]int{"metric": 10},
	}

	l := &nodes.LeafNode[*routeEntry]{Prefix: prefix, Value: route}
	got := l.CloneLeaf(nil)

	if got == l {
		t.Fatalf("expected new leaf instance")
	}
	if got.Prefix != l.Prefix {
		t.Fatalf("prefix must be copied as-is: want %v got %v", l.Prefix, got.Prefix)
	}
	if got.Value != l.Value {
		t.Fatalf("value must be copied when cloneFn is nil")
	}
}

func TestCloneLeaf_WithCloneFn(t *testing.T) {
	t.Parallel()
	prefix := netip.MustParsePrefix("198.51.100.0/24")
	route := &routeEntry{
		nextHop:    netip.MustParseAddr("198.51.100.1"),
		exitIF:     "ppp0",
		attributes: map[string]int{"mtu": 1500, "delay": 10},
	}

	l := &nodes.LeafNode[*routeEntry]{Prefix: prefix, Value: route}
	cloneFn := func(v *routeEntry) *routeEntry { return v.Clone() }
	got := l.CloneLeaf(cloneFn)

	expected := l.Value.Clone()
	if got.Value.nextHop != expected.nextHop || got.Value.exitIF != expected.exitIF {
		t.Fatalf("expected leaf value to be cloned")
	}
	if got.Value == l.Value {
		t.Fatalf("cloned value should be different instance")
	}
	// prefix is copied as-is
	if got.Prefix != l.Prefix {
		t.Fatalf("prefix must be copied unchanged")
	}
}

func TestCloneFringe_NilAndWithCloneFn(t *testing.T) {
	t.Parallel()
	route := &routeEntry{
		nextHop:    netip.MustParseAddr("10.1.1.1"),
		exitIF:     "bond0",
		attributes: map[string]int{"weight": 100, "priority": 5},
	}

	f := &nodes.FringeNode[*routeEntry]{Value: route}

	// nil cloneFn
	got := f.CloneFringe(nil)
	if got == f {
		t.Fatalf("expected a new fringe instance")
	}
	if got.Value != f.Value {
		t.Fatalf("value must be copied when cloneFn is nil")
	}

	// with cloneFn
	got2 := f.CloneFringe(func(v *routeEntry) *routeEntry { return v.Clone() })
	want := f.Value.Clone()
	if got2.Value.nextHop != want.nextHop || got2.Value.exitIF != want.exitIF {
		t.Fatalf("expected cloned value")
	}
	if got2.Value == f.Value {
		t.Fatalf("cloned value should be different instance")
	}
}

// ---- node.cloneFlat / node.cloneRec ----

func TestNodeCloneFlat_ShallowChildrenDeepValues(t *testing.T) {
	t.Parallel()
	parent := &nodes.BartNode[*routeEntry]{}

	// Add prefix values
	route1 := &routeEntry{
		nextHop:    netip.MustParseAddr("10.1.0.1"),
		exitIF:     "eth0",
		attributes: map[string]int{"metric": 10},
	}
	route2 := &routeEntry{
		nextHop:    netip.MustParseAddr("10.2.0.1"),
		exitIF:     "eth1",
		attributes: map[string]int{"metric": 20},
	}

	parent.InsertPrefix(10, route1)
	parent.InsertPrefix(20, route2)

	// Create child nodes
	pfx := netip.MustParsePrefix("10.0.0.0/8")
	leafRoute := &routeEntry{
		nextHop:    netip.MustParseAddr("10.0.0.1"),
		exitIF:     "lo",
		attributes: map[string]int{"metric": 1},
	}
	fringeRoute := &routeEntry{
		nextHop:    netip.MustParseAddr("10.3.0.1"),
		exitIF:     "vlan10",
		attributes: map[string]int{"vlan": 10},
	}

	leaf := &nodes.LeafNode[*routeEntry]{Prefix: pfx, Value: leafRoute}
	fringe := &nodes.FringeNode[*routeEntry]{Value: fringeRoute}
	childNode := &nodes.BartNode[*routeEntry]{}

	parent.InsertChild(1, childNode)
	parent.InsertChild(2, leaf)
	parent.InsertChild(3, fringe)

	fn := cloneFnFactory[*routeEntry]() // should not be nil
	got := parent.CloneFlat(fn)

	if got == parent {
		t.Fatalf("expected a new node instance")
	}

	// Verify prefixes are cloned (different array, cloned values)
	if got.PrefixCount() != 2 {
		t.Fatalf("expected 2 prefixes, got %d", got.PrefixCount())
	}

	// Values should be cloned (different instances)
	if v, ok := got.GetPrefix(10); !ok || v == route1 {
		t.Fatalf("expected cloned prefix value at index 10")
	} else if v.nextHop != route1.nextHop {
		t.Fatalf("cloned route should have same nextHop")
	}

	// Verify children are processed correctly
	if got.ChildCount() != 3 {
		t.Fatalf("expected 3 children, got %d", got.ChildCount())
	}

	// *bartNode child should be same reference (shallow)
	if gotNode, ok := got.GetChild(1); !ok || gotNode != childNode {
		t.Fatalf("expected shallow reference for *bartNode child")
	}

	// leaf should be cloned
	if gotLeaf, ok := got.GetChild(2); !ok {
		t.Fatalf("expected leaf at index 2")
	} else if l2, ok := gotLeaf.(*nodes.LeafNode[*routeEntry]); !ok || l2 == leaf {
		t.Fatalf("expected new leaf instance")
	} else if l2.Value == leaf.Value {
		t.Fatalf("expected cloned leaf value")
	} else if l2.Value.nextHop != leaf.Value.nextHop {
		t.Fatalf("cloned leaf value should have same nextHop")
	}

	// fringe should be cloned
	if gotFringe, ok := got.GetChild(3); !ok {
		t.Fatalf("expected fringe at index 3")
	} else if f2, ok := gotFringe.(*nodes.FringeNode[*routeEntry]); !ok || f2 == fringe {
		t.Fatalf("expected new fringe instance")
	} else if f2.Value == fringe.Value {
		t.Fatalf("expected cloned fringe value")
	} else if f2.Value.nextHop != fringe.Value.nextHop {
		t.Fatalf("cloned fringe value should have same nextHop")
	}

	// Structural independence: mutating the clone must not affect the original.
	origPC, origCC := parent.PrefixCount(), parent.ChildCount()
	got.InsertPrefix(30, &routeEntry{nextHop: netip.MustParseAddr("10.9.0.1"), exitIF: "tmp"})
	got.InsertChild(99, &nodes.BartNode[*routeEntry]{})
	if parent.PrefixCount() != origPC {
		t.Fatalf("parent prefixCount changed after mutating clone: got %d want %d", parent.PrefixCount(), origPC)
	}
	if parent.ChildCount() != origCC {
		t.Fatalf("parent childCount changed after mutating clone: got %d want %d", parent.ChildCount(), origCC)
	}
}

func TestNodeCloneFlat_PanicOnWrongType(t *testing.T) {
	t.Parallel()
	n := &nodes.BartNode[*routeEntry]{}
	n.Children = *n.Children.Copy()

	// insert a wrong type into children to trigger panic branch
	n.InsertChild(0, &struct{}{}) // not a recognized node type

	defer func() {
		if r := recover(); r == nil {
			t.Fatalf("expected panic on wrong node type")
		}
	}()

	_ = n.CloneFlat(nil)
}

func TestNodeCloneRec_DeepCopiesNodeChildren(t *testing.T) {
	t.Parallel()
	// chain of *bartNode: parent[0] -> child[0] -> grandchild
	parent := &nodes.BartNode[*routeEntry]{}
	child := &nodes.BartNode[*routeEntry]{}
	grand := &nodes.BartNode[*routeEntry]{}

	// Add a route to the grandchild to verify deep cloning
	grandRoute := &routeEntry{
		nextHop:    netip.MustParseAddr("192.168.0.1"),
		exitIF:     "eth0",
		attributes: map[string]int{"metric": 5},
	}
	grand.InsertPrefix(100, grandRoute)

	// build hierarchy
	parent.InsertChild(10, child)
	child.InsertChild(20, grand)

	cloneFn := cloneFnFactory[*routeEntry]()
	got := parent.CloneRec(cloneFn)

	// Must be a new parent
	if got == parent {
		t.Fatalf("expected different parent instance")
	}

	// verify deep clone
	kidAny, ok := got.GetChild(10)
	if !ok {
		t.Fatalf("expected child at index 10")
	}
	kid, ok := kidAny.(*nodes.BartNode[*routeEntry])
	if !ok || kid == child {
		t.Fatalf("expected deep-cloned child node")
	}

	gkAny, ok := kid.GetChild(20)
	if !ok {
		t.Fatalf("expected grandchild at index 20")
	}
	gk, ok := gkAny.(*nodes.BartNode[*routeEntry])
	if !ok || gk == grand {
		t.Fatalf("expected deep-cloned grandchild node")
	}

	// Verify the route in grandchild is also cloned
	clonedRoute, ok := gk.GetPrefix(100)
	if !ok {
		t.Fatalf("expected route in cloned grandchild")
	}
	if clonedRoute == grandRoute {
		t.Fatalf("route should be cloned in recursive clone")
	}
	if clonedRoute.nextHop != grandRoute.nextHop {
		t.Fatalf("cloned route should have same nextHop")
	}
}

// ---- fastNode.cloneFlat / cloneRec ----

func TestFastNodeCloneFlat_ValuesClonedAndChildrenFlat(t *testing.T) {
	t.Parallel()
	fn := &nodes.FastNode[*routeEntry]{}

	// insert prefix - default route
	defaultRoute := &routeEntry{
		nextHop:    netip.MustParseAddr("0.0.0.0"),
		exitIF:     "eth0",
		attributes: map[string]int{"metric": 1000},
	}
	fn.InsertPrefix(42, defaultRoute)

	// Create child nodes
	pfx := netip.MustParsePrefix("192.0.2.0/24")
	leafRoute := &routeEntry{
		nextHop:    netip.MustParseAddr("192.0.2.1"),
		exitIF:     "eth1",
		attributes: map[string]int{"metric": 100},
	}
	fringeRoute := &routeEntry{
		nextHop:    netip.MustParseAddr("172.16.0.1"),
		exitIF:     "eth2",
		attributes: map[string]int{"metric": 200},
	}

	leaf := &nodes.LeafNode[*routeEntry]{Prefix: pfx, Value: leafRoute}
	fringe := &nodes.FringeNode[*routeEntry]{Value: fringeRoute}
	childFast := &nodes.FastNode[*routeEntry]{}

	// insert children at addrs: 0,1,2
	fn.InsertChild(0, leaf)
	fn.InsertChild(1, fringe)
	fn.InsertChild(2, childFast)

	got := fn.CloneFlat(cloneFnFactory[*routeEntry]())
	if got == fn {
		t.Fatalf("expected new fast node instance")
	}

	// Check that prefixes are cloned
	if got.PrefixCount() != 1 {
		t.Fatalf("expected 1 prefix in cloned node")
	}
	if v, ok := got.GetPrefix(42); !ok || v == defaultRoute {
		t.Fatalf("expected cloned prefix value at index 42")
	} else if v.nextHop != defaultRoute.nextHop {
		t.Fatalf("cloned route should have same nextHop")
	}

	// Check children counts
	if got.ChildCount() != 3 {
		t.Fatalf("expected 3 children in cloned node")
	}

	// leaf should be cloned
	if gotLeaf, ok := got.GetChild(0); !ok {
		t.Fatalf("expected cloned leaf child")
	} else if l2, ok := gotLeaf.(*nodes.LeafNode[*routeEntry]); !ok || l2 == leaf {
		t.Fatalf("expected new leaf instance")
	} else if l2.Value == leaf.Value {
		t.Fatalf("expected cloned leaf value")
	} else if l2.Value.nextHop != leafRoute.nextHop {
		t.Fatalf("cloned leaf value should have same nextHop")
	}

	// fringe should be cloned
	if gotFringe, ok := got.GetChild(1); !ok {
		t.Fatalf("expected cloned fringe child")
	} else if f2, ok := gotFringe.(*nodes.FringeNode[*routeEntry]); !ok || f2 == fringe {
		t.Fatalf("expected new fringe instance")
	} else if f2.Value == fringe.Value {
		t.Fatalf("expected cloned fringe value")
	} else if f2.Value.nextHop != fringeRoute.nextHop {
		t.Fatalf("cloned fringe value should have same nextHop")
	}

	// fastNode child should be shallow copied (same pointer)
	if gotChild, ok := got.GetChild(2); !ok || gotChild != childFast {
		t.Fatalf("expected shallow copy of fastNode child")
	}
}

func TestFastNodeCloneRec_DeepCopiesFastNodeChildren(t *testing.T) {
	t.Parallel()
	parent := &nodes.FastNode[*routeEntry]{}
	child := &nodes.FastNode[*routeEntry]{}
	grand := &nodes.FastNode[*routeEntry]{}

	// Add route to grandchild to verify deep cloning
	route := &routeEntry{
		nextHop:    netip.MustParseAddr("203.0.113.1"),
		exitIF:     "wan0",
		attributes: map[string]int{"preference": 100},
	}
	grand.InsertPrefix(1, route)

	// Build hierarchy
	parent.InsertChild(10, child)
	child.InsertChild(20, grand)

	cloneFn := cloneFnFactory[*routeEntry]()
	got := parent.CloneRec(cloneFn)
	if got == parent {
		t.Fatalf("expected new parent")
	}

	// verify deep clone
	kidAny, ok := got.GetChild(10)
	if !ok {
		t.Fatalf("expected child at index 10")
	}
	kid, ok := kidAny.(*nodes.FastNode[*routeEntry])
	if !ok || kid == child {
		t.Fatalf("expected deep-cloned child fastNode")
	}

	gkAny, ok := kid.GetChild(20)
	if !ok {
		t.Fatalf("expected grandchild at index 20")
	}
	gk, ok := gkAny.(*nodes.FastNode[*routeEntry])
	if !ok || gk == grand {
		t.Fatalf("expected deep-cloned grandchild fastNode")
	}

	// Verify route is cloned at deepest level
	clonedRoute, ok := gk.GetPrefix(1)
	if !ok {
		t.Fatalf("expected route in cloned grandchild")
	}
	if clonedRoute == route {
		t.Fatalf("route should be cloned in recursive clone")
	}
	if clonedRoute.nextHop != route.nextHop {
		t.Fatalf("cloned route should have same nextHop")
	}
}
