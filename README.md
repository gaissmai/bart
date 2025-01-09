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

| ![example 4bit stride](https://github.com/user-attachments/assets/3b9f22c1-cfcc-4e01-bb13-864713ce5ae8) |
|:--:|
| *example from artlookup.pdf for a 4bit stride* |

The CBT is implemented as a bit-vector, backtracking is just
a matter of fast cache friendly bitmask operations.

The Table is implemented with popcount compressed sparse arrays
together with path compression. This reduces storage consumption
by almost two orders of magnitude in comparison to ART with
similar lookup times for the longest prefix match.

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
- to Daniel Lemire, and
- to Donald E. Knuth for the **ART** routing algorithm and

all the rest of his *Art* and for keeping important algorithms
in the public domain!

And last but not least to the Go team who do a wonderful job!

## LICENSE

MIT
