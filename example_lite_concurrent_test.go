// Copyright (c) 2025 Karl Gaissmaier
// SPDX-License-Identifier: MIT

package bart_test

import (
	"sync"
	"sync/atomic"

	"github.com/gaissmai/bart"
)

// ExampleLite_concurrent demonstrates safe concurrent usage of bart.Lite.
//
// This example is intended to be run with the Go race detector enabled
// (use `go test -race -run=ExampleLite_concurrent`)
// to verify that concurrent access is safe and free of data races.
//
// This example demonstrates how multiple goroutines perform lock-free, concurrent reads
// via an atomic pointer, while synchronizing writers with a mutex to ensure exclusive access.
// This concurrency pattern is useful when reads are frequent and writes are rare
// or take a long time in comparison to reads,
// providing high performance for concurrent workloads.
func ExampleLite_concurrent() {
	var liteAtomicPtr atomic.Pointer[bart.Lite]
	var liteMutex sync.Mutex

	baseTbl := new(bart.Lite)
	liteAtomicPtr.Store(baseTbl)

	wg := sync.WaitGroup{}
	wg.Add(1)
	go func() {
		defer wg.Done()
		for range 10_000 {
			for _, ip := range exampleIPs {
				_ = liteAtomicPtr.Load().Contains(ip)
			}
		}
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		for range 1000 {
			liteMutex.Lock()
			cur := liteAtomicPtr.Load()

			// batch of inserts
			next := cur
			for _, pfx := range examplePrefixes {
				next = next.InsertPersist(pfx)
			}

			liteAtomicPtr.Store(next)
			liteMutex.Unlock()
		}
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		for range 1000 {
			liteMutex.Lock()
			cur := liteAtomicPtr.Load()

			// batch of deletes
			next := cur
			for _, pfx := range examplePrefixes {
				next, _ = next.DeletePersist(pfx)
			}

			liteAtomicPtr.Store(next)
			liteMutex.Unlock()
		}
	}()

	wg.Wait()

	// Output:
}
