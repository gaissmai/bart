package value

import (
	"slices"
	"testing"
)

// ============================================================================
// Helper to assert that a function panics.
// ============================================================================
func expectPanic(t *testing.T, fn func()) {
	t.Helper()
	defer func() {
		if r := recover(); r == nil {
			t.Errorf("expected code to panic, but it did not")
		}
	}()
	fn()
}

// ============================================================================
// Helper Types for Testing IsEmptyStruct
// ============================================================================

// NamedEmpty is a named type defined as struct{}.
// It occupies 0 bytes, but is distinct from the unnamed struct{}.
type NamedEmpty struct{}

type NonEmpty struct {
	Value int
}

func TestIsEmptyStruct(t *testing.T) {
	tests := []struct {
		name string
		run  func(t *testing.T)
	}{
		// --------------------------------------------------------------------
		// Positive Case (Expect true)
		// --------------------------------------------------------------------
		{
			name: "unnamed empty struct struct{} returns true",
			run: func(t *testing.T) {
				if !IsEmptyStruct[struct{}]() {
					t.Errorf("expected IsEmptyStruct[struct{}]() to be true")
				}
			},
		},

		// --------------------------------------------------------------------
		// Negative Cases (Expect false)
		// --------------------------------------------------------------------
		{
			name: "named empty struct returns false (distinct type from struct{})",
			run: func(t *testing.T) {
				if IsEmptyStruct[NamedEmpty]() {
					t.Errorf("expected IsEmptyStruct[NamedEmpty]() to be false")
				}
			},
		},
		{
			name: "primitive type int returns false",
			run: func(t *testing.T) {
				if IsEmptyStruct[int]() {
					t.Errorf("expected IsEmptyStruct[int]() to be false")
				}
			},
		},
		{
			name: "primitive type string returns false",
			run: func(t *testing.T) {
				if IsEmptyStruct[string]() {
					t.Errorf("expected IsEmptyStruct[string]() to be false")
				}
			},
		},
		{
			name: "non-empty struct returns false",
			run: func(t *testing.T) {
				if IsEmptyStruct[NonEmpty]() {
					t.Errorf("expected IsEmptyStruct[NonEmpty]() to be false")
				}
			},
		},
		{
			name: "zero-sized array [0]byte returns false (is ZST, but not struct{})",
			run: func(t *testing.T) {
				if IsEmptyStruct[[0]byte]() {
					t.Errorf("expected IsEmptyStruct[[0]byte]() to be false")
				}
			},
		},
		{
			name: "array of empty structs [1]struct{} returns false",
			run: func(t *testing.T) {
				if IsEmptyStruct[[1]struct{}]() {
					t.Errorf("expected IsEmptyStruct[[1]struct{}]() to be false")
				}
			},
		},
		{
			name: "pointer to empty struct *struct{} returns false",
			run: func(t *testing.T) {
				if IsEmptyStruct[*struct{}]() {
					t.Errorf("expected IsEmptyStruct[*struct{}]() to be false")
				}
			},
		},
		{
			name: "interface type any returns false",
			run: func(t *testing.T) {
				if IsEmptyStruct[any]() {
					t.Errorf("expected IsEmptyStruct[any]() to be false")
				}
			},
		},
		{
			name: "slice of empty structs []struct{} returns false",
			run: func(t *testing.T) {
				if IsEmptyStruct[[]struct{}]() {
					t.Errorf("expected IsEmptyStruct[[]struct{}]() to be false")
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, tt.run)
	}
}

// ============================================================================
// Helper Types for Testing
// ============================================================================

// CustomVal implements Equal(CustomVal) bool with a value receiver.
type CustomVal struct {
	ID int
}

func (c CustomVal) Equal(other CustomVal) bool {
	return c.ID == other.ID
}

// CustomPtr implements Equal(*CustomPtr) bool with a pointer receiver.
type CustomPtr struct {
	ID int
}

func (c *CustomPtr) Equal(other *CustomPtr) bool {
	if c == nil && other == nil {
		return true
	}
	if c == nil || other == nil {
		return false
	}
	return c.ID == other.ID
}

// UncomparableWithEqual contains a slice (uncomparable), but implements Equal.
type UncomparableWithEqual struct {
	Data []int
}

func (u UncomparableWithEqual) Equal(other UncomparableWithEqual) bool {
	return slices.Equal(u.Data, other.Data)
}

// PlainStruct is a standard comparable struct without an Equal method.
type PlainStruct struct {
	A int
	B string
}

// ============================================================================
// Test Suite
// ============================================================================

func TestEqual_ComparableAndCustom(t *testing.T) {
	t.Run("primitive types", func(t *testing.T) {
		if !Equal(42, 42) {
			t.Errorf("expected 42 == 42")
		}
		if Equal(42, 43) {
			t.Errorf("expected 42 != 43")
		}
		if !Equal("hello", "hello") {
			t.Errorf("expected 'hello' == 'hello'")
		}
		if Equal("hello", "world") {
			t.Errorf("expected 'hello' != 'world'")
		}
	})

	t.Run("plain comparable structs", func(t *testing.T) {
		s1 := PlainStruct{A: 1, B: "test"}
		s2 := PlainStruct{A: 1, B: "test"}
		s3 := PlainStruct{A: 2, B: "test"}

		if !Equal(s1, s2) {
			t.Errorf("expected structs to be equal")
		}
		if Equal(s1, s3) {
			t.Errorf("expected structs to be unequal")
		}
	})

	t.Run("empty struct struct{}", func(t *testing.T) {
		if !Equal(struct{}{}, struct{}{}) {
			t.Errorf("expected struct{}{} == struct{}{}")
		}
	})

	t.Run("custom Equal with value receiver", func(t *testing.T) {
		v1 := CustomVal{ID: 10}
		v2 := CustomVal{ID: 10}
		v3 := CustomVal{ID: 20}

		if !Equal(v1, v2) {
			t.Errorf("expected custom equal to return true")
		}
		if Equal(v1, v3) {
			t.Errorf("expected custom equal to return false")
		}
	})

	t.Run("custom Equal with pointer receiver", func(t *testing.T) {
		p1 := &CustomPtr{ID: 10}
		p2 := &CustomPtr{ID: 10}
		p3 := &CustomPtr{ID: 20}

		if !Equal(p1, p2) {
			t.Errorf("expected pointers to be equal via custom method")
		}
		if Equal(p1, p3) {
			t.Errorf("expected pointers to be unequal via custom method")
		}

		// nil receiver checks
		var nil1, nil2 *CustomPtr
		if !Equal(nil1, nil2) {
			t.Errorf("expected two nil pointers to be equal")
		}
		if Equal(p1, nil1) {
			t.Errorf("expected non-nil and nil pointer to be unequal")
		}
	})

	t.Run("uncomparable type WITH Equal method (must not panic)", func(t *testing.T) {
		u1 := UncomparableWithEqual{Data: []int{1, 2, 3}}
		u2 := UncomparableWithEqual{Data: []int{1, 2, 3}}
		u3 := UncomparableWithEqual{Data: []int{1, 2, 4}}

		if !Equal(u1, u2) {
			t.Errorf("expected u1 and u2 to be equal")
		}
		if Equal(u1, u3) {
			t.Errorf("expected u1 and u3 to be unequal")
		}
	})
}

func TestEqual_PanicsOnUncomparable(t *testing.T) {
	t.Run("slice without Equal method panics", func(t *testing.T) {
		expectPanic(t, func() {
			s1 := []int{1, 2, 3}
			s2 := []int{1, 2, 3}
			_ = Equal(s1, s2)
		})
	})

	t.Run("map without Equal method panics", func(t *testing.T) {
		expectPanic(t, func() {
			m1 := map[string]int{"a": 1}
			m2 := map[string]int{"a": 1}
			_ = Equal(m1, m2)
		})
	})
}

// ============================================================================
// Helper Types for Testing CloneFnFactory
// ============================================================================

// PointerCloner implements Clone() with a pointer receiver.
type PointerCloner struct {
	Value string
}

func (p *PointerCloner) Clone() *PointerCloner {
	if p == nil {
		return nil // Native nil-safety
	}
	return &PointerCloner{Value: p.Value}
}

// ValueCloner implements Clone() with a value receiver.
type ValueCloner struct {
	Value int
}

func (v ValueCloner) Clone() ValueCloner {
	return ValueCloner{Value: v.Value * 2} // Multiply by 2 to prove Clone() ran
}

// NonCloner is a struct that does NOT implement Clone().
type NonCloner struct {
	Value string
}

func TestCloneFnFactory(t *testing.T) {
	tests := []struct {
		name string
		// We use a closure for execution because Go generic type parameters
		// cannot be passed dynamically as struct fields.
		run func(t *testing.T)
	}{
		{
			name: "Pointer type implementing Clone() returns functional closure",
			run: func(t *testing.T) {
				fn := CloneFnFactory[*PointerCloner]()
				if fn == nil {
					t.Fatalf("expected a function, got nil")
				}

				orig := &PointerCloner{Value: "test"}
				cloned := fn(orig)

				if cloned == orig {
					t.Errorf("expected a deep copy, but got the exact same memory address")
				}
				if cloned.Value != orig.Value {
					t.Errorf("expected value %q, got %q", orig.Value, cloned.Value)
				}
			},
		},
		{
			name: "Closure handles typed nil pointers gracefully without panic",
			run: func(t *testing.T) {
				fn := CloneFnFactory[*PointerCloner]()

				var orig *PointerCloner = nil
				cloned := fn(orig)

				if cloned != nil {
					t.Errorf("expected nil result, got %v", cloned)
				}
			},
		},
		{
			name: "Value type implementing Clone() works correctly",
			run: func(t *testing.T) {
				fn := CloneFnFactory[ValueCloner]()
				if fn == nil {
					t.Fatalf("expected a function, got nil")
				}

				orig := ValueCloner{Value: 21}
				cloned := fn(orig)

				// ValueCloner multiplies by 2 in its Clone method to prove execution
				if cloned.Value != 42 {
					t.Errorf("expected cloned value to be 42, got %d", cloned.Value)
				}
			},
		},
		{
			name: "Type not implementing Clone() returns nil factory",
			run: func(t *testing.T) {
				// string does not have a Clone() method
				fn := CloneFnFactory[string]()
				if fn != nil {
					t.Errorf("expected nil function, got %T", fn)
				}

				// custom struct without Clone() method
				fnStruct := CloneFnFactory[NonCloner]()
				if fnStruct != nil {
					t.Errorf("expected nil function, got %T", fnStruct)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, tt.run)
	}
}
