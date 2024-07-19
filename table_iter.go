//go:build go1.23

// rangefunc iterators for 1.23 and above

// to edit it with vim-go and gotip do ...
//
//  $ export GOROOT=$(gotip env GOROOT)
//  $ export PATH=${GOROOT}/bin:${PATH}
//  $ vim filename.go

package bart

import (
	"iter"
	"net/netip"
)

// LookupPrefixIter returns an iterator for each CIDR covering pfx.
// The iteration is in reverse CIDR sort order, from longest-prefix-match to shortest-prefix-match.
func (t *Table[V]) LookupPrefixIter(pfx netip.Prefix) iter.Seq2[netip.Prefix, V] {
	return func(yield func(netip.Prefix, V) bool) {

		// iterator setup
		if !pfx.IsValid() {
			return
		}

		// values derived from pfx
		ip := pfx.Addr()
		is4 := ip.Is4()
		bits := pfx.Bits()

		n := t.rootNodeByVersion(is4)
		if n == nil {
			return
		}

		// do not allocate
		path := ip.As16()
		octets := path[:]
		if is4 {
			octets = octets[12:]
		}
		copy(path[:], octets[:])

		// see comment in Insert()
		lastOctetIdx := (bits - 1) / strideLen
		lastOctet := octets[lastOctetIdx]
		lastOctetBits := bits - (lastOctetIdx * strideLen)

		// mask the prefix
		lastOctet = lastOctet & netMask(lastOctetBits)
		octets[lastOctetIdx] = lastOctet

		// stack of the traversed nodes for reverse ordering of supernets
		stack := [maxTreeDepth]*node[V]{}

		// run variable, used after for loop
		var i int
		var octet byte

		// find last node
		for i, octet = range octets[:lastOctetIdx+1] {
			// push current node on stack
			stack[i] = n

			// go down in tight loop
			c := n.getChild(octet)
			if c == nil {
				break
			}
			n = c
		}

		// start backtracking, unwind the stack
		for depth := i; depth >= 0; depth-- {
			n = stack[depth]

			// microbenchmarking
			if len(n.prefixes) == 0 {
				continue
			}

			// only the lastOctet may have a different prefix len
			if depth == lastOctetIdx {
				if !n.eachLookupPrefix(path, depth, is4, lastOctet, lastOctetBits, yield) {
					// early exit
					return
				}
				continue
			}

			// all others are just host routes
			if !n.eachLookupPrefix(path, depth, is4, octets[depth], strideLen, yield) {
				// early exit
				return
			}
		}
	}

}

// SubnetIter returns an iterator over all CIDRs covered by pfx.
// The iteration is in natural CIDR sort order.
func (t *Table[V]) SubnetIter(pfx netip.Prefix) iter.Seq2[netip.Prefix, V] {
	return func(yield func(netip.Prefix, V) bool) {

		// iterator setup
		if !pfx.IsValid() {
			return
		}

		// values derived from pfx
		ip := pfx.Addr()
		is4 := ip.Is4()
		bits := pfx.Bits()

		n := t.rootNodeByVersion(is4)
		if n == nil {
			return
		}

		// do not allocate
		path := ip.As16()
		octets := path[:]
		if is4 {
			octets = octets[12:]
		}
		copy(path[:], octets[:])

		// see comment in Insert()
		lastOctetIdx := (bits - 1) / strideLen
		lastOctet := octets[lastOctetIdx]
		lastOctetBits := bits - (lastOctetIdx * strideLen)

		// mask the prefix
		lastOctet = lastOctet & netMask(lastOctetBits)
		octets[lastOctetIdx] = lastOctet

		// find the trie node
		for i, octet := range octets {
			if i == lastOctetIdx {
				_ = n.eachSubnet(path, i, is4, lastOctet, lastOctetBits, yield)
				return
			}

			c := n.getChild(octet)
			if c == nil {
				break
			}

			n = c
		}
	}

}

// AllIter returns an iterator over all prefxes.
// The iteration order is undefined and you must not rely on it!
//
// Prefixes must not be inserted or deleted during iteration, otherwise
// the behavior is undefined. However, value updates are permitted.
func (t *Table[V]) AllIter() iter.Seq2[netip.Prefix, V] {
	t.init()

	return func(yield func(netip.Prefix, V) bool) {
		// respect early exit
		_ = t.rootV4.allRec(zeroPath, 0, true, yield) &&
			t.rootV6.allRec(zeroPath, 0, false, yield)
	}
}

// All4Iter, like [Table.AllIter] but only for the v4 routing table.
func (t *Table[V]) All4Iter() iter.Seq2[netip.Prefix, V] {
	t.init()

	return func(yield func(netip.Prefix, V) bool) {
		t.rootV4.allRec(zeroPath, 0, true, yield)
	}
}

// All6Iter, like [Table.AllIter] but only for the v6 routing table.
func (t *Table[V]) All6Iter() iter.Seq2[netip.Prefix, V] {
	t.init()

	return func(yield func(netip.Prefix, V) bool) {
		t.rootV6.allRec(zeroPath, 0, false, yield)
	}
}

// AllSortedIter returns an iterator over all prefxes.
// The iteration is in natural CIDR sort order.
//
// Prefixes must not be inserted or deleted during iteration, otherwise
// the behavior is undefined. However, value updates are permitted.
func (t *Table[V]) AllSortedIter() iter.Seq2[netip.Prefix, V] {
	t.init()

	return func(yield func(netip.Prefix, V) bool) {
		// respect early exit
		_ = t.rootV4.allRecSorted(zeroPath, 0, true, yield) &&
			t.rootV6.allRecSorted(zeroPath, 0, false, yield)
	}
}

// All4SortedIter, like [Table.AllSortedIter] but only for the v4 routing table.
func (t *Table[V]) All4SortedIter() iter.Seq2[netip.Prefix, V] {
	t.init()

	return func(yield func(netip.Prefix, V) bool) {
		t.rootV4.allRecSorted(zeroPath, 0, true, yield)
	}
}

// All6SortedIter, like [Table.AllSortedIter] but only for the v6 routing table.
func (t *Table[V]) All6SortedIter() iter.Seq2[netip.Prefix, V] {
	t.init()

	return func(yield func(netip.Prefix, V) bool) {
		t.rootV6.allRecSorted(zeroPath, 0, false, yield)
	}
}
