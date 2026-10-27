// Copyright (c) 2026 Karl Gaissmaier
// SPDX-License-Identifier: MIT

package art

import (
	"math/rand/v2"
	"sync"
	"testing"
)

// #####################################################
//  helpers
// #####################################################

// refIdx is the definition of the base index: walk down the complete binary
// tree from the root (idx 1) and go to the left child (2*idx) for a 0 bit or
// to the right child (2*idx+1) for a 1 bit of the prefix.
//
// It is independent of the formula in PfxToIdx and serves as the oracle.
func refIdx(octet uint8, pfxLen int) int {
	idx := 1
	for i := range pfxLen {
		idx = idx*2 + int(octet>>(7-i))&1
	}
	return idx
}

// refMask is the left-aligned network mask with pfxLen bits set.
func refMask(pfxLen int) uint8 {
	mask := 0xff << (8 - pfxLen) // int, may exceed 8 bits
	return uint8(mask)           // keep the low 8 bits
}

// assertPanics fails the test if fn does not panic.
func assertPanics(t *testing.T, name string, fn func()) {
	t.Helper()

	defer func() {
		if recover() == nil {
			t.Errorf("%s must panic", name)
		}
	}()
	fn()
}

// #####################################################
//  PfxToIdx, IdxToPfx, OctetToIdx
// #####################################################

func TestPfxToIdx(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		octet  uint8
		pfxLen uint8
		want   uint8
	}{
		{"default route", 0, 0, 1},
		{"default route ignores all host bits", 255, 0, 1},
		{"first /1", 0, 1, 2},
		{"second /1", 128, 1, 3},
		{"160/3 from the documentation", 0b1010_0000, 3, 13},
		{"host bits are ignored", 0b1011_1111, 3, 13},
		{"first /7", 0, 7, 128},
		{"last /7", 254, 7, 255},
		{"last /7 with host bit set", 255, 7, 255},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			if got := PfxToIdx(tt.octet, tt.pfxLen); got != tt.want {
				t.Errorf("PfxToIdx(%d, %d) = %d, want %d", tt.octet, tt.pfxLen, got, tt.want)
			}
		})
	}
}

// TestPfxToIdxMatchesDefinition checks all 8*256 valid inputs against the
// definition of the base index as walk in the complete binary tree.
func TestPfxToIdxMatchesDefinition(t *testing.T) {
	t.Parallel()

	for pfxLen := range 8 {
		for octet := range 256 {
			got := int(PfxToIdx(uint8(octet), uint8(pfxLen)))

			if want := refIdx(uint8(octet), pfxLen); got != want {
				t.Fatalf("PfxToIdx(%d, %d) = %d, want %d", octet, pfxLen, got, want)
			}
		}
	}
}

func TestIdxToPfx(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		idx        uint8
		wantOctet  uint8
		wantPfxLen uint8
	}{
		{"root, the default route", 1, 0, 0},
		{"first /1", 2, 0, 1},
		{"second /1", 3, 128, 1},
		{"13 from the documentation", 13, 160, 3},
		{"first /7", 128, 0, 7},
		{"last /7", 255, 254, 7},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			octet, pfxLen := IdxToPfx(tt.idx)
			if octet != tt.wantOctet || pfxLen != tt.wantPfxLen {
				t.Errorf("IdxToPfx(%d) = (%d, %d), want (%d, %d)",
					tt.idx, octet, pfxLen, tt.wantOctet, tt.wantPfxLen)
			}
		})
	}
}

// TestIdxToPfxRoundTrip verifies that PfxToIdx and IdxToPfx are inverse
// functions, in both directions and for all valid inputs.
func TestIdxToPfxRoundTrip(t *testing.T) {
	t.Parallel()

	// idx -> (octet, pfxLen) -> idx
	for i := 1; i < 256; i++ {
		idx := uint8(i)

		octet, pfxLen := IdxToPfx(idx)

		if octet != octet&refMask(int(pfxLen)) {
			t.Fatalf("IdxToPfx(%d): octet %d has host bits set for pfxLen %d", idx, octet, pfxLen)
		}

		if got := PfxToIdx(octet, pfxLen); got != idx {
			t.Fatalf("PfxToIdx(IdxToPfx(%d)) = %d, want %d", idx, got, idx)
		}
	}

	// (octet, pfxLen) -> idx -> (octet, pfxLen), the host bits get lost
	for pfxLen := range 8 {
		for octet := range 256 {
			idx := PfxToIdx(uint8(octet), uint8(pfxLen))

			gotOctet, gotPfxLen := IdxToPfx(idx)

			wantOctet := uint8(octet) & refMask(pfxLen)
			if gotOctet != wantOctet || gotPfxLen != uint8(pfxLen) {
				t.Fatalf("IdxToPfx(PfxToIdx(%d, %d)) = (%d, %d), want (%d, %d)",
					octet, pfxLen, gotOctet, gotPfxLen, wantOctet, pfxLen)
			}
		}
	}
}

func TestOctetToIdx(t *testing.T) {
	t.Parallel()

	for octet := range 256 {
		o := uint8(octet)
		got := OctetToIdx(o)

		// the parent of the /8 prefix with the index 256+octet
		if want := uint8((256 + octet) >> 1); got != want {
			t.Fatalf("OctetToIdx(%d) = %d, want %d (parent of 256+octet)", o, got, want)
		}

		// the same as the /7 prefix of the octet
		if want := PfxToIdx(o, 7); got != want {
			t.Fatalf("OctetToIdx(%d) = %d, want PfxToIdx(%d, 7) = %d", o, got, o, want)
		}

		if want := refIdx(o, 7); int(got) != want {
			t.Fatalf("OctetToIdx(%d) = %d, want %d (definition)", o, got, want)
		}
	}
}

// TestTreeStructure verifies the structure of the complete binary tree:
// the parent of idx is idx>>1 with one prefix bit less, the children of idx
// are 2*idx and 2*idx+1 with one prefix bit more.
func TestTreeStructure(t *testing.T) {
	t.Parallel()

	// parent
	for i := 2; i < 256; i++ {
		idx := uint8(i)
		parent := idx >> 1

		octet, pfxLen := IdxToPfx(idx)
		parentOctet, parentPfxLen := IdxToPfx(parent)

		if parentPfxLen != pfxLen-1 {
			t.Fatalf("parent of %d (%d): pfxLen = %d, want %d", idx, parent, parentPfxLen, pfxLen-1)
		}

		if want := octet & refMask(int(parentPfxLen)); parentOctet != want {
			t.Fatalf("parent of %d (%d): octet = %d, want %d", idx, parent, parentOctet, want)
		}

		// an ancestor always has a smaller index than its descendants,
		// lpm.LookupTbl relies on this to find the longest match
		if parent >= idx {
			t.Fatalf("parent %d is not smaller than idx %d", parent, idx)
		}
	}

	// children, only the indices 1..127 have children in 1..255
	for i := 1; i < 128; i++ {
		idx := uint8(i)

		octet, pfxLen := IdxToPfx(idx)

		leftOctet, leftPfxLen := IdxToPfx(2 * idx)
		rightOctet, rightPfxLen := IdxToPfx(2*idx + 1)

		if leftPfxLen != pfxLen+1 || rightPfxLen != pfxLen+1 {
			t.Fatalf("children of %d: pfxLen = (%d, %d), want %d", idx, leftPfxLen, rightPfxLen, pfxLen+1)
		}

		// the new prefix bit is 0 for the left and 1 for the right child
		if leftOctet != octet {
			t.Fatalf("left child of %d: octet = %d, want %d", idx, leftOctet, octet)
		}

		if want := octet | 0x80>>pfxLen; rightOctet != want {
			t.Fatalf("right child of %d: octet = %d, want %d", idx, rightOctet, want)
		}
	}
}

func TestIdxToPfxPanics(t *testing.T) {
	t.Parallel()

	assertPanics(t, "IdxToPfx(0)", func() { IdxToPfx(0) })
}

// #####################################################
//  PfxBits
// #####################################################

func TestPfxBits(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		depth int
		idx   uint8
		want  uint8
	}{
		{"default route in the root node", 0, 1, 0},
		{"longest prefix in the root node", 0, 255, 7},
		{"13 at depth 2 from the documentation", 2, 13, 19},
		{"default route of the stride at depth 3", 3, 1, 24},
		{"longest IPv4 prefix in a node", 3, 255, 31},
		{"first /7 at depth 1", 1, 128, 15},
		{"longest IPv6 prefix in a node", 15, 255, 127},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			if got := PfxBits(tt.depth, tt.idx); got != tt.want {
				t.Errorf("PfxBits(%d, %d) = %d, want %d", tt.depth, tt.idx, got, tt.want)
			}
		})
	}
}

// TestPfxBitsAllInputs checks PfxBits against the sum of the complete strides
// above and the prefix length within the stride, for all depths of IPv6.
func TestPfxBitsAllInputs(t *testing.T) {
	t.Parallel()

	for depth := range 16 {
		for i := 1; i < 256; i++ {
			idx := uint8(i)

			_, pfxLen := IdxToPfx(idx)
			want := depth*8 + int(pfxLen)

			if got := int(PfxBits(depth, idx)); got != want {
				t.Fatalf("PfxBits(%d, %d) = %d, want %d", depth, idx, got, want)
			}

			// the host routes /32 and /128 are never stored as prefix in a node
			if want > 127 {
				t.Fatalf("PfxBits(%d, %d) = %d, more than 127", depth, idx, want)
			}
		}
	}
}

// #####################################################
//  IdxToRange
// #####################################################

func TestIdxToRange(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		idx       uint8
		wantFirst uint8
		wantLast  uint8
	}{
		{"default route covers all octets", 1, 0, 255},
		{"first /1", 2, 0, 127},
		{"second /1", 3, 128, 255},
		{"13 from the documentation", 13, 160, 191},
		{"first /7", 128, 0, 1},
		{"second /7", 129, 2, 3},
		{"last /7", 255, 254, 255},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			first, last := IdxToRange(tt.idx)
			if first != tt.wantFirst || last != tt.wantLast {
				t.Errorf("IdxToRange(%d) = (%d, %d), want (%d, %d)",
					tt.idx, first, last, tt.wantFirst, tt.wantLast)
			}
		})
	}
}

// TestIdxToRangeProperties checks the properties of the octet ranges for all
// valid indices: the size and alignment of the range, the range of the
// children as partition of the range of the parent, and that every octet
// in the range maps back to the index.
func TestIdxToRangeProperties(t *testing.T) {
	t.Parallel()

	for i := 1; i < 256; i++ {
		idx := uint8(i)

		octet, pfxLen := IdxToPfx(idx)
		first, last := IdxToRange(idx)

		if first != octet {
			t.Fatalf("IdxToRange(%d): first = %d, want the octet %d of the prefix", idx, first, octet)
		}

		size := int(last) - int(first) + 1
		if want := 1 << (8 - pfxLen); size != want {
			t.Fatalf("IdxToRange(%d): size = %d, want %d", idx, size, want)
		}

		// the range is aligned to its size
		if int(first)%size != 0 {
			t.Fatalf("IdxToRange(%d): first = %d is not aligned to the size %d", idx, first, size)
		}

		// every octet in the range belongs to the prefix
		for o := int(first); o <= int(last); o++ {
			if got := PfxToIdx(uint8(o), pfxLen); got != idx {
				t.Fatalf("PfxToIdx(%d, %d) = %d, want %d, the octet is in the range of %d",
					o, pfxLen, got, idx, idx)
			}
		}
	}

	// the ranges of the two children partition the range of the parent
	for i := 1; i < 128; i++ {
		idx := uint8(i)

		first, last := IdxToRange(idx)
		leftFirst, leftLast := IdxToRange(2 * idx)
		rightFirst, rightLast := IdxToRange(2*idx + 1)

		if leftFirst != first || rightLast != last || int(leftLast)+1 != int(rightFirst) {
			t.Fatalf("children of %d: (%d..%d) and (%d..%d) do not partition (%d..%d)",
				idx, leftFirst, leftLast, rightFirst, rightLast, first, last)
		}
	}
}

func TestIdxToRangePanics(t *testing.T) {
	t.Parallel()

	// the invalid index 0 panics in IdxToPfx, which IdxToRange calls
	assertPanics(t, "IdxToRange(0)", func() { IdxToRange(0) })
}

// #####################################################
//  Benchmarks for the base index functions
// #####################################################
//
//  Design notes:
//
//  - The functions are tiny and inlined, a single call takes well below a
//    nanosecond. The loop overhead (index calculation, load of the probe,
//    store of the result) is a large part of the measured time.
//    BenchmarkLoopOverhead measures the loop alone, use it as the baseline
//    and compare the variants with each other not the absolute values.
//  - The probes are random, so the compiler can neither fold the arguments
//    nor the results.
// #####################################################

const (
	// probeCount probes are generated per kind, a power of two for cheap masking.
	probeCount = 1 << 16
	probeMask  = probeCount - 1
)

// probeSet holds the random inputs, only valid arguments are generated.
type probeSet struct {
	octets  []uint8 // 0..255
	pfxLens []uint8 // 0..7
	idxs    []uint8 // 1..255
	depths  []int   // 0..15, the depths of an IPv6 trie
}

// probes generates the inputs once, on first use.
var probes = sync.OnceValue(func() *probeSet {
	prng := rand.New(rand.NewPCG(1, 2))

	p := &probeSet{
		octets:  make([]uint8, probeCount),
		pfxLens: make([]uint8, probeCount),
		idxs:    make([]uint8, probeCount),
		depths:  make([]int, probeCount),
	}

	for i := range probeCount {
		p.octets[i] = uint8(prng.UintN(256))
		p.pfxLens[i] = uint8(prng.UintN(8))
		p.idxs[i] = uint8(1 + prng.UintN(255))
		p.depths[i] = prng.IntN(16)
	}

	return p
})

// sinks keep the results alive, the compiler must not drop the calls.
var (
	sinkA uint8
	sinkB uint8
)

// BenchmarkLoopOverhead measures the benchmark loop alone: load of a probe and
// store into the sink. It is the baseline for all benchmarks below.
func BenchmarkLoopOverhead(b *testing.B) {
	p := probes()

	for i := 0; b.Loop(); i++ {
		sinkA = p.idxs[i&probeMask]
	}
}

func BenchmarkPfxToIdx(b *testing.B) {
	p := probes()

	for i := 0; b.Loop(); i++ {
		sinkA = PfxToIdx(p.octets[i&probeMask], p.pfxLens[i&probeMask])
	}
}

func BenchmarkOctetToIdx(b *testing.B) {
	p := probes()

	for i := 0; b.Loop(); i++ {
		sinkA = OctetToIdx(p.octets[i&probeMask])
	}
}

func BenchmarkIdxToPfx(b *testing.B) {
	p := probes()

	for i := 0; b.Loop(); i++ {
		sinkA, sinkB = IdxToPfx(p.idxs[i&probeMask])
	}
}

func BenchmarkPfxBits(b *testing.B) {
	p := probes()

	for i := 0; b.Loop(); i++ {
		sinkA = PfxBits(p.depths[i&probeMask], p.idxs[i&probeMask])
	}
}

func BenchmarkIdxToRange(b *testing.B) {
	p := probes()

	for i := 0; b.Loop(); i++ {
		sinkA, sinkB = IdxToRange(p.idxs[i&probeMask])
	}
}
