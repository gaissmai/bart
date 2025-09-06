package bart

import (
	"net/netip"
	"sync"

	"github.com/gaissmai/bart/internal/art"
)

// Table TODO
type Table[V any] struct {
	// used by -copylocks checker from `go vet`.
	_ [0]sync.Mutex

	// the root nodes
	root4 node[V]
	root6 node[V]

	// the number of prefixes in the routing table
	size4 int
	size6 int
}

func (t *Table[V]) Insert(pfx netip.Prefix, val V) {
	if !pfx.IsValid() {
		return
	}

	// canonicalize prefix
	pfx = pfx.Masked()

	// values derived from pfx
	ip := pfx.Addr()
	is4 := ip.Is4()
	bits := pfx.Bits()
	maxDepth, lastBits := maxDepthAndLastBits(bits)
	octets := ip.AsSlice()
	lastOctet := uint8(0)
	if maxDepth < len(octets) {
		lastOctet = octets[maxDepth]
	}

	n := t.rootNodeByVersion(is4)
	n.initOnceRootPath(is4)

	// find the trie node
	idx := 0
	for idx < len(octets) {
		octet := octets[idx]

		// insert this prefix
		if idx == maxDepth {
			if exists := n.insertPrefix(art.PfxToIdx(lastOctet, lastBits), val); !exists {
				t.sizeUpdate(is4, 1)
			}
			return
		}

		// proceed to next trie level
		kid := n.getChild(octet)

		// insert new path compressed node for this prefix
		if kid == nil {
			kid = newNode[V](pfx)
			// insert this prefix into new kid
			kid.insertPrefix(art.PfxToIdx(lastOctet, lastBits), val)

			// insert new kid into n's children
			n.insertChild(octet, kid)

			t.sizeUpdate(is4, 1)
			return
		}

		// kid already exists and share same octet path
		if kid.containsPrefix(pfx) {

			// pfx is in kids trie level
			if pfx.Bits()-kid.basePath.Bits() < 8 {
				if exists := kid.insertPrefix(art.PfxToIdx(lastOctet, lastBits), val); !exists {
					t.sizeUpdate(is4, 1)
				}
				return
			}

			// pfx is some levels deeper
			idx = kid.basePath.Bits() / 8
			n = kid // proceed with this kid
			continue
		}

		panic("not implemented")
		// TODO
		// get common prefix bits
		// make intermediate node
		// insert this kid
		// insert this pfx
		// size update
	}
}

/*
// Modify TODO
func (t *Table[V]) Modify(pfx netip.Prefix, cb func(val V, ok bool) (newVal V, del bool)) (newVal V, deleted bool) {
	var zero V

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

	n := d.rootNodeByVersion(is4)

	// record the nodes on the path to the deleted node, needed to purge
	// and/or path compress nodes after the deletion of a prefix
	stack := [maxTreeDepth]*node[V]{}

	// find the trie node
	for depth, octet := range octets {
		depth = depth & 0xf // BCE

		// push current node on stack for path recording
		stack[depth] = n

		if depth == maxDepth {
			idx := art.PfxToIdx(octet, lastBits)

			oldVal, existed := n.getPrefix(idx)
			newVal, del := cb(oldVal, existed)

			// update size if necessary
			switch {
			case !existed && del: // no-op
				return zero, false

			case existed && del: // delete
				n.deletePrefix(idx)
				d.sizeUpdate(is4, -1)
				n.purgeAndCompress(stack[:depth], octets, is4)
				return oldVal, true

			case !existed: // insert
				n.insertPrefix(idx, newVal)
				d.sizeUpdate(is4, 1)
				return newVal, false

			case existed: // update
				n.insertPrefix(idx, newVal)
				return oldVal, false

			default:
				panic("unreachable")
			}
		}

		anyPtr := n.getChild(octet)
		if anyPtr == nil {
			// insert prefix path compressed

			newVal, del := cb(zero, false)
			if del {
				return zero, false // no-op
			}

			// insert
			if isFringe(depth, bits) {
				n.insertChild(octet, newFringeNode(newVal))
			} else {
				n.insertChild(octet, newLeafNode(pfx, newVal))
			}

			d.sizeUpdate(is4, 1)
			return newVal, false
		}

		// kid is node or leaf or fringe at octet
		kidAny := *anyPtr
		switch kid := kidAny.(type) {
		case *node[V]:
			n = kid // descend down to next trie level

		case *fringeNode[V]:
			oldVal := kid.value

			// update existing value if prefix is fringe
			if isFringe(depth, bits) {
				newVal, del := cb(kid.value, true)
				if !del {
					kid.value = newVal
					return oldVal, false // update
				}

				// delete
				n.deleteChild(octet)

				d.sizeUpdate(is4, -1)
				n.purgeAndCompress(stack[:depth], octets, is4)

				return oldVal, true // delete
			}

			// create new node ART node
			// push the fringe down, it becomes a default route (idx=1)
			// insert new child at current leaf position (addr)
			// descend down, replace n with new child
			newNode := new(node[V])
			_ = newNode.insertPrefix(1, kid.value)
			_ = n.insertChild(octet, newNode)
			n = newNode

		case *leafNode[V]:
			oldVal := kid.value

			// update existing value if prefixes are equal
			if kid.prefix == pfx {
				newVal, del := cb(kid.value, true)
				if !del {
					kid.value = newVal
					return oldVal, false // update
				}

				// delete
				n.deleteChild(octet)

				d.sizeUpdate(is4, -1)
				n.purgeAndCompress(stack[:depth], octets, is4)

				return oldVal, true // delete
			}

			// create new node
			// push the leaf down
			// insert new child at current leaf position (addr)
			// descend down, replace n with new child
			newNode := new(node[V])
			_ = newNode.insertAtDepth(kid.prefix, kid.value, depth+1)
			_ = n.insertChild(octet, newNode)
			n = newNode

		default:
			panic("logic error, wrong node type")
		}
	}

	return
}

// Delete deletes the prefix and returns the associated payload for prefix and true,
// or the zero value and false if prefix is not set in the routing table.
func (t *Table[V]) Delete(pfx netip.Prefix) (val V, exists bool) {
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

	n := d.rootNodeByVersion(is4)

	// record the nodes on the path to the deleted node, needed to purge
	// and/or path compress nodes after the deletion of a prefix
	stack := [maxTreeDepth]*node[V]{}

	// find the trie node
	for depth, octet := range octets {
		depth = depth & 0xf // BCE, Delete must be fast

		// push current node on stack for path recording
		stack[depth] = n

		if depth == maxDepth {
			// try to delete prefix in trie node
			val, exists = n.deletePrefix(art.PfxToIdx(octet, lastBits))
			if !exists {
				return val, false
			}

			d.sizeUpdate(is4, -1)
			n.purgeAndCompress(stack[:depth], octets, is4)
			return val, true
		}

		anyPtr := n.getChild(octet)
		if anyPtr == nil {
			return
		}
		kidAny := *anyPtr

		// kid is node or leaf or fringe at octet
		switch kid := kidAny.(type) {
		case *node[V]:
			n = kid // descend down to next trie level

		case *fringeNode[V]:
			// if pfx is no fringe at this depth, fast exit
			if !isFringe(depth, bits) {
				return
			}

			// pfx is fringe at depth, delete fringe
			n.deleteChild(octet)

			d.sizeUpdate(is4, -1)
			n.purgeAndCompress(stack[:depth], octets, is4)

			return kid.value, true

		case *leafNode[V]:
			// Attention: pfx must be masked to be comparable!
			if kid.prefix != pfx {
				return
			}

			// prefix is equal leaf, delete leaf
			n.deleteChild(octet)

			d.sizeUpdate(is4, -1)
			n.purgeAndCompress(stack[:depth], octets, is4)

			return kid.value, true

		default:
			panic("logic error, wrong node type")
		}
	}

	return
}

// Get returns the associated payload for prefix and true, or false if
// prefix is not set in the routing table.
func (t *Table[V]) Get(pfx netip.Prefix) (val V, ok bool) {
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

	n := d.rootNodeByVersion(is4)

	// find the trie node
	for depth, octet := range octets {
		if depth == maxDepth {
			return n.getPrefix(art.PfxToIdx(octet, lastBits))
		}

		anyPtr := n.getChild(octet)
		if anyPtr == nil {
			return
		}
		kidAny := *anyPtr

		// kid is node or leaf or fringe at octet
		switch kid := kidAny.(type) {
		case *node[V]:
			n = kid // descend down to next trie level

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
	}

	panic("unreachable")
}
*/

// Contains TODO
func (t *Table[V]) Contains(ip netip.Addr) bool {
	if !ip.IsValid() {
		return false
	}

	octets := ip.AsSlice()

	var n *node[V]
	if len(octets) == 4 {
		n = &t.root4
	} else {
		n = &t.root6
	}

	idx := uint(0) // use uint for BCE
	for idx < uint(len(octets)) {
		octet := octets[idx]

		if n.contains(uint(octet) + 256) {
			return true
		}

		kid := n.getChild(octet)
		if kid == nil {
			// no next node
			return false
		}
		n = kid

		idx++ // proceed to next octet, but ...
		if next := n.basePath.Bits() >> 3; next > int(idx) {
			// fast forward for path compression?
			if !n.basePath.Contains(ip) {
				return false
			}
			idx = uint(next) // larger step
		}
	}

	// last kid has default route
	return true
}

// Lookup TODO
func (t *Table[V]) Lookup(ip netip.Addr) (val V, ok bool) {
	if !ip.IsValid() {
		return
	}

	octets := ip.AsSlice()

	var n *node[V]
	if len(octets) == 4 {
		n = &t.root4
	} else {
		n = &t.root6
	}

	idx := uint(0) // use uint for BCE
	for idx < uint(len(octets)) {
		octet := octets[idx]

		// get and save current LPM val
		if tmpVal, tmpOk := n.lookup(uint(octet) + 256); tmpOk {
			val = tmpVal
			ok = tmpOk
		}

		// go to next trie level
		n = n.getChild(octet)
		if n == nil { // no next node
			return
		}

		idx++ // proceed to next octet, but ...
		if next := n.basePath.Bits() >> 3; next > int(idx) {
			// fast forward for path compression?
			if !n.basePath.Contains(ip) {
				return val, ok // return current best LPM val
			}
			idx = uint(next) // larger step
		}

	}

	// last fringe node is default route
	return *n.prefixes[1], true
}

func (t *Table[V]) sizeUpdate(is4 bool, n int) {
	if is4 {
		t.size4 += n
		return
	}
	t.size6 += n
}

// Size returns the prefix count.
func (t *Table[V]) Size() int {
	return t.size4 + t.size6
}

// Size4 returns the IPv4 prefix count.
func (t *Table[V]) Size4() int {
	return t.size4
}

// Size6 returns the IPv6 prefix count.
func (t *Table[V]) Size6() int {
	return t.size6
}

func maxDepthAndLastBits(bits int) (maxDepth int, lastBits uint8) {
	// maxDepth:  range from 0..4 or 0..16 !ATTENTION: not 0..3 or 0..15
	// lastBits:  range from 0..7
	return bits >> 3, uint8(bits & 7)
}

// rootNodeByVersion, root node getter for ip version and ART levels.
func (t *Table[V]) rootNodeByVersion(is4 bool) *node[V] {
	if is4 {
		return &t.root4
	}
	return &t.root6
}
