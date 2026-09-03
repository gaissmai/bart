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
	z    unsafe.Pointer // Mirrors unique.Handle[addrDetail]
}

// Prefix mirrors the memory layout of net/netip.Prefix.
type Prefix struct {
	ip          Addr
	bitsPlusOne uint8
}

// Contains4 reports whether pfx contains ip.
//
// Both pfx and ip must be valid IPv4 instances. When compiled with unsafe
// optimizations, family, validity, and zone checks are bypassed for speed.
//
//nolint:gosec // G115: integer overflow conversion uint64 -> uint32
func Contains4(pfx *netip.Prefix, ip *netip.Addr) bool {
	ip4 := (*Addr)(unsafe.Pointer(ip))
	p := (*Prefix)(unsafe.Pointer(pfx))

	bits := p.bitsPlusOne - 1
	return uint32((ip4.addr.lo^p.ip.addr.lo)>>((32-bits)&63)) == 0
}

// Contains6 reports whether pfx contains ip.
//
// Both pfx and ip must be valid IPv6 instances without scoping zones.
// When compiled with unsafe optimizations, family, validity, and zone checks
// are bypassed for speed.
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
