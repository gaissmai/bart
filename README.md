# package bart

[![Go Reference](https://pkg.go.dev/badge/github.com/gaissmai/bart.svg)](https://pkg.go.dev/github.com/gaissmai/bart#section-documentation)
![GitHub release (latest SemVer)](https://img.shields.io/github/v/release/gaissmai/bart)
[![CI](https://github.com/gaissmai/bart/actions/workflows/go.yml/badge.svg)](https://github.com/gaissmai/bart/actions/workflows/go.yml)
[![Coverage Status](https://coveralls.io/repos/github/gaissmai/bart/badge.svg)](https://coveralls.io/github/gaissmai/bart)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)
[![Stand With Ukraine](https://raw.githubusercontent.com/vshymanskyy/StandWithUkraine/main/badges/StandWithUkraine.svg)](https://stand-with-ukraine.pp.ua)

## ATTENTION: API change!!!

API change, Walk() got a signatur change and renamded to All(), ready for range-over-func iterations.

## Overview

`package bart` provides a Balanced-Routing-Table (BART).

BART is balanced in terms of memory consumption versus
lookup time.

The lookup time is by a factor of ~2 slower on average as the
routing algorithms ART, SMART, CPE, ... but reduces the memory
consumption by an order of magnitude in comparison.

BART is a multibit-trie with fixed stride length of 8 bits,
using the _baseIndex_ function from the ART algorithm to
build the complete-binary-tree (CBT) of prefixes for each stride.

The second key factor is popcount array compression at each stride level
of the CBT prefix tree and backtracking along the CBT in O(k).

The CBT is implemented as a bitvector, backtracking is just
a matter of fast cache friendly bitmask operations.

The child array at each stride level is also popcount compressed.

## API

The API has changed since v0.4.2 and 0.5.3.

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
  func (t *Table[V]) Delete(pfx netip.Prefix)
  func (t *Table[V]) Get(pfx netip.Prefix) (val V, ok bool)
  func (t *Table[V]) Update(pfx netip.Prefix, cb func(val V, ok bool) V) V

  func (t *Table[V]) Union(o *Table[V])
  func (t *Table[V]) Clone() *Table[V]
  
  func (t *Table[V]) Lookup(ip netip.Addr) (val V, ok bool)
  func (t *Table[V]) LookupPrefix(pfx netip.Prefix) (val V, ok bool)
  func (t *Table[V]) LookupPrefixLPM(pfx netip.Prefix) (lpm netip.Prefix, val V, ok bool)

  func (t *Table[V]) Subnets(pfx netip.Prefix) []netip.Prefix
  func (t *Table[V]) Supernets(pfx netip.Prefix) []netip.Prefix

  func (t *Table[V]) Overlaps(o *Table[V]) bool
  func (t *Table[V]) OverlapsPrefix(pfx netip.Prefix) bool
  
  func (t *Table[V]) String() string
  func (t *Table[V]) Fprint(w io.Writer) error
  func (t *Table[V]) MarshalText() ([]byte, error)
  func (t *Table[V]) MarshalJSON() ([]byte, error)

  func (t *Table[V]) All(yield func(pfx netip.Prefix, val V) bool) bool
  func (t *Table[V]) All4(yield func(pfx netip.Prefix, val V) bool) bool
  func (t *Table[V]) All6(yield func(pfx netip.Prefix, val V) bool) bool

  func (t *Table[V]) DumpList4() []DumpListNode[V]
  func (t *Table[V]) DumpList6() []DumpListNode[V]
```

## benchmarks

Please see the extensive [benchmarks](https://github.com/gaissmai/iprbench) comparing `bart` with other IP routing table implementations.

Just a teaser, LPM lookups against the full Internet routing table with random probes:

```
cpu: Intel(R) Core(TM) i5-8250U CPU @ 1.60GHz

BenchmarkFullMatchV4/Lookup                  28140715        41.95 ns/op
BenchmarkFullMatchV4/LookupPrefix            24648212        48.73 ns/op
BenchmarkFullMatchV4/LookupPrefixLPM         21412228        56.06 ns/op

BenchmarkFullMatchV6/Lookup                  29225397        41.06 ns/op
BenchmarkFullMatchV6/LookupPrefix            24992281        48.01 ns/op
BenchmarkFullMatchV6/LookupPrefixLPM         21743133        55.25 ns/op

BenchmarkFullMissV4/Lookup                   15246050        78.84 ns/op
BenchmarkFullMissV4/LookupPrefix             13382380        89.76 ns/op
BenchmarkFullMissV4/LookupPrefixLPM          12887918        93.09 ns/op

BenchmarkFullMissV6/Lookup                   69248640        17.31 ns/op
BenchmarkFullMissV6/LookupPrefix             51542642        23.29 ns/op
BenchmarkFullMissV6/LookupPrefixLPM          48444040        24.79 ns/op
```

## CONTRIBUTION

Please open an issue for discussion before sending a pull request.

## CREDIT

Credits for many inspirations go to the clever guys at tailscale,
to Daniel Lemire for the super-fast bitset package and
to Donald E. Knuth for the **ART** routing algorithm and
all the rest of his *Art* and for keeping important algorithms
in the public domain!

## LICENSE

MIT
