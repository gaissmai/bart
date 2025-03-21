package bart

import (
	"testing"
)

func BenchmarkMy(b *testing.B) {
	b.ResetTimer()
	for range b.N {
		tbl := new(Lite)
		for _, pfx := range routes {
			tbl.Insert(pfx.CIDR)
		}
	}
}
