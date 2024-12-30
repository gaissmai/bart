package bart

import (
	"runtime"
	"testing"
)

// count the nodes and leaves
func (t *Table2[V]) nodeAndLeafCount() (int, int) {
	n4, l4 := t.root4.nodeAndLeafCount()
	n6, l6 := t.root6.nodeAndLeafCount()
	return n4 + n6, l4 + l6
}

// nodes, count the nodes
func (t *Table2[V]) nodes() int {
	n4, _ := t.root4.nodeAndLeafCount()
	n6, _ := t.root6.nodeAndLeafCount()
	return n4 + n6
}

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
