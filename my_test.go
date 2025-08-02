package bart

import (
	"math/rand/v2"
	"net/netip"
	"sync/atomic"
	"testing"

	"github.com/gaissmai/bart/internal/sparse"
)

var exampleIPs = []netip.Addr{
	mpa("192.168.1.100"),
	mpa("192.168.2.1"),
	mpa("172.16.0.1"),
	mpa("2001:7c0:3100:1::1"),
	mpa("2001:7c0:3100:2::1"),
	mpa("fc00::1"),
	mpa("2003:dead:beef::1"),
}

var examplePrefixes = []netip.Prefix{
	mpp("192.168.0.0/16"),
	mpp("192.168.1.0/24"),
	mpp("2001:7c0:3100::/40"),
	mpp("2001:7c0:3100:1::/64"),
	mpp("fc00::/7"),
}

func BenchmarkMy(b *testing.B) {
	foo := atomic.Pointer[sparse.Array256[string]]{}
	b.Run("atomicPtrLoad", func(b *testing.B) {
		for b.Loop() {
			foo.Load()
		}
	})

	b.Run("atomicPtrStore", func(b *testing.B) {
		foo := atomic.Pointer[sparse.Array256[string]]{}
		for b.Loop() {
			foo.Store(&sparse.Array256[string]{})
		}
	})
}

/*
func TestMyDelete(t *testing.T) {
	rt := new(Table[int])

	rt.Insert(mpp("192.168.0.0/16"), 1)
	t.Log(rt.dumpString())

	rt.Insert(mpp("192.168.1.0/24"), 1)
	t.Log(rt.dumpString())

	rt.Delete(mpp("192.168.0.0/16"))
	t.Log(rt.dumpString())
}
*/

func TestMy(t *testing.T) {
	t.Parallel()

	prng := rand.New(rand.NewPCG(42, 42))
	k := 1_000_000
	pfxs := randomRealWorldPrefixes(prng, k)

	rt := new(Table[*testVal])
	for i, pfx := range pfxs {
		rt.Insert(pfx, &testVal{Data: i})
	}

	t.Run("contains", func(t *testing.T) {
		t.Parallel()
		for range k {
			for _, ip := range exampleIPs {
				_ = rt.Contains(ip)
			}
		}
	})

	t.Run("lookup", func(t *testing.T) {
		t.Parallel()
		for range k {
			for _, ip := range exampleIPs {
				_, _ = rt.Lookup(ip)
			}
		}
	})

	t.Run("sync Insert", func(t *testing.T) {
		t.Parallel()
		for i, pfx := range pfxs {
			rt.InsertSync(pfx, &testVal{Data: i})
		}
	})
}
