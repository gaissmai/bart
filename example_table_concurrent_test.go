package bart_test

import (
	"net/netip"
	"sync"
	"sync/atomic"

	"github.com/gaissmai/bart"
)

// testVal is a simple sample value type.
// We use *testVal as the generic payload type V, which is a pointer type,
// so it must implement Cloner[*testVal].
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
	tblAtomicPtr atomic.Pointer[bart.Table[*testVal]]
	tblMutex     sync.Mutex
)

// #######################################

// ExampleTable_concurrent demonstrates safe concurrent usage of bart.
// This example is intended to be run with the Go race detector enabled
// (use `go test -race -run=ExampleTable_concurrent`)
// to verify that concurrent access is safe and free of data races.
//
// This example demonstrates how multiple goroutines perform lock-free, concurrent reads
// via an atomic pointer, while synchronizing writers with a mutex to ensure exclusive access.
// This concurrency pattern is useful when reads are frequent and writes are rare
// or take a long time in comparison to reads,
// providing high performance for concurrent workloads.
//
// If the payload V either contains a pointer or is a pointer,
// it must implement the [bart.Cloner] interface.
func ExampleTable_concurrent() {
	baseTbl := new(bart.Table[*testVal]) // .WithPool()
	tblAtomicPtr.Store(baseTbl)

	wg := sync.WaitGroup{}

	wg.Add(1)
	go func() {
		defer wg.Done()
		for range 10_000_000 {
			for _, s := range exampleIPs {
				ip := netip.MustParseAddr(s)
				_, _ = tblAtomicPtr.Load().Lookup(ip)
			}
		}
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		for range 10_000 {
			for _, s := range examplePrefixes {
				pfx := netip.MustParsePrefix(s)
				tblMutex.Lock()

				oldTbl := tblAtomicPtr.Load()
				newTbl := oldTbl.InsertPersist(pfx, &testVal{data: 0})
				tblAtomicPtr.Store(newTbl)

				tblMutex.Unlock()
			}
		}
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		for range 10_000 {
			for _, s := range examplePrefixes {
				pfx := netip.MustParsePrefix(s)

				tblMutex.Lock()
				oldTbl := tblAtomicPtr.Load()
				newTbl := oldTbl.DeletePersist(pfx)
				tblAtomicPtr.Store(newTbl)
				tblMutex.Unlock()
			}
		}
	}()

	wg.Wait()

	// Output:
}
