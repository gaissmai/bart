package bart_test

import (
	"net/netip"
	"sync"
	"sync/atomic"

	"github.com/gaissmai/bart"
)

// SyncTable demonstrates how to wrap a [bart.Table] for safe concurrent access in Go.
//
// This example struct allows multiple goroutines to perform lock-free, concurrent reads
// via an atomic pointer, while synchronizing writers with a mutex to ensure exclusive access.
// This concurrency pattern is useful when reads are frequent and writes are rare
// or take a long time in comparison to reads,
// providing high performance for concurrent workloads.
type SyncTable[V any] struct {
	// Atomic pointer to the current table version.
	// Enables lock-free, concurrent reads by multiple goroutines.
	atomicPtr atomic.Pointer[bart.Table[V]]

	// Mutex for synchronizing concurrent writers.
	// Writers must acquire the lock before modifying the table.
	// No CAS is used for writers; only one writer at a time is allowed.
	mutex sync.Mutex
}

// NewSyncTable creates and initializes a new SyncTable.
// The underlying table is initialized and stored atomically.
func NewSyncTable[V any]() *SyncTable[V] {
	lf := new(SyncTable[V])
	lf.atomicPtr.Store(new(bart.Table[V]))
	return lf
}

// Contains is a sync adapter for [bart.Table.Contains].
func (lf *SyncTable[V]) Contains(ip netip.Addr) bool {
	rt := lf.atomicPtr.Load() // lock-free read of the current table version
	return rt.Contains(ip)
}

// Lookup is a sync adapter for [bart.Table.Lookup].
func (lf *SyncTable[V]) Lookup(ip netip.Addr) (val V, ok bool) {
	rt := lf.atomicPtr.Load() // lock-free read of the current table version
	return rt.Lookup(ip)
}

// Insert is a sync adapter for [bart.Table.Insert].
// This method acquires a writer lock to ensure exclusive access for writers.
// It creates a new persistent table version and atomically updates the pointer.
// Concurrent readers remain lock-free and always see a consistent table.
func (lf *SyncTable[V]) Insert(pfx netip.Prefix, val V) {
	lf.mutex.Lock() // acquire writer lock to exclude other writers
	defer lf.mutex.Unlock()

	oldPtr := lf.atomicPtr.Load()            // get current table version
	newPtr := oldPtr.InsertPersist(pfx, val) // create new persistent table version

	lf.atomicPtr.Store(newPtr) // atomically publish new version for readers
}

// Update is a sync adapter for [bart.Table.Update].
func (lf *SyncTable[V]) Update(pfx netip.Prefix, cb func(val V, ok bool) V) (newVal V) {
	lf.mutex.Lock()
	defer lf.mutex.Unlock()

	oldPtr := lf.atomicPtr.Load()
	newPtr, val := oldPtr.UpdatePersist(pfx, cb)

	lf.atomicPtr.Store(newPtr)
	return val
}

// Delete is a sync adapter for [bart.Table.Delete].
func (lf *SyncTable[V]) Delete(pfx netip.Prefix) {
	lf.mutex.Lock()
	defer lf.mutex.Unlock()

	oldPtr := lf.atomicPtr.Load()
	newPtr := oldPtr.DeletePersist(pfx)

	lf.atomicPtr.Store(newPtr)
}

// ExampleTable_concurrent demonstrates safe concurrent usage of bart.
// This example is intended to be run with the Go race detector enabled
// (use `go test -race -run=ExampleTable_concurrent`)
// to verify that concurrent access is safe and free of data races.
func ExampleTable_concurrent() {
	wg := sync.WaitGroup{}

	syncTbl := NewSyncTable[int]()

	wg.Add(1)
	go func() {
		defer wg.Done()
		for _, s := range examplePrefixes {
			pfx := netip.MustParsePrefix(s)
			syncTbl.Insert(pfx, 5)
		}
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		cb := func(val int, _ bool) int {
			val++
			return val
		}

		for _, s := range examplePrefixes {
			pfx := netip.MustParsePrefix(s)
			syncTbl.Update(pfx, cb)
		}
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		for _, s := range examplePrefixes {
			pfx := netip.MustParsePrefix(s)
			syncTbl.Delete(pfx)
		}
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		for _, s := range exampleIPs {
			ip := netip.MustParseAddr(s)
			syncTbl.Contains(ip)
		}
	}()

	wg.Wait()

	// Output:
}
