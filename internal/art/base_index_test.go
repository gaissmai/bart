// Copyright (c) 2025 Karl Gaissmaier
// SPDX-License-Identifier: MIT

package art

import "testing"

func TestOctetToIdx(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		octet uint8
		want  uint8
	}{
		{
			octet: 0,
			want:  128,
		},
		{
			octet: 255,
			want:  255,
		},
		{
			octet: 128,
			want:  192,
		},
	}

	for _, tc := range testCases {
		got := OctetToIdx(tc.octet)
		if got != tc.want {
			t.Errorf("OctetToIdx(%d), want: %d, got: %d", tc.octet, tc.want, got)
		}
	}
}

func TestPfxBits(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		depth int
		idx   uint8
		want  uint8
	}{
		// idx: 0 is invalid
		{
			depth: 0,
			idx:   1,
			want:  0,
		},
		{
			depth: 0,
			idx:   19,
			want:  4,
		},
		{
			depth: 15,
			idx:   19,
			want:  124,
		},
	}

	for _, tc := range testCases {
		got := PfxBits(tc.depth, tc.idx)
		if got != tc.want {
			t.Errorf("PfxBits(%d, %d), want: %d, got: %d", tc.depth, tc.idx, tc.want, got)
		}
	}
}

func TestPfxToIdx(t *testing.T) {
	t.Parallel()
	testCases := []struct {
		octet  uint8
		pfxLen uint8
		want   uint8
	}{
		{
			octet:  0,
			pfxLen: 0,
			want:   1,
		},
		{
			octet:  0,
			pfxLen: 1,
			want:   2,
		},
		{
			octet:  128,
			pfxLen: 1,
			want:   3,
		},
		{
			octet:  80,
			pfxLen: 4,
			want:   21,
		},
		{
			octet:  255,
			pfxLen: 7,
			want:   255,
		},
	}

	for _, tc := range testCases {
		got := PfxToIdx(tc.octet, tc.pfxLen)
		if got != tc.want {
			t.Errorf("PfxToIdx(%d, %d), want: %d, got: %d", tc.octet, tc.pfxLen, tc.want, got)
		}
	}
}

func TestIdxToPfx(t *testing.T) {
	t.Parallel()
	testCases := []struct {
		idx        uint8
		wantOctet  uint8
		wantPfxLen uint8
	}{
		// idx: 0 is invalid
		{
			idx:        1,
			wantOctet:  0,
			wantPfxLen: 0,
		},
		{
			idx:        15,
			wantOctet:  224,
			wantPfxLen: 3,
		},
		{
			idx:        255,
			wantOctet:  254,
			wantPfxLen: 7,
		},
	}

	for _, tc := range testCases {
		gotOctet, gotPfxLen := IdxToPfx(tc.idx)
		if gotOctet != tc.wantOctet || gotPfxLen != tc.wantPfxLen {
			t.Errorf("IdxToPfx(%d), want: (%d, %d), got: (%d, %d)", tc.idx, tc.wantOctet, tc.wantPfxLen, gotOctet, gotPfxLen)
		}
	}
}

func TestIdxToRange(t *testing.T) {
	t.Parallel()
	testCases := []struct {
		idx       uint8
		wantFirst uint8
		wantLast  uint8
	}{
		// idx: 0 is invalid
		{
			idx:       1,
			wantFirst: 0,
			wantLast:  255,
		},
		{
			idx:       2,
			wantFirst: 0,
			wantLast:  127,
		},
		{
			idx:       3,
			wantFirst: 128,
			wantLast:  255,
		},
		{
			idx:       4,
			wantFirst: 0,
			wantLast:  63,
		},
		{
			idx:       8,
			wantFirst: 0,
			wantLast:  31,
		},
		{
			idx:       13,
			wantFirst: 160,
			wantLast:  191,
		},
		{
			idx:       81,
			wantFirst: 68,
			wantLast:  71,
		},
		{
			idx:       254,
			wantFirst: 252,
			wantLast:  253,
		},
		{
			idx:       255,
			wantFirst: 254,
			wantLast:  255,
		},
	}

	for _, tc := range testCases {
		gotFirst, gotLast := IdxToRange(tc.idx)
		if gotFirst != tc.wantFirst || gotLast != tc.wantLast {
			t.Errorf("IdxToRange(%d), want: (%d, %d), got: (%d, %d)",
				tc.idx, tc.wantFirst, tc.wantLast, gotFirst, gotLast)
		}
	}
}

func TestNetMask(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		bits uint8
		want uint8
	}{
		{
			bits: 0,
			want: 0x00, // 00000000
		},
		{
			bits: 1,
			want: 0x80, // 10000000
		},
		{
			bits: 2,
			want: 0xC0, // 11000000
		},
		{
			bits: 3,
			want: 0xE0, // 11100000
		},
		{
			bits: 4,
			want: 0xF0, // 11110000
		},
		{
			bits: 5,
			want: 0xF8, // 11111000
		},
		{
			bits: 6,
			want: 0xFC, // 11111100
		},
		{
			bits: 7,
			want: 0xFE, // 11111110
		},
		{
			bits: 8,
			want: 0xFF, // 11111111
		},
	}

	for _, tc := range testCases {
		got := NetMask(tc.bits)
		if got != tc.want {
			t.Errorf("NetMask(%d), want: 0x%02X, got: 0x%02X", tc.bits, tc.want, got)
		}
	}
}

func TestNetMask_PanicOnInvalidInput(t *testing.T) {
	t.Parallel()

	defer func() {
		if r := recover(); r == nil {
			t.Errorf("NetMask(9) should have panicked")
		}
	}()
	NetMask(9)
}
