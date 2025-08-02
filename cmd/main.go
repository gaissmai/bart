package main

import (
	"bufio"
	"compress/gzip"
	"fmt"
	"log"
	"math/rand/v2"
	"net/netip"
	"os"
	"slices"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/gaissmai/bart"
)

// full internet prefix list, gzipped
const prefixFile = "../testdata/prefixes.txt.gz"

type testVal struct {
	data int
}

// Clone ensures deep copying for use with ...Persist.
func (v *testVal) Clone() *testVal {
	if v == nil {
		return nil
	}
	return &testVal{data: v.data}
}

var (
	boolSink bool
	//
	contains atomic.Int64
	lookups  atomic.Int64
	//
	inserts atomic.Int64
	deletes atomic.Int64
)

func main() {
	pfxs := tier1Pfxs()

	rt := new(bart.Table[*testVal])

	wg := sync.WaitGroup{}

	wg.Add(1)
	go func() {
		defer wg.Done()
		prng := rand.New(rand.NewPCG(42, 42))

		for {
			ip := pfxs[prng.IntN(len(pfxs))].Addr()
			boolSink = rt.Contains(ip)
			contains.Add(1)
		}
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		prng := rand.New(rand.NewPCG(24, 24))

		for {
			ip := pfxs[prng.IntN(len(pfxs))].Addr()
			_, boolSink = rt.Lookup(ip)
			lookups.Add(1)
		}
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		prng := rand.New(rand.NewPCG(2442, 2442))
		pfxs := shuffle(prng, pfxs)

		for {
			for i, pfx := range pfxs {
				rt.Insert(pfx, &testVal{data: i})
				inserts.Add(1)
			}
		}
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		prng := rand.New(rand.NewPCG(1418, 1418))
		pfxs := shuffle(prng, pfxs)

		for {
			for _, pfx := range pfxs {
				rt.Delete(pfx)
				deletes.Add(1)
			}
		}
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()

		// skip the first tick
		tick := time.NewTicker(time.Second * 1)
		<-tick.C

		lastContains := contains.Load()
		lastLookups := lookups.Load()
		lastInserts := inserts.Load()
		lastDeletes := deletes.Load()

		last := time.Now()
		for {
			<-tick.C

			nowContains := contains.Load()
			nowLookups := lookups.Load()
			nowInserts := inserts.Load()
			nowDeletes := deletes.Load()

			now := time.Now()
			deltaS := now.Sub(last).Seconds()
			last = now

			deltaContains := float64(nowContains-lastContains) / deltaS
			deltaLookups := float64(nowLookups-lastLookups) / deltaS
			deltaInserts := float64(nowInserts-lastInserts) / deltaS
			deltaDeletes := float64(nowDeletes-lastDeletes) / deltaS

			fmt.Printf(">>>> contains/s: %10d, lookups/s: %10d, inserts/s: %7d, deletes/s: %7d\n",
				int(deltaContains), int(deltaLookups), int(deltaInserts), int(deltaDeletes))

			lastContains = nowContains
			lastLookups = nowLookups
			lastInserts = nowInserts
			lastDeletes = nowDeletes

		}
	}()

	wg.Wait()
}

func tier1Pfxs() (pfxs []netip.Prefix) {
	file, err := os.Open(prefixFile)
	if err != nil {
		log.Fatal(err)
	}

	rgz, err := gzip.NewReader(file)
	if err != nil {
		log.Fatal(err)
	}

	scanner := bufio.NewScanner(rgz)
	for scanner.Scan() {
		line := scanner.Text()
		line = strings.TrimSpace(line)

		pfx := netip.MustParsePrefix(line)
		pfx = pfx.Masked()

		pfxs = append(pfxs, pfx)
	}

	if err = scanner.Err(); err != nil {
		log.Printf("reading from %v, %v", rgz, err)
	}

	return pfxs
}

func shuffle(prng *rand.Rand, in []netip.Prefix) []netip.Prefix {
	out := slices.Clone(in)
	prng.Shuffle(len(out), func(i, j int) {
		out[i], out[j] = out[j], out[i]
	})
	return out
}
