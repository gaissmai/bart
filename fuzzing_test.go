package bart

import (
	"math/rand/v2"
	"net/netip"
	"slices"
	"testing"

	"github.com/gaissmai/bart/internal/tests/golden"
)

// extractPfxs4 extracts as many valid IPv4 netip.Prefix values as possible
// from a raw, randomized byte slice.
func extractPfxs4(data []byte) []netip.Prefix {
	// Each IPv4 prefix requires 5 bytes: 4 for the IPv4 address + 1 for prefix depth (0-32).
	const chunkSize = 5

	maxEntries := len(data) / chunkSize
	if maxEntries == 0 {
		return nil
	}

	prefixes := make([]netip.Prefix, 0, maxEntries)

	for i := 0; i+chunkSize <= len(data); i += chunkSize {
		chunk := data[i : i+chunkSize]

		ip := netip.AddrFrom4([4]byte(chunk[:4]))
		bits := int(chunk[4]) % 33

		prefixes = append(prefixes, netip.PrefixFrom(ip, bits))
	}

	return prefixes
}

// extractPfxs6 extracts as many valid IPv6 netip.Prefix values as possible
// from a raw, randomized byte slice.
func extractPfxs6(data []byte) []netip.Prefix {
	// Each IPv6 prefix requires 17 bytes: 16 for the IPv6 address + 1 for prefix depth (0-128).
	const chunkSize = 17
	maxEntries := len(data) / chunkSize

	if maxEntries == 0 {
		return nil
	}

	prefixes := make([]netip.Prefix, 0, maxEntries)

	for i := 0; i+chunkSize <= len(data); i += chunkSize {
		chunk := data[i : i+chunkSize]

		ip := netip.AddrFrom16([16]byte(chunk[:16]))
		bits := int(chunk[16]) % 129

		prefixes = append(prefixes, netip.PrefixFrom(ip, bits))
	}

	return prefixes
}

var (
	seedIP4 = mpa("10.0.0.1")
	seedIP6 = mpa("2001:db8::1")

	seedIP = slices.Concat(
		seedIP4.AsSlice(),
		seedIP6.AsSlice(),
		[]byte("eth"),
	)

	seedPfx4 = slices.Concat(
		seedIP4.AsSlice(), []byte{32},
		seedIP4.AsSlice(), []byte{8},
	)

	seedPfx6 = slices.Concat(
		seedIP6.AsSlice(), []byte{96},
		seedIP6.AsSlice(), []byte{32},
	)
)

func FuzzFastACL_Contains_Differential(f *testing.F) {
	const maxData = 256
	// Seed corpus
	f.Add(seedIP, seedPfx4, seedPfx6)

	f.Fuzz(func(t *testing.T, ipData, pfx4Data, pfx6Data []byte) {
		t.Parallel()

		// Clamp payload if it exceeds the threshold.
		if len(pfx4Data) > maxData {
			pfx4Data = pfx4Data[:maxData]
		}
		if len(pfx6Data) > maxData {
			pfx6Data = pfx6Data[:maxData]
		}

		ipBuf := [23]byte{}
		copy(ipBuf[:], ipData)

		ip4 := netip.AddrFrom4([4]byte(ipBuf[:4]))
		ip6 := netip.AddrFrom16([16]byte(ipBuf[4:20]))

		// Determine random zone usage via a fast bitwise check on the first byte.
		if ipBuf[0]&1 == 1 {
			ip6 = ip6.WithZone(string(ipBuf[20:23]))
		}

		pfxs4 := extractPfxs4(pfx4Data)
		pfxs6 := extractPfxs6(pfx6Data)

		gold := new(golden.Table[any])
		facl := new(FastACL)

		for _, p := range slices.Concat(pfxs4, pfxs6) {
			gold.Insert(p, nil)
			facl.Insert(p)
		}

		goldOK := gold.Contains(ip4)
		faclOK := facl.Contains(ip4)

		if goldOK != faclOK {
			t.Fatalf("Differential mismatch for IPv4: Contains(%s) = %v, want %v", ip4, faclOK, goldOK)
		}

		goldOK = gold.Contains(ip6)
		faclOK = facl.Contains(ip6)

		if goldOK != faclOK {
			t.Fatalf("Differential mismatch for IPv6: Contains(%s) = %v, want %v", ip6, faclOK, goldOK)
		}

	})
}

func FuzzFastACL_ContainsPrefix_Differential(f *testing.F) {
	// Seed corpus with random byte stream payload.
	const maxData = 512
	f.Add(seedPfx4, seedPfx6)

	f.Fuzz(func(t *testing.T, pfx4Data, pfx6Data []byte) {
		t.Parallel()

		// Clamp payload if it exceeds the threshold.
		if len(pfx4Data) > maxData {
			pfx4Data = pfx4Data[:maxData]
		}
		if len(pfx6Data) > maxData {
			pfx6Data = pfx6Data[:maxData]
		}

		pfxs4 := extractPfxs4(pfx4Data)
		pfxs6 := extractPfxs6(pfx6Data)

		gold := new(golden.Table[any])
		facl := new(FastACL)

		var probe4, probe6 netip.Prefix
		if len(pfxs4) > 0 {
			probe4 = pfxs4[0]
			pfxs4 = pfxs4[1:]
		}
		if len(pfxs6) > 0 {
			probe6 = pfxs6[0]
			pfxs6 = pfxs6[1:]
		}

		for _, p := range slices.Concat(pfxs4, pfxs6) {
			gold.Insert(p, nil)
			facl.Insert(p)
		}

		_, goldOK := gold.LookupPrefix(probe4)
		faclOK := facl.ContainsPrefix(probe4)

		if goldOK != faclOK {
			t.Fatalf("Differential mismatch for IPv4: ContainsPrefix(%s) = %v, want %v", probe4, faclOK, goldOK)
		}

		_, goldOK = gold.LookupPrefix(probe6)
		faclOK = facl.ContainsPrefix(probe6)

		if goldOK != faclOK {
			t.Fatalf("Differential mismatch for IPv6: ContainsPrefix(%s) = %v, want %v", probe6, faclOK, goldOK)
		}

	})
}

func FuzzFastACL_LookupLPM_Differential(f *testing.F) {
	// Seed corpus with random byte stream payload.
	const maxData = 512
	f.Add(seedPfx4, seedPfx6)

	f.Fuzz(func(t *testing.T, pfx4Data, pfx6Data []byte) {
		t.Parallel()

		// Clamp payload if it exceeds the threshold.
		if len(pfx4Data) > maxData {
			pfx4Data = pfx4Data[:maxData]
		}
		if len(pfx6Data) > maxData {
			pfx6Data = pfx6Data[:maxData]
		}

		pfxs4 := extractPfxs4(pfx4Data)
		pfxs6 := extractPfxs6(pfx6Data)

		gold := new(golden.Table[any])
		facl := new(FastACL)

		var probe4, probe6 netip.Prefix
		if len(pfxs4) > 0 {
			probe4 = pfxs4[0]
			pfxs4 = pfxs4[1:]
		}
		if len(pfxs6) > 0 {
			probe6 = pfxs6[0]
			pfxs6 = pfxs6[1:]
		}

		for _, p := range slices.Concat(pfxs4, pfxs6) {
			gold.Insert(p, nil)
			facl.Insert(p)
		}

		goldLPM, _, goldOK := gold.LookupPrefixLPM(probe4)
		faclLPM, faclOK := facl.LookupPrefixLPM(probe4)

		if goldOK != faclOK || goldLPM != faclLPM {
			t.Fatalf("Differential mismatch for IPv4: LookupPrefixLPM(%s) = (%v, %v), want (%v, %v)",
				probe4, faclLPM, faclOK, goldLPM, goldOK)
		}

		goldLPM, _, goldOK = gold.LookupPrefixLPM(probe6)
		faclLPM, faclOK = facl.LookupPrefixLPM(probe6)

		if goldOK != faclOK || goldLPM != faclLPM {
			t.Fatalf("Differential mismatch for IPv6: LookupPrefixLPM(%s) = (%v, %v), want (%v, %v)",
				probe6, faclLPM, faclOK, goldLPM, goldOK)
		}

	})
}

func FuzzFastACL_Subnets_Differential(f *testing.F) {
	// Seed corpus with random byte stream payload.
	const maxData = 512
	f.Add(seedPfx4, seedPfx6)

	f.Fuzz(func(t *testing.T, pfx4Data, pfx6Data []byte) {
		t.Parallel()

		// Clamp payload if it exceeds the threshold.
		if len(pfx4Data) > maxData {
			pfx4Data = pfx4Data[:maxData]
		}
		if len(pfx6Data) > maxData {
			pfx6Data = pfx6Data[:maxData]
		}

		pfxs4 := extractPfxs4(pfx4Data)
		pfxs6 := extractPfxs6(pfx6Data)

		gold := new(golden.Table[any])
		facl := new(FastACL)

		var probe4, probe6 netip.Prefix
		if len(pfxs4) > 0 {
			probe4 = pfxs4[0]
			pfxs4 = pfxs4[1:]
		}
		if len(pfxs6) > 0 {
			probe6 = pfxs6[0]
			pfxs6 = pfxs6[1:]
		}

		for _, p := range slices.Concat(pfxs4, pfxs6) {
			gold.Insert(p, nil)
			facl.Insert(p)
		}

		goldSubnets := gold.Subnets(probe4)
		faclSubnets := slices.Collect(facl.Subnets(probe4))

		if !slices.Equal(goldSubnets, faclSubnets) {
			t.Fatalf("Differential mismatch for IPv4: Subnets(%s)", probe4)
		}

		goldSubnets = gold.Subnets(probe6)
		faclSubnets = slices.Collect(facl.Subnets(probe6))

		if !slices.Equal(goldSubnets, faclSubnets) {
			t.Fatalf("Differential mismatch for IPv6: Subnets(%s)", probe6)
		}

	})
}

func FuzzFastACL_Supernets_Differential(f *testing.F) {
	// Seed corpus with random byte stream payload.
	const maxData = 512
	f.Add(seedPfx4, seedPfx6)

	f.Fuzz(func(t *testing.T, pfx4Data, pfx6Data []byte) {
		t.Parallel()

		// Clamp payload if it exceeds the threshold.
		if len(pfx4Data) > maxData {
			pfx4Data = pfx4Data[:maxData]
		}
		if len(pfx6Data) > maxData {
			pfx6Data = pfx6Data[:maxData]
		}

		pfxs4 := extractPfxs4(pfx4Data)
		pfxs6 := extractPfxs6(pfx6Data)

		gold := new(golden.Table[any])
		facl := new(FastACL)

		var probe4, probe6 netip.Prefix
		if len(pfxs4) > 0 {
			probe4 = pfxs4[0]
			pfxs4 = pfxs4[1:]
		}
		if len(pfxs6) > 0 {
			probe6 = pfxs6[0]
			pfxs6 = pfxs6[1:]
		}

		for _, p := range slices.Concat(pfxs4, pfxs6) {
			gold.Insert(p, nil)
			facl.Insert(p)
		}

		goldSupernets := gold.Supernets(probe4)
		faclSupernets := slices.Collect(facl.Supernets(probe4))

		if !slices.Equal(goldSupernets, faclSupernets) {
			t.Fatalf("Differential mismatch for IPv4: Supernets(%s)", probe4)
		}

		goldSupernets = gold.Supernets(probe6)
		faclSupernets = slices.Collect(facl.Supernets(probe6))

		if !slices.Equal(goldSupernets, faclSupernets) {
			t.Fatalf("Differential mismatch for IPv6: Supernets(%s)", probe6)
		}

	})
}

func FuzzFastACL_Clone_Differential(f *testing.F) {
	const maxData = 256
	// Seed corpus
	f.Add(seedPfx4, seedPfx6)

	f.Fuzz(func(t *testing.T, pfx4Data, pfx6Data []byte) {
		t.Parallel()

		// Clamp payload if it exceeds the threshold.
		if len(pfx4Data) > maxData {
			pfx4Data = pfx4Data[:maxData]
		}
		if len(pfx6Data) > maxData {
			pfx6Data = pfx6Data[:maxData]
		}

		pfxs4 := extractPfxs4(pfx4Data)
		pfxs6 := extractPfxs6(pfx6Data)

		facl := new(FastACL)

		pfxs := slices.Concat(pfxs4, pfxs6)
		if len(pfxs) == 0 {
			return
		}

		for _, p := range pfxs {
			facl.Insert(p)
		}

		prng := rand.New(rand.NewPCG(42, 42))
		probe := pfxs[prng.IntN(len(pfxs))]

		clone := facl.Clone()

		if !clone.Equal(facl) {
			t.Fatal("Equal mismatch for: Clone()")
		}

		// delete the probe pfx
		clone.Delete(probe)
		if clone.Equal(facl) {
			t.Fatal("Equal after delete: Clone()")
		}
	})
}

func FuzzFastACL_Equal(f *testing.F) {
	const maxData = 4096
	// Seed corpus
	f.Add(seedPfx4, seedPfx6)

	f.Fuzz(func(t *testing.T, pfx4Data, pfx6Data []byte) {
		t.Parallel()

		// Clamp payload if it exceeds the threshold.
		if len(pfx4Data) > maxData {
			pfx4Data = pfx4Data[:maxData]
		}
		if len(pfx6Data) > maxData {
			pfx6Data = pfx6Data[:maxData]
		}

		pfxs4 := extractPfxs4(pfx4Data)
		pfxs6 := extractPfxs6(pfx6Data)

		facl := new(FastACL)

		pfxs := slices.Concat(pfxs4, pfxs6)
		if len(pfxs) == 0 {
			return
		}

		for _, p := range pfxs {
			facl.Insert(p)
		}

		pfxs4 = slices.Collect(facl.All4())
		pfxs6 = slices.Collect(facl.All6())

		facl1 := new(FastACL)
		facl2 := new(FastACL)

		mid4 := len(pfxs4) / 2
		mid6 := len(pfxs6) / 2

		for _, pfx := range slices.Concat(pfxs4[:mid4], pfxs6[:mid6]) {
			facl1.Insert(pfx)
		}
		for _, pfx := range slices.Concat(pfxs4[mid4:], pfxs6[mid6:]) {
			facl2.Insert(pfx)
		}

		if facl1.Equal(facl2) {
			t.Fatal("Equal after separating pfxs")
		}
	})
}
