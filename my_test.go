package bart

import (
	"runtime"
	"testing"
)

func BenchmarkFullTableInsertMy(b *testing.B) {
	var startMem, endMem runtime.MemStats

	var rt Table2[struct{}]

	runtime.GC()
	runtime.ReadMemStats(&startMem)
	b.ResetTimer()
	b.Run("Insert", func(b *testing.B) {
		for range b.N {
			for _, route := range routes {
				rt.Insert(route.CIDR, struct{}{})
			}
		}
		runtime.GC()
		runtime.ReadMemStats(&endMem)

		nodes, _ := rt.nodeAndLeafCount()

		b.ReportMetric(float64(endMem.HeapAlloc-startMem.HeapAlloc), "Bytes")
		b.ReportMetric(float64(rt.Size())/float64(nodes), "Prefix/Node")
	})
}
