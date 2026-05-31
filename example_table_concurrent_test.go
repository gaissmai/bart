// Copyright (c) 2026 Karl Gaissmaier
// SPDX-License-Identifier: MIT

package bart_test

import (
	"sync"
	"sync/atomic"

	"github.com/gaissmai/bart"
)

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
// If the payload V either contains pointers or is a pointer,
// implement a Clone method (structural typing is used).
func ExampleTable_concurrent() {
	var tblAtomicPtr atomic.Pointer[bart.Table[*testVal]]
	var tblMutex sync.Mutex

	baseTbl := new(bart.Table[*testVal])
	tblAtomicPtr.Store(baseTbl)

	var readerWg sync.WaitGroup
	var writerWg sync.WaitGroup

	// Unbuffered channel acts as a synchronized starting gun.
	// All goroutines will block on this channel until it is closed.
	startSignal := make(chan struct{})

	// Channel to signal when all writers have finished.
	writersDone := make(chan struct{})

	// 1. GOROUTINE: READERS
	// Tracked by the reader wg. Runs until writersDone is closed.
	readerWg.Add(1)
	go func() {
		defer readerWg.Done()
		<-startSignal // Block until the starting gun fires

		var localSink bool
		for {
			select {
			case <-writersDone: // stop when all writers finished
				_ = localSink
				return
			default:
				for _, ip := range exampleIPs {
					localSink = tblAtomicPtr.Load().Contains(ip)
				}
			}
		}
	}()

	// 2. GOROUTINE: WRITER (INSERTS)
	// Tracked only by writer wg
	writerWg.Add(1)
	go func() {
		defer writerWg.Done()
		<-startSignal // Block until the starting gun fires

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

	// 3. GOROUTINE: WRITER (DELETES)
	// Tracked only by writer wg
	writerWg.Add(1)
	go func() {
		defer writerWg.Done()
		<-startSignal // Block until the starting gun fires

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

	// Orchestration: Monitor the writers, signal the reader, and orchestrate shutdown.
	go func() {
		<-startSignal // Block until the starting gun fires
		writerWg.Wait()
		close(writersDone)
	}()

	// At this point, all goroutines are initialized and waiting.
	// Closing the channel releases all of them at the exact same fraction of a second.
	close(startSignal)

	// We only need to wait for the reader (wg), because the reader will
	// only exit after writersDone is closed, which only happens after all writers are done.
	readerWg.Wait()

	// Output:
}
