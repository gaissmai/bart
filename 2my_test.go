package bart

import (
	"testing"
)

func TestMy(t *testing.T) {
	var rt Table[int]
	for i, route := range routes4 {
		rt.Insert(route.CIDR, i)
	}

	//probe := mpp("185.152.168.0/21")
	probe := mpp("0.0.0.0/0")
	inter := new(Table[int])
	inter.Insert(probe, 0)
	_ = rt.Overlaps(inter)

}

/*
func BenchmarkMy4(b *testing.B) {
	var rt Table[int]

	for i, route := range routes4 {
		rt.Insert(route.CIDR, i)
	}

	// probe := mpp("240.0.0.0/7")
	probe := mpp("185.152.168.0/21")
	inter := new(Table[int])
	inter.Insert(probe, 0)

	b.Run("Overlp", func(b *testing.B) {
		b.ResetTimer()
		for k := 0; k < b.N; k++ {
			boolSink = rt.Overlaps(inter)
		}
		if boolSink {
			b.ReportMetric(float64(1), probe.String())
		} else {
			b.ReportMetric(float64(-1), probe.String())
		}
	})

	b.Run("Prefix", func(b *testing.B) {
		b.ResetTimer()
		for k := 0; k < b.N; k++ {
			boolSink = rt.OverlapsPrefix(probe)
		}
		if boolSink {
			b.ReportMetric(float64(1), probe.String())
		} else {
			b.ReportMetric(float64(-1), probe.String())
		}
	})
}
*/

/*
func BenchmarkMy4(b *testing.B) {
	var rt Table[int]

	for i, route := range routes4 {
		rt.Insert(route.CIDR, i)
	}

	for {
		inter := new(Table[int])
		pfx := randomPrefix4()
		inter.Insert(pfx, 0)

		b.Run("Overlp", func(b *testing.B) {
			b.ResetTimer()
			for k := 0; k < b.N; k++ {
				boolSink = rt.Overlaps(inter)
			}
			if boolSink {
				b.ReportMetric(float64(1), pfx.String())
			} else {
				b.ReportMetric(float64(-1), pfx.String())
			}
		})

		b.Run("Prefix", func(b *testing.B) {
			b.ResetTimer()
			for k := 0; k < b.N; k++ {
				boolSink = rt.OverlapsPrefix(pfx)
			}
			if boolSink {
				b.ReportMetric(float64(1), pfx.String())
			} else {
				b.ReportMetric(float64(-1), pfx.String())
			}
		})
	}
}
*/

func BenchmarkMy(b *testing.B) {
	var rt Table[int]

	for i, route := range routes {
		rt.Insert(route.CIDR, i)
	}

	for {
		inter := new(Table[int])
		pfx := randomPrefix()
		inter.Insert(pfx, 0)

		b.Run("Overlp", func(b *testing.B) {
			b.ResetTimer()
			for k := 0; k < b.N; k++ {
				boolSink = rt.Overlaps(inter)
			}
			if boolSink {
				b.ReportMetric(float64(1), pfx.String())
			} else {
				b.ReportMetric(float64(-1), pfx.String())
			}
		})

		b.Run("Prefix", func(b *testing.B) {
			b.ResetTimer()
			for k := 0; k < b.N; k++ {
				boolSink = rt.OverlapsPrefix(pfx)
			}
			if boolSink {
				b.ReportMetric(float64(1), pfx.String())
			} else {
				b.ReportMetric(float64(-1), pfx.String())
			}
		})
	}
}
