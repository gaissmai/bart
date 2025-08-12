package prefix

import (
	"bytes"
	"net/netip"

	"github.com/gaissmai/bart/internal/art"
)

// prefixBounds is a compact representation of a network prefix (netip.Prefix)
// optimized for fast containment checks.
//
// Storage details:
//   - ip: the first bytes contain the IP address (4 bytes for IPv4, 16 bytes for IPv6).
//     The array is always 16 bytes long, but unused tail bytes are left as zero.
//   - bits: prefix length in bits (CIDR notation).
//   - lastIdx: index of the last byte in the address that is affected by the prefix mask.
//   - lower: the minimum allowed value for the byte at lastIdx according to the mask.
//   - upper: the maximum allowed value for the byte at lastIdx according to the mask.
//
// For IPv4 addresses:
// - only the first 4 bytes of ip are filled
// - contains() receives an slice of exactly 4 bytes for IPv4
//
// This format avoids re-computing mask logic in contains(), enabling the Go compiler
// to inline the method for maximum speed.
type prefixBounds struct {
	ip    [16]byte // address bytes, only first 4 are used for IPv4
	bits  uint8    // prefix length in bits
	lower uint8    // min value allowed at lastIdx
	upper uint8    // max value allowed at lastIdx
	is4   bool
}

// newPrefixBounds converts a netip.Prefix into the compact tinyPrefix format.
//
// Steps:
// - Get the address bytes from p (4 bytes for IPv4, 16 bytes for IPv6).
// - Store those bytes in the beginning of ip (rest of ip is zero).
// - Compute lastIdx: the byte position of the last byte covered by the prefix mask.
// - Compute lastBits: how many bits are used in that final masked byte.
// - Use art.NetMask(lastBits) to build the mask and calculate lower and upper bounds.
//
// All expensive calculations are performed here so that contains() can be minimal.
func newPrefixBounds(p netip.Prefix) *prefixBounds {
	pfx := new(prefixBounds)

	ip := p.Addr()          // IP address from prefix
	bits := uint8(p.Bits()) // prefix length in bits
	octets := ip.AsSlice()  // 4 bytes if IPv4, 16 bytes if IPv6

	pfx.is4 = ip.Is4()
	pfx.bits = bits

	lastIdx := (bits - 1) >> 3        // divide by 8 to get index of last covered byte
	lastBits := bits - (lastIdx << 3) // bits in that last byte

	copy(pfx.ip[:], octets) // copy address bytes into pfx.ip

	// lower bound: preserve only the network bits in the last covered byte
	pfx.lower = octets[lastIdx] & art.NetMask(lastBits)

	// upper bound: network bits fixed, remaining bits set to 1
	pfx.upper = octets[lastIdx] | ^art.NetMask(lastBits)

	return pfx
}

// contains determines whether the given address (as octets) is inside this network prefix.
//
// Rules:
//   - Compare bytes before lastIdx for exact equality.
//   - Check that the byte at lastIdx is >= lower and <= upper bounds.
//
// For IPv4 prefixes:
//   - octets must have length 4
//   - pfx.ip only has the first 4 bytes populated
//   - comparison only involves those bytes
//
// This makes the method very fast and suitable for compiler inlining.
func (pfx *prefixBounds) contains(octets []byte) bool {
	lastIdx := (pfx.bits - 1) >> 3

	return bytes.Equal(pfx.ip[:lastIdx], octets[:lastIdx]) &&
		octets[lastIdx] >= pfx.lower &&
		octets[lastIdx] <= pfx.upper
}
