package sparse

import (
	"math/rand/v2"
	"slices"
	"testing"
)

func TestNewArray(t *testing.T) {
	a := NewArray[int]()

	if c := a.Count(); c != 0 {
		t.Errorf("Count, expected 0, got %d", c)
	}
}

func TestSparseArrayCount(t *testing.T) {
	a := NewArray[int]()

	for i := range 10_000 {
		a.InsertAt(uint(i), i)
		a.InsertAt(uint(i), i)
	}
	if c := a.Count(); c != 10_000 {
		t.Errorf("Count, expected 10_000, got %d", c)
	}

	for i := range 5_000 {
		a.DeleteAt(uint(i))
		a.DeleteAt(uint(i))
	}
	if c := a.Count(); c != 5_000 {
		t.Errorf("Count, expected 5_000, got %d", c)
	}
}

func TestSparseArrayGet(t *testing.T) {
	a := NewArray[int]()

	for i := range 10_000 {
		a.InsertAt(uint(i), i)
	}

	for range 100 {
		i := rand.IntN(10_000)
		v, ok := a.Get(uint(i))
		if !ok {
			t.Errorf("Get, expected true, got %v", ok)
		}
		if v != i {
			t.Errorf("Get, expected %d, got %d", i, v)
		}

		v = a.MustGet(uint(i))
		if v != i {
			t.Errorf("MustGet, expected %d, got %d", i, v)
		}
	}

	_, ok := a.Get(20_000)
	if ok {
		t.Errorf("Get, expected false, got %v", ok)
	}
}

func TestSparseArrayMustGetPanic(t *testing.T) {
	defer func() {
		r := recover()
		if r == nil {
			t.Errorf("MustGet, expected panic")
		}
	}()

	a := NewArray[int]()

	for i := 5; i <= 10; i++ {
		a.InsertAt(uint(i), i)
	}

	// must panic, runtime error: index out of range [-1]
	a.MustGet(0)
}

func TestSparseArrayUpdate(t *testing.T) {
	a := NewArray[int]()

	for i := range 10_000 {
		a.InsertAt(uint(i), i)
	}

	// mult all values * 2
	for i := 15_000; i >= 0; i-- {
		a.UpdateAt(uint(i), func(oldVal int, existsOld bool) int {
			newVal := i * 3
			if existsOld {
				newVal = oldVal * 2
			}
			return newVal
		})
	}

	for i := range 10_000 {
		v, _ := a.Get(uint(i))
		if v != 2*i {
			t.Errorf("UpdateAt, expected %d, got %d", 2*i, v)
		}
	}

	for i := 10_000; i <= 15_000; i++ {
		v, _ := a.Get(uint(i))
		if v != 3*i {
			t.Errorf("UpdateAt, expected %d, got %d", 3*i, v)
		}
	}
}

func TestSparseArrayAllSetBits(t *testing.T) {
	defer func() {
		r := recover()
		if r == nil {
			t.Errorf("buffer too small, expected panic")
		}
	}()

	const n = 32
	a := NewArray[int]()

	want := make([]uint, 0, n)

	for i := range n {
		a.InsertAt(uint(i), i)
		want = append(want, uint(i))
	}

	backingBuf := make([]uint, n)
	got := a.AllSetBits(backingBuf)

	if !slices.Equal(want, got) {
		t.Errorf("AllSetBits, want:\n%v\ngot:\n%v\n", want, got)
	}

	// must panic
	backingBuf = make([]uint, n/2)
	a.AllSetBits(backingBuf)
}
