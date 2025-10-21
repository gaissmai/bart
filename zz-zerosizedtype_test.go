// Copyright (c) 2024 Karl Gaissmaier
// SPDX-License-Identifier: MIT

package bart

import (
	"testing"
)

func TestZeroSizedType_MustPanic(t *testing.T) {
	t.Parallel()

	t.Run("struct{}: Insert()", func(t *testing.T) {
		t.Parallel()

		defer func(name string) {
			if r := recover(); r == nil {
				t.Errorf("%s must panic", name)
			}
		}("struct{}: Insert()")

		fast := new(Fast[struct{}])
		fast.Insert(mpp("::1/128"), struct{}{})
	})

	t.Run("struct{}: InsertPersist()", func(t *testing.T) {
		t.Parallel()

		defer func(name string) {
			if r := recover(); r == nil {
				t.Errorf("%s must panic", name)
			}
		}("struct{}: InsertPersist()")

		fast := new(Fast[struct{}])
		fast.InsertPersist(mpp("::1/128"), struct{}{})
	})

	t.Run("[0]byte: Insert()", func(t *testing.T) {
		t.Parallel()

		defer func(name string) {
			if r := recover(); r == nil {
				t.Errorf("%s must panic", name)
			}
		}("[0]byte: Insert()")

		fast := new(Fast[[0]byte])
		fast.Insert(mpp("::1/128"), [0]byte{})
	})

	t.Run("[0]byte: InsertPersist()", func(t *testing.T) {
		t.Parallel()

		defer func(name string) {
			if r := recover(); r == nil {
				t.Errorf("%s must panic", name)
			}
		}("[0]byte: InsertPersist()")

		fast := new(Fast[[0]byte])
		fast.InsertPersist(mpp("::1/128"), [0]byte{})
	})
}
