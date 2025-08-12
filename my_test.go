package bart

import (
	"fmt"
	"net/netip"
	"testing"
	"unsafe"
)

func TestFoo(t *testing.T) {
	t.Errorf("%d", unsafe.Sizeof(netip.Prefix{}))
}

func TestPrefixContains(t *testing.T) {
	tests := []struct {
		name   string
		prefix netip.Prefix
		ip     netip.Addr
		want   bool
	}{
		{
			name:   "IPv4-Adresse innerhalb des Prefix",
			prefix: mpp("192.168.0.0/24"),
			ip:     mpa("192.168.0.42"),
			want:   true,
		},
		{
			name:   "IPv4-Adresse außerhalb des Prefix",
			prefix: mpp("192.168.0.0/24"),
			ip:     mpa("192.168.1.5"),
			want:   false,
		},
		{
			name:   "IPv6-Adresse innerhalb des Prefix",
			prefix: mpp("2001:db8::/32"),
			ip:     mpa("2001:db8::1"),
			want:   true,
		},
		{
			name:   "IPv6-Adresse außerhalb des Prefix",
			prefix: mpp("2001:db8::/32"),
			ip:     mpa("2001:dead::1"),
			want:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			myPrefix := newPrefixBounds(tt.prefix)
			got := myPrefix.contains(tt.ip.AsSlice())
			if got != tt.want {
				t.Errorf("Prefix.Contains(%q) = %v, erwartet %v", tt.ip, got, tt.want)
			}
		})
	}
}

func BenchmarkPrefixContains(b *testing.B) {
	tests := []struct {
		name   string
		prefix netip.Prefix
		ip     netip.Addr
	}{
		{
			name:   "IPv4",
			prefix: netip.MustParsePrefix("192.168.1.0/24"),
			ip:     netip.MustParseAddr("192.168.1.100"),
		},
		{
			name:   "IPv6",
			prefix: netip.MustParsePrefix("2001:db8::/32"),
			ip:     netip.MustParseAddr("2001:db8::abcd"),
		},
	}

	for _, tt := range tests {
		b.Run(tt.name, func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				_ = tt.prefix.Contains(tt.ip)
			}
		})
	}
}

func BenchmarkPrefixContains2(b *testing.B) {
	tests := []struct {
		name   string
		prefix netip.Prefix
		ip     netip.Addr
	}{
		{
			name:   "IPv4",
			prefix: netip.MustParsePrefix("192.168.1.0/30"),
			ip:     netip.MustParseAddr("192.168.1.100"),
		},
		{
			name:   "IPv6",
			prefix: netip.MustParsePrefix("2001:db8::/126"),
			ip:     netip.MustParseAddr("2001:db8::4"),
		},
	}

	for _, tt := range tests {
		b.Run(tt.name, func(b *testing.B) {
			pfx := newPrefixBounds(tt.prefix)
			octets := tt.ip.AsSlice()
			for i := 0; i < b.N; i++ {
				_ = pfx.contains(octets)
			}
		})
	}
}

func TestMyWorst(t *testing.T) {
	tbl := new(Table[string])
	for _, p := range worstCasePfxsIP4 {
		tbl.Insert(p, p.String())
		fmt.Println(tbl.dumpString())
	}

	_ = tbl.Contains(worstCaseProbeIP4)
}

func TestMy(t *testing.T) {
	d := new(Dart[any])

	d.Insert(mpp("1.2.3.4/32"), nil)
	d.Insert(mpp("1.2.3.5/32"), nil)

	fmt.Printf("stats v4: %#v\n", d.root4.nodeStatsRec())
	fmt.Println(d.dumpString())

	b := new(Table[any])

	b.Insert(mpp("1.2.3.4/32"), nil)
	b.Insert(mpp("1.2.3.5/32"), nil)

	fmt.Printf("stats v4: %#v\n", d.root4.nodeStatsRec())
	fmt.Println(b.dumpString())
}

func BenchmarkDartFullMatch4(b *testing.B) {
	rt := new(Dart[struct{}])

	for _, route := range routes {
		rt.Insert(route.CIDR, struct{}{})
	}

	b.Log(matchIP4)
	b.Log(matchPfx4)

	b.Run("Contains", func(b *testing.B) {
		b.ResetTimer()
		for range b.N {
			boolSink = rt.Contains(matchIP4)
		}
	})

	b.Run("Lookup", func(b *testing.B) {
		b.ResetTimer()
		for range b.N {
			_, boolSink = rt.Lookup(matchIP4)
		}
	})
}

func BenchmarkDartFullMatch6(b *testing.B) {
	rt := new(Dart[struct{}])

	for _, route := range routes {
		rt.Insert(route.CIDR, struct{}{})
	}

	b.Log(matchIP6)
	b.Log(matchPfx6)

	b.Run("Contains", func(b *testing.B) {
		b.ResetTimer()
		for range b.N {
			boolSink = rt.Contains(matchIP6)
		}
	})

	b.Run("Lookup", func(b *testing.B) {
		b.ResetTimer()
		for range b.N {
			_, boolSink = rt.Lookup(matchIP6)
		}
	})
}

func BenchmarkDartFullMiss4(b *testing.B) {
	rt := new(Dart[struct{}])

	for _, route := range routes {
		rt.Insert(route.CIDR, struct{}{})
	}

	b.Log(missIP4)
	b.Log(missPfx4)

	b.Run("Contains", func(b *testing.B) {
		b.ResetTimer()
		for range b.N {
			boolSink = rt.Contains(missIP4)
		}
	})

	b.Run("Lookup", func(b *testing.B) {
		b.ResetTimer()
		for range b.N {
			_, boolSink = rt.Lookup(missIP4)
		}
	})
}

func BenchmarkDartFullMiss6(b *testing.B) {
	rt := new(Dart[struct{}])

	for _, route := range routes {
		rt.Insert(route.CIDR, struct{}{})
	}

	b.Log(missIP6)
	b.Log(missPfx6)

	b.Run("Contains", func(b *testing.B) {
		b.ResetTimer()
		for range b.N {
			boolSink = rt.Contains(missIP6)
		}
	})

	b.Run("Lookup", func(b *testing.B) {
		b.ResetTimer()
		for range b.N {
			_, boolSink = rt.Lookup(missIP6)
		}
	})
}
