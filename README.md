# package bart

![GitHub release (latest SemVer)](https://img.shields.io/github/v/release/gaissmai/bart)
[![Go Reference](https://pkg.go.dev/badge/github.com/gaissmai/bart.svg)](https://pkg.go.dev/github.com/gaissmai/bart#section-documentation)
[![Mentioned in Awesome Go](https://awesome.re/mentioned-badge-flat.svg)](https://github.com/avelino/awesome-go)
[![CI](https://github.com/gaissmai/bart/actions/workflows/go.yml/badge.svg)](https://github.com/gaissmai/bart/actions/workflows/go.yml)
[![Coverage Status](https://coveralls.io/repos/github/gaissmai/bart/badge.svg)](https://coveralls.io/github/gaissmai/bart)
[![Go Report Card](https://goreportcard.com/badge/github.com/gaissmai/bart)](https://goreportcard.com/report/github.com/gaissmai/bart)
[![License: MIT](https://img.shields.io/badge/License-MIT-green.svg)](https://opensource.org/licenses/MIT)
[![Stand With Ukraine](https://raw.githubusercontent.com/vshymanskyy/StandWithUkraine/main/badges/StandWithUkraine.svg)](https://stand-with-ukraine.pp.ua)

## Overview

`package bart` provides a Balanced-Routing-Table (BART).

BART is balanced in terms of memory usage and lookup time
for the longest-prefix match.

BART is a multibit-trie with fixed stride length of 8 bits,
using the _baseIndex_ function from the ART algorithm to
build the complete-binary-tree (CBT) of prefixes for each stride.

The CBT is implemented as a bit-vector, backtracking is just
a matter of fast cache friendly bitmask operations.

The Table is implemented with popcount compressed sparse arrays
together with path compression. This reduces storage consumption
by almost two orders of magnitude in comparison to ART with
comparable or even better lookup times for longest prefix match.

The algorithm is also excellent for determining whether two tables
contain overlapping IP addresses.
All this happens within nanoseconds without memory allocation.

## Example

```golang
func ExampleTable_Contains() {
	// Create a new routing table
	table := new(bart.Table[struct{}])

	// Insert some prefixes
	prefixes := []string{
		"192.168.0.0/16",       // corporate
		"192.168.1.0/24",       // department
		"2001:7c0:3100::/40",   // corporate
		"2001:7c0:3100:1::/64", // department
		"fc00::/7",             // unique local
	}

	for _, s := range prefixes {
		pfx := netip.MustParsePrefix(s)
		table.Insert(pfx, struct{}{})
	}

	// Test some IP addresses for black/whitelist containment
	ips := []string{
		"192.168.1.100",      // must match, department
		"192.168.2.1",        // must match, corporate
		"2001:7c0:3100:1::1", // must match, department
		"2001:7c0:3100:2::1", // must match, corporate
		"fc00::1",            // must match, unique local
		//
		"172.16.0.1",        // must NOT match
		"2003:dead:beef::1", // must NOT match
	}

	for _, s := range ips {
		ip := netip.MustParseAddr(s)
		fmt.Printf("%-20s is contained: %t\n", ip, table.Contains(ip))
	}

	// Output:
	// 192.168.1.100        is contained: true
	// 192.168.2.1          is contained: true
	// 2001:7c0:3100:1::1   is contained: true
	// 2001:7c0:3100:2::1   is contained: true
	// fc00::1              is contained: true
	// 172.16.0.1           is contained: false
	// 2003:dead:beef::1    is contained: false
}
```
## API

The API has changed in ..., v0.10.1, v0.11.0, v0.12.0, v0.12.6, v0.16.0

```golang
  import "github.com/gaissmai/bart"
  
  type Table[V any] struct {
  	// Has unexported fields.
  }
    Table is an IPv4 and IPv6 routing table with payload V. The zero value is
    ready to use.

    The Table is safe for concurrent readers but not for concurrent readers
    and/or writers.

  func (t *Table[V]) Insert(pfx netip.Prefix, val V)
  func (t *Table[V]) Update(pfx netip.Prefix, cb func(val V, ok bool) V) (newVal V)
  func (t *Table[V]) Delete(pfx netip.Prefix)

  func (t *Table[V]) Get(pfx netip.Prefix) (val V, ok bool)
  func (t *Table[V]) GetAndDelete(pfx netip.Prefix) (val V, ok bool)

  func (t *Table[V]) Union(o *Table[V])
  func (t *Table[V]) Clone() *Table[V]

  func (t *Table[V]) Contains(ip netip.Addr) bool
  func (t *Table[V]) Lookup(ip netip.Addr) (val V, ok bool)
  func (t *Table[V]) LookupPrefix(pfx netip.Prefix) (val V, ok bool)
  func (t *Table[V]) LookupPrefixLPM(pfx netip.Prefix) (lpm netip.Prefix, val V, ok bool)

  func (t *Table[V]) OverlapsPrefix(pfx netip.Prefix) bool

  func (t *Table[V]) Overlaps(o *Table[V])  bool
  func (t *Table[V]) Overlaps4(o *Table[V]) bool
  func (t *Table[V]) Overlaps6(o *Table[V]) bool

  func (t *Table[V]) Subnets(pfx netip.Prefix)   func(yield func(netip.Prefix, V) bool)
  func (t *Table[V]) Supernets(pfx netip.Prefix) func(yield func(netip.Prefix, V) bool)

  func (t *Table[V]) All()  func(yield func(pfx netip.Prefix, val V) bool)
  func (t *Table[V]) All4() func(yield func(pfx netip.Prefix, val V) bool)
  func (t *Table[V]) All6() func(yield func(pfx netip.Prefix, val V) bool)

  func (t *Table[V]) AllSorted()  func(yield func(pfx netip.Prefix, val V) bool)
  func (t *Table[V]) AllSorted4() func(yield func(pfx netip.Prefix, val V) bool)
  func (t *Table[V]) AllSorted6() func(yield func(pfx netip.Prefix, val V) bool)

  func (t *Table[V]) Size()  int
  func (t *Table[V]) Size4() int
  func (t *Table[V]) Size6() int

  func (t *Table[V]) String() string
  func (t *Table[V]) Fprint(w io.Writer) error
  func (t *Table[V]) MarshalText() ([]byte, error)
  func (t *Table[V]) MarshalJSON() ([]byte, error)

  func (t *Table[V]) DumpList4() []DumpListNode[V]
  func (t *Table[V]) DumpList6() []DumpListNode[V]
```

## benchmarks

Please see the extensive [benchmarks](https://github.com/gaissmai/iprbench) comparing `bart` with other IP routing table implementations.

Just a teaser, Contains and Lookups against the full Internet routing table with random IP address probes:

```
goos: linux
goarch: amd64
pkg: github.com/gaissmai/bart
cpu: Intel(R) Core(TM) i5-7500T CPU @ 2.70GHz
BenchmarkFullMatchV4/Contains    49814167        23.22  ns/op   0 B/op   0 allocs/op
BenchmarkFullMatchV6/Contains    94662561        11.90  ns/op   0 B/op   0 allocs/op
BenchmarkFullMissV4/Contains     46916434        24.32  ns/op   0 B/op   0 allocs/op
BenchmarkFullMissV6/Contains     239470936        5.023 ns/op   0 B/op   0 allocs/op
PASS
ok  	github.com/gaissmai/bart	15.343s

goos: linux
goarch: amd64
pkg: github.com/gaissmai/bart
cpu: Intel(R) Core(TM) i5-7500T CPU @ 2.70GHz
BenchmarkFullMatchV4/Lookup      52110546        22.65  ns/op   0 B/op   0 allocs/op
BenchmarkFullMatchV6/Lookup      52083624        22.09  ns/op   0 B/op   0 allocs/op
BenchmarkFullMissV4/Lookup       40740790        27.80  ns/op   0 B/op   0 allocs/op
BenchmarkFullMissV6/Lookup       148526529        8.076 ns/op   0 B/op   0 allocs/op
PASS
ok  	github.com/gaissmai/bart	15.646s
```

and Overlaps with randomly generated tables of different size:
```
goos: linux
goarch: amd64
pkg: github.com/gaissmai/bart
cpu: Intel(R) Core(TM) i5-8250U CPU @ 1.60GHz
BenchmarkFullTableOverlapsV4/With____1           9086344     123.5   ns/op    0 B/op    0 allocs/op
BenchmarkFullTableOverlapsV4/With____2          68859405      17.27  ns/op    0 B/op    0 allocs/op
BenchmarkFullTableOverlapsV4/With____4          68697332      17.29  ns/op    0 B/op    0 allocs/op
BenchmarkFullTableOverlapsV4/With____8           6341209     189.8   ns/op    0 B/op    0 allocs/op
BenchmarkFullTableOverlapsV4/With___16           5453186     221.0   ns/op    0 B/op    0 allocs/op
BenchmarkFullTableOverlapsV4/With___32          58935297      20.47  ns/op    0 B/op    0 allocs/op
BenchmarkFullTableOverlapsV4/With___64          43856942      27.76  ns/op    0 B/op    0 allocs/op
BenchmarkFullTableOverlapsV4/With__128          42872038      27.63  ns/op    0 B/op    0 allocs/op
BenchmarkFullTableOverlapsV4/With__256          42910443      27.62  ns/op    0 B/op    0 allocs/op
BenchmarkFullTableOverlapsV4/With__512          126998767      9.362 ns/op    0 B/op    0 allocs/op
BenchmarkFullTableOverlapsV4/With_1024          128460864      9.363 ns/op    0 B/op    0 allocs/op
BenchmarkFullTableOverlapsV6/With____1          146886393      8.216 ns/op    0 B/op    0 allocs/op
BenchmarkFullTableOverlapsV6/With____2          146285103      8.183 ns/op    0 B/op    0 allocs/op
BenchmarkFullTableOverlapsV6/With____4          18488910      64.98  ns/op    0 B/op    0 allocs/op
BenchmarkFullTableOverlapsV6/With____8          144183597      8.258 ns/op    0 B/op    0 allocs/op
BenchmarkFullTableOverlapsV6/With___16          14775404      80.97  ns/op    0 B/op    0 allocs/op
BenchmarkFullTableOverlapsV6/With___32          21450390      55.98  ns/op    0 B/op    0 allocs/op
BenchmarkFullTableOverlapsV6/With___64          23702264      51.76  ns/op    0 B/op    0 allocs/op
BenchmarkFullTableOverlapsV6/With__128          22386841      53.63  ns/op    0 B/op    0 allocs/op
BenchmarkFullTableOverlapsV6/With__256          22390033      54.09  ns/op    0 B/op    0 allocs/op
BenchmarkFullTableOverlapsV6/With__512          22338945      53.57  ns/op    0 B/op    0 allocs/op
BenchmarkFullTableOverlapsV6/With_1024          22369528      53.67  ns/op    0 B/op    0 allocs/op
PASS
ok      github.com/gaissmai/bart    48.594s
```

## Compatibility Guarantees

The package is currently released as a pre-v1 version, which gives the author the freedom to break
backward compatibility to help improve the API as he learns which initial design decisions would need
to be revisited to better support the use cases that the library solves for.

These occurrences are expected to be rare in frequency and the API is already quite stable.

## CONTRIBUTION

Please open an issue for discussion before sending a pull request.

## CREDIT

Standing on the shoulders of giants.

Credits for many inspirations go to

- the clever guys at tailscale,
- to Daniel Lemire for his inspiring blog,
- to Donald E. Knuth for the **ART** routing algorithm and
- to Yoichi Hariguchi who deciphered it for us mere mortals

And last but not least to the Go team who do a wonderful job!

## LICENSE

MIT
