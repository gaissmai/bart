// Copyright (c) 2026 Karl Gaissmaier
// SPDX-License-Identifier: MIT

package bitset

import (
	"math/rand/v2"
	"slices"
	"testing"
)

var prng = rand.New(rand.NewPCG(42, 42))

func randomBitSet256() BitSet256 {
	return BitSet256{
		prng.Uint64(),
		prng.Uint64(),
		prng.Uint64(),
		prng.Uint64(),
	}
}

var (
	sinkSliceUint8 []uint8
	sinkBitSet256  BitSet256
)

func TestZeroValue(t *testing.T) {
	t.Parallel()
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("A zero value bitset must not panic: %v", r)
		}
	}()

	var b BitSet256

	b = BitSet256{}
	b.Set(0)

	b = BitSet256{}
	b.Clear(100)

	b = BitSet256{}
	b.OnesCount()

	b = BitSet256{}
	b.Rank(100)

	b = BitSet256{}
	b.Test(42)

	b = BitSet256{}
	b.NextSet(0)

	b = BitSet256{}
	b.Bits()

	b = BitSet256{}
	c := BitSet256{}
	b.Or(&c)

	b = BitSet256{}
	c = BitSet256{}
	b = b.And(&c)

	b = BitSet256{}
	c = BitSet256{}
	b.Overlaps(&c)

	b = BitSet256{}
	c = BitSet256{}
	b.AndTop(&c)
}

func TestSetClearTest(t *testing.T) {
	t.Parallel()
	var b BitSet256
	for i := range 256 {
		bit := uint8(i)
		if b.Test(bit) {
			t.Errorf("expected bit %d to be clear initially", bit)
		}
		b.Set(bit)
		if !b.Test(bit) {
			t.Errorf("expected bit %d to be set after Set", bit)
		}
		b.Clear(bit)
		if b.Test(bit) {
			t.Errorf("expected bit %d to be clear after Clear", bit)
		}
	}
}

func TestFirstSet(t *testing.T) {
	t.Parallel()
	testCases := []struct {
		name    string
		set     []uint8
		wantIdx uint8
		wantOk  bool
	}{
		{
			name:    "null",
			set:     []uint8{},
			wantIdx: 0,
			wantOk:  false,
		},
		{
			name:    "zero",
			set:     []uint8{0},
			wantIdx: 0,
			wantOk:  true,
		},
		{
			name:    "1,5",
			set:     []uint8{1, 5},
			wantIdx: 1,
			wantOk:  true,
		},
		{
			name:    "5,7",
			set:     []uint8{5, 7},
			wantIdx: 5,
			wantOk:  true,
		},
		{
			name:    "2. word",
			set:     []uint8{70, 255},
			wantIdx: 70,
			wantOk:  true,
		},
		{
			name:    "3. word",
			set:     []uint8{150, 255},
			wantIdx: 150,
			wantOk:  true,
		},
		{
			name:    "4. word",
			set:     []uint8{233, 255},
			wantIdx: 233,
			wantOk:  true,
		},
	}

	for _, tc := range testCases {
		var b BitSet256
		for _, u := range tc.set {
			b.Set(u)
		}

		idx, ok := b.FirstSet()

		if ok != tc.wantOk {
			t.Errorf("FirstSet, %s: got ok: %v, want: %v", tc.name, ok, tc.wantOk)
		}

		if idx != tc.wantIdx {
			t.Errorf("FirstSet, %s: got idx: %d, want: %d", tc.name, idx, tc.wantIdx)
		}
	}
}

func TestLastSet(t *testing.T) {
	t.Parallel()
	testCases := []struct {
		name    string
		set     []uint8
		wantIdx uint8
		wantOk  bool
	}{
		{
			name:    "null",
			set:     []uint8{},
			wantIdx: 0,
			wantOk:  false,
		},
		{
			name:    "zero",
			set:     []uint8{0},
			wantIdx: 0,
			wantOk:  true,
		},
		{
			name:    "1,5",
			set:     []uint8{1, 5},
			wantIdx: 5,
			wantOk:  true,
		},
		{
			name:    "5,7",
			set:     []uint8{5, 7},
			wantIdx: 7,
			wantOk:  true,
		},
		{
			name:    "2. word",
			set:     []uint8{70, 126},
			wantIdx: 126,
			wantOk:  true,
		},
		{
			name:    "3. word",
			set:     []uint8{1, 34, 150},
			wantIdx: 150,
			wantOk:  true,
		},
		{
			name:    "4. word",
			set:     []uint8{1, 70, 150, 233},
			wantIdx: 233,
			wantOk:  true,
		},
		{
			name:    "very last",
			set:     []uint8{1, 70, 150, 233, 255},
			wantIdx: 255,
			wantOk:  true,
		},
	}

	for _, tc := range testCases {
		var b BitSet256
		for _, u := range tc.set {
			b.Set(u)
		}

		idx, ok := b.LastSet()

		if ok != tc.wantOk {
			t.Errorf("LastSet, %s: got ok: %v, want: %v", tc.name, ok, tc.wantOk)
		}

		if idx != tc.wantIdx {
			t.Errorf("LastSet, %s: got idx: %d, want: %d", tc.name, idx, tc.wantIdx)
		}
	}
}

func TestNextSet(t *testing.T) {
	t.Parallel()
	testCases := []struct {
		name string
		//
		set   []uint8
		del   []uint8
		start uint8
		//
		wantIdx uint8
		wantOk  bool
	}{
		{
			name:    "null",
			set:     []uint8{},
			del:     []uint8{},
			start:   0,
			wantIdx: 0,
			wantOk:  false,
		},
		{
			name:    "zero",
			set:     []uint8{0},
			del:     []uint8{},
			start:   0,
			wantIdx: 0,
			wantOk:  true,
		},
		{
			name:    "1,5",
			set:     []uint8{1, 5},
			del:     []uint8{},
			start:   0,
			wantIdx: 1,
			wantOk:  true,
		},
		{
			name:    "1,5",
			set:     []uint8{1, 5},
			del:     []uint8{},
			start:   2,
			wantIdx: 5,
			wantOk:  true,
		},
		{
			name:    "1,5",
			set:     []uint8{1, 5},
			del:     []uint8{},
			start:   6,
			wantIdx: 0,
			wantOk:  false,
		},
		{
			name:    "1,5,7",
			set:     []uint8{1, 5, 7},
			del:     []uint8{5},
			start:   2,
			wantIdx: 7,
			wantOk:  true,
		},
		{
			name:    "2. word",
			set:     []uint8{1, 70, 255},
			del:     []uint8{},
			start:   2,
			wantIdx: 70,
			wantOk:  true,
		},
	}

	for _, tc := range testCases {
		var b BitSet256
		for _, u := range tc.set {
			b.Set(u)
		}

		for _, u := range tc.del {
			b.Clear(u) // without compact
		}

		idx, ok := b.NextSet(tc.start)

		if ok != tc.wantOk {
			t.Errorf("NextSet, %s: got ok: %v, want: %v", tc.name, ok, tc.wantOk)
		}

		if idx != tc.wantIdx {
			t.Errorf("NextSet, %s: got idx: %d, want: %d", tc.name, idx, tc.wantIdx)
		}
	}
}

func TestIsEmpty(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name string
		//
		set []uint8
		del []uint8
		//
		want bool
	}{
		{
			name: "null",
			set:  []uint8{},
			del:  []uint8{},
			want: true,
		},
		{
			name: "zero",
			set:  []uint8{0},
			del:  []uint8{},
			want: false,
		},
		{
			name: "1,5",
			set:  []uint8{1, 5},
			del:  []uint8{},
			want: false,
		},
		{
			name: "many",
			set:  []uint8{1, 65, 130, 190, 250},
			del:  []uint8{},
			want: false,
		},
		{
			name: "set clear",
			set:  []uint8{1},
			del:  []uint8{1},
			want: true,
		},
	}

	for _, tc := range testCases {
		var b BitSet256
		for _, u := range tc.set {
			b.Set(u)
		}

		for _, u := range tc.del {
			b.Clear(u) // without compact
		}

		got := b.IsEmpty()

		if got != tc.want {
			t.Errorf("IsEmpty, %s: got: %v, want: %v", tc.name, got, tc.want)
		}
	}
}

func TestAll(t *testing.T) {
	t.Parallel()
	testCases := []struct {
		name string
		//
		set []uint8
		del []uint8
		//
		wantData []uint8
	}{
		{
			name:     "null",
			set:      []uint8{},
			del:      []uint8{},
			wantData: []uint8{},
		},
		{
			name:     "zero",
			set:      []uint8{0},
			del:      []uint8{},
			wantData: []uint8{0}, // bit #0 is set
		},
		{
			name:     "1,5",
			set:      []uint8{1, 5},
			del:      []uint8{},
			wantData: []uint8{1, 5},
		},
		{
			name:     "many",
			set:      []uint8{1, 65, 130, 190, 250},
			del:      []uint8{},
			wantData: []uint8{1, 65, 130, 190, 250},
		},
		{
			name:     "special, last return",
			set:      []uint8{1},
			del:      []uint8{1}, // delete without compact
			wantData: []uint8{},
		},
	}

	for _, tc := range testCases {
		var b BitSet256
		for _, u := range tc.set {
			b.Set(u)
		}

		for _, u := range tc.del {
			b.Clear(u) // without compact
		}

		buf := b.Bits()

		if !slices.Equal(buf, tc.wantData) {
			t.Errorf("All, %s: returned buf is not equal as expected:\ngot:  %v\nwant: %v",
				tc.name, buf, tc.wantData)
		}
	}
}

// Helper to generate consecutive uint8 slices
func makeSequence(start, length uint8) []uint8 {
	seq := make([]uint8, length)
	for i := range length {
		seq[i] = start + i
	}
	return seq
}

func TestIterators(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		bitset       BitSet256
		wantBits     []uint8
		wantSequence []uint8 // Zero-based sequence index for AllEnumerate
	}{
		{
			name:         "empty bitset",
			bitset:       BitSet256{0, 0, 0, 0},
			wantBits:     nil,
			wantSequence: nil,
		},
		{
			name:         "single bit set at start",
			bitset:       BitSet256{1, 0, 0, 0},
			wantBits:     []uint8{0},
			wantSequence: []uint8{0},
		},
		{
			name:         "single bit set at boundary (255)",
			bitset:       BitSet256{0, 0, 0, 1 << 63},
			wantBits:     []uint8{255},
			wantSequence: []uint8{0},
		},
		{
			name:         "sparse bits across words",
			bitset:       BitSet256{1 << 0, 1 << 10, 1 << 20, 1 << 30},
			wantBits:     []uint8{0, 64 + 10, 128 + 20, 192 + 30},
			wantSequence: []uint8{0, 1, 2, 3},
		},
		{
			name:         "multiple bits in same word",
			bitset:       BitSet256{0b1011, 0, 0, 0}, // Bits 0, 1, 3
			wantBits:     []uint8{0, 1, 3},
			wantSequence: []uint8{0, 1, 2},
		},
		{
			name:         "dense word",
			bitset:       BitSet256{0xFFFFFFFFFFFFFFFF, 0, 0, 0},
			wantBits:     makeSequence(0, 64),
			wantSequence: makeSequence(0, 64),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			t.Run("All", func(t *testing.T) {
				t.Parallel()
				var gotBits []uint8
				for bit := range tt.bitset.All() {
					gotBits = append(gotBits, bit)
				}
				if !slices.Equal(gotBits, tt.wantBits) {
					t.Errorf("All() = %v, want %v", gotBits, tt.wantBits)
				}
			})

			t.Run("AllEnumerate", func(t *testing.T) {
				t.Parallel()
				var gotSeq, gotBits []uint8
				for i, bit := range tt.bitset.AllEnumerate() {
					gotSeq = append(gotSeq, i)
					gotBits = append(gotBits, bit)
				}
				if !slices.Equal(gotBits, tt.wantBits) {
					t.Errorf("AllEnumerate() bits = %v, want %v", gotBits, tt.wantBits)
				}
				if !slices.Equal(gotSeq, tt.wantSequence) {
					t.Errorf("AllEnumerate() sequence = %v, want %v", gotSeq, tt.wantSequence)
				}
			})
		})
	}

	t.Run("early break", func(t *testing.T) {
		t.Parallel()
		bs := BitSet256{0b1111, 0, 0, 0} // Bits 0, 1, 2, 3
		var count int

		for range bs.All() {
			count++
			if count == 2 {
				break
			}
		}
		if count != 2 {
			t.Errorf("All() early break failed: processed %d bits, want 2", count)
		}

		count = 0
		for range bs.AllEnumerate() {
			count++
			if count == 2 {
				break
			}
		}
		if count != 2 {
			t.Errorf("AllEnumerate() early break failed: processed %d bits, want 2", count)
		}
	})
}

func TestAppendBits(t *testing.T) {
	t.Parallel()
	testCases := []struct {
		name string
		//
		set []uint8
		del []uint8
		//
		wantData []uint8
	}{
		{
			name:     "null",
			set:      []uint8{},
			del:      []uint8{},
			wantData: []uint8{},
		},
		{
			name:     "zero",
			set:      []uint8{0},
			del:      []uint8{},
			wantData: []uint8{0}, // bit #0 is set
		},
		{
			name:     "1,5",
			set:      []uint8{1, 5},
			del:      []uint8{},
			wantData: []uint8{1, 5},
		},
		{
			name:     "many",
			set:      []uint8{1, 65, 130, 190, 250},
			del:      []uint8{},
			wantData: []uint8{1, 65, 130, 190, 250},
		},
		{
			name:     "special, last return",
			set:      []uint8{1},
			del:      []uint8{1}, // delete without compact
			wantData: []uint8{},
		},
	}

	for _, tc := range testCases {
		var b BitSet256
		for _, u := range tc.set {
			b.Set(u)
		}

		for _, u := range tc.del {
			b.Clear(u) // without compact
		}

		buf := b.AppendBits(make([]uint8, 0, 256))

		if !slices.Equal(buf, tc.wantData) {
			t.Errorf("AppendBits, %s: returned buf is not equal as expected:\ngot:  %v\nwant: %v",
				tc.name, buf, tc.wantData)
		}
	}
}

func TestAlignedPairs(t *testing.T) {
	const evenBits uint64 = 0x5555555555555555

	tests := []struct {
		name  string
		input BitSet256
		want  BitSet256
	}{
		{
			name:  "Empty bitset",
			input: BitSet256{0, 0, 0, 0},
			want:  BitSet256{0, 0, 0, 0},
		},
		{
			name:  "Aligned pair at LSB (bits 0, 1) -> MATCH at index 0",
			input: BitSet256{3, 0, 0, 0}, // 3 = 0b0011 (bits 0 and 1 set)
			want:  BitSet256{1, 0, 0, 0}, // Result bit 0 set
		},
		{
			name:  "Unaligned pair (bits 1, 2) -> NO MATCH (odd index 1)",
			input: BitSet256{(1 << 1) | (1 << 2), 0, 0, 0}, // Bits 1 and 2 set
			want:  BitSet256{0, 0, 0, 0},                   // Must be filtered out
		},
		{
			name:  "Multiple aligned pairs in word 0 (bits 0,1 and 4,5)",
			input: BitSet256{(1 << 0) | (1 << 1) | (1 << 4) | (1 << 5), 0, 0, 0},
			want:  BitSet256{(1 << 0) | (1 << 4), 0, 0, 0},
		},
		{
			name:  "Word 0 upper boundary aligned pair (bits 62, 63) -> MATCH at index 62",
			input: BitSet256{(1 << 62) | (1 << 63), 0, 0, 0},
			want:  BitSet256{1 << 62, 0, 0, 0},
		},
		{
			name:  "Cross-word boundary pair (bits 63, 64) -> NO MATCH (odd index 63)",
			input: BitSet256{1 << 63, 1, 0, 0}, // Bit 63 (word 0) and Bit 0 (word 1)
			want:  BitSet256{0, 0, 0, 0},       // Discarded (n=63 is odd)
		},
		{
			name:  "Word 1 lower boundary aligned pair (bits 64, 65) -> MATCH at index 64",
			input: BitSet256{0, 3, 0, 0}, // Bits 0 and 1 set in word 1
			want:  BitSet256{0, 1, 0, 0}, // Result bit 0 set in word 1 (overall bit 64)
		},
		{
			name:  "Word 3 upper boundary aligned pair (bits 254, 255) -> MATCH at index 254",
			input: BitSet256{0, 0, 0, (1 << 62) | (1 << 63)},
			want:  BitSet256{0, 0, 0, 1 << 62},
		},
		{
			name:  "Dense sequential run (bits 0, 1, 2, 3) -> MATCH at 0 and 2",
			input: BitSet256{0xF, 0, 0, 0}, // Bits 0, 1, 2, 3 set
			want:  BitSet256{0x5, 0, 0, 0}, // Bits 0 and 2 set (pairs (0,1) and (2,3))
		},
		{
			name:  "All bits set",
			input: BitSet256{^uint64(0), ^uint64(0), ^uint64(0), ^uint64(0)},
			want:  BitSet256{evenBits, evenBits, evenBits, evenBits},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.input.AlignedPairs()
			if got != tt.want {
				t.Errorf("AlignedPairs() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestShifts(t *testing.T) {
	tests := []struct {
		name      string
		input     BitSet256
		wantRight BitSet256
		wantLeft  BitSet256
	}{
		{
			name:      "Empty bitset",
			input:     BitSet256{0, 0, 0, 0},
			wantRight: BitSet256{0, 0, 0, 0},
			wantLeft:  BitSet256{0, 0, 0, 0},
		},
		{
			name:      "Single bit at position 0 (LSB of word 0)",
			input:     BitSet256{1, 0, 0, 0},
			wantRight: BitSet256{0, 0, 0, 0},
			wantLeft:  BitSet256{2, 0, 0, 0},
		},
		{
			name:      "Cross word boundary 0 -> 1 (bit 63 shifted left to bit 64)",
			input:     BitSet256{1 << 63, 0, 0, 0},
			wantRight: BitSet256{1 << 62, 0, 0, 0},
			wantLeft:  BitSet256{0, 1, 0, 0},
		},
		{
			name:      "Cross word boundary 1 -> 0 (bit 64 shifted right to bit 63)",
			input:     BitSet256{0, 1, 0, 0},
			wantRight: BitSet256{1 << 63, 0, 0, 0},
			wantLeft:  BitSet256{0, 2, 0, 0},
		},
		{
			name:      "Cross word boundary 3 -> 2 (bit 192 shifted right to bit 191)",
			input:     BitSet256{0, 0, 0, 1},
			wantRight: BitSet256{0, 0, 1 << 63, 0},
			wantLeft:  BitSet256{0, 0, 0, 2},
		},
		{
			name:      "Single bit at position 255 (MSB of word 3)",
			input:     BitSet256{0, 0, 0, 1 << 63},
			wantRight: BitSet256{0, 0, 0, 1 << 62},
			wantLeft:  BitSet256{0, 0, 0, 0}, // Shifts out into nothingness
		},
		{
			name:      "All bits set",
			input:     BitSet256{^uint64(0), ^uint64(0), ^uint64(0), ^uint64(0)},
			wantRight: BitSet256{^uint64(0), ^uint64(0), ^uint64(0), ^uint64(0) >> 1},
			wantLeft:  BitSet256{^uint64(1), ^uint64(0), ^uint64(0), ^uint64(0)},
		},
	}

	for _, tt := range tests {
		t.Run("RightShift/"+tt.name, func(t *testing.T) {
			got := tt.input
			got.RightShift()
			if got != tt.wantRight {
				t.Errorf("RightShift() = %v, want %v", got, tt.wantRight)
			}
		})

		t.Run("LeftShift/"+tt.name, func(t *testing.T) {
			got := tt.input
			got.LeftShift()
			if got != tt.wantLeft {
				t.Errorf("LeftShift() = %v, want %v", got, tt.wantLeft)
			}
		})
	}
}

// test setting every 3rd bit, just in case something odd is happening
func TestOnesCount(t *testing.T) {
	t.Parallel()
	var b BitSet256
	tot := uint8(64*3 + 11)
	for i := uint8(0); i < tot; i += 3 {
		sz := b.OnesCount()
		if sz != int(i)/3 {
			t.Errorf("Count reported as %d, but it should be %d", sz, i/3)
			break
		}
		b.Set(i)
	}
}

func TestOr(t *testing.T) {
	t.Parallel()

	zero := BitSet256{0, 0, 0, 0}
	full := BitSet256{
		0xFFFF_FFFF_FFFF_FFFF,
		0xFFFF_FFFF_FFFF_FFFF,
		0xFFFF_FFFF_FFFF_FFFF,
		0xFFFF_FFFF_FFFF_FFFF,
	}

	tests := []struct {
		name string
		b    BitSet256
		c    BitSet256
		want BitSet256
	}{
		{
			name: "zero values",
			b:    zero,
			c:    zero,
			want: zero,
		},
		{
			name: "identity with zero (b | 0 == b)",
			b:    BitSet256{1, 2, 3, 4},
			c:    zero,
			want: BitSet256{1, 2, 3, 4},
		},
		{
			name: "identity with zero (0 | c == c)",
			b:    zero,
			c:    BitSet256{1, 2, 3, 4},
			want: BitSet256{1, 2, 3, 4},
		},
		{
			name: "idempotency (b | b == b)",
			b:    BitSet256{0xAA, 0xBB, 0xCC, 0xDD},
			c:    BitSet256{0xAA, 0xBB, 0xCC, 0xDD},
			want: BitSet256{0xAA, 0xBB, 0xCC, 0xDD},
		},
		{
			name: "disjoint words OR",
			b:    BitSet256{0xF000_0000_0000_0000, 0x0, 0x00FF_0000_0000_0000, 0x0},
			c:    BitSet256{0x0F00_0000_0000_0000, 0x0, 0x0000_FF00_0000_0000, 0x0},
			want: BitSet256{0xFF00_0000_0000_0000, 0x0, 0x00FF_FF00_0000_0000, 0x0},
		},
		{
			name: "union with full set resulting in full set",
			b:    BitSet256{0x1234, 0x5678, 0x9ABC, 0xDEF0},
			c:    full,
			want: full,
		},
		{
			name: "interleaved bit patterns",
			b:    BitSet256{0xAAAA_AAAA_AAAA_AAAA, 0x5555_5555_5555_5555, 0xAAAA_AAAA_AAAA_AAAA, 0x5555_5555_5555_5555},
			c:    BitSet256{0x5555_5555_5555_5555, 0xAAAA_AAAA_AAAA_AAAA, 0x5555_5555_5555_5555, 0xAAAA_AAAA_AAAA_AAAA},
			want: full,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			// Test b.Or(c)
			got := tt.b.Or(&tt.c)
			if got != tt.want {
				t.Errorf("got %v, want %v", got, tt.want)
			}

			// Test commutativity: b | c == c | b
			gotCommutative := tt.c.Or(&tt.b)
			if gotCommutative != tt.want {
				t.Errorf("commutativity check failed: got %v, want %v", gotCommutative, tt.want)
			}
		})
	}
}

func TestAnd(t *testing.T) {
	t.Parallel()

	zero := BitSet256{0, 0, 0, 0}
	full := BitSet256{
		0xFFFF_FFFF_FFFF_FFFF,
		0xFFFF_FFFF_FFFF_FFFF,
		0xFFFF_FFFF_FFFF_FFFF,
		0xFFFF_FFFF_FFFF_FFFF,
	}

	tests := []struct {
		name string
		b    BitSet256
		c    BitSet256
		want BitSet256
	}{
		{
			name: "zero values",
			b:    zero,
			c:    zero,
			want: zero,
		},
		{
			name: "annihilation with zero (b & 0 == 0)",
			b:    BitSet256{1, 2, 3, 4},
			c:    zero,
			want: zero,
		},
		{
			name: "annihilation with zero (0 & c == 0)",
			b:    zero,
			c:    BitSet256{1, 2, 3, 4},
			want: zero,
		},
		{
			name: "identity with full set (b & full == b)",
			b:    BitSet256{0x1234, 0x5678, 0x9ABC, 0xDEF0},
			c:    full,
			want: BitSet256{0x1234, 0x5678, 0x9ABC, 0xDEF0},
		},
		{
			name: "idempotency (b & b == b)",
			b:    BitSet256{0xAA, 0xBB, 0xCC, 0xDD},
			c:    BitSet256{0xAA, 0xBB, 0xCC, 0xDD},
			want: BitSet256{0xAA, 0xBB, 0xCC, 0xDD},
		},
		{
			name: "disjoint sets resulting in zero",
			b:    BitSet256{0xF000_0000_0000_0000, 0x0, 0x00FF_0000_0000_0000, 0x0},
			c:    BitSet256{0x0F00_0000_0000_0000, 0x0, 0x0000_FF00_0000_0000, 0x0},
			want: zero,
		},
		{
			name: "partial bit overlap across words",
			b:    BitSet256{0xFF00_FF00_FF00_FF00, 0x00FF_00FF_00FF_00FF, 0xF0F0_F0F0_F0F0_F0F0, 0x0F0F_0F0F_0F0F_0F0F},
			c:    BitSet256{0xF000_F000_F000_F000, 0x00F0_00F0_00F0_00F0, 0xFF00_FF00_FF00_FF00, 0xFF00_FF00_FF00_FF00},
			want: BitSet256{0xF000_F000_F000_F000, 0x00F0_00F0_00F0_00F0, 0xF000_F000_F000_F000, 0x0F00_0F00_0F00_0F00},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			// Test b.And(c)
			got := tt.b.And(&tt.c)
			if got != tt.want {
				t.Errorf("got %v, want %v", got, tt.want)
			}

			// Test commutativity: b & c == c & b
			gotCommutative := tt.c.And(&tt.b)
			if gotCommutative != tt.want {
				t.Errorf("commutativity check failed: got %v, want %v", gotCommutative, tt.want)
			}
		})
	}
}

func TestOverlaps(t *testing.T) {
	t.Parallel()

	zero := BitSet256{0, 0, 0, 0}
	full := BitSet256{
		0xFFFF_FFFF_FFFF_FFFF,
		0xFFFF_FFFF_FFFF_FFFF,
		0xFFFF_FFFF_FFFF_FFFF,
		0xFFFF_FFFF_FFFF_FFFF,
	}

	tests := []struct {
		name string
		b    BitSet256
		c    BitSet256
		want bool
	}{
		{
			name: "both zero values",
			b:    zero,
			c:    zero,
			want: false,
		},
		{
			name: "receiver zero against populated set",
			b:    zero,
			c:    BitSet256{1, 0, 0, 0},
			want: false,
		},
		{
			name: "disjoint sets (no common set bits)",
			b:    BitSet256{0xAAAA_AAAA_AAAA_AAAA, 0x5555_5555_5555_5555, 0xAAAA_AAAA_AAAA_AAAA, 0x5555_5555_5555_5555},
			c:    BitSet256{0x5555_5555_5555_5555, 0xAAAA_AAAA_AAAA_AAAA, 0x5555_5555_5555_5555, 0xAAAA_AAAA_AAAA_AAAA},
			want: false,
		},
		{
			name: "full set intersection",
			b:    BitSet256{1, 0, 0, 0},
			c:    full,
			want: true,
		},
		{
			name: "single bit match in word 0",
			b:    BitSet256{1, 0, 0, 0},
			c:    BitSet256{1, 0, 0, 0},
			want: true,
		},
		{
			name: "single bit match in word 1",
			b:    BitSet256{0, 1 << 31, 0, 0},
			c:    BitSet256{0, 1 << 31, 0, 0},
			want: true,
		},
		{
			name: "single bit match in word 2",
			b:    BitSet256{0, 0, 1 << 63, 0},
			c:    BitSet256{0, 0, 1 << 63, 0},
			want: true,
		},
		{
			name: "single bit match in word 3",
			b:    BitSet256{0, 0, 0, 1},
			c:    BitSet256{0, 0, 0, 1},
			want: true,
		},
		{
			name: "identical non-zero sets",
			b:    BitSet256{0x1234, 0x5678, 0x9ABC, 0xDEF0},
			c:    BitSet256{0x1234, 0x5678, 0x9ABC, 0xDEF0},
			want: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := tt.b.Overlaps(&tt.c)
			if got != tt.want {
				t.Errorf("got %v, want %v", got, tt.want)
			}

			// Test symmetry
			gotSymmetric := tt.c.Overlaps(&tt.b)
			if gotSymmetric != tt.want {
				t.Errorf("symmetry check failed: got %v, want %v", gotSymmetric, tt.want)
			}
		})
	}
}

func TestAndTop(t *testing.T) {
	t.Parallel()
	testCases := []struct {
		name    string
		a, b    []uint8
		wantIdx uint8
		wantOk  bool
	}{
		{
			name:    "both empty",
			a:       []uint8{},
			b:       []uint8{},
			wantIdx: 0,
			wantOk:  false,
		},
		{
			name:    "disjoint",
			a:       []uint8{10, 20},
			b:       []uint8{30, 40},
			wantIdx: 0,
			wantOk:  false,
		},
		{
			name:    "match word 0",
			a:       []uint8{5, 10},
			b:       []uint8{10, 15},
			wantIdx: 10,
			wantOk:  true,
		},
		{
			name:    "match word 1",
			a:       []uint8{70, 80},
			b:       []uint8{60, 70},
			wantIdx: 70,
			wantOk:  true,
		},
		{
			name:    "match word 2",
			a:       []uint8{130, 140},
			b:       []uint8{140, 150},
			wantIdx: 140,
			wantOk:  true,
		},
		{
			name:    "match word 3",
			a:       []uint8{200, 210},
			b:       []uint8{210, 220},
			wantIdx: 210,
			wantOk:  true,
		},
		{
			name:    "multiple matches",
			a:       []uint8{10, 70, 130, 200},
			b:       []uint8{10, 70, 130, 200},
			wantIdx: 200,
			wantOk:  true,
		},
	}

	for _, tc := range testCases {
		var a, b BitSet256
		for _, v := range tc.a {
			a.Set(v)
		}
		for _, v := range tc.b {
			b.Set(v)
		}

		gotIdx, gotOk := a.AndTop(&b)
		if gotOk != tc.wantOk {
			t.Errorf("AndTop, %s: got ok %v, want %v", tc.name, gotOk, tc.wantOk)
		}
		if gotIdx != tc.wantIdx {
			t.Errorf("AndTop, %s: got idx %d, want %d", tc.name, gotIdx, tc.wantIdx)
		}

		// Commutative check
		gotIdx2, gotOk2 := b.AndTop(&a)
		if gotOk2 != tc.wantOk {
			t.Errorf("AndTop (commutative), %s: got ok %v, want %v", tc.name, gotOk2, tc.wantOk)
		}
		if gotIdx2 != tc.wantIdx {
			t.Errorf("AndTop (commutative), %s: got idx %d, want %d", tc.name, gotIdx2, tc.wantIdx)
		}
	}
}

func TestXor(t *testing.T) {
	t.Parallel()

	zero := BitSet256{0, 0, 0, 0}
	full := BitSet256{
		0xFFFF_FFFF_FFFF_FFFF,
		0xFFFF_FFFF_FFFF_FFFF,
		0xFFFF_FFFF_FFFF_FFFF,
		0xFFFF_FFFF_FFFF_FFFF,
	}

	tests := []struct {
		name string
		b    BitSet256
		c    BitSet256
		want BitSet256
	}{
		{
			name: "zero values",
			b:    zero,
			c:    zero,
			want: zero,
		},
		{
			name: "identity with zero (b ^ 0 == b)",
			b:    BitSet256{1, 2, 3, 4},
			c:    zero,
			want: BitSet256{1, 2, 3, 4},
		},
		{
			name: "self-inversion / annihilation (b ^ b == 0)",
			b:    BitSet256{0x1234, 0x5678, 0x9ABC, 0xDEF0},
			c:    BitSet256{0x1234, 0x5678, 0x9ABC, 0xDEF0},
			want: zero,
		},
		{
			name: "complement with full set (b ^ full == ^b)",
			b:    BitSet256{0x0000_0000_0000_0000, 0xFFFF_FFFF_FFFF_FFFF, 0xAAAA_AAAA_AAAA_AAAA, 0x5555_5555_5555_5555},
			c:    full,
			want: BitSet256{0xFFFF_FFFF_FFFF_FFFF, 0x0000_0000_0000_0000, 0x5555_5555_5555_5555, 0xAAAA_AAAA_AAAA_AAAA},
		},
		{
			name: "disjoint words XOR",
			b:    BitSet256{0xF000_0000_0000_0000, 0x0, 0x00FF_0000_0000_0000, 0x0},
			c:    BitSet256{0x0F00_0000_0000_0000, 0x0, 0x0000_FF00_0000_0000, 0x0},
			want: BitSet256{0xFF00_0000_0000_0000, 0x0, 0x00FF_FF00_0000_0000, 0x0},
		},
		{
			name: "overlapping bit toggle",
			b:    BitSet256{0xFF00_FF00_FF00_FF00, 0x00FF_00FF_00FF_00FF, 0xF0F0_F0F0_F0F0_F0F0, 0x0F0F_0F0F_0F0F_0F0F},
			c:    BitSet256{0xF000_F000_F000_F000, 0x00F0_00F0_00F0_00F0, 0xFF00_FF00_FF00_FF00, 0xFF00_FF00_FF00_FF00},
			want: BitSet256{0x0F00_0F00_0F00_0F00, 0x000F_000F_000F_000F, 0x0FF0_0FF0_0FF0_0FF0, 0xF00F_F00F_F00F_F00F},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			// Test b.Xor(c)
			got := tt.b.Xor(&tt.c)
			if got != tt.want {
				t.Errorf("got %v, want %v", got, tt.want)
			}

			// Test commutativity: b ^ c == c ^ b
			gotCommutative := tt.c.Xor(&tt.b)
			if gotCommutative != tt.want {
				t.Errorf("commutativity check failed: got %v, want %v", gotCommutative, tt.want)
			}
		})
	}
}

func TestRank(t *testing.T) {
	t.Parallel()
	u := []uint8{0, 3, 5, 7, 11, 62, 63, 64, 70, 150, 255}

	tests := []struct {
		idx  uint8
		want int
	}{
		{
			idx:  0,
			want: 1,
		},
		{
			idx:  1,
			want: 1,
		},
		{
			idx:  2,
			want: 1,
		},
		{
			idx:  3,
			want: 2,
		},
		{
			idx:  4,
			want: 2,
		},
		{
			idx:  62,
			want: 6,
		},
		{
			idx:  63,
			want: 7,
		},
		{
			idx:  64,
			want: 8,
		},
		{
			idx:  150,
			want: 10,
		},
		{
			idx:  254,
			want: 10,
		},
		{
			idx:  255,
			want: 11,
		},
	}

	var b BitSet256
	for _, v := range u {
		b.Set(v)
	}

	for _, tc := range tests {
		if got := b.Rank(tc.idx); got != tc.want {
			t.Errorf("Rank(%d): want: %d, got: %d", tc.idx, tc.want, got)
		}
	}
}

func BenchmarkIsEmpty(b *testing.B) {
	aa := []BitSet256{
		randomBitSet256(),
		randomBitSet256(),
		randomBitSet256(),
		randomBitSet256(),
	}

	for i := uint8(0); b.Loop(); i++ {
		aa[i&3].IsEmpty()
	}
}

func BenchmarkSet(b *testing.B) {
	bs := randomBitSet256()
	for i := uint8(0); b.Loop(); i++ {
		bs.Set(i)
	}
}

func BenchmarkTest(b *testing.B) {
	bs := randomBitSet256()
	for i := uint8(0); b.Loop(); i++ {
		_ = bs.Test(i)
	}
}

func BenchmarkClear(b *testing.B) {
	bs := randomBitSet256()
	for i := uint8(0); b.Loop(); i++ {
		bs.Clear(i)
	}
}

func BenchmarkOnesCount(b *testing.B) {
	aa := []BitSet256{
		randomBitSet256(),
		randomBitSet256(),
		randomBitSet256(),
		randomBitSet256(),
	}

	for i := uint8(0); b.Loop(); i++ {
		aa[i&3].OnesCount()
	}
}

func BenchmarkRank(b *testing.B) {
	aa := []BitSet256{
		randomBitSet256(),
		randomBitSet256(),
		randomBitSet256(),
		randomBitSet256(),
	}

	for i := uint8(0); b.Loop(); i++ {
		aa[i&3].Rank(i)
	}
}

func BenchmarkFirstSet(b *testing.B) {
	b.Run("Sparse", func(b *testing.B) {
		aa := []BitSet256{
			{0, 0, 0, 1},
			{0, 0, 0, 1},
			{0, 0, 0, 1},
			{0, 0, 0, 1},
		}

		for i := uint8(0); b.Loop(); i++ {
			aa[i&3].FirstSet()
		}
	})

	b.Run("Dense", func(b *testing.B) {
		aa := []BitSet256{
			randomBitSet256(),
			randomBitSet256(),
			randomBitSet256(),
			randomBitSet256(),
		}

		for i := uint8(0); b.Loop(); i++ {
			aa[i&3].FirstSet()
		}
	})
}

func BenchmarkNextSet(b *testing.B) {
	b.Run("Sparse", func(b *testing.B) {
		aa := []BitSet256{
			{0, 0, 0, 1},
			{0, 0, 0, 1},
			{0, 0, 0, 1},
			{0, 0, 0, 1},
		}

		for i := uint8(0); b.Loop(); i++ {
			aa[i&3].NextSet(i)
		}
	})

	b.Run("Dense", func(b *testing.B) {
		aa := []BitSet256{
			randomBitSet256(),
			randomBitSet256(),
			randomBitSet256(),
			randomBitSet256(),
		}

		for i := uint8(0); b.Loop(); i++ {
			aa[i&3].NextSet(i)
		}
	})
}

func BenchmarkLastSet(b *testing.B) {
	b.Run("Sparse", func(b *testing.B) {
		aa := []BitSet256{
			{1, 0, 0, 0},
			{1, 0, 0, 0},
			{1, 0, 0, 0},
			{1, 0, 0, 0},
		}

		for i := uint8(0); b.Loop(); i++ {
			aa[i&3].LastSet()
		}
	})

	b.Run("Dense", func(b *testing.B) {
		aa := []BitSet256{
			randomBitSet256(),
			randomBitSet256(),
			randomBitSet256(),
			randomBitSet256(),
		}

		for i := uint8(0); b.Loop(); i++ {
			aa[i&3].LastSet()
		}
	})
}

func BenchmarkAndTop(b *testing.B) {
	b.Run("Sparse", func(b *testing.B) {
		aa := []BitSet256{
			{1, 0, 0, 0},
			{1, 0, 0, 0},
			{1, 0, 0, 0},
			{1, 0, 0, 0},
		}

		var i uint8
		for b.Loop() {
			aa[i&3].AndTop(&aa[i&3])
			i++
		}
	})

	b.Run("Dense", func(b *testing.B) {
		aa := []BitSet256{
			randomBitSet256(),
			randomBitSet256(),
			randomBitSet256(),
			randomBitSet256(),
		}

		var i uint8
		for b.Loop() {
			aa[i&3].AndTop(&aa[i&3])
			i++
		}
	})
}

func BenchmarkOverlaps(b *testing.B) {
	aa := randomBitSet256()
	bb := []BitSet256{
		randomBitSet256(),
		randomBitSet256(),
		randomBitSet256(),
		randomBitSet256(),
	}

	var i uint8
	for b.Loop() {
		aa.Overlaps(&bb[i&3])
		i++
	}
}

func BenchmarkOr(b *testing.B) {
	aa := randomBitSet256()
	bb := []BitSet256{
		randomBitSet256(),
		randomBitSet256(),
		randomBitSet256(),
		randomBitSet256(),
	}

	var i uint8
	for b.Loop() {
		aa.Or(&bb[i&3])
		i++
	}
}

func BenchmarkAnd(b *testing.B) {
	aa := randomBitSet256()
	bb := []BitSet256{
		randomBitSet256(),
		randomBitSet256(),
		randomBitSet256(),
		randomBitSet256(),
	}

	var i uint8
	for b.Loop() {
		aa.And(&bb[i&3])
		i++
	}
}

func BenchmarkXor(b *testing.B) {
	aa := randomBitSet256()
	bb := []BitSet256{
		randomBitSet256(),
		randomBitSet256(),
		randomBitSet256(),
		randomBitSet256(),
	}

	var i uint8
	for b.Loop() {
		aa.Xor(&bb[i&3])
		i++
	}
}

func BenchmarkAll(b *testing.B) {
	var sink uint8

	b.Run("Sparse", func(b *testing.B) {
		aa := []BitSet256{
			{0, 0, 0, 1},
			{0, 0, 0, 1},
			{0, 0, 0, 1},
			{0, 0, 0, 1},
		}

		var i uint8
		for b.Loop() {
			for bit := range aa[i&3].All() {
				sink = bit
			}
			i++
		}
		sinkSliceUint8 = append(sinkSliceUint8, sink)
	})

	b.Run("Dense", func(b *testing.B) {
		aa := []BitSet256{
			randomBitSet256(),
			randomBitSet256(),
			randomBitSet256(),
			randomBitSet256(),
		}

		var i uint8
		for b.Loop() {
			for bit := range aa[i&3].All() {
				sink = bit
			}
			i++
		}
		sinkSliceUint8 = append(sinkSliceUint8, sink)
	})
}

func BenchmarkAppendBits(b *testing.B) {
	b.Run("Sparse", func(b *testing.B) {
		aa := []BitSet256{
			{0, 0, 0, 1},
			{0, 0, 0, 1},
			{0, 0, 0, 1},
			{0, 0, 0, 1},
		}

		buf := make([]uint8, 0, 256)
		var i uint8
		for b.Loop() {
			buf = aa[i&3].AppendBits(buf[:0])
			i++
		}
		copy(sinkSliceUint8, buf)
	})

	b.Run("Dense", func(b *testing.B) {
		aa := []BitSet256{
			randomBitSet256(),
			randomBitSet256(),
			randomBitSet256(),
			randomBitSet256(),
		}

		buf := make([]uint8, 0, 256)
		var i uint8
		for b.Loop() {
			buf = aa[i&3].AppendBits(buf[:0])
			i++
		}
		copy(sinkSliceUint8, buf)
	})
}

func BenchmarkBits(b *testing.B) {
	var sink []uint8

	b.Run("Sparse", func(b *testing.B) {
		aa := []BitSet256{
			{0, 0, 0, 1},
			{0, 0, 0, 1},
			{0, 0, 0, 1},
			{0, 0, 0, 1},
		}

		var i uint8
		for b.Loop() {
			sink = aa[i&3].Bits()
			i++
		}
		copy(sinkSliceUint8, sink)
	})

	b.Run("Dense", func(b *testing.B) {
		aa := []BitSet256{
			randomBitSet256(),
			randomBitSet256(),
			randomBitSet256(),
			randomBitSet256(),
		}

		var i uint8
		for b.Loop() {
			sink = aa[i&3].Bits()
			i++
		}
		copy(sinkSliceUint8, sink)
	})
}

func BenchmarkRightShift(b *testing.B) {
	aa := []BitSet256{
		randomBitSet256(),
		randomBitSet256(),
		randomBitSet256(),
		randomBitSet256(),
	}

	var i uint8
	for b.Loop() {
		aa[i&3].RightShift()
		i++
	}
}

func BenchmarkLeftShift(b *testing.B) {
	aa := []BitSet256{
		randomBitSet256(),
		randomBitSet256(),
		randomBitSet256(),
		randomBitSet256(),
	}

	var i uint8
	for b.Loop() {
		aa[i&3].LeftShift()
		i++
	}
}

func BenchmarkAlignedPairs(b *testing.B) {
	sink := BitSet256{}

	aa := []BitSet256{
		randomBitSet256(),
		randomBitSet256(),
		randomBitSet256(),
		randomBitSet256(),
	}

	var i uint8
	for b.Loop() {
		sink = aa[i&3].AlignedPairs()
		i++
	}

	sinkBitSet256 = sink
}
