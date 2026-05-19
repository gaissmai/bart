// Copyright (c) 2025 Karl Gaissmaier
// SPDX-License-Identifier: MIT

package bart

import (
	"fmt"
	"math/rand/v2"
	"net/netip"
	"testing"

	"github.com/gaissmai/bart/internal/tests/random"
)

func BenchmarkFullFastMatch4(b *testing.B) {
	fast := new(Fast[bool])
	for _, pfx := range tier1.routes4() {
		fast.Insert(pfx, true)
	}

	matchIP4 := tier1.matchIP4()
	matchPfx4 := tier1.matchPfx4()

	b.Run("Contains", func(b *testing.B) {
		for b.Loop() {
			fast.Contains(matchIP4)
		}
	})

	b.Run("Lookup", func(b *testing.B) {
		for b.Loop() {
			fast.Lookup(matchIP4)
		}
	})

	b.Run("LookupPrefix", func(b *testing.B) {
		for b.Loop() {
			fast.LookupPrefix(matchPfx4)
		}
	})

	b.Run("LookupPfxLPM", func(b *testing.B) {
		for b.Loop() {
			fast.LookupPrefixLPM(matchPfx4)
		}
	})
}

func BenchmarkFullFastMatch6(b *testing.B) {
	fast := new(Fast[bool])
	for _, pfx := range tier1.routes6() {
		fast.Insert(pfx, true)
	}

	matchIP6 := tier1.matchIP6()
	matchPfx6 := tier1.matchPfx6()

	b.Run("Contains", func(b *testing.B) {
		for b.Loop() {
			fast.Contains(matchIP6)
		}
	})

	b.Run("Lookup", func(b *testing.B) {
		for b.Loop() {
			fast.Lookup(matchIP6)
		}
	})

	b.Run("LookupPrefix", func(b *testing.B) {
		for b.Loop() {
			fast.LookupPrefix(matchPfx6)
		}
	})

	b.Run("LookupPfxLPM", func(b *testing.B) {
		for b.Loop() {
			fast.LookupPrefixLPM(matchPfx6)
		}
	})
}

func BenchmarkFullFastMiss4(b *testing.B) {
	fast := new(Fast[bool])
	for _, pfx := range tier1.routes4() {
		fast.Insert(pfx, true)
	}

	missIP4 := tier1.missIP4()
	missPfx4 := tier1.missPfx4()

	b.Run("Contains", func(b *testing.B) {
		for b.Loop() {
			fast.Contains(missIP4)
		}
	})

	b.Run("Lookup", func(b *testing.B) {
		for b.Loop() {
			fast.Lookup(missIP4)
		}
	})

	b.Run("LookupPrefix", func(b *testing.B) {
		for b.Loop() {
			fast.LookupPrefix(missPfx4)
		}
	})

	b.Run("LookupPfxLPM", func(b *testing.B) {
		for b.Loop() {
			fast.LookupPrefixLPM(missPfx4)
		}
	})
}

func BenchmarkFullFastMiss6(b *testing.B) {
	fast := new(Fast[bool])
	for _, pfx := range tier1.routes6() {
		fast.Insert(pfx, true)
	}

	missIP6 := tier1.missIP6()
	missPfx6 := tier1.missPfx6()

	b.Run("Contains", func(b *testing.B) {
		for b.Loop() {
			fast.Contains(missIP6)
		}
	})

	b.Run("Lookup", func(b *testing.B) {
		for b.Loop() {
			fast.Lookup(missIP6)
		}
	})

	b.Run("LookupPrefix", func(b *testing.B) {
		for b.Loop() {
			fast.LookupPrefix(missPfx6)
		}
	})

	b.Run("LookupPfxLPM", func(b *testing.B) {
		for b.Loop() {
			fast.LookupPrefixLPM(missPfx6)
		}
	})
}

func BenchmarkFullBartMatch4(b *testing.B) {
	bart := new(Table[bool])
	for _, pfx := range tier1.routes4() {
		bart.Insert(pfx, true)
	}

	matchIP4 := tier1.matchIP4()
	matchPfx4 := tier1.matchPfx4()

	b.Run("Contains", func(b *testing.B) {
		for b.Loop() {
			bart.Contains(matchIP4)
		}
	})

	b.Run("Lookup", func(b *testing.B) {
		for b.Loop() {
			bart.Lookup(matchIP4)
		}
	})

	b.Run("LookupPrefix", func(b *testing.B) {
		for b.Loop() {
			bart.LookupPrefix(matchPfx4)
		}
	})

	b.Run("LookupPfxLPM", func(b *testing.B) {
		for b.Loop() {
			bart.LookupPrefixLPM(matchPfx4)
		}
	})
}

func BenchmarkFullBartMatch6(b *testing.B) {
	bart := new(Table[bool])
	for _, pfx := range tier1.routes6() {
		bart.Insert(pfx, true)
	}

	matchIP6 := tier1.matchIP6()
	matchPfx6 := tier1.matchPfx6()

	b.Run("Contains", func(b *testing.B) {
		for b.Loop() {
			bart.Contains(matchIP6)
		}
	})

	b.Run("Lookup", func(b *testing.B) {
		for b.Loop() {
			bart.Lookup(matchIP6)
		}
	})

	b.Run("LookupPrefix", func(b *testing.B) {
		for b.Loop() {
			bart.LookupPrefix(matchPfx6)
		}
	})

	b.Run("LookupPfxLPM", func(b *testing.B) {
		for b.Loop() {
			bart.LookupPrefixLPM(matchPfx6)
		}
	})
}

func BenchmarkFullBartMiss4(b *testing.B) {
	bart := new(Table[bool])
	for _, pfx := range tier1.routes4() {
		bart.Insert(pfx, true)
	}

	missIP4 := tier1.missIP4()
	missPfx4 := tier1.missPfx4()

	b.Run("Contains", func(b *testing.B) {
		for b.Loop() {
			bart.Contains(missIP4)
		}
	})

	b.Run("Lookup", func(b *testing.B) {
		for b.Loop() {
			bart.Lookup(missIP4)
		}
	})

	b.Run("LookupPrefix", func(b *testing.B) {
		for b.Loop() {
			bart.LookupPrefix(missPfx4)
		}
	})

	b.Run("LookupPfxLPM", func(b *testing.B) {
		for b.Loop() {
			bart.LookupPrefixLPM(missPfx4)
		}
	})
}

func BenchmarkFullBartMiss6(b *testing.B) {
	bart := new(Table[bool])
	for _, pfx := range tier1.routes6() {
		bart.Insert(pfx, true)
	}

	missIP6 := tier1.missIP6()
	missPfx6 := tier1.missPfx6()

	b.Run("Contains", func(b *testing.B) {
		for b.Loop() {
			bart.Contains(missIP6)
		}
	})

	b.Run("Lookup", func(b *testing.B) {
		for b.Loop() {
			bart.Lookup(missIP6)
		}
	})

	b.Run("LookupPrefix", func(b *testing.B) {
		for b.Loop() {
			bart.LookupPrefix(missPfx6)
		}
	})

	b.Run("LookupPfxLPM", func(b *testing.B) {
		for b.Loop() {
			bart.LookupPrefixLPM(missPfx6)
		}
	})
}

func BenchmarkTableOverlaps4(b *testing.B) {
	lt := new(Table[any])

	for _, route := range tier1.routes4() {
		lt.Insert(route, nil)
	}

	for i := 1; i <= 1<<20; i *= 2 {
		prng := rand.New(rand.NewPCG(42, 42))
		lt2 := new(Table[any])
		for _, pfx := range random.RealWorldPrefixes4(prng, i) {
			lt2.Insert(pfx, nil)
		}

		b.Run(fmt.Sprintf("With_%4d", i), func(b *testing.B) {
			for b.Loop() {
				lt.Overlaps4(lt2)
			}
		})
	}
}

func BenchmarkTableOverlaps6(b *testing.B) {
	lt := new(Table[any])

	for _, route := range tier1.routes6() {
		lt.Insert(route, nil)
	}

	for i := 1; i <= 1<<20; i *= 2 {
		prng := rand.New(rand.NewPCG(42, 42))
		lt2 := new(Table[any])
		for _, pfx := range random.RealWorldPrefixes6(prng, i) {
			lt2.Insert(pfx, nil)
		}

		b.Run(fmt.Sprintf("With_%4d", i), func(b *testing.B) {
			for b.Loop() {
				lt.Overlaps6(lt2)
			}
		})
	}
}

func BenchmarkFastOverlaps4(b *testing.B) {
	lt := new(Fast[any])

	for _, route := range tier1.routes4() {
		lt.Insert(route, nil)
	}

	for i := 1; i <= 1<<20; i *= 2 {
		prng := rand.New(rand.NewPCG(42, 42))
		lt2 := new(Fast[any])
		for _, pfx := range random.RealWorldPrefixes4(prng, i) {
			lt2.Insert(pfx, nil)
		}

		b.Run(fmt.Sprintf("With_%4d", i), func(b *testing.B) {
			for b.Loop() {
				lt.Overlaps4(lt2)
			}
		})
	}
}

func BenchmarkFastOverlaps6(b *testing.B) {
	lt := new(Fast[any])

	for _, route := range tier1.routes6() {
		lt.Insert(route, nil)
	}

	for i := 1; i <= 1<<20; i *= 2 {
		prng := rand.New(rand.NewPCG(42, 42))
		lt2 := new(Fast[any])
		for _, pfx := range random.RealWorldPrefixes6(prng, i) {
			lt2.Insert(pfx, nil)
		}

		b.Run(fmt.Sprintf("With_%4d", i), func(b *testing.B) {
			for b.Loop() {
				lt.Overlaps6(lt2)
			}
		})
	}
}

// worstcase benchmarks

var (
	worstCaseProbeIP4  = mpa("255.255.255.255")
	worstCaseProbePfx4 = mpp("255.255.255.255/32")

	ipv4DefaultRoute = mpp("0.0.0.0/0")
	worstCasePfxsIP4 = []netip.Prefix{
		mpp("0.0.0.0/1"),
		mpp("254.0.0.0/8"),
		mpp("255.0.0.0/9"),
		mpp("255.254.0.0/16"),
		mpp("255.255.0.0/17"),
		mpp("255.255.254.0/24"),
		mpp("255.255.255.0/25"),
		mpp("255.255.255.255/32"), // matching prefix
	}

	worstCaseProbeIP6  = mpa("ffff:ffff:ffff:ffff:ffff:ffff:ffff:ffff")
	worstCaseProbePfx6 = mpp("ffff:ffff:ffff:ffff:ffff:ffff:ffff:ffff/128")

	ipv6DefaultRoute = mpp("::/0")
	worstCasePfxsIP6 = []netip.Prefix{
		mpp("::/1"),
		mpp("fe00::/8"),
		mpp("ff00::/9"),
		mpp("fffe::/16"),
		mpp("ffff::/17"),
		mpp("ffff:fe00::/24"),
		mpp("ffff:ff00::/25"),
		mpp("ffff:fffe::/32"),
		mpp("ffff:ffff::/33"),
		mpp("ffff:ffff:fe00::/40"),
		mpp("ffff:ffff:ff00::/41"),
		mpp("ffff:ffff:fffe::/48"),
		mpp("ffff:ffff:ffff::/49"),
		mpp("ffff:ffff:ffff:fe00::/56"),
		mpp("ffff:ffff:ffff:ff00::/57"),
		mpp("ffff:ffff:ffff:fffe::/64"),
		mpp("ffff:ffff:ffff:ffff::/65"),
		mpp("ffff:ffff:ffff:ffff:fe00::/72"),
		mpp("ffff:ffff:ffff:ffff:ff00::/73"),
		mpp("ffff:ffff:ffff:ffff:fffe::/80"),
		mpp("ffff:ffff:ffff:ffff:ffff::/81"),
		mpp("ffff:ffff:ffff:ffff:ffff:fe00::/88"),
		mpp("ffff:ffff:ffff:ffff:ffff:ff00::/89"),
		mpp("ffff:ffff:ffff:ffff:ffff:fffe::/96"),
		mpp("ffff:ffff:ffff:ffff:ffff:ffff::/97"),
		mpp("ffff:ffff:ffff:ffff:ffff:ffff:fe00::/104"),
		mpp("ffff:ffff:ffff:ffff:ffff:ffff:ff00::/105"),
		mpp("ffff:ffff:ffff:ffff:ffff:ffff:fffe::/112"),
		mpp("ffff:ffff:ffff:ffff:ffff:ffff:ffff::/113"),
		mpp("ffff:ffff:ffff:ffff:ffff:ffff:ffff:fe00/120"),
		mpp("ffff:ffff:ffff:ffff:ffff:ffff:ffff:ff00/121"),
		mpp("ffff:ffff:ffff:ffff:ffff:ffff:ffff:fffe/128"),
		mpp("ffff:ffff:ffff:ffff:ffff:ffff:ffff:ffff/128"),
	}
)

func BenchmarkBartWorstCaseMatch4(b *testing.B) {
	b.Run("Contains", func(b *testing.B) {
		tbl := new(Table[any])
		for _, p := range worstCasePfxsIP4 {
			tbl.Insert(p, nil)
		}

		for b.Loop() {
			tbl.Contains(worstCaseProbeIP4)
		}
	})

	b.Run("Lookup", func(b *testing.B) {
		tbl := new(Table[any])
		for _, p := range worstCasePfxsIP4 {
			tbl.Insert(p, nil)
		}
		tbl.Insert(ipv4DefaultRoute, ipv4DefaultRoute.String())
		tbl.Delete(worstCaseProbePfx4)

		for b.Loop() {
			tbl.Lookup(worstCaseProbeIP4)
		}
	})

	b.Run("LookupPrefix", func(b *testing.B) {
		tbl := new(Table[any])
		for _, p := range worstCasePfxsIP4 {
			tbl.Insert(p, nil)
		}
		tbl.Insert(ipv4DefaultRoute, ipv4DefaultRoute.String())
		tbl.Delete(worstCaseProbePfx4)

		for b.Loop() {
			tbl.LookupPrefix(worstCaseProbePfx4)
		}
	})

	b.Run("LookupPrefixLPM", func(b *testing.B) {
		tbl := new(Table[any])
		for _, p := range worstCasePfxsIP4 {
			tbl.Insert(p, nil)
		}
		tbl.Insert(ipv4DefaultRoute, ipv4DefaultRoute.String())
		tbl.Delete(worstCaseProbePfx4)

		for b.Loop() {
			tbl.LookupPrefixLPM(worstCaseProbePfx4)
		}
	})
}

func BenchmarkFastWorstCaseMatch4(b *testing.B) {
	b.Run("Contains", func(b *testing.B) {
		tbl := new(Fast[any])
		for _, p := range worstCasePfxsIP4 {
			tbl.Insert(p, nil)
		}

		for b.Loop() {
			tbl.Contains(worstCaseProbeIP4)
		}
	})

	b.Run("Lookup", func(b *testing.B) {
		tbl := new(Fast[any])
		for _, p := range worstCasePfxsIP4 {
			tbl.Insert(p, nil)
		}
		tbl.Insert(ipv4DefaultRoute, ipv4DefaultRoute.String())
		tbl.Delete(worstCaseProbePfx4)

		for b.Loop() {
			tbl.Lookup(worstCaseProbeIP4)
		}
	})

	b.Run("LookupPrefix", func(b *testing.B) {
		tbl := new(Fast[any])
		for _, p := range worstCasePfxsIP4 {
			tbl.Insert(p, nil)
		}
		tbl.Insert(ipv4DefaultRoute, ipv4DefaultRoute.String())
		tbl.Delete(worstCaseProbePfx4)

		for b.Loop() {
			tbl.LookupPrefix(worstCaseProbePfx4)
		}
	})

	b.Run("LookupPrefixLPM", func(b *testing.B) {
		tbl := new(Fast[any])
		for _, p := range worstCasePfxsIP4 {
			tbl.Insert(p, nil)
		}
		tbl.Insert(ipv4DefaultRoute, ipv4DefaultRoute.String())
		tbl.Delete(worstCaseProbePfx4)

		for b.Loop() {
			tbl.LookupPrefixLPM(worstCaseProbePfx4)
		}
	})
}

func BenchmarkBartWorstCaseMiss4(b *testing.B) {
	b.Run("Contains", func(b *testing.B) {
		tbl := new(Table[any])
		for _, p := range worstCasePfxsIP4 {
			tbl.Insert(p, nil)
		}

		tbl.Delete(worstCaseProbePfx4) // delete matching prefix

		for b.Loop() {
			tbl.Contains(worstCaseProbeIP4)
		}
	})

	b.Run("Lookup", func(b *testing.B) {
		tbl := new(Table[any])
		for _, p := range worstCasePfxsIP4 {
			tbl.Insert(p, nil)
		}

		tbl.Delete(worstCaseProbePfx4) // delete matching prefix

		for b.Loop() {
			tbl.Lookup(worstCaseProbeIP4)
		}
	})

	b.Run("LookupPrefix", func(b *testing.B) {
		tbl := new(Table[any])
		for _, p := range worstCasePfxsIP4 {
			tbl.Insert(p, nil)
		}

		tbl.Delete(worstCaseProbePfx4) // delete matching prefix

		for b.Loop() {
			tbl.LookupPrefix(worstCaseProbePfx4)
		}
	})

	b.Run("LookupPrefixLPM", func(b *testing.B) {
		tbl := new(Table[any])
		for _, p := range worstCasePfxsIP4 {
			tbl.Insert(p, nil)
		}

		tbl.Delete(worstCaseProbePfx4) // delete matching prefix

		for b.Loop() {
			tbl.LookupPrefixLPM(worstCaseProbePfx4)
		}
	})
}

func BenchmarkFastWorstCaseMiss4(b *testing.B) {
	b.Run("Contains", func(b *testing.B) {
		tbl := new(Fast[any])
		for _, p := range worstCasePfxsIP4 {
			tbl.Insert(p, nil)
		}

		tbl.Delete(worstCaseProbePfx4) // delete matching prefix

		for b.Loop() {
			tbl.Contains(worstCaseProbeIP4)
		}
	})

	b.Run("Lookup", func(b *testing.B) {
		tbl := new(Fast[any])
		for _, p := range worstCasePfxsIP4 {
			tbl.Insert(p, nil)
		}

		tbl.Delete(worstCaseProbePfx4) // delete matching prefix

		for b.Loop() {
			tbl.Lookup(worstCaseProbeIP4)
		}
	})

	b.Run("LookupPrefix", func(b *testing.B) {
		tbl := new(Fast[any])
		for _, p := range worstCasePfxsIP4 {
			tbl.Insert(p, nil)
		}

		tbl.Delete(worstCaseProbePfx4) // delete matching prefix

		for b.Loop() {
			tbl.LookupPrefix(worstCaseProbePfx4)
		}
	})

	b.Run("LookupPrefixLPM", func(b *testing.B) {
		tbl := new(Fast[any])
		for _, p := range worstCasePfxsIP4 {
			tbl.Insert(p, nil)
		}

		tbl.Delete(worstCaseProbePfx4) // delete matching prefix

		for b.Loop() {
			tbl.LookupPrefixLPM(worstCaseProbePfx4)
		}
	})
}
func BenchmarkBartWorstCaseMatch6(b *testing.B) {
	b.Run("Contains", func(b *testing.B) {
		tbl := new(Table[any])
		for _, p := range worstCasePfxsIP6 {
			tbl.Insert(p, nil)
		}

		for b.Loop() {
			tbl.Contains(worstCaseProbeIP6)
		}
	})

	b.Run("Lookup", func(b *testing.B) {
		tbl := new(Table[any])
		for _, p := range worstCasePfxsIP6 {
			tbl.Insert(p, nil)
		}
		tbl.Insert(ipv6DefaultRoute, ipv6DefaultRoute.String())
		tbl.Delete(worstCaseProbePfx6)

		for b.Loop() {
			tbl.Lookup(worstCaseProbeIP6)
		}
	})

	b.Run("LookupPrefix", func(b *testing.B) {
		tbl := new(Table[any])
		for _, p := range worstCasePfxsIP6 {
			tbl.Insert(p, nil)
		}
		tbl.Insert(ipv6DefaultRoute, ipv6DefaultRoute.String())
		tbl.Delete(worstCaseProbePfx6)

		for b.Loop() {
			tbl.LookupPrefix(worstCaseProbePfx6)
		}
	})

	b.Run("LookupPrefixLPM", func(b *testing.B) {
		tbl := new(Table[any])
		for _, p := range worstCasePfxsIP6 {
			tbl.Insert(p, nil)
		}
		tbl.Insert(ipv6DefaultRoute, ipv6DefaultRoute.String())
		tbl.Delete(worstCaseProbePfx6)

		for b.Loop() {
			tbl.LookupPrefixLPM(worstCaseProbePfx6)
		}
	})
}

func BenchmarkFastWorstCaseMatch6(b *testing.B) {
	b.Run("Contains", func(b *testing.B) {
		tbl := new(Fast[any])
		for _, p := range worstCasePfxsIP6 {
			tbl.Insert(p, nil)
		}

		for b.Loop() {
			tbl.Contains(worstCaseProbeIP6)
		}
	})

	b.Run("Lookup", func(b *testing.B) {
		tbl := new(Fast[any])
		for _, p := range worstCasePfxsIP6 {
			tbl.Insert(p, nil)
		}
		tbl.Insert(ipv6DefaultRoute, ipv6DefaultRoute.String())
		tbl.Delete(worstCaseProbePfx6)

		for b.Loop() {
			tbl.Lookup(worstCaseProbeIP6)
		}
	})

	b.Run("LookupPrefix", func(b *testing.B) {
		tbl := new(Fast[any])
		for _, p := range worstCasePfxsIP6 {
			tbl.Insert(p, nil)
		}
		tbl.Insert(ipv6DefaultRoute, ipv6DefaultRoute.String())
		tbl.Delete(worstCaseProbePfx6)

		for b.Loop() {
			tbl.LookupPrefix(worstCaseProbePfx6)
		}
	})

	b.Run("LookupPrefixLPM", func(b *testing.B) {
		tbl := new(Fast[any])
		for _, p := range worstCasePfxsIP6 {
			tbl.Insert(p, nil)
		}
		tbl.Insert(ipv6DefaultRoute, ipv6DefaultRoute.String())
		tbl.Delete(worstCaseProbePfx6)

		for b.Loop() {
			tbl.LookupPrefixLPM(worstCaseProbePfx6)
		}
	})
}

func BenchmarkBartWorstCaseMiss6(b *testing.B) {
	b.Run("Contains", func(b *testing.B) {
		tbl := new(Table[any])
		for _, p := range worstCasePfxsIP6 {
			tbl.Insert(p, nil)
		}

		tbl.Delete(worstCaseProbePfx6) // delete matching prefix

		for b.Loop() {
			tbl.Contains(worstCaseProbeIP6)
		}
	})

	b.Run("Lookup", func(b *testing.B) {
		tbl := new(Table[any])
		for _, p := range worstCasePfxsIP6 {
			tbl.Insert(p, nil)
		}

		tbl.Delete(worstCaseProbePfx6) // delete matching prefix

		for b.Loop() {
			tbl.Lookup(worstCaseProbeIP6)
		}
	})

	b.Run("LookupPrefix", func(b *testing.B) {
		tbl := new(Table[any])
		for _, p := range worstCasePfxsIP6 {
			tbl.Insert(p, nil)
		}

		tbl.Delete(worstCaseProbePfx6) // delete matching prefix

		for b.Loop() {
			tbl.LookupPrefix(worstCaseProbePfx6)
		}
	})

	b.Run("LookupPrefixLPM", func(b *testing.B) {
		tbl := new(Table[any])
		for _, p := range worstCasePfxsIP6 {
			tbl.Insert(p, nil)
		}

		tbl.Delete(worstCaseProbePfx6) // delete matching prefix

		for b.Loop() {
			tbl.LookupPrefixLPM(worstCaseProbePfx6)
		}
	})
}

func BenchmarkFastWorstCaseMiss6(b *testing.B) {
	b.Run("Contains", func(b *testing.B) {
		tbl := new(Fast[any])
		for _, p := range worstCasePfxsIP6 {
			tbl.Insert(p, nil)
		}

		tbl.Delete(worstCaseProbePfx6) // delete matching prefix

		for b.Loop() {
			tbl.Contains(worstCaseProbeIP6)
		}
	})

	b.Run("Lookup", func(b *testing.B) {
		tbl := new(Fast[any])
		for _, p := range worstCasePfxsIP6 {
			tbl.Insert(p, nil)
		}

		tbl.Delete(worstCaseProbePfx6) // delete matching prefix

		for b.Loop() {
			tbl.Lookup(worstCaseProbeIP6)
		}
	})

	b.Run("LookupPrefix", func(b *testing.B) {
		tbl := new(Fast[any])
		for _, p := range worstCasePfxsIP6 {
			tbl.Insert(p, nil)
		}

		tbl.Delete(worstCaseProbePfx6) // delete matching prefix

		for b.Loop() {
			tbl.LookupPrefix(worstCaseProbePfx6)
		}
	})

	b.Run("LookupPrefixLPM", func(b *testing.B) {
		tbl := new(Fast[any])
		for _, p := range worstCasePfxsIP6 {
			tbl.Insert(p, nil)
		}

		tbl.Delete(worstCaseProbePfx6) // delete matching prefix

		for b.Loop() {
			tbl.LookupPrefixLPM(worstCaseProbePfx6)
		}
	})
}
