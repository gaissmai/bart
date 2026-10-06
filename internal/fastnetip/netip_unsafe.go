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
func Contains4(pfx netip.Prefix, ip netip.Addr) bool {
	ip4 := (*Addr)(unsafe.Pointer(&ip))
	p := (*Prefix)(unsafe.Pointer(&pfx))

	bits := p.bitsPlusOne - 1
	return uint32((ip4.addr.lo^p.ip.addr.lo)>>((32-bits)&63)) == 0
}

// Contains6 reports whether pfx contains ip.
//
// Both pfx and ip must be valid IPv6 instances. When compiled with unsafe
// optimizations, family, validity, and zone checks are bypassed for speed.
func Contains6(pfx netip.Prefix, ip6 netip.Addr) bool {
	ip := (*Addr)(unsafe.Pointer(&ip6))
	p := (*Prefix)(unsafe.Pointer(&pfx))
	bits := int(p.bitsPlusOne - 1)
	return ip.addr.xor(p.ip.addr).and(mask6[bits]).isZero()
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

// mask6 is a precalculated to keep Contains6 inlined
var mask6 = [129]uint128{
	0:   {0x0000000000000000, 0x0000000000000000},
	1:   {0x8000000000000000, 0x0000000000000000},
	2:   {0xc000000000000000, 0x0000000000000000},
	3:   {0xe000000000000000, 0x0000000000000000},
	4:   {0xf000000000000000, 0x0000000000000000},
	5:   {0xf800000000000000, 0x0000000000000000},
	6:   {0xfc00000000000000, 0x0000000000000000},
	7:   {0xfe00000000000000, 0x0000000000000000},
	8:   {0xff00000000000000, 0x0000000000000000},
	9:   {0xff80000000000000, 0x0000000000000000},
	10:  {0xffc0000000000000, 0x0000000000000000},
	11:  {0xffe0000000000000, 0x0000000000000000},
	12:  {0xfff0000000000000, 0x0000000000000000},
	13:  {0xfff8000000000000, 0x0000000000000000},
	14:  {0xfffc000000000000, 0x0000000000000000},
	15:  {0xfffe000000000000, 0x0000000000000000},
	16:  {0xffff000000000000, 0x0000000000000000},
	17:  {0xffff800000000000, 0x0000000000000000},
	18:  {0xffffc00000000000, 0x0000000000000000},
	19:  {0xffffe00000000000, 0x0000000000000000},
	20:  {0xfffff00000000000, 0x0000000000000000},
	21:  {0xfffff80000000000, 0x0000000000000000},
	22:  {0xfffffc0000000000, 0x0000000000000000},
	23:  {0xfffffe0000000000, 0x0000000000000000},
	24:  {0xffffff0000000000, 0x0000000000000000},
	25:  {0xffffff8000000000, 0x0000000000000000},
	26:  {0xffffffc000000000, 0x0000000000000000},
	27:  {0xffffffe000000000, 0x0000000000000000},
	28:  {0xfffffff000000000, 0x0000000000000000},
	29:  {0xfffffff800000000, 0x0000000000000000},
	30:  {0xfffffffc00000000, 0x0000000000000000},
	31:  {0xfffffffe00000000, 0x0000000000000000},
	32:  {0xffffffff00000000, 0x0000000000000000},
	33:  {0xffffffff80000000, 0x0000000000000000},
	34:  {0xffffffffc0000000, 0x0000000000000000},
	35:  {0xffffffffe0000000, 0x0000000000000000},
	36:  {0xfffffffff0000000, 0x0000000000000000},
	37:  {0xfffffffff8000000, 0x0000000000000000},
	38:  {0xfffffffffc000000, 0x0000000000000000},
	39:  {0xfffffffffe000000, 0x0000000000000000},
	40:  {0xffffffffff000000, 0x0000000000000000},
	41:  {0xffffffffff800000, 0x0000000000000000},
	42:  {0xffffffffffc00000, 0x0000000000000000},
	43:  {0xffffffffffe00000, 0x0000000000000000},
	44:  {0xfffffffffff00000, 0x0000000000000000},
	45:  {0xfffffffffff80000, 0x0000000000000000},
	46:  {0xfffffffffffc0000, 0x0000000000000000},
	47:  {0xfffffffffffe0000, 0x0000000000000000},
	48:  {0xffffffffffff0000, 0x0000000000000000},
	49:  {0xffffffffffff8000, 0x0000000000000000},
	50:  {0xffffffffffffc000, 0x0000000000000000},
	51:  {0xffffffffffffe000, 0x0000000000000000},
	52:  {0xfffffffffffff000, 0x0000000000000000},
	53:  {0xfffffffffffff800, 0x0000000000000000},
	54:  {0xfffffffffffffc00, 0x0000000000000000},
	55:  {0xfffffffffffffe00, 0x0000000000000000},
	56:  {0xffffffffffffff00, 0x0000000000000000},
	57:  {0xffffffffffffff80, 0x0000000000000000},
	58:  {0xffffffffffffffc0, 0x0000000000000000},
	59:  {0xffffffffffffffe0, 0x0000000000000000},
	60:  {0xfffffffffffffff0, 0x0000000000000000},
	61:  {0xfffffffffffffff8, 0x0000000000000000},
	62:  {0xfffffffffffffffc, 0x0000000000000000},
	63:  {0xfffffffffffffffe, 0x0000000000000000},
	64:  {0xffffffffffffffff, 0x0000000000000000},
	65:  {0xffffffffffffffff, 0x8000000000000000},
	66:  {0xffffffffffffffff, 0xc000000000000000},
	67:  {0xffffffffffffffff, 0xe000000000000000},
	68:  {0xffffffffffffffff, 0xf000000000000000},
	69:  {0xffffffffffffffff, 0xf800000000000000},
	70:  {0xffffffffffffffff, 0xfc00000000000000},
	71:  {0xffffffffffffffff, 0xfe00000000000000},
	72:  {0xffffffffffffffff, 0xff00000000000000},
	73:  {0xffffffffffffffff, 0xff80000000000000},
	74:  {0xffffffffffffffff, 0xffc0000000000000},
	75:  {0xffffffffffffffff, 0xffe0000000000000},
	76:  {0xffffffffffffffff, 0xfff0000000000000},
	77:  {0xffffffffffffffff, 0xfff8000000000000},
	78:  {0xffffffffffffffff, 0xfffc000000000000},
	79:  {0xffffffffffffffff, 0xfffe000000000000},
	80:  {0xffffffffffffffff, 0xffff000000000000},
	81:  {0xffffffffffffffff, 0xffff800000000000},
	82:  {0xffffffffffffffff, 0xffffc00000000000},
	83:  {0xffffffffffffffff, 0xffffe00000000000},
	84:  {0xffffffffffffffff, 0xfffff00000000000},
	85:  {0xffffffffffffffff, 0xfffff80000000000},
	86:  {0xffffffffffffffff, 0xfffffc0000000000},
	87:  {0xffffffffffffffff, 0xfffffe0000000000},
	88:  {0xffffffffffffffff, 0xffffff0000000000},
	89:  {0xffffffffffffffff, 0xffffff8000000000},
	90:  {0xffffffffffffffff, 0xffffffc000000000},
	91:  {0xffffffffffffffff, 0xffffffe000000000},
	92:  {0xffffffffffffffff, 0xfffffff000000000},
	93:  {0xffffffffffffffff, 0xfffffff800000000},
	94:  {0xffffffffffffffff, 0xfffffffc00000000},
	95:  {0xffffffffffffffff, 0xfffffffe00000000},
	96:  {0xffffffffffffffff, 0xffffffff00000000},
	97:  {0xffffffffffffffff, 0xffffffff80000000},
	98:  {0xffffffffffffffff, 0xffffffffc0000000},
	99:  {0xffffffffffffffff, 0xffffffffe0000000},
	100: {0xffffffffffffffff, 0xfffffffff0000000},
	101: {0xffffffffffffffff, 0xfffffffff8000000},
	102: {0xffffffffffffffff, 0xfffffffffc000000},
	103: {0xffffffffffffffff, 0xfffffffffe000000},
	104: {0xffffffffffffffff, 0xffffffffff000000},
	105: {0xffffffffffffffff, 0xffffffffff800000},
	106: {0xffffffffffffffff, 0xffffffffffc00000},
	107: {0xffffffffffffffff, 0xffffffffffe00000},
	108: {0xffffffffffffffff, 0xfffffffffff00000},
	109: {0xffffffffffffffff, 0xfffffffffff80000},
	110: {0xffffffffffffffff, 0xfffffffffffc0000},
	111: {0xffffffffffffffff, 0xfffffffffffe0000},
	112: {0xffffffffffffffff, 0xffffffffffff0000},
	113: {0xffffffffffffffff, 0xffffffffffff8000},
	114: {0xffffffffffffffff, 0xffffffffffffc000},
	115: {0xffffffffffffffff, 0xffffffffffffe000},
	116: {0xffffffffffffffff, 0xfffffffffffff000},
	117: {0xffffffffffffffff, 0xfffffffffffff800},
	118: {0xffffffffffffffff, 0xfffffffffffffc00},
	119: {0xffffffffffffffff, 0xfffffffffffffe00},
	120: {0xffffffffffffffff, 0xffffffffffffff00},
	121: {0xffffffffffffffff, 0xffffffffffffff80},
	122: {0xffffffffffffffff, 0xffffffffffffffc0},
	123: {0xffffffffffffffff, 0xffffffffffffffe0},
	124: {0xffffffffffffffff, 0xfffffffffffffff0},
	125: {0xffffffffffffffff, 0xfffffffffffffff8},
	126: {0xffffffffffffffff, 0xfffffffffffffffc},
	127: {0xffffffffffffffff, 0xfffffffffffffffe},
	128: {0xffffffffffffffff, 0xffffffffffffffff},
}
