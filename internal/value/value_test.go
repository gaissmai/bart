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
