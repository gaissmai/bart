package bart

import (
	"net/netip"
	"testing"
)

// ---- Test helper types ----

type clonerInt int

// Implement Cloner[clonerInt]
func (c clonerInt) Clone() clonerInt {
	// return a distinct value to prove cloning happened
	return clonerInt(int(c) + 1000)
}

type nonClonerInt int

// ---- cloneFnFactory / cloneVal / copyVal ----

func TestCloneFnFactory_WithCloner(t *testing.T) {
	fn := cloneFnFactory[clonerInt]()
	if fn == nil {
		t.Fatalf("expected non-nil clone func when V implements Cloner[V]")
	}
	in := clonerInt(7)
	out := fn(in)
	if out != in.Clone() {
		t.Fatalf("expected out=%v cloned, got %v", in.Clone(), out)
	}
}

func TestCloneFnFactory_WithoutCloner(t *testing.T) {
	fn := cloneFnFactory[nonClonerInt]()
	if fn != nil {
		t.Fatalf("expected nil clone func when V does not implement Cloner[V]")
	}
}

func TestCloneVal_WithCloner(t *testing.T) {
	in := clonerInt(3)
	got := cloneVal[clonerInt](in)
	if want := in.Clone(); got != want {
		t.Fatalf("expected %v, got %v", want, got)
	}
}

func TestCloneVal_WithoutCloner(t *testing.T) {
	in := nonClonerInt(5)
	got := cloneVal[nonClonerInt](in)
	if got != in {
		t.Fatalf("expected passthrough for non-cloner, got %v", got)
	}
}

func TestCopyVal_Passthrough(t *testing.T) {
	in := nonClonerInt(9)
	if got := copyVal[nonClonerInt](in); got != in {
		t.Fatalf("copyVal should return input; want %v got %v", in, got)
	}
}

// ---- leafNode.cloneLeaf / fringeNode.cloneFringe ----

func TestLeafCloneLeaf_NilCloneFn(t *testing.T) {
	prefix := netip.MustParsePrefix("192.0.2.0/24")
	l := &leafNode[nonClonerInt]{prefix: prefix, value: 42}
	got := l.cloneLeaf(nil)
	if got == l {
		t.Fatalf("expected new leaf instance")
	}
	if got.prefix != l.prefix {
		t.Fatalf("prefix must be copied as-is: want %v got %v", l.prefix, got.prefix)
	}
	if got.value != l.value {
		t.Fatalf("value must be copied: want %v got %v", l.value, got.value)
	}
}

func TestLeafCloneLeaf_WithCloneFn(t *testing.T) {
	prefix := netip.MustParsePrefix("198.51.100.0/24")
	l := &leafNode[clonerInt]{prefix: prefix, value: 7}
	cloneFn := func(v clonerInt) clonerInt { return v.Clone() }
	got := l.cloneLeaf(cloneFn)
	if got.value != l.value.Clone() {
		t.Fatalf("expected leaf value to be cloned; want %v got %v", l.value.Clone(), got.value)
	}
	// prefix is copied as-is
	if got.prefix != l.prefix {
		t.Fatalf("prefix must be copied unchanged")
	}
}

func TestFringeCloneFringe_NilAndWithCloneFn(t *testing.T) {
	f := &fringeNode[clonerInt]{value: 33}
	// nil cloneFn
	got := f.cloneFringe(nil)
	if got == f {
		t.Fatalf("expected a new fringe instance")
	}
	if got.value != f.value {
		t.Fatalf("value must be copied when cloneFn is nil")
	}
	// with cloneFn
	got2 := f.cloneFringe(func(v clonerInt) clonerInt { return v.Clone() })
	if want := f.value.Clone(); got2.value != want {
		t.Fatalf("expected cloned value: want %v got %v", want, got2.value)
	}
}

// ---- node.cloneFlat / node.cloneRec ----

func TestNodeCloneFlat_ShallowChildrenDeepValues(t *testing.T) {
	t.Parallel()
	parent := &node[clonerInt]{}

	// Add prefix values using InsertAt
	parent.prefixes.InsertAt(10, clonerInt(1))
	parent.prefixes.InsertAt(20, clonerInt(2))

	// Create child nodes
	pfx := netip.MustParsePrefix("10.0.0.0/8")
	leaf := &leafNode[clonerInt]{prefix: pfx, value: 10}
	fringe := &fringeNode[clonerInt]{value: 20}
	childNode := &node[clonerInt]{}

	parent.children.InsertAt(1, childNode)
	parent.children.InsertAt(2, leaf)
	parent.children.InsertAt(3, fringe)

	fn := cloneFnFactory[clonerInt]() // should not be nil
	got := parent.cloneFlat(fn)

	if got == parent {
		t.Fatalf("expected a new node instance")
	}

	if &got.prefixes == &parent.prefixes {
		t.Fatalf("expected a new prefixes backing slice")
	}

	if &got.children == &parent.children {
		t.Fatalf("expected a new children backing slice")
	}

	// Verify prefixes are cloned (different array, cloned values)
	if got.prefixes.Len() != 2 {
		t.Fatalf("expected 2 prefixes, got %d", got.prefixes.Len())
	}
	// Values should be cloned (+1000)
	if v, ok := got.prefixes.Get(10); !ok || v != clonerInt(1001) {
		t.Fatalf("expected cloned prefix value 1001; got %v ok=%v", v, ok)
	}
	if v, ok := got.prefixes.Get(20); !ok || v != clonerInt(1002) {
		t.Fatalf("expected cloned prefix value 1002; got %v ok=%v", v, ok)
	}

	// Verify children are processed correctly
	if got.children.Len() != 3 {
		t.Fatalf("expected 3 children, got %d", got.children.Len())
	}

	// *node child should be same reference (shallow)
	if gotNode, ok := got.children.Get(1); !ok || gotNode != childNode {
		t.Fatalf("expected shallow reference for *node child")
	}

	// leaf should be cloned
	if gotLeaf, ok := got.children.Get(2); !ok {
		t.Fatalf("expected leaf at index 2")
	} else if l2, ok := gotLeaf.(*leafNode[clonerInt]); !ok || l2 == leaf {
		t.Fatalf("expected new leaf instance")
	} else if l2.value != leaf.value.Clone() {
		t.Fatalf("expected cloned leaf value")
	}

	// fringe should be cloned
	if gotFringe, ok := got.children.Get(3); !ok {
		t.Fatalf("expected fringe at index 3")
	} else if f2, ok := gotFringe.(*fringeNode[clonerInt]); !ok || f2 == fringe {
		t.Fatalf("expected new fringe instance")
	} else if f2.value != fringe.value.Clone() {
		t.Fatalf("expected cloned fringe value")
	}
}

func TestNodeCloneFlat_PanicOnWrongType(t *testing.T) {
	n := &node[nonClonerInt]{}
	n.children = *n.children.Copy()
	// insert a wrong type into children.Items to trigger panic branch
	n.children.Items = append(n.children.Items, struct{}{}) // not a recognized node type
	defer func() {
		if r := recover(); r == nil {
			t.Fatalf("expected panic on wrong node type")
		}
	}()
	_ = n.cloneFlat(nil)
}

func TestNodeCloneRec_DeepCopiesNodeChildren(t *testing.T) {
	// parent -> child -> grandchild chain of *node
	grand := &node[clonerInt]{}
	child := &node[clonerInt]{}
	child.children = *child.children.Copy()
	child.children.Items = append(child.children.Items, grand)

	parent := &node[clonerInt]{}
	parent.children = *parent.children.Copy()
	parent.children.Items = append(parent.children.Items, child)

	got := parent.cloneRec(cloneFnFactory[clonerInt]())

	// Must be a new parent
	if got == parent {
		t.Fatalf("expected different parent instance")
	}
	// child must be a different instance too
	gotChild, _ := got.children.Items[0].(*node[clonerInt])
	if gotChild == nil || gotChild == child {
		t.Fatalf("expected deep-cloned child node")
	}
	// grandchild must be deep-cloned too
	gotGrand, _ := gotChild.children.Items[0].(*node[clonerInt])
	if gotGrand == nil || gotGrand == grand {
		t.Fatalf("expected deep-cloned grandchild node")
	}
}

// ---- fatNode.cloneFlat / cloneRec ----

func TestFatNodeCloneFlat_ValuesClonedAndChildrenFlat(t *testing.T) {
	fn := &fatNode[clonerInt]{}

	// Use the proper insertPrefix method
	fn.insertPrefix(42, clonerInt(11))

	// Create child nodes
	pfx := netip.MustParsePrefix("192.0.2.0/24")
	leaf := &leafNode[clonerInt]{prefix: pfx, value: 21}
	fringe := &fringeNode[clonerInt]{value: 31}
	childFat := &fatNode[clonerInt]{}

	// Use the proper insertChild method
	fn.insertChild(1, leaf)
	fn.insertChild(2, fringe)
	fn.insertChild(3, childFat)

	got := fn.cloneFlat(cloneFnFactory[clonerInt]())
	if got == fn {
		t.Fatalf("expected new fat node instance")
	}

	// Check that prefixes are cloned
	if got.prefixCount() != 1 {
		t.Fatalf("expected 1 prefix in cloned node")
	}
	if v, ok := got.getPrefix(42); !ok || v != clonerInt(1011) {
		t.Fatalf("expected cloned prefix value 1011 at index 42; got %v ok=%v", v, ok)
	}

	// Check children counts
	if got.childCount() != 3 {
		t.Fatalf("expected 3 children in cloned node")
	}

	// fatNode child should be shallow (same pointer)
	if gotChild, ok := got.getChild(3); !ok || gotChild != childFat {
		t.Fatalf("expected shallow copy of fatNode child")
	}

	// leaf should be cloned
	if gotLeaf, ok := got.getChild(1); !ok {
		t.Fatalf("expected cloned leaf child")
	} else if l2, ok := gotLeaf.(*leafNode[clonerInt]); !ok || l2 == leaf || l2.value != leaf.value.Clone() {
		t.Fatalf("expected cloned leaf with cloned value")
	}

	// fringe should be cloned
	if gotFringe, ok := got.getChild(2); !ok {
		t.Fatalf("expected cloned fringe child")
	} else if f2, ok := gotFringe.(*fringeNode[clonerInt]); !ok || f2 == fringe || f2.value != fringe.value.Clone() {
		t.Fatalf("expected cloned fringe with cloned value")
	}
}

func TestFatNodeCloneRec_DeepCopiesFatNodeChildren(t *testing.T) {
	parent := &fatNode[clonerInt]{}
	child := &fatNode[clonerInt]{}
	grand := &fatNode[clonerInt]{}

	// Build hierarchy using proper methods
	child.insertChild(20, grand)
	parent.insertChild(10, child)

	got := parent.cloneRec(cloneFnFactory[clonerInt]())
	if got == parent {
		t.Fatalf("expected new parent")
	}

	// verify deep clone
	kidAny, ok := got.getChild(10)
	if !ok {
		t.Fatalf("expected child at index 10")
	}
	kid, ok := kidAny.(*fatNode[clonerInt])
	if !ok || kid == child {
		t.Fatalf("expected deep-cloned child fatNode")
	}

	gkAny, ok := kid.getChild(20)
	if !ok {
		t.Fatalf("expected grandchild at index 20")
	}
	gk, ok := gkAny.(*fatNode[clonerInt])
	if !ok || gk == grand {
		t.Fatalf("expected deep-cloned grandchild fatNode")
	}
}
