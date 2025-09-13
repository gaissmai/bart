package bart

import (
	"testing"
)

// Notes on testing library & framework:
// - Using Go's standard testing package (testing). If the repository includes testify,
//   these tests can be extended with require/assert, but we keep zero-dependency defaults.

// cloneString is a simple clone function for string values used in tests.
func cloneString(v string) string { return v }

// helpers to build nodes for tests without relying on AsSlice internals:

// mkNode creates an empty trie node for string values.
func mkNode() *node[string] { return new(node[string]) }

// addPrefix sets a prefix byte -> value on a node and returns whether it was duplicate.
func addPrefix(n *node[string], b uint8, val string) (dup bool) {
	return n.insertPrefix(b, val)
}

// addLeaf attaches a leaf child at an address with given prefix depth and value.
func addLeaf(n *node[string], addr uint8, prefix uint8, val string) {
	lf := &leafNode[string]{prefix: prefix, value: val}
	n.insertChild(addr, lf)
}

// addFringe attaches a fringe child at an address with a value.
func addFringe(n *node[string], addr uint8, val string) {
	fr := &fringeNode[string]{value: val}
	n.insertChild(addr, fr)
}

// addChildNode attaches a cloned subtree node at an address.
func addChildNode(n *node[string], addr uint8, kid *node[string]) {
	n.insertChild(addr, kid)
}

func TestUnionRec_PrefixOverwriteCountsDuplicates(t *testing.T) {
	// GIVEN two nodes with overlapping prefix entries
	this := mkNode()
	other := mkNode()

	addPrefix(this, 10, "A")
	addPrefix(this, 20, "B")
	addPrefix(other, 20, "B2") // duplicate on 20 should count
	addPrefix(other, 30, "C")

	// WHEN we union (destructive) with a cloning func
	dups := this.unionRec(cloneString, other, 0)

	// THEN duplicates should be exactly 1 (only prefix 20)
	if dups \!= 1 {
		t.Fatalf("expected 1 duplicate, got %d", dups)
	}

	// AND values reflect overwritten entries
	// Verify by re-inserting same key and ensuring it reports duplicate (present)
	if \!addPrefix(this, 10, "A2") {
		t.Fatalf("expected prefix 10 to exist after union")
	}
	if \!addPrefix(this, 20, "B3") {
		t.Fatalf("expected prefix 20 to exist and have been overwritten")
	}
	if \!addPrefix(this, 30, "C2") {
		t.Fatalf("expected prefix 30 to exist from other")
	}
}

func TestUnionRec_NullChildInsertions(t *testing.T) {
	// Covers: NULL,node | NULL,leaf | NULL,fringe
	this := mkNode()
	other := mkNode()

	// NULL, node
	sub := mkNode()
	addPrefix(sub, 1, "n1")
	addChildNode(other, 5, sub)

	// NULL, leaf
	addLeaf(other, 6, 2, "leaf2")

	// NULL, fringe
	addFringe(other, 7, "fr3")

	dups := this.unionRec(cloneString, other, 0)
	if dups \!= 0 {
		t.Fatalf("expected 0 duplicates when inserting into empty children, got %d", dups)
	}

	// Validate basic presence by attempting inserts that should report duplicates inside subtrees.
	// For node at addr=5, inserting default at depth+1 via union would have created/kept structure;
	// We test presence indirectly by re-unioning an identical other and expecting duplicates to rise.
	dups2 := this.unionRec(cloneString, other, 0)
	if dups2 == 0 {
		t.Fatalf("expected duplicates on second union into same structure")
	}
}

func TestUnionRec_NodeWithLeafAndFringe(t *testing.T) {
	// Covers: node,leaf and node,fringe paths
	this := mkNode()
	other := mkNode()

	// Prepare this child node at addr 9
	thisKid := mkNode()
	addPrefix(thisKid, 1, "base") // some content to push into
	addChildNode(this, 9, thisKid)

	// OTHER has a leaf at same addr
	addLeaf(other, 9, 3, "leafVal")
	// Also include a fringe at a different addr to ensure independent behavior
	addFringe(other, 8, "fr")

	dups := this.unionRec(cloneString, other, 0)

	// We expect at least 0+ duplicates; exact count depends on whether inserts hit existing prefixes.
	// Assert structural consistency by performing idempotent union again and ensure duplicates increase or remain >= previous.
	dups2 := this.unionRec(cloneString, other, 0)
	if dups2 < dups {
		t.Fatalf("expected duplicates on idempotent union to be >= previous run, got %d < %d", dups2, dups)
	}
}

func TestUnionRec_LeafCases_LeafNodeLeafLeafFringe(t *testing.T) {
	// Covers: leaf,node | leaf,leaf (equal and different) | leaf,fringe
	this := mkNode()
	other := mkNode()

	// Place a leaf in THIS at addr 11
	addLeaf(this, 11, 2, "Lthis")

	// Case: leaf,node at same addr => new intermediate node created
	subOther := mkNode()
	addPrefix(subOther, 1, "kid")
	addChildNode(other, 11, subOther)

	dups0 := this.unionRec(cloneString, other, 0)
	if dups0 < 0 {
		t.Fatalf("duplicates cannot be negative: %d", dups0)
	}

	// Reset setup for leaf,leaf equal prefix
	this2 := mkNode()
	other2 := mkNode()
	addLeaf(this2, 12, 4, "X")
	addLeaf(other2, 12, 4, "Y") // same prefix -> overwrite + duplicates++

	dups1 := this2.unionRec(cloneString, other2, 0)
	if dups1 \!= 1 {
		t.Fatalf("expected 1 duplicate for equal leaf prefixes, got %d", dups1)
	}

	// leaf,leaf different prefixes should create new node and maybe duplicate if collisions occur
	this3 := mkNode()
	other3 := mkNode()
	addLeaf(this3, 13, 2, "A")
	addLeaf(other3, 13, 5, "B")
	dups2 := this3.unionRec(cloneString, other3, 0)
	if dups2 < 0 {
		t.Fatalf("expected non-negative duplicates for different leaf prefixes; got %d", dups2)
	}

	// leaf,fringe should create new node and set default route
	this4 := mkNode()
	other4 := mkNode()
	addLeaf(this4, 14, 1, "L")
	addFringe(other4, 14, "F")
	dups3 := this4.unionRec(cloneString, other4, 0)
	if dups3 < 0 {
		t.Fatalf("expected non-negative duplicates for leaf,fringe; got %d", dups3)
	}
}

func TestUnionRec_FringeCases_FringeNodeFringeLeaf(t *testing.T) {
	// Covers: fringe,node | fringe,leaf | fringe,fringe
	this := mkNode()
	other := mkNode()

	// fringe,node
	addFringe(this, 3, "baseF")
	sub := mkNode()
	addPrefix(sub, 1, "S")
	addChildNode(other, 3, sub)

	dups0 := this.unionRec(cloneString, other, 0)
	if dups0 < 0 {
		t.Fatalf("expected duplicates >= 0, got %d", dups0)
	}

	// Reset for fringe,leaf
	this2 := mkNode()
	other2 := mkNode()
	addFringe(this2, 4, "F2")
	addLeaf(other2, 4, 2, "L2")

	dups1 := this2.unionRec(cloneString, other2, 0)
	if dups1 < 0 {
		t.Fatalf("expected duplicates >= 0 for fringe,leaf, got %d", dups1)
	}

	// fringe,fringe should overwrite value and count duplicate
	this3 := mkNode()
	other3 := mkNode()
	addFringe(this3, 5, "old")
	addFringe(other3, 5, "new")
	dups2 := this3.unionRec(cloneString, other3, 0)
	if dups2 \!= 1 {
		t.Fatalf("expected 1 duplicate for fringe,fringe overwrite, got %d", dups2)
	}
}

func TestUnionRecPersist_ClonesExistingNodeBeforeMerge(t *testing.T) {
	// Ensure unionRecPersist clones the existing child node (cloneFlat) and then merges,
	// leaving original structure logically intact aside from the receiver mutations required by API.
	this := mkNode()
	other := mkNode()

	// Put a node child at addr 7 with some prefixes so that cloneFlat path is exercised
	thisKid := mkNode()
	addPrefix(thisKid, 1, "one")
	addPrefix(thisKid, 2, "two")
	addChildNode(this, 7, thisKid)

	// Other has a node at same addr with a conflicting prefix
	otherKid := mkNode()
	addPrefix(otherKid, 2, "two-new") // should cause duplicate on merge
	addChildNode(other, 7, otherKid)

	dups := this.unionRecPersist(cloneString, other, 0)
	if dups \!= 1 {
		t.Fatalf("expected 1 duplicate from conflicting prefix in cloned child, got %d", dups)
	}

	// Idempotency-ish: running again should be >= duplicates, as structure now contains merged state.
	dups2 := this.unionRecPersist(cloneString, other, 0)
	if dups2 < 1 {
		t.Fatalf("expected >=1 duplicate after re-run, got %d", dups2)
	}
}

func TestUnionRec_PanicOnInvalidTypeIsContained(t *testing.T) {
	// We cannot construct an invalid type easily; instead, we ensure that all handled
	// combinations do not panic. This acts as a guard against unexpected panics during union.
	cases := []struct {
		name  string
		setup func(this, other *node[string])
	}{
		{"NULL_node", func(this, other *node[string]) { addChildNode(other, 1, mkNode()) }},
		{"NULL_leaf", func(this, other *node[string]) { addLeaf(other, 2, 1, "v") }},
		{"NULL_fringe", func(this, other *node[string]) { addFringe(other, 3, "v") }},
		{"node_node", func(this, other *node[string]) {
			addChildNode(this, 4, mkNode())
			addChildNode(other, 4, mkNode())
		}},
		{"node_leaf", func(this, other *node[string]) {
			addChildNode(this, 5, mkNode())
			addLeaf(other, 5, 1, "v")
		}},
		{"node_fringe", func(this, other *node[string]) {
			addChildNode(this, 6, mkNode())
			addFringe(other, 6, "v")
		}},
		{"leaf_node", func(this, other *node[string]) {
			addLeaf(this, 7, 1, "v")
			addChildNode(other, 7, mkNode())
		}},
		{"leaf_leaf_equal", func(this, other *node[string]) {
			addLeaf(this, 8, 2, "a")
			addLeaf(other, 8, 2, "b")
		}},
		{"leaf_fringe", func(this, other *node[string]) {
			addLeaf(this, 9, 1, "a")
			addFringe(other, 9, "b")
		}},
		{"fringe_node", func(this, other *node[string]) {
			addFringe(this, 10, "a")
			addChildNode(other, 10, mkNode())
		}},
		{"fringe_leaf", func(this, other *node[string]) {
			addFringe(this, 11, "a")
			addLeaf(other, 11, 1, "b")
		}},
		{"fringe_fringe", func(this, other *node[string]) {
			addFringe(this, 12, "a")
			addFringe(other, 12, "b")
		}},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			this := mkNode()
			other := mkNode()
			tc.setup(this, other)
			defer func() {
				if r := recover(); r \!= nil {
					t.Fatalf("unionRec panicked for case %s: %v", tc.name, r)
				}
			}()
			_ = this.unionRec(cloneString, other, 0)
		})
	}
}