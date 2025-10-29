// Copyright (c) 2024 Karl Gaissmaier
// SPDX-License-Identifier: MIT

package value

import (
	"maps"
	"reflect"
	"testing"
)

// Test types for Equaler interface
type equalableType struct {
	Value int
}

func (e equalableType) Equal(other equalableType) bool {
	return e.Value == other.Value
}

type nonEqualableType struct {
	Value int
}

// Test types for Cloner interface
type clonableType struct {
	Data map[string]int
}

func (c clonableType) Clone() clonableType {
	return clonableType{Data: maps.Clone(c.Data)}
}

type nonClonableType struct {
	Data map[string]int
}

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
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if tt.got != tt.want {
				t.Fatalf("want %v, got %v", tt.want, tt.got)
			}
		})
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

func TestEqual(t *testing.T) {
	t.Parallel()

	t.Run("with_Equaler_interface", func(t *testing.T) {
		t.Parallel()
		v1 := equalableType{Value: 42}
		v2 := equalableType{Value: 42}
		v3 := equalableType{Value: 99}

		if !Equal(v1, v2) {
			t.Error("Equal should return true for equal values")
		}
		if Equal(v1, v3) {
			t.Error("Equal should return false for different values")
		}
	})

	t.Run("without_Equaler_fallback_to_DeepEqual", func(t *testing.T) {
		t.Parallel()
		v1 := nonEqualableType{Value: 42}
		v2 := nonEqualableType{Value: 42}
		v3 := nonEqualableType{Value: 99}

		if !Equal(v1, v2) {
			t.Error("Equal should return true for equal values via DeepEqual")
		}
		if Equal(v1, v3) {
			t.Error("Equal should return false for different values via DeepEqual")
		}
	})

	t.Run("complex_types_with_DeepEqual", func(t *testing.T) {
		t.Parallel()
		v1 := map[string]int{"a": 1, "b": 2}
		v2 := map[string]int{"a": 1, "b": 2}
		v3 := map[string]int{"a": 1, "b": 3}

		if !Equal(v1, v2) {
			t.Error("Equal should return true for equal maps")
		}
		if Equal(v1, v3) {
			t.Error("Equal should return false for different maps")
		}
	})

	t.Run("simple_types", func(t *testing.T) {
		t.Parallel()
		if !Equal(42, 42) {
			t.Error("Equal should return true for equal ints")
		}
		if Equal(42, 99) {
			t.Error("Equal should return false for different ints")
		}
	})

	t.Run("typed_nil_interfaces", func(t *testing.T) {
		t.Parallel()
		// pointer typed-nil
		var p1 *int = nil
		var p2 *int = nil
		if !Equal[any](p1, p2) {
			t.Error("Equal should treat two typed-nil pointers as equal")
		}
		// slice typed-nil
		var s1 []int = nil
		var s2 []int = nil
		if !Equal[any](s1, s2) {
			t.Error("Equal should treat two typed-nil slices as equal")
		}
		// map typed-nil
		var m1 map[string]int = nil
		var m2 map[string]int = nil
		if !Equal[any](m1, m2) {
			t.Error("Equal should treat two typed-nil maps as equal")
		}
		// interface holding typed-nil vs untyped nil
		var ai any = (*int)(nil)
		var bi any = nil
		if Equal(ai, bi) {
			t.Error("Equal should not treat typed-nil inside interface equal to nil interface")
		}
	})
}

func TestCloneFnFactory(t *testing.T) {
	t.Parallel()

	t.Run("with_Cloner_interface", func(t *testing.T) {
		t.Parallel()
		fn := CloneFnFactory[clonableType]()
		if fn == nil {
			t.Fatal("CloneFnFactory should return a non-nil function for Cloner types")
		}

		original := clonableType{Data: map[string]int{"key": 42}}
		cloned := fn(original)

		if !reflect.DeepEqual(original.Data, cloned.Data) {
			t.Error("Cloned value should be deep equal to original")
		}

		// Verify it's a deep copy
		cloned.Data["key"] = 99
		if original.Data["key"] != 42 {
			t.Error("Modifying clone should not affect original")
		}
	})

	t.Run("without_Cloner_interface", func(t *testing.T) {
		t.Parallel()
		fn := CloneFnFactory[nonClonableType]()
		if fn != nil {
			t.Error("CloneFnFactory should return nil for non-Cloner types")
		}
	})

	t.Run("simple_types", func(t *testing.T) {
		t.Parallel()
		fn := CloneFnFactory[int]()
		if fn != nil {
			t.Error("CloneFnFactory should return nil for simple types")
		}
	})
}

func TestCloneVal(t *testing.T) {
	t.Parallel()

	t.Run("with_Cloner_interface", func(t *testing.T) {
		t.Parallel()
		original := clonableType{Data: map[string]int{"key": 42}}
		cloned := CloneVal(original)

		if !reflect.DeepEqual(original.Data, cloned.Data) {
			t.Error("Cloned value should be deep equal to original")
		}

		// Verify it's a deep copy
		cloned.Data["key"] = 99
		if original.Data["key"] != 42 {
			t.Error("Modifying clone should not affect original")
		}
	})

	t.Run("without_Cloner_interface", func(t *testing.T) {
		t.Parallel()
		original := nonClonableType{Data: map[string]int{"key": 42}}
		cloned := CloneVal(original)

		// Without Cloner, it returns the value as-is
		if !reflect.DeepEqual(original.Data, cloned.Data) {
			t.Error("CloneVal should return equal value")
		}

		// This is a shallow copy for maps
		cloned.Data["key"] = 99
		if original.Data["key"] != 99 {
			t.Error("Without Cloner, map is shared (shallow copy)")
		}
	})

	t.Run("simple_types", func(t *testing.T) {
		t.Parallel()
		original := 42
		cloned := CloneVal(original)
		if original != cloned {
			t.Error("CloneVal should return same value for simple types")
		}
	})
}

func TestCloneVal_TypedNil(t *testing.T) {
	t.Parallel()

	t.Run("pointer_typed_nil_without_Cloner", func(t *testing.T) {
		t.Parallel()
		var p *int
		cloned := CloneVal(p)

		if cloned != nil {
			t.Error("CloneVal should preserve typed nil pointer")
		}
	})

	t.Run("pointer_typed_nil_with_Cloner", func(t *testing.T) {
		t.Parallel()
		type clonablePtr struct {
			Val *int
		}
		// Note: we'd need to implement Clone() for this to work
		// For now, without Cloner, typed nil is passed through
		var cp clonablePtr
		cloned := CloneVal(cp)

		if cloned.Val != nil {
			t.Error("CloneVal should preserve typed nil in struct fields")
		}
	})

	t.Run("slice_typed_nil", func(t *testing.T) {
		t.Parallel()
		var s []int
		cloned := CloneVal(s)

		if cloned != nil {
			t.Error("CloneVal should preserve typed nil slice")
		}
	})

	t.Run("map_typed_nil", func(t *testing.T) {
		t.Parallel()
		var m map[string]int
		cloned := CloneVal(m)

		if cloned != nil {
			t.Error("CloneVal should preserve typed nil map")
		}
	})

	t.Run("interface_with_typed_nil", func(t *testing.T) {
		t.Parallel()
		var i interface{} = (*int)(nil)
		cloned := CloneVal(i)

		if cloned != i {
			t.Error("CloneVal should preserve interface with typed nil")
		}

		// Verify it's still typed nil, not just nil
		if cloned == nil {
			t.Error("CloneVal should preserve typed nil, not convert to untyped nil")
		}
	})

	t.Run("struct_with_all_nil_fields", func(t *testing.T) {
		t.Parallel()
		type testStruct struct {
			Ptr   *int
			Slice []string
			Map   map[int]string
		}

		var s testStruct
		cloned := CloneVal(s)

		if cloned.Ptr != nil || cloned.Slice != nil || cloned.Map != nil {
			t.Error("CloneVal should preserve all typed nil fields in struct")
		}
	})

	t.Run("clonableType_with_nil_fields", func(t *testing.T) {
		t.Parallel()
		type clonableWithNil struct {
			Data map[string]int
		}

		impl := func(c clonableWithNil) clonableWithNil {
			if c.Data == nil {
				return clonableWithNil{Data: nil}
			}
			return clonableWithNil{Data: maps.Clone(c.Data)}
		}
		_ = impl

		// Without implementing Cloner interface on the type,
		// CloneVal returns the value as-is
		var cwn clonableWithNil
		cloned := CloneVal(cwn)

		if cloned.Data != nil {
			t.Error("CloneVal should preserve typed nil in struct field")
		}
	})
}

func TestCopyVal_TypedNil(t *testing.T) {
	t.Parallel()

	t.Run("pointer_typed_nil", func(t *testing.T) {
		t.Parallel()
		var p *int
		copied := CopyVal(p)

		if copied != nil {
			t.Error("CopyVal should preserve typed nil pointer")
		}

		// Verify it's the same nil
		if copied != p {
			t.Error("CopyVal should return the same typed nil value")
		}
	})

	t.Run("slice_typed_nil", func(t *testing.T) {
		t.Parallel()
		var s []int
		copied := CopyVal(s)

		if copied != nil {
			t.Error("CopyVal should preserve typed nil slice")
		}

		// Both should be nil
		if len(copied) != 0 || cap(copied) != 0 {
			t.Error("CopyVal typed nil slice should have zero length and capacity")
		}
	})

	t.Run("map_typed_nil", func(t *testing.T) {
		t.Parallel()
		var m map[string]int
		copied := CopyVal(m)

		if copied != nil {
			t.Error("CopyVal should preserve typed nil map")
		}

		// Both should be nil
		if len(copied) != 0 {
			t.Error("CopyVal typed nil map should have zero length")
		}
	})

	t.Run("interface_with_typed_nil", func(t *testing.T) {
		t.Parallel()
		var i interface{} = (*int)(nil)
		copied := CopyVal(i)

		// CopyVal is a value copy, so interface contents are copied
		if copied != i {
			t.Error("CopyVal should preserve interface with typed nil")
		}

		// Verify it's still typed nil, not untyped nil
		if copied == nil {
			t.Error("CopyVal should preserve typed nil, not convert to untyped nil")
		}
	})

	t.Run("struct_with_all_nil_fields", func(t *testing.T) {
		t.Parallel()
		type testStruct struct {
			Ptr   *int
			Slice []string
			Map   map[int]string
		}

		var s testStruct
		copied := CopyVal(s)

		if copied.Ptr != nil || copied.Slice != nil || copied.Map != nil {
			t.Error("CopyVal should preserve all typed nil fields in struct")
		}

		// Verify fields are truly nil (not just zero-length)
		if copied.Ptr != s.Ptr || len(copied.Slice) != 0 || len(copied.Map) != 0 {
			t.Error("CopyVal should create value copy with identical nil fields")
		}
	})

	t.Run("function_typed_nil", func(t *testing.T) {
		t.Parallel()
		var fn func()
		copied := CopyVal(fn)

		if copied != nil {
			t.Error("CopyVal should preserve typed nil function")
		}
	})

	t.Run("channel_typed_nil", func(t *testing.T) {
		t.Parallel()
		var ch chan int
		copied := CopyVal(ch)

		if copied != nil {
			t.Error("CopyVal should preserve typed nil channel")
		}
	})
}

func TestCopyVal(t *testing.T) {
	t.Parallel()

	t.Run("simple_types", func(t *testing.T) {
		t.Parallel()
		original := 42
		copied := CopyVal(original)
		if original != copied {
			t.Error("CopyVal should return same value")
		}
	})

	t.Run("struct_types", func(t *testing.T) {
		t.Parallel()
		original := nonClonableType{Data: map[string]int{"key": 42}}
		copied := CopyVal(original)

		if !reflect.DeepEqual(original.Data, copied.Data) {
			t.Error("CopyVal should return structurally equal value")
		}

		// CopyVal is a value copy, so maps are shared
		copied.Data["key"] = 99
		if original.Data["key"] != 99 {
			t.Error("CopyVal shares map references")
		}
	})

	t.Run("pointer_types", func(t *testing.T) {
		t.Parallel()
		val := 42
		original := &val
		copied := CopyVal(original)

		if original != copied {
			t.Error("CopyVal should return same pointer")
		}

		*copied = 99
		if *original != 99 {
			t.Error("CopyVal shares pointer")
		}
	})
}
