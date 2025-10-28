// Copyright (c) 2024 Karl Gaissmaier
// SPDX-License-Identifier: MIT

package value

import (
	"testing"
)

func TestIsZeroSizedType(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		got  bool
		want bool
	}{
		{
			name: "struct{}",
			got:  IsZST[struct{}](),
			want: true,
		},
		{
			name: "[0]byte",
			got:  IsZST[[0]byte](),
			want: true,
		},
		{
			name: "int",
			got:  IsZST[int](),
			want: false,
		},
	}

	for _, tt := range tests {
		if tt.got != tt.want {
			t.Errorf("%s, want %v, got %v", tt.name, tt.want, tt.got)
		}
	}
}
