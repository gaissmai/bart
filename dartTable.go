package bart

import (
	"net/netip"
	"sync"

	"github.com/gaissmai/bart/internal/art"
	"github.com/gaissmai/bart/internal/lpm"
)

type Dart[V any] struct {
	// used by -copylocks checker from `go vet`.
	_ [0]sync.Mutex

	// the root nodes
	root4 artNode[V]
	root6 artNode[V]

	// the number of prefixes in the routing table
	size4 int
	size6 int
}

// rootNodeByVersion, root node getter for ip version.
func (f *Dart[V]) rootNodeByVersion(is4 bool) *artNode[V] {
	if is4 {
		return &f.root4
	}

	return &f.root6
}

func (f *Dart[V]) Insert(pfx netip.Prefix, val V) {
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

	n0 := f.rootNodeByVersion(is4)

	// ###
	// insert prefix into ART

	// level 0
	if bits <= 8 {
		exists := n0.insertPrefix(octets[0], bits, val)
		if !exists {
			f.sizeUpdate(is4, 1)
		}
		return
	}

	// level 1
	if bits <= 16 {
		n1, ok := n0.getChild(octets[0]).(*artNode[V])
		if !ok {
			n1 = &artNode[V]{}
		}
		n0.setChild(octets[0], n1)

		exists := n1.insertPrefix(octets[1], bits-8, val)
		if !exists {
			f.sizeUpdate(is4, 1)
		}
		return
	}

	// ###
	// bits > 16, insert prefix into BART

	// level 0, get or create child, an artNode
	n1, ok := n0.getChild(octets[0]).(*artNode[V])
	if !ok {
		n1 = &artNode[V]{}
	}
	n0.children[octets[0]] = n1

	// level 1, get or create child, now a bartNode
	n2, ok := n1.getChild(octets[1]).(*bartNode[V])
	if !ok {
		n2 = &bartNode[V]{}
	}
	n1.children[octets[1]] = n2

	// level 2, insert prefix into bartNode
	if exists := n2.insertAtDepth(pfx, val, 2); exists {
		return
	}

	// true insert, update size
	f.sizeUpdate(is4, 1)
}

func (f *Dart[V]) Delete(pfx netip.Prefix) {
	panic("Delete() TODO")
}

func (f *Dart[V]) Contains(ip netip.Addr) bool {
	// if ip is invalid, Is4() returns false and AsSlice() returns nil
	is4 := ip.Is4()
	octets := ip.AsSlice()

	n0 := f.rootNodeByVersion(is4)
	if n0.contains(octets[0]) {
		return true
	}

	n1, ok := n0.getChild(octets[0]).(*artNode[V])
	if !ok {
		return false
	}
	if n1.contains(octets[1]) {
		return true
	}

	n, ok := n1.getChild(octets[1]).(*bartNode[V])
	if !ok {
		return false
	}

	for _, octet := range octets[2:] {
		// for contains, any lpm match is good enough, no backtracking needed
		if n.prefixes.Len() != 0 && n.lpmTest(art.OctetToIdx(octet)) {
			return true
		}

		// stop traversing?
		if !n.children.Test(octet) {
			return false
		}
		kid := n.children.MustGet(octet)

		// kid is node or leaf or fringe at octet
		switch kid := kid.(type) {
		case *bartNode[V]:
			n = kid
			continue // descend down to next trie level

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

func (f *Dart[V]) Lookup(ip netip.Addr) (val V, ok bool) {
	// if ip is invalid, Is4() returns false and AsSlice() returns nil
	is4 := ip.Is4()
	octets := ip.AsSlice()

	// run variables, used after for loop
	var depth int
	var octet byte

	// stack of the traversed nodes for fast backtracking, if needed
	stack := [maxTreeDepth]*bartNode[V]{}

	n0 := f.rootNodeByVersion(is4)

	n1, exists := n0.getChild(octets[0]).(*artNode[V])
	if !exists {
		return n0.lookup(octets[0])
	}

	n, exists := n1.getChild(octets[1]).(*bartNode[V])
	if !exists {
		goto LookupInART
	}

LookupInBART:
	for depth, octet = range octets {
		if depth < 2 {
			continue
		}

		depth = depth & 0xf // BCE, Lookup must be fast
		octet = octets[depth]

		// push current node on stack for fast backtracking
		stack[depth] = n

		// go down in tight loop to last octet
		if !n.children.Test(octet) {
			// no more nodes below octet
			break LookupInBART
		}
		kid := n.children.MustGet(octet)

		// kid is node or leaf or fringe at octet
		switch kid := kid.(type) {
		case *bartNode[V]:
			n = kid
			continue // descend down to next trie level

		case *fringeNode[V]:
			// fringe is the default-route for all possible nodes below
			return kid.value, true

		case *leafNode[V]:
			if kid.prefix.Contains(ip) {
				return kid.value, true
			}
			// reached a path compressed prefix, stop traversing
			break LookupInBART

		default:
			panic("logic error, wrong node type")
		}
	}

	// start backtracking, unwind the stack, bounds check eliminated
	for ; depth >= 2; depth-- {
		depth = depth & 0xf // BCE

		n = stack[depth]

		// longest prefix match, skip if node has no prefixes
		if n.prefixes.Len() != 0 {
			idx := art.OctetToIdx(octets[depth])
			// lpmGet(idx), manually inlined
			// --------------------------------------------------------------
			if topIdx, ok := n.prefixes.IntersectionTop(lpm.BackTrackingBitset(idx)); ok {
				return n.prefixes.MustGet(topIdx), true
			}
			// --------------------------------------------------------------
		}
	}

LookupInART:
	if val, ok = n1.lookup(octets[1]); ok {
		return val, true
	}

	val, ok = n0.lookup(octets[0])
	return
}

func (f *Dart[V]) sizeUpdate(is4 bool, n int) {
	if is4 {
		f.size4 += n
		return
	}
	f.size6 += n
}

// Size returns the prefix count.
func (f *Dart[V]) Size() int {
	return f.size4 + f.size6
}

// Size4 returns the IPv4 prefix count.
func (f *Dart[V]) Size4() int {
	return f.size4
}

// Size6 returns the IPv6 prefix count.
func (f *Dart[V]) Size6() int {
	return f.size6
}
