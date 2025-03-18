package bart

import "net/netip"

// Lite is just a convenience wrapper for [Table], instantiated with an
// empty struct as payload. Lite is ideal for simple IP ACLs
// (access-control-lists) with plain true/false results without a payload.
//
// Lite delegates almost all methods unmodified to the underlying [Table].
type Lite struct {
	Table[struct{}]
}

// Insert a pfx into the tree.
func (l *Lite) Insert(pfx netip.Prefix) {
	l.Table.Insert(pfx, struct{}{})
}

// InsertPersist is similar to Insert but the receiver isn't modified.
func (l *Lite) InsertPersist(pfx netip.Prefix) *Lite {
	tbl := l.Table.InsertPersist(pfx, struct{}{})
	// copy of *tbl is here by intention
	//nolint:govet
	return &Lite{*tbl}
}

// DeletePersist is similar to Delete but the receiver isn't modified.
func (l *Lite) DeletePersist(pfx netip.Prefix) *Lite {
	tbl := l.Table.DeletePersist(pfx)
	// copy of *tbl is here by intention
	//nolint:govet
	return &Lite{*tbl}
}

// Deprecated: update is pointless without payload and panics.
func (l *Lite) Update() {
	panic("update is pointless without payload")
}

// Deprecated: update is pointless without payload and panics.
func (l *Lite) UpdatePersist() {
	panic("update is pointless without payload")
}
