package lpm

import (
	"testing"

	"github.com/gaissmai/bart/internal/bitset"
)

func genPath(i uint8) (path bitset.BitSet256) {
	for ; i > 0; i >>= 1 {
		path.Set(i)
	}
	return
}

func TestLookupTbl_Length(t *testing.T) {
	t.Parallel()
	if got := len(LookupTbl); got != 256 {
		t.Fatalf("LookupTbl length=%d, want 256", got)
	}
}

//nolint:gosec
func TestLookupTbl_PathInvariant(t *testing.T) {
	t.Parallel()
	for i := range 256 {
		want := genPath(uint8(i))
		if LookupTbl[i] != want {
			t.Fatalf("entry %d mismatch:\n got=%v\nwant=%v", i, LookupTbl[i], want)
		}
	}
}
