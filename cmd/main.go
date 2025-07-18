package main

import (
	"log"
	"time"

	"github.com/gaissmai/bart"
)

func main() {
	log.SetFlags(log.Lmicroseconds)

	lt := new(bart.Lite)
	ts := time.Now()
	for _, pfx := range tier1Pfxs() {
		lt.Insert(pfx)
	}

	syncLite := SyncLiteFrom(lt)
	log.Printf("insert full tier1 table and clone: %v, size: %d", time.Since(ts), syncLite.Load().Size())
	log.Printf("len tier1pfxs: %d", len(tier1Pfxs()))

	/*
		wg := sync.WaitGroup{}

			wg.Add(1)
			go func() {
				defer wg.Done()
				for range 1_000_000 {
					for _, s := range exampleIPs {
						ip := netip.MustParseAddr(s)
						_ = syncTbl.Contains(ip)
					}
				}
			}()

			wg.Add(1)
			go func() {
				defer wg.Done()
				for range 10_000 {
					for _, s := range examplePrefixes {
						pfx := netip.MustParsePrefix(s)
						syncTbl.Insert(pfx)
					}
				}
			}()

			wg.Add(1)
			go func() {
				defer wg.Done()
				for range 10_000 {
					for _, s := range examplePrefixes {
						pfx := netip.MustParsePrefix(s)
						syncTbl.Delete(pfx)
					}
				}
			}()

			wg.Wait()
	*/
}
