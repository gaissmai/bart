//go:build go1.23

// Copyright (c) 2024 Karl Gaissmaier
// SPDX-License-Identifier: MIT

package bitset

import (
	"iter"
	"math/bits"
)

// All iterates over all the set bits.
func (b BitSet) All() iter.Seq[uint] {
	return func(yield func(u uint) bool) {
		for idx, word := range b {
			for word != 0 {
				u := uint(idx<<log2WordSize + bits.TrailingZeros64(word))

				if !yield(u) {
					return
				}

				// clear the rightmost set bit
				word &= word - 1
			}
		}
	}
}
