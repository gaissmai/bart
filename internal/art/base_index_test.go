// Copyright (c) 2025 Karl Gaissmaier
// SPDX-License-Identifier: MIT

package art

import "testing"

func TestIdx256OutOfBounds(t *testing.T) {
	t.Parallel()

	t.Run("IdxToPfx256(0)", func(t *testing.T) {
		t.Parallel()
		defer func() {
			if r := recover(); r == nil {
				t.Error("An idx out of bounds MUST panic")
			}
		}()

		IdxToPfx256(0)
	})

	t.Run("PfxLen256(0,0)", func(t *testing.T) {
		t.Parallel()
		defer func() {
			if r := recover(); r == nil {
				t.Error("An idx out of bounds MUST panic")
			}
		}()

		PfxLen256(0, 0)
	})
}

func TestHostIdx(t *testing.T) {
	testCases := []struct {
		octet uint8
		want  uint
	}{
		{
			octet: 0,
			want:  256,
		},
		{
			octet: 255,
			want:  511,
		},
	}

	for _, tc := range testCases {
		got := HostIdx(tc.octet)
		if got != tc.want {
			t.Errorf("HostIdx(%d), want: %d, got: %d", tc.octet, tc.want, got)
		}
	}
}

func TestPfxLen256(t *testing.T) {
	testCases := []struct {
		depth int
		idx   uint8
		want  uint8
	}{
		/*
			{
				depth: 0,
				idx:   0,  // invalid
				want:  -1, // invalid
			},
		*/
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
		got := PfxLen256(tc.depth, tc.idx)
		if got != tc.want {
			t.Errorf("PfxLen256(%d, %d), want: %d, got: %d", tc.depth, tc.idx, tc.want, got)
		}
	}
}

func TestPfxToIdx(t *testing.T) {
	testCases := []struct {
		octet  uint8
		pfxLen uint8
		want   uint
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
			octet:  254,
			pfxLen: 7,
			want:   255,
		},
		{
			octet:  255,
			pfxLen: 7,
			want:   255,
		},
		{
			octet:  0,
			pfxLen: 8,
			want:   256,
		},
		{
			octet:  255,
			pfxLen: 8,
			want:   511,
		},
	}

	for _, tc := range testCases {
		got := pfxToIdx(tc.octet, tc.pfxLen)
		if got != tc.want {
			t.Errorf("PfxToIdx(%d, %d), want: %d, got: %d", tc.octet, tc.pfxLen, tc.want, got)
		}
	}
}

func TestPfxToIdx256(t *testing.T) {
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
		// pfcLen 8, idx gets shifted >> 1
		{
			octet:  0,
			pfxLen: 8,
			want:   128,
		},
		{
			octet:  255,
			pfxLen: 8,
			want:   255,
		},
	}

	for _, tc := range testCases {
		got := PfxToIdx256(tc.octet, tc.pfxLen)
		if got != tc.want {
			t.Errorf("PfxToIdx256(%d, %d), want: %d, got: %d", tc.octet, tc.pfxLen, tc.want, got)
		}
	}
}

func TestIdxToPfx256(t *testing.T) {
	testCases := []struct {
		idx        uint8
		wantOctet  uint8
		wantPfxLen uint8
	}{
		/*
			{
				idx:        0,  // invalid
				wantOctet:  0,  // invalid
				wantPfxLen: -1, // invalid
			},
		*/
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
		gotOctet, gotPfxLen := IdxToPfx256(tc.idx)
		if gotOctet != tc.wantOctet || gotPfxLen != tc.wantPfxLen {
			t.Errorf("IdxToPfx256(%d), want: (%d, %d), got: (%d, %d)", tc.idx, tc.wantOctet, tc.wantPfxLen, gotOctet, gotPfxLen)
		}
	}
}

func TestIdxToRange256(t *testing.T) {
	testCases := []struct {
		idx       uint8
		wantFirst uint8
		wantLast  uint8
	}{
		/*
			{
				idx:       0, // invalid
				wantFirst: 0,
				wantLast:  255,
			},
		*/
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
		gotFirst, gotLast := IdxToRange256(tc.idx)
		if gotFirst != tc.wantFirst || gotLast != tc.wantLast {
			t.Errorf("IdxToRange256(%d), want: (%d, %d), got: (%d, %d)",
				tc.idx, tc.wantFirst, tc.wantLast, gotFirst, gotLast)
		}
	}
}
