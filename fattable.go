package bart

import (
	"net/netip"
	"sync"

	"github.com/gaissmai/bart/internal/art"
)

// Fat follows the original ART design by Knuth in using fixed
// 256-slot arrays at each level.
// In contrast to the original, this variant introduces a new form of path
// compression. This keeps memory usage within a reasonable range while
// preserving the high lookup speed of the pure array-based ART algorithm.
//
// Both [bart.Fat] and [bart.Table] use the same path compression, but they
// differ in how levels are represented:
//
//   - [bart.Fat]:   uncompressed  fixed level arrays + path compression
//   - [bart.Table]: popcount-compressed level arrays + path compression
//
// As a result:
//   - [bart.Fat] sacrifices memory efficiency to achieve about 2x higher speed
//   - [bart.Table] minimizes memory consumption as much as possible
//
// Which variant is preferable depends on the use case: [bart.Fat] is most
// beneficial when maximum speed for longest-prefix-match is the top priority,
// for example in a Forwarding Information Base (FIB).
//
// For the full Internet routing table, the [bart.Fat] structure alone requires
// about 250 MB of memory, with additional space needed for payload such as
// next hop, interface, and further attributes.
type Fat[V any] struct {
	// used by -copylocks checker from `go vet`.
	_ [0]sync.Mutex

	// the root nodes are fat nodes with fixed size arrays
	root4 fatNode[V]
	root6 fatNode[V]

	// the number of prefixes in the routing table
	size4 int
	size6 int
}

// rootNodeByVersion, root node getter for ip version and trie levels.
func (f *Fat[V]) rootNodeByVersion(is4 bool) *fatNode[V] {
	if is4 {
		return &f.root4
	}
	return &f.root6
}

// Insert adds a prefix with the given value.
// Its semantics are identical to [Table.Insert].
func (f *Fat[V]) Insert(pfx netip.Prefix, val V) {
	if !pfx.IsValid() {
		return
	}
	// canonicalize prefix
	pfx = pfx.Masked()
	is4 := pfx.Addr().Is4()

	n := f.rootNodeByVersion(is4)

	// insert prefix
	if exists := n.insertAtDepth(pfx, val, 0); exists {
		return
	}

	// true insert, update size
	f.sizeUpdate(is4, 1)
}

// Modify applies a callback to the value of the given prefix.
// Its semantics are identical to [Table.Modify].
func (f *Fat[V]) Modify(pfx netip.Prefix, cb func(val V, found bool) (_ V, del bool)) (_ V, deleted bool) {
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

	n := f.rootNodeByVersion(is4)

	// record the nodes on the path to the deleted node, needed to purge
	// and/or path compress nodes after the deletion of a prefix
	stack := [maxTreeDepth]*fatNode[V]{}

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
				f.sizeUpdate(is4, -1)
				n.purgeAndCompress(stack[:depth], octets, is4)
				return oldVal, true

			case !existed: // insert
				n.insertPrefix(idx, newVal)
				f.sizeUpdate(is4, 1)
				return newVal, false

			case existed: // update
				n.insertPrefix(idx, newVal)
				return oldVal, false

			default:
				panic("unreachable")
			}
		}

		kidAny, exists := n.getChild(octet)
		if !exists {
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

			f.sizeUpdate(is4, 1)
			return newVal, false
		}

		// kid is node or leaf or fringe at octet
		switch kid := kidAny.(type) {
		case *fatNode[V]:
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

				f.sizeUpdate(is4, -1)
				n.purgeAndCompress(stack[:depth], octets, is4)

				return oldVal, true // delete
			}

			// create new node
			// push the fringe down, it becomes a default route (idx=1)
			// insert new child at current leaf position (addr)
			// descend down, replace n with new child
			newNode := new(fatNode[V])
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

				f.sizeUpdate(is4, -1)
				n.purgeAndCompress(stack[:depth], octets, is4)

				return oldVal, true // delete
			}

			// create new node
			// push the leaf down
			// insert new child at current leaf position (addr)
			// descend down, replace n with new child
			newNode := new(fatNode[V])
			_ = newNode.insertAtDepth(kid.prefix, kid.value, depth+1)
			_ = n.insertChild(octet, newNode)
			n = newNode

		default:
			panic("logic error, wrong node type")
		}
	}

	return
}

// Delete removes the given prefix and returns its value and whether it existed.
// Its semantics are identical to [Table.Delete].
func (f *Fat[V]) Delete(pfx netip.Prefix) (val V, exists bool) {
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

	n := f.rootNodeByVersion(is4)

	// record the nodes on the path to the deleted node, needed to purge
	// and/or path compress nodes after the deletion of a prefix
	stack := [maxTreeDepth]*fatNode[V]{}

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

			f.sizeUpdate(is4, -1)
			n.purgeAndCompress(stack[:depth], octets, is4)
			return val, true
		}

		kidAny, ok := n.getChild(octet)
		if !ok {
			return
		}

		// kid is node or leaf or fringe at octet
		switch kid := kidAny.(type) {
		case *fatNode[V]:
			n = kid // descend down to next trie level

		case *fringeNode[V]:
			// if pfx is no fringe at this depth, fast exit
			if !isFringe(depth, bits) {
				return
			}

			// pfx is fringe at depth, delete fringe
			n.deleteChild(octet)

			f.sizeUpdate(is4, -1)
			n.purgeAndCompress(stack[:depth], octets, is4)

			return kid.value, true

		case *leafNode[V]:
			// Attention: pfx must be masked to be comparable!
			if kid.prefix != pfx {
				return
			}

			// prefix is equal leaf, delete leaf
			n.deleteChild(octet)

			f.sizeUpdate(is4, -1)
			n.purgeAndCompress(stack[:depth], octets, is4)

			return kid.value, true

		default:
			panic("logic error, wrong node type")
		}
	}

	return
}

// Get looks up the given prefix and returns its value and whether it exists.
// Its semantics are identical to [Table.Get].
func (f *Fat[V]) Get(pfx netip.Prefix) (val V, ok bool) {
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

	n := f.rootNodeByVersion(is4)

	// find the trie node
	for depth, octet := range octets {
		if depth == maxDepth {
			return n.getPrefix(art.PfxToIdx(octet, lastBits))
		}

		kidAny, exists := n.getChild(octet)
		if !exists {
			return
		}

		// kid is node or leaf or fringe at octet
		switch kid := kidAny.(type) {
		case *fatNode[V]:
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

// Contains reports whether the given address is covered by any stored prefix.
// Its semantics are identical to [Table.Contains].
func (f *Fat[V]) Contains(ip netip.Addr) bool {
	if !ip.IsValid() {
		return false
	}

	is4 := ip.Is4()
	n := f.rootNodeByVersion(is4)

	for _, octet := range ip.AsSlice() {
		if n.contains(uint(octet) + 256) {
			return true
		}

		kidAny, exists := n.getChild(octet)
		if !exists {
			// no next node
			return false
		}

		// kid is node or leaf or fringe at octet
		switch kid := kidAny.(type) {
		case *fatNode[V]:
			n = kid // continue

		case *fringeNode[V]:
			// fringe is the default-route for all possible octets below
			return true

		case *leafNode[V]:
			// due to path compression, the octet path between
			// leaf and prefix may diverge
			return kid.prefix.Contains(ip)

		default:
			panic("logic error, wrong node type")
		}
	}

	return false
}

// Lookup returns the value of the longest prefix match for the given address.
// Its semantics are identical to [Table.Lookup].
func (f *Fat[V]) Lookup(ip netip.Addr) (val V, ok bool) {
	if !ip.IsValid() {
		return
	}

	is4 := ip.Is4()
	n := f.rootNodeByVersion(is4)

	for _, octet := range ip.AsSlice() {
		// save the current best LPM val, lookup is cheap in Fat
		if bestLPM, tmpOk := n.lookup(uint(octet) + 256); tmpOk {
			val = bestLPM
			ok = tmpOk
		}

		kidAny, exists := n.getChild(octet)
		if !exists {
			// no next node
			return val, ok
		}

		// next kid is fat, fringe or leaf node.
		switch kid := kidAny.(type) {
		case *fatNode[V]:
			n = kid

		case *fringeNode[V]:
			// fringe is the default-route for all possible nodes below
			return kid.value, true

		case *leafNode[V]:
			// due to path compression, the octet path between
			// leaf and prefix may diverge
			if kid.prefix.Contains(ip) {
				return kid.value, true
			}
			// maybe there is a current best value from upper levels
			return val, ok

		default:
			panic("logic error, wrong node type")
		}
	}

	panic("unreachable")
}

// Clone returns a copy of the routing table.
// Its semantics are identical to [Table.Clone].
func (f *Fat[V]) Clone() *Fat[V] {
	if f == nil {
		return nil
	}

	c := new(Fat[V])

	cloneFn := cloneFnFactory[V]()

	c.root4 = *f.root4.cloneRec(cloneFn)
	c.root6 = *f.root6.cloneRec(cloneFn)

	c.size4 = f.size4
	c.size6 = f.size6

	return c
}

func (f *Fat[V]) sizeUpdate(is4 bool, n int) {
	if is4 {
		f.size4 += n
		return
	}
	f.size6 += n
}

// Size returns the prefix count.
func (f *Fat[V]) Size() int {
	return f.size4 + f.size6
}

// Size4 returns the IPv4 prefix count.
func (f *Fat[V]) Size4() int {
	return f.size4
}

// Size6 returns the IPv6 prefix count.
func (f *Fat[V]) Size6() int {
	return f.size6
}
