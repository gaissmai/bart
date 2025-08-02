package bart_test

import (
	"net/netip"
	"sync"

	"github.com/gaissmai/bart"
)

// ExampleLite_concurrent demonstrates safe concurrent usage of bart.
//
// This example is intended to be run with the Go race detector enabled
// (use `go test -race -run=ExampleTable_concurrent`)
// to verify that concurrent access is safe and free of data races.
func ExampleLite_sync() {
	rt := new(bart.Lite)

	wg := sync.WaitGroup{}
	wg.Add(1)
	go func() {
		defer wg.Done()
		for range 1_000_000 {
			for _, s := range exampleIPs {
				ip := netip.MustParseAddr(s)
				_ = rt.Contains(ip)
			}
		}
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		for range 10_000 {
			for _, s := range examplePrefixes {
				pfx := netip.MustParsePrefix(s)
				rt.InsertSync(pfx)
			}
		}
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		for range 10_000 {
			for _, s := range examplePrefixes {
				pfx := netip.MustParsePrefix(s)
				rt.DeleteSync(pfx)
			}
		}
	}()

	wg.Wait()

	// Output:
}
