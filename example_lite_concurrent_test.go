package bart_test

import (
	"net/netip"
	"sync"
	"sync/atomic"

	"github.com/gaissmai/bart"
)

var (
	liteAtomicPtr atomic.Pointer[bart.Lite]
	liteMutex     sync.Mutex
)

// ExampleLite_concurrent demonstrates safe concurrent usage of bart.
//
// This example is intended to be run with the Go race detector enabled
// (use `go test -race -run=ExampleTable_concurrent`)
// to verify that concurrent access is safe and free of data races.
//
// This example demonstrates how multiple goroutines perform lock-free, concurrent reads
// via an atomic pointer, while synchronizing writers with a mutex to ensure exclusive access.
// This concurrency pattern is useful when reads are frequent and writes are rare
// or take a long time in comparison to reads,
// providing high performance for concurrent workloads.
func ExampleLite_concurrent() {
	baseTbl := new(bart.Lite) // .WithPool()
	liteAtomicPtr.Store(baseTbl)

	wg := sync.WaitGroup{}
	wg.Add(1)
	go func() {
		defer wg.Done()
		for range 1_000_000 {
			for _, s := range exampleIPs {
				ip := netip.MustParseAddr(s)
				_ = liteAtomicPtr.Load().Contains(ip)
			}
		}
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		for range 10_000 {
			for _, s := range examplePrefixes {
				pfx := netip.MustParsePrefix(s)

				liteMutex.Lock()
				oldTbl := liteAtomicPtr.Load()
				newTbl := oldTbl.InsertPersist(pfx)
				liteAtomicPtr.Store(newTbl)
				liteMutex.Unlock()
			}
		}
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		for range 10_000 {
			for _, s := range examplePrefixes {
				pfx := netip.MustParsePrefix(s)

				liteMutex.Lock()
				oldTbl := liteAtomicPtr.Load()
				newTbl := oldTbl.DeletePersist(pfx)
				liteAtomicPtr.Store(newTbl)
				liteMutex.Unlock()
			}
		}
	}()

	wg.Wait()

	// Output:
}
