package fastnetip

import (
	"net/netip"
	"unsafe"
)

type uint128 struct {
	hi uint64
	lo uint64
}

type Addr struct {
	addr uint128
	z    uintptr
}

type Prefix struct {
	ip          Addr
	bitsPlusOne uint8
}

//nolint:gosec // G115: integer overflow conversion uint64 -> uint32
func Contains4(pfx *netip.Prefix, ip4 *netip.Addr) bool {
	ip := (*Addr)(unsafe.Pointer(ip4))
	p := (*Prefix)(unsafe.Pointer(pfx))

	bits := p.bitsPlusOne - 1
	return uint32((ip.addr.lo^p.ip.addr.lo)>>((32-bits)&63)) == 0
}

func Contains6(pfx *netip.Prefix, ip6 *netip.Addr) bool {
	ip := (*Addr)(unsafe.Pointer(ip6))
	p := (*Prefix)(unsafe.Pointer(pfx))

	return ip.addr.xor(p.ip.addr).and(mask6(int(p.bitsPlusOne) - 1)).isZero()
}

// ##################################################################################

func (u uint128) isZero() bool {
	return u.hi|u.lo == 0
}

func (u uint128) and(v uint128) uint128 {
	return uint128{
		hi: u.hi & v.hi,
		lo: u.lo & v.lo,
	}
}

func (u uint128) xor(v uint128) uint128 {
	return uint128{
		hi: u.hi ^ v.hi,
		lo: u.lo ^ v.lo,
	}
}

func mask6(n int) uint128 {
	return uint128{
		hi: ^(^uint64(0) >> n),
		lo: ^uint64(0) << (128 - n),
	}
}
