// Copyright (c) 2025 Karl Gaissmaier
// SPDX-License-Identifier: MIT

package bart_test

import (
	"sync"
	"sync/atomic"

	"github.com/gaissmai/bart"
)

// testVal is a simple sample value type.
// We use *testVal as the generic payload type V, which is a pointer type,
// so it must implement Cloner[V].
type testVal struct {
	data int
}

// Clone ensures deep copying for use with ...Persist implementing the
// Cloner interface.
func (v *testVal) Clone() *testVal {
	if v == nil {
		return nil
	}
	return &testVal{data: v.data}
}

// #######################################

// ExampleTable_concurrent demonstrates safe concurrent usage of bart.Table.
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
// it must implement the Cloner interface.
func ExampleTable_concurrent() {
	var tblAtomicPtr atomic.Pointer[bart.Table[*testVal]]
	var tblMutex sync.Mutex

	baseTbl := new(bart.Table[*testVal])
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
			cur := tblAtomicPtr.Load()

			// batch of inserts
			next := cur
			for _, pfx := range examplePrefixes {
				next = next.InsertPersist(pfx, &testVal{data: 0})
			}

			tblAtomicPtr.Store(next)
			tblMutex.Unlock()
		}
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		for range 1_000 {
			tblMutex.Lock()
			cur := tblAtomicPtr.Load()

			// batch of deletes
			next := cur
			for _, pfx := range examplePrefixes {
				next = next.DeletePersist(pfx)
			}

			tblAtomicPtr.Store(next)
			tblMutex.Unlock()
		}
	}()

	wg.Wait()

	// Output:
}
