package bart

import (
	"net/netip"
	"sync"

	"github.com/gaissmai/bart/internal/art"
	"github.com/gaissmai/bart/internal/lpm"
)

// DART is a dual routing table design that combines ART and BART.
//
// The first two levels of the trie - covering prefixes up to /16 - are
// implemented using ART with fixed-size arrays.
//
// By combining ART and BART, DART shifts the balance towards better lookup speed
// but with higher memory usage.
//
// DART is specifically optimized for Internet routing tables where prefixes are densely packed up to /16.
// Conceptually, this architecture can be thought of as ART functioning as a software TCAM lookup engine
// for the initial /16 bits in front of BART.
//
// Every empty Dart table uses at least 4MB. So if you only have a few routing entries
// and need to use little memory, you should use [bart.Table].
type Dart[V any] struct {
	// used by -copylocks checker from `go vet`.
	_ [0]sync.Mutex

	// the root nodes are ART nodes, fixed size arrays
	// nodes starting with level 2 are BART nodes, popcount compressed arrays.
	root4 artNode[V]
	root6 artNode[V]

	// the number of prefixes in the routing table
	size4 int
	size6 int
}

// rootNodeByVersion, root node getter for ip version and ART levels.
func (d *Dart[V]) rootNodeByVersion(is4 bool) (node *artNode[V], artLevels int) {
	if is4 {
		return &d.root4, 2 // default
	}
	return &d.root6, 4 // default
}

func (d *Dart[V]) Insert(pfx netip.Prefix, val V) {
	if !pfx.IsValid() {
		return
	}

	// canonicalize prefix
	pfx = pfx.Masked()

	// values derived from pfx
	ip := pfx.Addr()
	is4 := ip.Is4()
	bits := pfx.Bits()
	octets := ip.AsSlice()
	maxDepth, lastBits := finalArt(bits)

	nArt, artLevels := d.rootNodeByVersion(is4)

	var nBart *bartNode[V]

	// insert prefix into ART or fast forward over ART levels
	for depth, octet := range octets[:artLevels] {
		if depth == maxDepth {
			if exists := nArt.insertPrefix(octet, lastBits, val); !exists {
				d.sizeUpdate(is4, 1)
			}
			return
		}

		// maybe nil
		kidAny := nArt.getChild(octet)

		// insert leafNode path compressed
		if kidAny == nil {
			nArt.insertChild(octet, newLeafNode(pfx, val))
			d.sizeUpdate(is4, 1)
			return
		}

		switch kid := kidAny.(type) {
		case *artNode[V]:
			nArt = kid
		case *bartNode[V]:
			if depth != artLevels-1 {
				panic("TODE REMOVE: logic error, bartNode on wrong ART level")
			}
			nBart = kid
		case *leafNode[V]:
			if kid.prefix == pfx {
				kid.value = val
				return
			}

			d.sizeUpdate(is4, 1)

			if depth == artLevels-1 {
				// create new node BART node
				// push the leaf down, maybe as leaf or fringe
				// insert new child at current leaf position (addr)
				// descend down, replace node with new child
				// exit ART loop
				newNode := new(bartNode[V])
				newNode.insertAtDepth(kid.prefix, kid.value, depth+1)

				nArt.insertChild(octet, newNode)
				nBart = newNode
			}

			// create new node ART node
			// push the leaf down, ART has no fringes, only leaves
			// insert new child at current leaf position
			// descend down, replace node with new child
			newNode := new(artNode[V])
			newNode.insertAtDepth(kid.prefix, kid.value, depth+1)

			nArt.insertChild(octet, newNode)
			nArt = newNode

		default:
			panic("logic error, wrong node type in ART level")
		}
	}

	// insert prefix into BART
	if exists := nBart.insertAtDepth(pfx, val, artLevels); exists {
		return
	}

	// true insert, update size
	d.sizeUpdate(is4, 1)
}

// Delete removes pfx from the tree, pfx does not have to be present.
func (d *Dart[V]) Delete(pfx netip.Prefix) {
	_, _ = d.getAndDelete(pfx)
}

// GetAndDelete deletes the prefix and returns the associated payload for prefix and true,
// or the zero value and false if prefix is not set in the routing table.
func (d *Dart[V]) GetAndDelete(pfx netip.Prefix) (val V, ok bool) {
	return d.getAndDelete(pfx)
}

func (d *Dart[V]) getAndDelete(pfx netip.Prefix) (val V, exists bool) {
	if !pfx.IsValid() {
		return
	}

	// canonicalize prefix
	pfx = pfx.Masked()

	// values derived from pfx
	ip := pfx.Addr()
	is4 := ip.Is4()
	bits := pfx.Bits()
	octets := ip.AsSlice()
	maxDepth, lastBits := maxDepthAndLastBits(bits)

	nArt, artLevels := d.rootNodeByVersion(is4)

	var nBart *bartNode[V]

	// delete prefix from ART or fast forward over ART levels
	for depth, octet := range octets[:artLevels] {
		levelBits := bits - (depth * 8)

		if levelBits <= 8 {
			val, exists = nArt.deletePrefix(octet, levelBits)
			if !exists {
				d.sizeUpdate(is4, -1)
			}
			return val, exists
		}

		// get next child in ART levels, maybe nil
		next := nArt.getChild(octet)
		if next == nil {
			// nothing to delete
			return
		}

		// last ART level, assert BART node and break loop
		if depth == artLevels-1 {
			nBart = next.(*bartNode[V])
			break
		}

		// assert ART node and move forward
		nArt = next.(*artNode[V])
	}

	// record the nodes on the path to the deleted node, needed to purge
	// and/or path compress nodes after the deletion of a prefix
	stack := [maxTreeDepth]*bartNode[V]{}

	// find the BART trie node
	depth := artLevels
	for _, octet := range octets[artLevels:] {
		// push current node on stack for path recording
		stack[depth] = nBart

		if depth == maxDepth {
			// try to delete prefix in trie node
			val, exists = nBart.prefixes.DeleteAt(art.PfxToIdx(octet, lastBits))
			if !exists {
				return
			}

			d.sizeUpdate(is4, -1)
			nBart.purgeAndCompress(stack[:depth], octets, is4)
			return val, true
		}

		if !nBart.children.Test(octet) {
			return
		}
		kid := nBart.children.MustGet(octet)

		// kid is node or leaf or fringe at octet
		switch kid := kid.(type) {
		case *bartNode[V]:
			nBart = kid
			break // descend down to next trie level

		case *fringeNode[V]:
			// if pfx is no fringe at this depth, fast exit
			if !isFringe(depth, bits) {
				return
			}

			// pfx is fringe at depth, delete fringe
			nBart.children.DeleteAt(octet)

			d.sizeUpdate(is4, -1)
			nBart.purgeAndCompress(stack[:depth], octets, is4)

			return kid.value, true

		case *leafNode[V]:
			// Attention: pfx must be masked to be comparable!
			if kid.prefix != pfx {
				return
			}

			// prefix is equal leaf, delete leaf
			nBart.children.DeleteAt(octet)

			d.sizeUpdate(is4, -1)
			nBart.purgeAndCompress(stack[:depth], octets, is4)

			return kid.value, true

		default:
			panic("logic error, wrong node type")
		}

		depth++
	}

	return
}

// Get returns the associated payload for prefix and true, or false if
// prefix is not set in the routing table.
func (d *Dart[V]) Get(pfx netip.Prefix) (val V, ok bool) {
	if !pfx.IsValid() {
		return
	}

	// canonicalize the prefix
	pfx = pfx.Masked()

	// values derived from pfx
	ip := pfx.Addr()
	is4 := ip.Is4()
	bits := pfx.Bits()
	octets := ip.AsSlice()
	maxDepth, lastBits := maxDepthAndLastBits(bits)

	nArt, artLevels := d.rootNodeByVersion(is4)

	var nBart *bartNode[V]

	// get prefix from ART or fast forward to BART
	for depth, octet := range octets[:artLevels] {
		levelBits := bits - (depth * 8)

		if levelBits <= 8 {
			return nArt.getPrefix(octet, levelBits)
		}

		// get next child in ART levels, maybe nil
		next := nArt.getChild(octet)
		if next == nil {
			return
		}

		// last ART level, assert BART node and break loop
		if depth == artLevels-1 {
			nBart = next.(*bartNode[V])
			break
		}

		// assert ART node and move forward
		nArt = next.(*artNode[V])
	}

	depth := artLevels
	// find the trie node
	for _, octet := range octets[artLevels:] {
		if depth == maxDepth {
			return nBart.prefixes.Get(art.PfxToIdx(octet, lastBits))
		}

		if !nBart.children.Test(octet) {
			return
		}
		kid := nBart.children.MustGet(octet)

		// kid is node or leaf or fringe at octet
		switch kid := kid.(type) {
		case *bartNode[V]:
			nBart = kid
			break // descend down to next trie level

		case *fringeNode[V]:
			// reached a path compressed fringe, stop traversing
			if isFringe(depth, bits) {
				return kid.value, true
			}
			return

		case *leafNode[V]:
			// reached a path compressed prefix, stop traversing
			if kid.prefix == pfx {
				return kid.value, true
			}
			return

		default:
			panic("logic error, wrong node type")
		}

		depth++
	}

	panic("unreachable")
}

// Contains TODO
func (d *Dart[V]) Contains(ip netip.Addr) bool {
	if !ip.IsValid() {
		return false
	}

	is4 := ip.Is4()
	octets := ip.AsSlice()

	nArt, artLevels := d.rootNodeByVersion(is4)

	var nBart *bartNode[V]

	// first test in ART levels and if not fount in BART levels
	for depth, octet := range octets[:artLevels] {
		if nArt.contains(octet) {
			return true
		}

		next := nArt.getChild(octet)
		if next == nil {
			return false
		}

		// last ART level, assert the BART node
		if depth == artLevels-1 {
			nBart = next.(*bartNode[V])
			break
		}

		// proceed to next ART level
		nArt = next.(*artNode[V])
	}

	for _, octet := range octets[artLevels:] {
		// for contains, any lpm match is good enough, no backtracking needed
		if nBart.prefixes.Len() != 0 && nBart.lpmTest(art.OctetToIdx(octet)) {
			return true
		}

		// stop traversing?
		if !nBart.children.Test(octet) {
			return false
		}
		kid := nBart.children.MustGet(octet)

		// kid is node or leaf or fringe at octet
		switch kid := kid.(type) {
		case *bartNode[V]:
			nBart = kid
			break // descend down to next trie level

		case *fringeNode[V]:
			// fringe is the default-route for all possible octets below
			return true

		case *leafNode[V]:
			return kid.prefix.Contains(ip)

		default:
			panic("logic error, wrong node type")
		}
	}

	return false
}

// Lookup TODO
func (d *Dart[V]) Lookup(ip netip.Addr) (val V, ok bool) {
	if !ip.IsValid() {
		return
	}

	is4 := ip.Is4()
	octets := ip.AsSlice()

	nArt, artLevels := d.rootNodeByVersion(is4)

	var nBart *bartNode[V]

	// fast forward to BART levels, but record LPM matches in ART
	for depth, octet := range octets[:artLevels] {

		// save the current best LPM val, lookup is cheap in ART
		if tmpVal, tmpOk := nArt.lookup(octet); tmpOk {
			val = tmpVal
			ok = tmpOk
		}

		next := nArt.getChild(octet)
		if next == nil {
			// get next node, or return
			return val, ok
		}

		// last ART level, assert the BART node
		if depth == artLevels-1 {
			nBart = next.(*bartNode[V])
			break
		}

		// proceed to next ART level
		nArt = next.(*artNode[V])
	}

	// stack of the traversed nodes for fast backtracking, if needed
	stack := [maxTreeDepth]*bartNode[V]{}

	depth := artLevels
LOOP:
	for _, octet := range octets[artLevels:] {
		// push current node on stack for fast backtracking
		stack[depth] = nBart

		// go down in tight loop to last octet
		if !nBart.children.Test(octet) {
			// no more nodes below octet
			break LOOP
		}
		kid := nBart.children.MustGet(octet)

		// kid is node or leaf or fringe at octet
		switch kid := kid.(type) {
		case *bartNode[V]:
			nBart = kid
			break // descend down to next trie level

		case *fringeNode[V]:
			// fringe is the default-route for all possible nodes below
			return kid.value, true

		case *leafNode[V]:
			if kid.prefix.Contains(ip) {
				return kid.value, true
			}
			// reached a path compressed prefix, stop traversing
			break LOOP

		default:
			panic("logic error, wrong node type")
		}

		depth++
	}

	// start backtracking, unwind the stack, bounds check eliminated
	for ; depth >= artLevels; depth-- {
		depth = depth & 0xf // BCE

		nBart = stack[depth]

		// longest prefix match, skip if node has no prefixes
		if nBart.prefixes.Len() != 0 {
			idx := art.OctetToIdx(octets[depth])
			// --------------------------------------------------------------
			// ---------------- lpmGet(idx), manually inlined ---------------
			// --------------------------------------------------------------
			if topIdx, ok := nBart.prefixes.IntersectionTop(lpm.BackTrackingBitset(idx)); ok {
				return nBart.prefixes.MustGet(topIdx), true
			}
		}
	}

	// no match in BART, maybe ART stored a best match in val/ok
	return val, ok
}

func (d *Dart[V]) sizeUpdate(is4 bool, n int) {
	if is4 {
		d.size4 += n
		return
	}
	d.size6 += n
}

// Size returns the prefix count.
func (d *Dart[V]) Size() int {
	return d.size4 + d.size6
}

// Size4 returns the IPv4 prefix count.
func (d *Dart[V]) Size4() int {
	return d.size4
}

// Size6 returns the IPv6 prefix count.
func (d *Dart[V]) Size6() int {
	return d.size6
}

func finalArt(bits int) (maxDepth int, lastBits int) {
	// maxDepth:  range from 0..3 or 0..15
	// lastBits:  range from 0..8
	maxDepth = (bits - 1) / 8
	lastBits = bits - (maxDepth * 8)
	return
}
