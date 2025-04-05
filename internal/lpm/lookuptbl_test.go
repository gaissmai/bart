// Copyright (c) 2025 Karl Gaissmaier
// SPDX-License-Identifier: MIT

package lpm

import (
	"slices"
	"testing"
)

func TestBackTrackingBitset(t *testing.T) {
	tests := []struct {
		idx  uint
		want []uint8
	}{
		{
			idx:  0, // invalid
			want: []uint8{},
		},
		{
			idx:  1,
			want: []uint8{1}, // default route
		},
		{
			idx:  2,
			want: []uint8{1, 2},
		},
		{
			idx:  3,
			want: []uint8{1, 3},
		},
		{
			idx:  15,
			want: []uint8{1, 3, 7, 15},
		},
		{
			idx:  16,
			want: []uint8{1, 2, 4, 8, 16},
		},
		{
			idx:  509,
			want: []uint8{1, 3, 7, 15, 31, 63, 127, 254},
		},
		{
			idx:  510,
			want: []uint8{1, 3, 7, 15, 31, 63, 127, 255},
		},
		{
			idx:  511,
			want: []uint8{1, 3, 7, 15, 31, 63, 127, 255},
		},
		{
			idx:  512,
			want: []uint8{}, // overflow
		},
		{
			idx:  513,
			want: []uint8{1}, // overflow
		},
	}

	for _, tc := range tests {
		got := BackTrackingBitset(tc.idx).All()
		if !slices.Equal(got, tc.want) {
			t.Errorf("BackTrackingBitset(%d), want: %v, got: %v", tc.idx, tc.want, got)
		}
	}
}
