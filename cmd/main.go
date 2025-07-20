package main

import (
	"log"
	"math/rand/v2"
	"net/netip"
	"sync"
	"time"

	"github.com/gaissmai/bart"
)

func main() {
	prng := rand.New(rand.NewPCG(42, 42))
	log.SetFlags(log.Lmicroseconds)

	lt := new(bart.Lite)
	ts := time.Now()
	for _, pfx := range tier1Pfxs() {
		lt.Insert(pfx)
	}

	syncLite := SyncLiteFrom(lt).WithPool()
	log.Printf("insert full tier1 table and clone: %v, size: %d", time.Since(ts), syncLite.Load().Size())
	log.Printf("len tier1pfxs: %d", len(tier1Pfxs()))

	wg := sync.WaitGroup{}

	wg.Add(1)
	go func() {
		defer wg.Done()
		for {
			log.Printf("Lite.Size(): %d", syncLite.Load().Size())
			time.Sleep(time.Second * 1)
		}
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		var ok bool
		for {
			ip := randomRealWorldPrefixes(prng, 1)[0].Addr().Next()
			ok = syncLite.Contains(ip)
			log.Printf("Lite.Contains(): %v, %s", ok, ip)
			time.Sleep(time.Millisecond * 5_005)
		}
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		for {
			for _, pfx := range randomRealWorldPrefixes(prng, 1_000) {
				syncLite.Insert(pfx)
			}
			time.Sleep(time.Second * 1)
		}
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		for {
			pfxs := collectShuffeled(prng, syncLite.Load())
			p10 := len(pfxs) / 100 * 10
			for _, pfx := range pfxs[:p10] {
				syncLite.Delete(pfx)
			}
			time.Sleep(time.Second * 1)
		}
	}()

	wg.Wait()
}

func collectShuffeled(prng *rand.Rand, rt *bart.Lite) []netip.Prefix {
	pfxs := make([]netip.Prefix, 0, rt.Size())
	for pfx := range rt.All() {
		pfxs = append(pfxs, pfx)
	}

	prng.Shuffle(len(pfxs), func(i, j int) {
		pfxs[i], pfxs[j] = pfxs[j], pfxs[i]
	})

	return pfxs
}
