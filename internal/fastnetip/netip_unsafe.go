//go:build unsafe

// Package fastnetip provides high-performance, zero-allocation CIDR containment
// checks by mirroring the memory layout of net/netip.Addr and net/netip.Prefix.
//
// By using unsafe pointer casting and separating logic into IP family-specific
// functions (IPv4 vs IPv6), this package bypasses runtime validity checks,
// zone validations, and branch mispredictions, enabling full compiler inlining.
//
// Prerequisites:
// Callers must ensure that the provided netip.Prefix and netip.Addr pointers
// are valid, non-zero, unzoned, and match the corresponding IP version family
// (IPv4 for Contains4, IPv6 for Contains6).
package fastnetip

import (
	"net/netip"
	"unsafe"
)

// uint128 represents an unsigned 128-bit integer split across two 64-bit words (hi and lo).
// It mirrors the unexported uint128 type used internally by net/netip.
type uint128 struct {
	hi uint64
	lo uint64
}

// Addr mirrors the memory layout of net/netip.Addr.
type Addr struct {
	addr uint128
	z    uintptr // Mirrors unique.Handle[addrDetail]
}

// Prefix mirrors the memory layout of net/netip.Prefix.
type Prefix struct {
	ip          Addr
	bitsPlusOne uint8
}

// Contains4 reports whether the IPv4 prefix pfx contains the IPv4 address ip4.
//
// It performs no validity, family, or zone checks. The caller must guarantee
// that both pfx and ip4 are valid IPv4 instances.
//
//nolint:gosec // G115: integer overflow conversion uint64 -> uint32
func Contains4(pfx *netip.Prefix, ip4 *netip.Addr) bool {
	ip := (*Addr)(unsafe.Pointer(ip4))
	p := (*Prefix)(unsafe.Pointer(pfx))

	bits := p.bitsPlusOne - 1
	return uint32((ip.addr.lo^p.ip.addr.lo)>>((32-bits)&63)) == 0
}

// Contains6 reports whether the IPv6 prefix pfx contains the IPv6 address ip6.
//
// It performs no validity, family, or zone checks. The caller must guarantee
// that both pfx and ip6 are valid IPv6 instances without scoping zones.
func Contains6(pfx *netip.Prefix, ip6 *netip.Addr) bool {
	ip := (*Addr)(unsafe.Pointer(ip6))
	p := (*Prefix)(unsafe.Pointer(pfx))
	bits := int(p.bitsPlusOne - 1)
	return ip.addr.xor(p.ip.addr).and(mask6(bits)).isZero()
}

// ##################################################################################

// isZero reports whether u represents the value zero.
func (u uint128) isZero() bool {
	return u.hi|u.lo == 0
}

// and returns the bitwise AND operation of u and v.
func (u uint128) and(v uint128) uint128 {
	return uint128{
		hi: u.hi & v.hi,
		lo: u.lo & v.lo,
	}
}

// xor returns the bitwise XOR operation of u and v.
func (u uint128) xor(v uint128) uint128 {
	return uint128{
		hi: u.hi ^ v.hi,
		lo: u.lo ^ v.lo,
	}
}

// mask6 returns a 128-bit mask with the topmost n bits set to 1.
func mask6(n int) uint128 {
	return uint128{
		hi: ^(^uint64(0) >> n),
		lo: ^uint64(0) << (128 - n),
	}
}
