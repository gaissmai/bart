/*
Copyright 2014 Will Fitzgerald. All rights reserved.
Use of this source code is governed by a BSD-style
license that can be found in the LICENSE file.
*/

// Package bitset implements bitsets, a mapping
// between non-negative integers and boolean values.
//
// This is a simplified and stripped down version of:
//
//	github.com/bits-and-blooms/bitset
//
// All bugs belong to me.
package bitset

import (
	"math/bits"
)

// the wordSize of a bit set
const wordSize = uint(64)

// log2WordSize is lg(wordSize)
const log2WordSize = uint(6)

// A BitSet is a set of bits.
type BitSet struct {
	set []uint64
}

// From is a constructor used to create a BitSet from a slice of words
func From(buf []uint64) BitSet {
	return BitSet{buf}
}

// Len returns the len in uint64 words.
func (b BitSet) Len() int {
	return len(b.set)
}

// Cap returns the cap in uint64 words.
func (b BitSet) Cap() int {
	return cap(b.set)
}

// extendSet adds additional words to incorporate new bits if needed.
func (b *BitSet) extendSet(i uint) {
	nsize := wordsNeeded(i)
	if b.set == nil {
		b.set = make([]uint64, nsize)
	} else if b.Cap() >= nsize {
		b.set = b.set[:nsize]
	} else if b.Len() < nsize {
		newset := make([]uint64, nsize)
		copy(newset, b.set)
		b.set = newset
	}
}

// bitsCapacity returns the number of possible bits in the current set.
func (b BitSet) bitsCapacity() uint {
	return uint(b.Len() * 64)
}

// wordsNeeded calculates the number of words needed for i bits.
func wordsNeeded(i uint) int {
	return int(i+wordSize) >> log2WordSize
}

// wordsIndex calculates the index of words in a `uint64`
func wordsIndex(i uint) uint {
	return i & (wordSize - 1)
}

// Words returns the bitset as a slice of uint64 words, giving direct access to the internal representation.
// It is meant for advanced users.
// It is not a copy, so changes to the returned slice will affect the bitset.
func (b BitSet) Words() []uint64 {
	return b.set
}

// Test whether bit i is set.
func (b BitSet) Test(i uint) bool {
	if i >= b.bitsCapacity() {
		return false
	}
	return b.set[i>>log2WordSize]&(1<<wordsIndex(i)) != 0
}

// Set bit i to 1, the capacity of the bitset is increased accordingly.
func (b *BitSet) Set(i uint) {
	if i >= b.bitsCapacity() {
		b.extendSet(i)
	}
	b.set[i>>log2WordSize] |= 1 << wordsIndex(i)
}

// Clear bit i to 0.
func (b *BitSet) Clear(i uint) {
	if i >= b.bitsCapacity() {
		return
	}
	b.set[i>>log2WordSize] &^= 1 << wordsIndex(i)
}

// Compact shrinks BitSet so that we preserve all set bits, while minimizing
// memory usage.
// A new slice is allocated to store the new bits, so you may see an increase in
// memory usage until the GC runs. Normally this should not be a problem, but if you
func (b *BitSet) Compact() {
	idx := b.Len() - 1

	// find last word with at least one bit set.
	for ; idx >= 0; idx-- {
		if b.set[idx] != 0 {
			newset := make([]uint64, idx+1)
			copy(newset, b.set[:idx+1])
			b.set = newset
			return
		}
	}

	// not found
	b.set = nil
}

// NextSet returns the next bit set from the specified index,
// including possibly the current index
// along with an error code (true = valid, false = no set bit found)
// for i,e := b.NextSet(0); e; i,e = b.NextSet(i + 1) {...}
func (b BitSet) NextSet(i uint) (uint, bool) {
	x := int(i >> log2WordSize)
	if x >= b.Len() {
		return 0, false
	}
	w := b.set[x]
	w = w >> wordsIndex(i)
	if w != 0 {
		return i + uint(bits.TrailingZeros64(w)), true
	}
	x++
	// bounds check elimination in the loop
	if x < 0 {
		return 0, false
	}
	for x < b.Len() {
		if b.set[x] != 0 {
			return uint(x)*wordSize + uint(bits.TrailingZeros64(b.set[x])), true
		}
		x++

	}
	return 0, false
}

// NextSetMany returns many next bit sets from the specified index,
// including possibly the current index and up to cap(buffer).
// If the returned slice has len zero, then no more set bits were found
//
// It is possible to retrieve all set bits as follow:
//
//	indices := make([]uint, b.Count())
//	b.NextSetMany(0, indices)
func (b BitSet) NextSetMany(i uint, buffer []uint) (uint, []uint) {
	myanswer := buffer
	capacity := cap(buffer)
	x := int(i >> log2WordSize)
	if x >= b.Len() || capacity == 0 {
		return 0, myanswer[:0]
	}
	skip := wordsIndex(i)
	word := b.set[x] >> skip
	myanswer = myanswer[:capacity]
	size := int(0)
	for word != 0 {
		r := uint(bits.TrailingZeros64(word))
		t := word & ((^word) + 1)
		myanswer[size] = r + i
		size++
		if size == capacity {
			goto End
		}
		word = word ^ t
	}
	x++
	for idx, word := range b.set[x:] {
		for word != 0 {
			r := uint(bits.TrailingZeros64(word))
			t := word & ((^word) + 1)
			myanswer[size] = r + (uint(x+idx) << 6)
			size++
			if size == capacity {
				goto End
			}
			word = word ^ t
		}
	}
End:
	if size > 0 {
		return myanswer[size-1], myanswer[:size]
	}
	return 0, myanswer[:0]
}

// Clone this BitSet, returning a new BitSet that has the same bits set.
func (b BitSet) Clone() BitSet {
	if b.set == nil {
		return BitSet{}
	}

	c := BitSet{}
	c.set = make([]uint64, b.Len())
	copy(c.set, b.set)
	return c
}

// IntersectionCardinality computes the cardinality of the intersection
func (b BitSet) IntersectionCardinality(c BitSet) uint {
	if b.Len() <= c.Len() {
		return uint(popcntAndSlice(b.set, c.set))
	}
	return uint(popcntAndSlice(c.set, b.set))
}

// InPlaceIntersection overwrites and computes the intersection of
// base set with the compare set.
// This is the BitSet equivalent of & (and)
func (b *BitSet) InPlaceIntersection(c BitSet) {
	bLen := b.Len()
	cLen := c.Len()

	// intersect b with shorter or equal c
	if bLen >= cLen {
		// bounds check elimination
		_ = b.set[cLen-1]
		_ = c.set[cLen-1]

		for i := range cLen {
			b.set[i] &= c.set[i]
		}
		for i := cLen; i < bLen; i++ {
			b.set[i] = 0
		}
		return
	}

	// intersect b with longer c
	// bounds check elimination
	_ = b.set[bLen-1]
	_ = c.set[bLen-1]

	for i := range bLen {
		b.set[i] &= c.set[i]
	}

	newset := make([]uint64, cLen)
	copy(newset, b.set)
	b.set = newset
}

// InPlaceUnion creates the destructive union of base set with compare set.
// This is the BitSet equivalent of | (or).
func (b *BitSet) InPlaceUnion(c BitSet) {
	bLen := b.Len()
	cLen := c.Len()

	// union b with shorter or equal c
	if bLen >= cLen {
		// bounds check elimination
		_ = b.set[cLen-1]
		_ = c.set[cLen-1]

		for i := range cLen {
			b.set[i] |= c.set[i]
		}
		return
	}

	// union b with longer c
	newset := make([]uint64, cLen)
	copy(newset, b.set)
	b.set = newset
	// bounds check elimination
	_ = b.set[cLen-1]
	_ = c.set[cLen-1]

	for i := range cLen {
		b.set[i] |= c.set[i]
	}
}

// Count (number of set bits).
// Also known as "popcount" or "population count".
func (b BitSet) Count() int {
	return popcntSlice(b.set)
}

// Rank returns the number of set bits up to and including the index
// that are set in the bitset.
func (b BitSet) Rank(index uint) int {
	if index >= b.bitsCapacity() {
		return popcntSlice(b.set)
	}
	leftover := (index + 1) & 63
	answer := popcntSlice(b.set[:(index+1)>>6])
	if leftover != 0 {
		answer += bits.OnesCount64(b.set[(index+1)>>6] << (64 - leftover))
	}
	return answer
}

func popcntSlice(s []uint64) int {
	var cnt int
	for _, x := range s {
		cnt += bits.OnesCount64(x)
	}
	return cnt
}

func popcntAndSlice(s, m []uint64) int {
	var cnt int
	for i := range s {
		// panics if mask slice m is too short
		cnt += bits.OnesCount64(s[i] & m[i])
	}
	return cnt
}
