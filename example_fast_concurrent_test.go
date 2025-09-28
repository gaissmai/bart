// Copyright (c) 2025 Karl Gaissmaier
// SPDX-License-Identifier: MIT

package bart_test

import (
	"sync"
	"sync/atomic"

	"github.com/gaissmai/bart"
)

// #######################################

// ExampleFast_concurrent demonstrates safe concurrent usage of bart.Fast.
// This example is intended to be run with the Go race detector enabled
// (use `go test -race -run=ExampleFast_concurrent`)
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
func ExampleFast_concurrent() {
	var tblAtomicPtr atomic.Pointer[bart.Fast[*testVal]]
	var tblMutex sync.Mutex

	baseTbl := new(bart.Fast[*testVal])
	tblAtomicPtr.Store(baseTbl)

	wg := sync.WaitGroup{}

	wg.Add(1)
	go func() {
		defer wg.Done()
		for range 100_000 {
			for _, ip := range exampleIPs {
				_, _ = tblAtomicPtr.Load().Lookup(ip)
			}
		}
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		for range 1_000 {
			tblMutex.Lock()
			tbl := tblAtomicPtr.Load()

			// batch of inserts
			for _, pfx := range examplePrefixes {
				tbl = tbl.InsertPersist(pfx, &testVal{data: 0})
			}

			tblAtomicPtr.Store(tbl)
			tblMutex.Unlock()
		}
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		for range 1_000 {
			tblMutex.Lock()
			tbl := tblAtomicPtr.Load()

			// batch of deletes
			for _, pfx := range examplePrefixes {
				tbl, _, _ = tbl.DeletePersist(pfx)
			}

			tblAtomicPtr.Store(tbl)
			tblMutex.Unlock()
		}
	}()

	wg.Wait()

	// Output:
}
