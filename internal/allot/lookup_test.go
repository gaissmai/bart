// Copyright (c) 2025 Karl Gaissmaier
// SPDX-License-Identifier: MIT

package allot

import (
	"slices"
	"testing"
)

func TestIdxToHostRoutes(t *testing.T) {
	tests := []struct {
		idx  uint
		want []uint
	}{
		{
			idx:  0, // invalid
			want: []uint{},
		},
		{
			idx:  63,
			want: []uint{248, 249, 250, 251, 252, 253, 254, 255},
		},
		{
			idx:  127,
			want: []uint{252, 253, 254, 255},
		},
		{
			idx:  128,
			want: []uint{0, 1},
		},
		{
			idx:  199,
			want: []uint{142, 143},
		},
		{
			idx:  255,
			want: []uint{254, 255},
		},
		{
			idx:  256, // uint8 overflow, no panic by intention!
			want: []uint{},
		},
	}

	for _, tc := range tests {
		got := IdxToHostRoutes(tc.idx).All()
		if !slices.Equal(got, tc.want) {
			t.Errorf("IdxToHostRoutes(%d), want: %v, got: %v", tc.idx, tc.want, got)
		}
	}
}

func TestIdxToPrefixRoutes(t *testing.T) {
	tests := []struct {
		idx  uint
		want []uint
	}{
		{
			idx:  0, // invalid
			want: []uint{},
		},
		{
			idx:  41,
			want: []uint{41, 82, 83, 164, 165, 166, 167},
		},
		{
			idx:  63,
			want: []uint{63, 126, 127, 252, 253, 254, 255},
		},
		{
			idx:  127,
			want: []uint{127, 254, 255},
		},
		{
			idx:  128,
			want: []uint{128},
		},
		{
			idx:  199,
			want: []uint{199},
		},
		{
			idx:  255,
			want: []uint{255},
		},
		{
			idx:  256, // uint8 overflow, no panic by intention!
			want: []uint{},
		},
	}

	for _, tc := range tests {
		got := IdxToPrefixRoutes(tc.idx).All()
		if !slices.Equal(got, tc.want) {
			t.Errorf("IdxToPrefixRoutes(%d), want: %v, got: %v", tc.idx, tc.want, got)
		}
	}
}
