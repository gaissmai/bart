// Copyright (c) 2025 Karl Gaissmaier
// SPDX-License-Identifier: MIT

// Usage: go generate -tags=ignore ./...
//go:generate ./scripts/generate-node-methods.sh
//go:build ignore

package bart

// ### GENERATE DELETE START ###

// stub code for generator types and methods
// useful for gopls during development, deleted during go generate

import (
	"net/netip"

	"github.com/gaissmai/bart/internal/bitset"
)

type _NODE_TYPE[V any] struct {
	prefixes struct{ bitset.BitSet256 }
	children struct{ bitset.BitSet256 }
}

func (n *_NODE_TYPE[V]) mustGetPrefix(uint8) (val V)                                    { return val }
func (n *_NODE_TYPE[V]) mustGetChild(uint8) (child any)                                 { return child }
func (n *_NODE_TYPE[V]) insertPrefix(uint8, V) (exists bool)                            { return exists }
func (n *_NODE_TYPE[V]) getChild(uint8) (child any, ok bool)                            { return child, ok }
func (n *_NODE_TYPE[V]) insertChild(uint8, any) (exists bool)                           { return exists }
func (n *_NODE_TYPE[V]) cloneRec(cloneFunc[V]) (c *_NODE_TYPE[V])                       { return c }
func (n *_NODE_TYPE[V]) cloneFlat(cloneFunc[V]) (c *_NODE_TYPE[V])                      { return c }
func (n *_NODE_TYPE[V]) insert(netip.Prefix, V, int) (exists bool)                      { return }
func (n *_NODE_TYPE[V]) insertPersist(cloneFunc[V], netip.Prefix, V, int) (exists bool) { return }

// ### GENERATE DELETE END ###

// unionRec recursively merges another node o into the receiver node n.
//
// All prefix and child entries from o are cloned and inserted into n.
// If a prefix already exists in n, its value is overwritten by the value from o,
// and the duplicate is counted in the return value. This count can later be used
// to update size-related metadata in the parent trie.
//
// The union handles all possible combinations of child node types (node, leaf, fringe)
// between the two nodes. Structural conflicts are resolved by creating new intermediate
// *_NODE_TYPE[V] objects and pushing both children further down the trie. Leaves and fringes
// are also recursively relocated as needed to preserve prefix semantics.
//
// The merge operation is destructive on the receiver n, but leaves the source node o unchanged.
//
// Returns the number of duplicate prefixes that were overwritten during merging.
func (n *_NODE_TYPE[V]) unionRec(cloneFn cloneFunc[V], o *_NODE_TYPE[V], depth int) (duplicates int) {
	buf := [256]uint8{}

	// for all prefixes in other node do ...
	for _, oIdx := range o.prefixes.AsSlice(&buf) {
		// clone/copy the value from other node at idx
		val := o.mustGetPrefix(oIdx)
		clonedVal := cloneFn(val)

		// insert/overwrite cloned value from o into n
		if n.insertPrefix(oIdx, clonedVal) {
			// this prefix is duplicate in n and o
			duplicates++
		}
	}

	// for all child addrs in other node do ...
	for _, addr := range o.children.AsSlice(&buf) {
		otherChild := o.mustGetChild(addr)
		thisChild, thisExists := n.getChild(addr)

		// Use helper function to handle all 4x3 combinations
		duplicates += n.handleMatrix(cloneFn, thisExists, thisChild, otherChild, addr, depth)
	}

	return duplicates
}

// unionRecPersist is similar to unionRec but performs an immutable union of nodes.
func (n *_NODE_TYPE[V]) unionRecPersist(cloneFn cloneFunc[V], o *_NODE_TYPE[V], depth int) (duplicates int) {
	buf := [256]uint8{}

	// for all prefixes in other node do ...
	for _, oIdx := range o.prefixes.AsSlice(&buf) {
		// clone/copy the value from other node
		val := o.mustGetPrefix(oIdx)
		clonedVal := cloneFn(val)

		// insert/overwrite cloned value from o into n
		if exists := n.insertPrefix(oIdx, clonedVal); exists {
			// this prefix is duplicate in n and o
			duplicates++
		}
	}

	// for all child addrs in other node do ...
	for _, addr := range o.children.AsSlice(&buf) {
		otherChild := o.mustGetChild(addr)
		thisChild, thisExists := n.getChild(addr)

		// Use helper function to handle all 4x3 combinations
		duplicates += n.handleMatrixPersist(cloneFn, thisExists, thisChild, otherChild, addr, depth)
	}

	return duplicates
}

// handleMatrix, 12 possible combinations to union this child and other child
//
//	THIS,   OTHER: (always clone the other kid!)
//	--------------
//	NULL,   node    <-- insert node at addr
//	NULL,   leaf    <-- insert leaf at addr
//	NULL,   fringe  <-- insert fringe at addr
//
//	node,   node    <-- union rec-descent with node
//	node,   leaf    <-- insert leaf at depth+1
//	node,   fringe  <-- insert fringe at depth+1
//
//	leaf,   node    <-- insert new node, push this leaf down, union rec-descent
//	leaf,   leaf    <-- insert new node, push both leaves down (!first check equality)
//	leaf,   fringe  <-- insert new node, push this leaf and fringe down
//
//	fringe, node    <-- insert new node, push this fringe down, union rec-descent
//	fringe, leaf    <-- insert new node, push this fringe down, insert other leaf at depth+1
//	fringe, fringe  <-- just overwrite value
func (n *_NODE_TYPE[V]) handleMatrix(cloneFn cloneFunc[V], thisExists bool, thisChild, otherChild any, addr uint8, depth int) int {
	// Do ALL type assertions upfront - reduces line noise
	var (
		thisNode, thisIsNode     = thisChild.(*_NODE_TYPE[V])
		thisLeaf, thisIsLeaf     = thisChild.(*leafNode[V])
		thisFringe, thisIsFringe = thisChild.(*fringeNode[V])

		otherNode, otherIsNode     = otherChild.(*_NODE_TYPE[V])
		otherLeaf, otherIsLeaf     = otherChild.(*leafNode[V])
		otherFringe, otherIsFringe = otherChild.(*fringeNode[V])
	)

	// just insert cloned child at this empty slot
	if !thisExists {
		switch {
		case otherIsNode:
			n.insertChild(addr, otherNode.cloneRec(cloneFn))
		case otherIsLeaf:
			n.insertChild(addr, &leafNode[V]{prefix: otherLeaf.prefix, value: cloneFn(otherLeaf.value)})
		case otherIsFringe:
			n.insertChild(addr, &fringeNode[V]{value: cloneFn(otherFringe.value)})
		default:
			panic("logic error, wrong node type")
		}
		return 0
	}

	// Case 1: Special cases that DON'T need a new node

	// Special case: fringe + fringe -> just overwrite value
	if thisIsFringe && otherIsFringe {
		thisFringe.value = cloneFn(otherFringe.value)
		return 1
	}

	// Special case: leaf + leaf with same prefix -> just overwrite value
	if thisIsLeaf && otherIsLeaf && thisLeaf.prefix == otherLeaf.prefix {
		thisLeaf.value = cloneFn(otherLeaf.value)
		return 1
	}

	// Case 2: thisChild is already a node - insert into it, no new node needed
	if thisIsNode {
		switch {
		case otherIsNode:
			return thisNode.unionRec(cloneFn, otherNode, depth+1)
		case otherIsLeaf:
			if thisNode.insert(otherLeaf.prefix, cloneFn(otherLeaf.value), depth+1) {
				return 1
			}
			return 0
		case otherIsFringe:
			if thisNode.insertPrefix(1, cloneFn(otherFringe.value)) {
				return 1
			}
			return 0
		default:
			panic("logic error, wrong node type")
		}
	}

	// Case 3: All remaining cases need a new node
	// (thisChild is leaf or fringe, and we didn't hit the special cases above)

	nc := new(_NODE_TYPE[V])

	// Push existing child down into new node
	switch {
	case thisIsLeaf:
		nc.insert(thisLeaf.prefix, thisLeaf.value, depth+1)
	case thisIsFringe:
		nc.insertPrefix(1, thisFringe.value)
	default:
		panic("logic error, unexpected this child type")
	}

	// Replace child with new node
	n.insertChild(addr, nc)

	// Now handle other child
	switch {
	case otherIsNode:
		return nc.unionRec(cloneFn, otherNode, depth+1)
	case otherIsLeaf:
		if nc.insert(otherLeaf.prefix, cloneFn(otherLeaf.value), depth+1) {
			return 1
		}
		return 0
	case otherIsFringe:
		if nc.insertPrefix(1, cloneFn(otherFringe.value)) {
			return 1
		}
		return 0
	default:
		panic("logic error, wrong other node type")
	}
}

// handleMatrixPersist, 12 possible combinations to union this child and other child
//
//	THIS,   OTHER: (always clone the other kid!)
//	--------------
//	NULL,   node    <-- insert node at addr
//	NULL,   leaf    <-- insert leaf at addr
//	NULL,   fringe  <-- insert fringe at addr
//
//	node,   node    <-- union rec-descent with node
//	node,   leaf    <-- insert leaf at depth+1
//	node,   fringe  <-- insert fringe at depth+1
//
//	leaf,   node    <-- insert new node, push this leaf down, union rec-descent
//	leaf,   leaf    <-- insert new node, push both leaves down (!first check equality)
//	leaf,   fringe  <-- insert new node, push this leaf and fringe down
//
//	fringe, node    <-- insert new node, push this fringe down, union rec-descent
//	fringe, leaf    <-- insert new node, push this fringe down, insert other leaf at depth+1
//	fringe, fringe  <-- just overwrite value
func (n *_NODE_TYPE[V]) handleMatrixPersist(cloneFn cloneFunc[V], thisExists bool, thisChild, otherChild any, addr uint8, depth int) int {
	// Do ALL type assertions upfront - reduces line noise
	var (
		thisNode, thisIsNode     = thisChild.(*_NODE_TYPE[V])
		thisLeaf, thisIsLeaf     = thisChild.(*leafNode[V])
		thisFringe, thisIsFringe = thisChild.(*fringeNode[V])

		otherNode, otherIsNode     = otherChild.(*_NODE_TYPE[V])
		otherLeaf, otherIsLeaf     = otherChild.(*leafNode[V])
		otherFringe, otherIsFringe = otherChild.(*fringeNode[V])
	)

	// just insert cloned child at this empty slot
	if !thisExists {
		switch {
		case otherIsNode:
			n.insertChild(addr, otherNode.cloneRec(cloneFn))
		case otherIsLeaf:
			n.insertChild(addr, &leafNode[V]{prefix: otherLeaf.prefix, value: cloneFn(otherLeaf.value)})
		case otherIsFringe:
			n.insertChild(addr, &fringeNode[V]{value: cloneFn(otherFringe.value)})
		default:
			panic("logic error, wrong node type")
		}
		return 0
	}

	// Case 1: Special cases that DON'T need a new node

	// Special case: fringe + fringe -> just overwrite value
	if thisIsFringe && otherIsFringe {
		thisFringe.value = cloneFn(otherFringe.value)
		return 1
	}

	// Special case: leaf + leaf with same prefix -> just overwrite value
	if thisIsLeaf && otherIsLeaf && thisLeaf.prefix == otherLeaf.prefix {
		thisLeaf.value = cloneFn(otherLeaf.value)
		return 1
	}

	// Case 2: thisChild is already a node - clone this node, insert into it
	if thisIsNode {
		// CLONE the node

		// thisNode points now to cloned kid
		thisNode = thisNode.cloneFlat(cloneFn)

		// replace kid with cloned thisKid
		n.insertChild(addr, thisNode)

		switch {
		case otherIsNode:
			return thisNode.unionRecPersist(cloneFn, otherNode, depth+1)
		case otherIsLeaf:
			if thisNode.insertPersist(cloneFn, otherLeaf.prefix, cloneFn(otherLeaf.value), depth+1) {
				return 1
			}
			return 0
		case otherIsFringe:
			if thisNode.insertPrefix(1, cloneFn(otherFringe.value)) {
				return 1
			}
			return 0
		default:
			panic("logic error, wrong node type")
		}
	}

	// Case 3: All remaining cases need a new node
	// (thisChild is leaf or fringe, and we didn't hit the special cases above)

	nc := new(_NODE_TYPE[V])

	// Push existing child down into new node
	switch {
	case thisIsLeaf:
		nc.insert(thisLeaf.prefix, thisLeaf.value, depth+1)
	case thisIsFringe:
		nc.insertPrefix(1, thisFringe.value)
	default:
		panic("logic error, unexpected this child type")
	}

	// Replace child with new node
	n.insertChild(addr, nc)

	// Now handle other child
	switch {
	case otherIsNode:
		return nc.unionRec(cloneFn, otherNode, depth+1)
	case otherIsLeaf:
		if nc.insert(otherLeaf.prefix, cloneFn(otherLeaf.value), depth+1) {
			return 1
		}
		return 0
	case otherIsFringe:
		if nc.insertPrefix(1, cloneFn(otherFringe.value)) {
			return 1
		}
		return 0
	default:
		panic("logic error, wrong other node type")
	}
}
