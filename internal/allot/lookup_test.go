// Copyright (c) 2025 Karl Gaissmaier
// SPDX-License-Identifier: MIT

package allot

import (
	"slices"
	"testing"
)

func TestIdxToFringeRoutes(t *testing.T) {
	tests := []struct {
		idx  uint8
		want []uint8
	}{
		{
			idx:  0, // invalid
			want: []uint8{},
		},
		{
			idx:  63,
			want: []uint8{248, 249, 250, 251, 252, 253, 254, 255},
		},
		{
			idx:  127,
			want: []uint8{252, 253, 254, 255},
		},
		{
			idx:  128,
			want: []uint8{0, 1},
		},
		{
			idx:  199,
			want: []uint8{142, 143},
		},
		{
			idx:  255,
			want: []uint8{254, 255},
		},
	}

	for _, tc := range tests {
		fringeRoutes := IdxToFringeRoutes(tc.idx)
		got := fringeRoutes.Bits()
		if !slices.Equal(got, tc.want) {
			t.Errorf("IdxToFringeRoutes(%d), want: %v, got: %v", tc.idx, tc.want, got)
		}
	}
}

func TestIdxToPrefixRoutes(t *testing.T) {
	tests := []struct {
		idx  uint8
		want []uint8
	}{
		{
			idx:  0, // invalid
			want: []uint8{},
		},
		{
			idx:  41,
			want: []uint8{41, 82, 83, 164, 165, 166, 167},
		},
		{
			idx:  63,
			want: []uint8{63, 126, 127, 252, 253, 254, 255},
		},
		{
			idx:  127,
			want: []uint8{127, 254, 255},
		},
		{
			idx:  128,
			want: []uint8{128},
		},
		{
			idx:  199,
			want: []uint8{199},
		},
		{
			idx:  255,
			want: []uint8{255},
		},
	}

	for _, tc := range tests {
		fringeRoutes := IdxToPrefixRoutes(tc.idx)
		got := fringeRoutes.Bits()
		if !slices.Equal(got, tc.want) {
			t.Errorf("IdxToPrefixRoutes(%d), want: %v, got: %v", tc.idx, tc.want, got)
		}
	}
}
