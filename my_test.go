package bart

import "testing"

func BenchmarkMyLite(b *testing.B) {
	for b.Loop() {
		l := new(Lite)
		for _, pfx := range routes {
			l.Insert(pfx.CIDR)
		}
	}
	b.ReportMetric(float64(b.Elapsed().Nanoseconds()/int64(len(routes))), "ns/insert")
}

func BenchmarkMyTable(b *testing.B) {
	for b.Loop() {
		l := new(Table[any])
		for _, pfx := range routes {
			l.Insert(pfx.CIDR, nil)
		}
	}
	b.ReportMetric(float64(b.Elapsed().Nanoseconds()/int64(len(routes))), "ns/insert")
}
