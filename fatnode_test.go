package bart

import (
	"testing"
)

func TestFatNode_EmptyCountsAndState(t *testing.T) {
	t.Parallel()
	n := &fatNode[int]{}

	if got := n.prefixCount(); got != 0 {
		t.Fatalf("prefixCount=%d, want 0", got)
	}
	if got := n.childCount(); got != 0 {
		t.Fatalf("childCount=%d, want 0", got)
	}
	if !n.isEmpty() {
		t.Fatalf("isEmpty=false, want true")
	}

	if _, ok := n.getChild(0); ok {
		t.Fatalf("getChild returned ok=true on empty node")
	}

	if _, ok := n.getPrefix(0); ok {
		t.Fatalf("getPrefix returned ok=true on empty node")
	}
}

func TestFatNode_Children_Insert_Get_Delete_Idempotent(t *testing.T) {
	t.Parallel()
	n := &fatNode[int]{}

	type dummy struct{ id int }
	child := &dummy{id: 42}

	if exists := n.insertChild(7, child); exists {
		t.Fatalf("insertChild first insertion exists=true, want false")
	}
	if got := n.childCount(); got != 1 {
		t.Fatalf("childCount=%d, want 1 after first insert", got)
	}

	if exists := n.insertChild(7, child); !exists {
		t.Fatalf("insertChild second insertion exists=false, want true")
	}
	if got := n.childCount(); got != 1 {
		t.Fatalf("childCount=%d, want 1 after second insert", got)
	}

	gotAny, ok := n.getChild(7)
	if !ok {
		t.Fatalf("getChild(7) ok=false, want true")
	}
	if got, ok := gotAny.(*dummy); !ok || got.id != 42 {
		t.Fatalf("getChild type/value mismatch: %T %#v", gotAny, gotAny)
	}

	n.deleteChild(7)
	if got := n.childCount(); got != 0 {
		t.Fatalf("childCount=%d, want 0 after delete", got)
	}
	if _, ok := n.getChild(7); ok {
		t.Fatalf("getChild(7) ok=true after delete, want false")
	}

	n.deleteChild(7) // idempotent
	if got := n.childCount(); got != 0 {
		t.Fatalf("childCount=%d, want 0 after idempotent delete", got)
	}
}

func TestFatNode_Prefix_Insert_Get_Contains_Lookup_Propagation(t *testing.T) {
	t.Parallel()
	n := &fatNode[int]{}

	if exists := n.insertPrefix(32, 100); exists {
		t.Fatalf("insertPrefix first insertion exists=true, want false")
	}
	if got := n.prefixCount(); got != 1 {
		t.Fatalf("prefixCount=%d, want 1 after first insert", got)
	}

	if v, ok := n.getPrefix(32); !ok || v != 100 {
		t.Fatalf("getPrefix(32)=(%d,%v), want (100,true)", v, ok)
	}

	// Descendant indices (lookup uses idx>>1)
	if !n.contains(64) {
		t.Fatalf("contains(64)=false, want true (inherited from 32)")
	}
	if val, ok := n.lookup(64); !ok || val != 100 {
		t.Fatalf("lookup(64)=(%d,%v), want (100,true)", val, ok)
	}
	if val, ok := n.lookup(65); !ok || val != 100 {
		t.Fatalf("lookup(65)=(%d,%v), want (100,true)", val, ok)
	}

	// Not covered indices
	if n.contains(66) {
		t.Fatalf("contains(66)=true, want false (33 remains nil)")
	}
	if _, ok := n.lookup(66); ok {
		t.Fatalf("lookup(66) ok=true, want false")
	}

	// Overwrite and propagate
	if exists := n.insertPrefix(32, 200); !exists {
		t.Fatalf("insertPrefix overwrite exists=false, want true")
	}
	if val, ok := n.lookup(64); !ok || val != 200 {
		t.Fatalf("lookup(64) after overwrite=(%d,%v), want (200,true)", val, ok)
	}

	// More specific route should not be overridden
	if exists := n.insertPrefix(64, 300); exists {
		t.Fatalf("insertPrefix(64) exists=true on first insert, want false")
	}
	if val, ok := n.lookup(128); !ok || val != 300 {
		t.Fatalf("lookup(128) after specific insert=(%d,%v), want (300,true)", val, ok)
	}
	if base, val, ok := n.lookupIdx(128); !ok || base != 64 || val != 300 {
		t.Fatalf("lookupIdx(128) got (base=%d,val=%d,ok=%v), want (base=64,val=300,ok=true)", base, val, ok)
	}

	// Parent change should not affect specific route
	n.insertPrefix(32, 400)
	if val, ok := n.lookup(128); !ok || val != 300 {
		t.Fatalf("lookup(128) after parent overwrite=(%d,%v), want (300,true)", val, ok)
	}
	if base, val, ok := n.lookupIdx(128); !ok || base != 64 || val != 300 {
		t.Fatalf("lookupIdx(128) got (base=%d,val=%d,ok=%v), want (base=64,val=300,ok=true)", base, val, ok)
	}
	if val, ok := n.lookup(64); !ok || val != 300 {
		t.Fatalf("lookup(64) (direct child of 32)=(%d,%v), want (300,true)", val, ok)
	}
	if base, val, ok := n.lookupIdx(66); ok {
		t.Fatalf("lookupIdx(66) ok=true (base=%d,val=%d), want false", base, val)
	}
}

func TestFatNode_DeletePrefix_Behavior(t *testing.T) {
	t.Parallel()
	n := &fatNode[int]{}

	n.insertPrefix(32, 111)
	n.insertPrefix(64, 222)

	if val, ok := n.lookup(128); !ok || val != 222 {
		t.Fatalf("pre-delete lookup(128)=(%d,%v), want (222,true)", val, ok)
	}
	if got := n.prefixCount(); got != 2 {
		t.Fatalf("prefixCount pre-delete=%d, want 2", got)
	}

	// Delete non-existent
	if _, ok := n.deletePrefix(10); ok {
		t.Fatalf("deletePrefix(10) ok=true, want false")
	}

	// Delete specific and fall back to parent
	if v, ok := n.deletePrefix(64); !ok || v != 222 {
		t.Fatalf("deletePrefix(64)=(%d,%v), want (222,true)", v, ok)
	}
	if _, ok := n.getPrefix(64); ok {
		t.Fatalf("getPrefix(64) after delete ok=true, want false")
	}
	if val, ok := n.lookup(128); !ok || val != 111 {
		t.Fatalf("lookup(128) after delete=(%d,%v), want (111,true)", val, ok)
	}

	// Delete parent
	if v, ok := n.deletePrefix(32); !ok || v != 111 {
		t.Fatalf("deletePrefix(32)=(%d,%v), want (111,true)", v, ok)
	}
	if n.contains(64) {
		t.Fatalf("contains(64)=true after deleting 32, want false")
	}
	if _, ok := n.lookup(64); ok {
		t.Fatalf("lookup(64) ok=true after deleting 32, want false")
	}
	if got := n.prefixCount(); got != 0 {
		t.Fatalf("prefixCount after deletions=%d, want 0", got)
	}
}

func TestFatNode_Allot_StopsAtSpecificRoutes(t *testing.T) {
	t.Parallel()
	n := &fatNode[int]{}

	n.insertPrefix(32, 1)
	n.insertPrefix(64, 2)

	// Overwrite parent should not affect specific child
	n.insertPrefix(32, 3)

	if v, ok := n.lookup(128); !ok || v != 2 {
		t.Fatalf("lookup(128) got (%d,%v), want (2,true)", v, ok)
	}
	if v, ok := n.lookup(66); ok {
		t.Fatalf("lookup(66) ok=true (%d), want false (unrelated branch)", v)
	}
	if v, ok := n.lookup(64); !ok || v != 2 {
		t.Fatalf("lookup(64) got (%d,%v), want (2,true)", v, ok)
	}
}
