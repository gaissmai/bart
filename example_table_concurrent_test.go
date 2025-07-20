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
//
// If the payload V either contains a pointer or is a pointer,
// it must implement the [bart.Cloner] interface.
type SyncTable[V bart.Cloner[V]] struct {
	// Atomic pointer to the current table version.
	// Enables lock-free, concurrent reads by multiple goroutines.
	atomic.Pointer[bart.Table[V]]

	// Mutex for synchronizing concurrent writers.
	// Writers must acquire the lock before modifying the table.
	// No CAS is used for writers; only one writer at a time is allowed.
	sync.Mutex
}

// NewSyncTable creates and initializes a new SyncTable.
// The underlying table is initialized and stored atomically.
func NewSyncTable[V bart.Cloner[V]]() *SyncTable[V] {
	lf := new(SyncTable[V])
	lf.Store(new(bart.Table[V]))
	return lf
}

// WithPool replaces the current table version with a new version with a sync.Pool for trie nodes.
//
// WithPool acquires an exclusive writer lock, ensuring no other writer can modify the value concurrently.
// It then retrieves the current table version, creates a new version using the table's WithPool method,
// and atomically publishes this new version for concurrent readers.
//
// This method is safe for concurrent use by multiple goroutines.
func (lf *SyncTable[V]) WithPool() *SyncTable[V] {
	lf.Lock() // acquire writer lock to exclude other writers
	defer lf.Unlock()

	oldPtr := lf.Load()         // get current table version
	newPtr := oldPtr.WithPool() // create new persistent table version

	lf.Store(newPtr) // atomically publish new version for readers
	return lf
}

// Contains is a sync adapter for [bart.Table.Contains].
func (lf *SyncTable[V]) Contains(ip netip.Addr) bool {
	return lf.Load().Contains(ip)
}

// Lookup is a sync adapter for [bart.Table.Lookup].
func (lf *SyncTable[V]) Lookup(ip netip.Addr) (val V, ok bool) {
	return lf.Load().Lookup(ip)
}

// Insert is a sync adapter for [bart.Table.Insert].
// This method acquires a writer lock to ensure exclusive access for writers.
// It creates a new persistent table version and atomically updates the pointer.
// Concurrent readers remain lock-free and always see a consistent table.
func (lf *SyncTable[V]) Insert(pfx netip.Prefix, val V) {
	lf.Lock() // acquire writer lock to exclude other writers
	defer lf.Unlock()

	oldPtr := lf.Load()                      // get current table version
	newPtr := oldPtr.InsertPersist(pfx, val) // create new persistent table version

	lf.Store(newPtr) // atomically publish new version for readers
}

// Update is a sync adapter for [bart.Table.Update].
func (lf *SyncTable[V]) Update(pfx netip.Prefix, cb func(val V, ok bool) V) (newVal V) {
	lf.Lock()
	defer lf.Unlock()

	oldPtr := lf.Load()
	newPtr, val := oldPtr.UpdatePersist(pfx, cb)

	lf.Store(newPtr)
	return val
}

// Delete is a sync adapter for [bart.Table.Delete].
func (lf *SyncTable[V]) Delete(pfx netip.Prefix) {
	lf.Lock()
	defer lf.Unlock()

	oldPtr := lf.Load()
	newPtr := oldPtr.DeletePersist(pfx)

	lf.Store(newPtr)
}

// #######################################
// just a very stupid example of a payload
// #######################################

// testVal is a sample value type.
// We use *testVal as the generic type V, which is a pointer type,
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

// example callback for Update, just decrement data
var cb = func(p *testVal, ok bool) *testVal {
	if ok && p != nil {
		p.data--
		return p
	}
	return &testVal{data: -1}
}

// #######################################

// ExampleTable_concurrent demonstrates safe concurrent usage of bart.
// This example is intended to be run with the Go race detector enabled
// (use `go test -race -run=ExampleTable_concurrent`)
// to verify that concurrent access is safe and free of data races.
func ExampleTable_concurrent() {
	wg := sync.WaitGroup{}

	syncTbl := NewSyncTable[*testVal]()
	syncTbl.WithPool()

	wg.Add(1)
	go func() {
		defer wg.Done()
		for range 1_000_000 {
			for _, s := range exampleIPs {
				ip := netip.MustParseAddr(s)
				_, _ = syncTbl.Lookup(ip)
			}
		}
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		for range 10_000 {
			for _, s := range examplePrefixes {
				pfx := netip.MustParsePrefix(s)
				syncTbl.Insert(pfx, &testVal{data: 0})
			}
		}
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		for range 10_000 {
			for _, s := range examplePrefixes {
				pfx := netip.MustParsePrefix(s)
				syncTbl.Update(pfx, cb)
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

	// Output:
}
