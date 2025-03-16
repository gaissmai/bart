package lpmbt

import (
	"slices"
	"testing"
)

func TestBackTrackingBitset(t *testing.T) {
	tests := []struct {
		idx  uint
		want []uint
	}{
		{
			idx:  0, // invalid
			want: []uint{},
		},
		{
			idx:  1,
			want: []uint{1}, // default route
		},
		{
			idx:  2,
			want: []uint{1, 2},
		},
		{
			idx:  3,
			want: []uint{1, 3},
		},
		{
			idx:  15,
			want: []uint{1, 3, 7, 15},
		},
		{
			idx:  16,
			want: []uint{1, 2, 4, 8, 16},
		},
		{
			idx:  509,
			want: []uint{1, 3, 7, 15, 31, 63, 127, 254},
		},
		{
			idx:  510,
			want: []uint{1, 3, 7, 15, 31, 63, 127, 255},
		},
		{
			idx:  511,
			want: []uint{1, 3, 7, 15, 31, 63, 127, 255},
		},
		{
			idx:  512,
			want: []uint{}, // overflow
		},
		{
			idx:  513,
			want: []uint{1}, // overflow
		},
	}

	for _, tc := range tests {
		got := BackTrackingBitset(tc.idx).All()
		if !slices.Equal(got, tc.want) {
			t.Errorf("BackTrackingBitset(%d), want: %v, got: %v", tc.idx, tc.want, got)
		}
	}
}
