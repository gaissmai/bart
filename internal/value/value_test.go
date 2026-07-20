package value

import (
	"reflect"
	"testing"
)

// ============================================================================
// Helper Types for Testing IsZST
// ============================================================================

type EmptyStruct struct{}

type EmptyStructWrapper struct {
	A EmptyStruct
	B [0]int
}

type NonEmptyStruct struct {
	Field int
}

func TestIsZST(t *testing.T) {
	tests := []struct {
		name string
		run  func(t *testing.T)
	}{
		// --------------------------------------------------------------------
		// Zero-Sized Types (Expect true)
		// --------------------------------------------------------------------
		{
			name: "empty struct struct{} is ZST",
			run: func(t *testing.T) {
				if !IsZST[struct{}]() {
					t.Errorf("expected IsZST[struct{}]() to be true")
				}
			},
		},
		{
			name: "named empty struct is ZST",
			run: func(t *testing.T) {
				if !IsZST[EmptyStruct]() {
					t.Errorf("expected IsZST[EmptyStruct]() to be true")
				}
			},
		},
		{
			name: "struct containing only ZST fields is ZST",
			run: func(t *testing.T) {
				if !IsZST[EmptyStructWrapper]() {
					t.Errorf("expected IsZST[EmptyStructWrapper]() to be true")
				}
			},
		},
		{
			name: "zero-length array [0]byte is ZST",
			run: func(t *testing.T) {
				if !IsZST[[0]byte]() {
					t.Errorf("expected IsZST[[0]byte]() to be true")
				}
			},
		},
		{
			name: "array of zero-sized elements [5]struct{} is ZST",
			run: func(t *testing.T) {
				if !IsZST[[5]struct{}]() {
					t.Errorf("expected IsZST[[5]struct{}]() to be true")
				}
			},
		},

		// --------------------------------------------------------------------
		// Non-Zero-Sized Types (Expect false)
		// --------------------------------------------------------------------
		{
			name: "primitive type int is non-ZST",
			run: func(t *testing.T) {
				if IsZST[int]() {
					t.Errorf("expected IsZST[int]() to be false")
				}
			},
		},
		{
			name: "struct with non-ZST fields is non-ZST",
			run: func(t *testing.T) {
				if IsZST[NonEmptyStruct]() {
					t.Errorf("expected IsZST[NonEmptyStruct]() to be false")
				}
			},
		},
		{
			name: "pointer to ZST *struct{} is non-ZST (occupies pointer size)",
			run: func(t *testing.T) {
				if IsZST[*struct{}]() {
					t.Errorf("expected IsZST[*struct{}]() to be false")
				}
			},
		},
		{
			name: "slice of ZST []struct{} is non-ZST (occupies slice header size)",
			run: func(t *testing.T) {
				if IsZST[[]struct{}]() {
					t.Errorf("expected IsZST[[]struct{}]() to be false")
				}
			},
		},
		{
			name: "interface type any is non-ZST (occupies interface header size)",
			run: func(t *testing.T) {
				if IsZST[any]() {
					t.Errorf("expected IsZST[any]() to be false")
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, tt.run)
	}
}

// ============================================================================
// Helper Types for Testing Equal
// ============================================================================

// ValueEqualer implements Equal(ValueEqualer) bool with a value receiver.
type ValueEqualer struct {
	ID int
}

func (v ValueEqualer) Equal(other ValueEqualer) bool {
	return v.ID == other.ID
}

// PointerEqualer implements Equal(*PointerEqualer) bool with a pointer receiver.
type PointerEqualer struct {
	ID int
}

func (p *PointerEqualer) Equal(other *PointerEqualer) bool {
	if p == nil || other == nil {
		return p == other
	}
	return p.ID == other.ID
}

// NonEqualer does NOT implement an Equal method.
type NonEqualer struct {
	Tags []string
}

func TestEqual(t *testing.T) {
	tests := []struct {
		name string
		run  func(t *testing.T)
	}{
		{
			name: "Value receiver implementing Equal returns true for matching values",
			run: func(t *testing.T) {
				v1 := ValueEqualer{ID: 42}
				v2 := ValueEqualer{ID: 42}

				if !Equal(v1, v2) {
					t.Errorf("expected Equal(%v, %v) to be true", v1, v2)
				}
			},
		},
		{
			name: "Value receiver implementing Equal returns false for differing values",
			run: func(t *testing.T) {
				v1 := ValueEqualer{ID: 42}
				v2 := ValueEqualer{ID: 100}

				if Equal(v1, v2) {
					t.Errorf("expected Equal(%v, %v) to be false", v1, v2)
				}
			},
		},
		{
			name: "Pointer receiver implementing Equal handles non-nil pointers",
			run: func(t *testing.T) {
				p1 := &PointerEqualer{ID: 1}
				p2 := &PointerEqualer{ID: 1}
				p3 := &PointerEqualer{ID: 2}

				if !Equal(p1, p2) {
					t.Errorf("expected Equal(%v, %v) to be true", p1, p2)
				}
				if Equal(p1, p3) {
					t.Errorf("expected Equal(%v, %v) to be false", p1, p3)
				}
			},
		},
		{
			name: "Pointer receiver implementing Equal handles nil pointers gracefully",
			run: func(t *testing.T) {
				var nil1 *PointerEqualer = nil
				var nil2 *PointerEqualer = nil
				p1 := &PointerEqualer{ID: 1}

				// Both nil
				if !Equal(nil1, nil2) {
					t.Errorf("expected Equal(nil, nil) to be true")
				}
				// One nil, one non-nil
				if Equal(nil1, p1) {
					t.Errorf("expected Equal(nil, non-nil) to be false")
				}
			},
		},
		{
			name: "Fallback to reflect.DeepEqual returns true for matching non-Equaler types",
			run: func(t *testing.T) {
				// Slices/Structs without Equal method
				n1 := NonEqualer{Tags: []string{"a", "b"}}
				n2 := NonEqualer{Tags: []string{"a", "b"}}

				if !Equal(n1, n2) {
					t.Errorf("expected Equal(%v, %v) to be true via DeepEqual", n1, n2)
				}
			},
		},
		{
			name: "Fallback to reflect.DeepEqual returns false for differing non-Equaler types",
			run: func(t *testing.T) {
				n1 := NonEqualer{Tags: []string{"a", "b"}}
				n2 := NonEqualer{Tags: []string{"a", "c"}}

				if Equal(n1, n2) {
					t.Errorf("expected Equal(%v, %v) to be false via DeepEqual", n1, n2)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, tt.run)
	}
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

// ============================================================================
// Table-Driven Tests for CopyVal
// ============================================================================

func TestCopyVal(t *testing.T) {
	type CustomStruct struct {
		ID   int
		Name string
	}

	tests := []struct {
		name string
		run  func(t *testing.T)
	}{
		{
			name: "copies primitive int value",
			run: func(t *testing.T) {
				orig := 42
				got := CopyVal(orig)

				if got != orig {
					t.Errorf("expected %d, got %d", orig, got)
				}
			},
		},
		{
			name: "copies string value",
			run: func(t *testing.T) {
				orig := "hello go"
				got := CopyVal(orig)

				if got != orig {
					t.Errorf("expected %q, got %q", orig, got)
				}
			},
		},
		{
			name: "copies struct by value",
			run: func(t *testing.T) {
				orig := CustomStruct{ID: 1, Name: "Test"}
				got := CopyVal(orig)

				if got != orig {
					t.Errorf("expected %+v, got %+v", orig, got)
				}
			},
		},
		{
			name: "copies pointer value (returns exact same address)",
			run: func(t *testing.T) {
				orig := &CustomStruct{ID: 1, Name: "Test"}
				got := CopyVal(orig)

				if got != orig {
					t.Errorf("expected pointer address %p, got %p", orig, got)
				}
			},
		},
		{
			name: "copies slice value (shallow copy of slice header)",
			run: func(t *testing.T) {
				orig := []int{1, 2, 3}
				got := CopyVal(orig)

				if !reflect.DeepEqual(got, orig) {
					t.Errorf("expected slice %v, got %v", orig, got)
				}
			},
		},
		{
			name: "handles nil pointer gracefully",
			run: func(t *testing.T) {
				var orig *CustomStruct = nil
				got := CopyVal(orig)

				if got != nil {
					t.Errorf("expected nil, got %v", got)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, tt.run)
	}
}
