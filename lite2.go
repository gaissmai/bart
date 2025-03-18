package bart

import "net/netip"

type Lite2 struct {
	Table[struct{}]
}

func (l *Lite2) Insert(pfx netip.Prefix) {
	l.Table.Insert(pfx, struct{}{})
}
