package bart_test

import (
	"net/netip"
	"sync"
	"sync/atomic"

	"github.com/gaissmai/bart"
)

// SyncLite demonstrates how to wrap a [bart.Lite] for safe concurrent access in Go.
//
// This example struct allows multiple goroutines to perform lock-free, concurrent reads
// via an atomic pointer, while synchronizing writers with a mutex to ensure exclusive access.
// This concurrency pattern is useful when reads are frequent and writes are rare
// or take a long time in comparison to reads,
// providing high performance for concurrent workloads.
type SyncLite struct {
	// Atomic pointer to the current table version.
	// Enables lock-free, concurrent reads by multiple goroutines.
	atomic.Pointer[bart.Lite]

	// Mutex for synchronizing concurrent writers.
	// Writers must acquire the lock before modifying the table.
	// No CAS is used for writers; only one writer at a time is allowed.
	sync.Mutex
}

// NewSyncLite creates and initializes a new SyncLite.
// The underlying table is initialized and stored atomically.
func NewSyncLite() *SyncLite {
	lf := new(SyncLite)
	lf.Store(new(bart.Lite))
	return lf
}

// WithPool replaces the current table version with a new version with a sync.Pool for trie nodes.
//
// WithPool acquires an exclusive writer lock, ensuring no other writer can modify the value concurrently.
// It then retrieves the current table version, creates a new version using the table's WithPool method,
// and atomically publishes this new version for concurrent readers.
//
// This method is safe for concurrent use by multiple goroutines.
func (lf *SyncLite) WithPool() *SyncLite {
	lf.Lock() // acquire writer lock to exclude other writers
	defer lf.Unlock()

	oldPtr := lf.Load()         // get current table version
	newPtr := oldPtr.WithPool() // create new persistent table version

	lf.Store(newPtr) // atomically publish new version for readers
	return lf
}

// Contains is a sync adapter for [bart.Lite.Contains].
func (lf *SyncLite) Contains(ip netip.Addr) bool {
	return lf.Load().Contains(ip)
}

// Insert is a sync adapter for [bart.Lite.Insert].
// This method acquires a writer lock to ensure exclusive access for writers.
// It creates a new persistent table version and atomically updates the pointer.
// Concurrent readers remain lock-free and always see a consistent table.
func (lf *SyncLite) Insert(pfx netip.Prefix) {
	lf.Lock() // acquire writer lock to exclude other writers
	defer lf.Unlock()

	oldPtr := lf.Load()                 // get current table version
	newPtr := oldPtr.InsertPersist(pfx) // create new persistent table version

	lf.Store(newPtr) // atomically publish new version for readers
}

// Delete is a sync adapter for [bart.Lite.Delete].
func (lf *SyncLite) Delete(pfx netip.Prefix) {
	lf.Lock()
	defer lf.Unlock()

	oldPtr := lf.Load()
	newPtr := oldPtr.DeletePersist(pfx)

	lf.Store(newPtr)
}

// ExampleLite_concurrent demonstrates safe concurrent usage of bart.
// This example is intended to be run with the Go race detector enabled
// (use `go test -race -run=ExampleLite_concurrent`)
// to verify that concurrent access is safe and free of data races.
func ExampleLite_concurrent() {
	wg := sync.WaitGroup{}

	syncTbl := NewSyncLite().WithPool()

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

	// Output:
}
