package main

import (
	"net/netip"
	"sync"
	"sync/atomic"

	"github.com/gaissmai/bart"
)

type SyncLite struct {
	atomic.Pointer[bart.Lite]
	sync.Mutex
}

func NewSyncLite() *SyncLite {
	lf := new(SyncLite)
	lf.Store(new(bart.Lite))
	return lf
}

func SyncLiteFrom(lite *bart.Lite) *SyncLite {
	lf := new(SyncLite)
	lf.Store(lite.Clone())
	return lf
}

func (lf *SyncLite) WithPool() *SyncLite {
	lf.Lock() // acquire writer lock to exclude other writers
	defer lf.Unlock()

	oldPtr := lf.Load()         // get current table version
	newPtr := oldPtr.WithPool() // create new persistent table version

	lf.Store(newPtr) // atomically publish new version for readers
	return lf
}

func (lf *SyncLite) Contains(ip netip.Addr) bool {
	return lf.Load().Contains(ip)
}

func (lf *SyncLite) Insert(pfx netip.Prefix) {
	lf.Lock() // acquire writer lock to exclude other writers
	defer lf.Unlock()

	oldPtr := lf.Load()                 // get current table version
	newPtr := oldPtr.InsertPersist(pfx) // create new persistent table version

	lf.Store(newPtr) // atomically publish new version for readers
}

func (lf *SyncLite) Delete(pfx netip.Prefix) {
	lf.Lock()
	defer lf.Unlock()

	oldPtr := lf.Load()
	newPtr := oldPtr.DeletePersist(pfx)

	lf.Store(newPtr)
}
