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

func TestPanicOnZST(t *testing.T) {
	t.Parallel()

	t.Run("struct{}", func(t *testing.T) {
		t.Parallel()

		defer func() {
			if r := recover(); r == nil {
				t.Error("struct{} must panic")
			}
		}()

		PanicOnZST[struct{}]()
	})

	t.Run("[0]byte", func(t *testing.T) {
		t.Parallel()

		defer func() {
			if r := recover(); r == nil {
				t.Error("[0]byte must panic")
			}
		}()

		PanicOnZST[[0]byte]()
	})

	t.Run("int", func(t *testing.T) {
		t.Parallel()

		defer func() {
			if r := recover(); r != nil {
				t.Error("int must not panic")
			}
		}()

		PanicOnZST[int]()
	})
}
