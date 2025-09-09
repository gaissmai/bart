![GitHub release (latest SemVer)](https://img.shields.io/github/v/release/gaissmai/bart)
[![Go Reference](https://pkg.go.dev/badge/github.com/gaissmai/bart.svg)](https://pkg.go.dev/github.com/gaissmai/bart#section-documentation)
[![Mentioned in Awesome Go](https://awesome.re/mentioned-badge-flat.svg)](https://github.com/avelino/awesome-go)
[![CI](https://github.com/gaissmai/bart/actions/workflows/go.yml/badge.svg)](https://github.com/gaissmai/bart/actions/workflows/go.yml)
[![Coverage Status](https://coveralls.io/repos/github/gaissmai/bart/badge.svg)](https://coveralls.io/github/gaissmai/bart)
[![Go Report Card](https://goreportcard.com/badge/github.com/gaissmai/bart)](https://goreportcard.com/report/github.com/gaissmai/bart)
[![License: MIT](https://img.shields.io/badge/License-MIT-green.svg)](https://opensource.org/licenses/MIT)
[![Stand With Ukraine](https://raw.githubusercontent.com/vshymanskyy/StandWithUkraine/main/badges/StandWithUkraine.svg)](https://stand-with-ukraine.pp.ua)

# package bart

The bart package provides a **Balanced Routing Table (BART)** for extremely
fast IP-to-CIDR lookups and related tasks such as:

- **ACL** fast checks, quickly determine whether an IP address matches any of millions of CIDR rules.
- **RIB** efficient storage, handle very large routing tables with low memory overhead, while keeping lookups fast.
- **FIB** high-speed lookups, achieve LPM in constant-time for packet forwarding in the datapath.

BART is designed for workloads where both speed and memory efficiency matter,
making it a good fit for firewalls, routers, or any system that needs large-scale
IP prefix matching.

## Overview

BART is balanced in terms of both memory usage and lookup time for longest-prefix matches.
It is implemented as a multibit trie with a fixed stride of 8 bits, using a fast mapping
function derived from Donald E. Knuth’s **Allotment Routing Table** (ART) algorithm, to map
the possible prefixes at each level into a complete binary tree.

This binary tree is represented with popcount‑compressed sparse arrays for **level compression**.
Combined with a **novel path compression**, this design reduces memory consumption by nearly
two [orders of magnitude](https://github.com/gaissmai/iprbench) compared to ART,
while delivering even faster lookup times for prefix searches (see linked benchmarks).

## Usage and Compilation

Example: simple ACL

```go
package main

import (
  "net/netip"

  "github.com/gaissmai/bart"
)

func main() {
  // Simple ACL with bart.Lite
  allowlist := new(bart.Lite)

  // Add allowed networks
  allowlist.Insert(netip.MustParsePrefix("192.168.0.0/16"))
  allowlist.Insert(netip.MustParsePrefix("2001:db8::/32"))

  // Test some IPs
  testIPs := []netip.Addr{
    netip.MustParseAddr("192.168.1.100"), // allowed
    netip.MustParseAddr("2001:db8::1"),   // allowed
    netip.MustParseAddr("172.16.0.1"),    // denied
  }

  for _, ip := range testIPs {
    if allowlist.Contains(ip) {
      // ALLOWED
    } else {
      // DENIED
    }
  }
}
```

The BART algorithm is based on fixed-size bit vectors and precomputed lookup tables.
Lookups are executed entirely with fast, cache-resident bitmask operations, which
modern CPUs accelerate using specialized instructions such as POPCNT, LZCNT, and TZCNT.

For maximum performance, specify the CPU feature set when compiling.
See the [Go minimum requirements](https://go.dev/wiki/MinimumRequirements#architectures) for details.

```bash
# On ARM64, Go auto-selects CPU instructions.
# Example for AMD64, choose v2/v3/v4 to match your CPU features.
GOAMD64=v3 go build
```


## Bitset Efficiency

Due to the novel path compression, BART always operates on a fixed internal 256-bit length.
Critical loops over these bitsets can be unrolled for additional speed,
ensuring predictable memory access and efficient use of CPU pipelines.

```go
func (b *BitSet256) popcnt() (cnt int) {
	cnt += bits.OnesCount64(b[0])
	cnt += bits.OnesCount64(b[1])
	cnt += bits.OnesCount64(b[2])
	cnt += bits.OnesCount64(b[3])
	return
}
```

Future Go versions with SIMD intrinsics for `uint64` vectors may unlock
additional speedups on compatible hardware.

## Concurrency model

There are examples demonstrating how to use bart concurrently with multiple readers and writers.
Readers can always access the table lock‑free. Writers synchronize with a mutex so that only one writer
modifies the persistent table at a time, without relying on CAS, which can be problematic with multiple
long‑running writers.

The combination of lock-free concurrency, fast lookup and update times and low memory consumption
provides clear advantages for any routing daemon.

But as always, it depends on the specific use case.

See the concurrent tests for concrete examples of this pattern:
- [ExampleLite](example_lite_concurrent_test.go)
- [ExampleTable](example_table_concurrent_test.go)


## Additional Use Cases

Beyond high-performance prefix matching, BART also excels at detecting overlaps
between two routing tables.
In internal benchmarks `BenchmarkTableOverlaps` the check runs in a few nanoseconds
per query with zero heap allocations on a modern CPU.

## API

```go
  import "github.com/gaissmai/bart"
  
  type Table[V any] struct {
  	// Has unexported fields.
  }

  func (t *Table[V]) Contains(ip netip.Addr) bool
  func (t *Table[V]) Lookup(ip netip.Addr) (V, bool)

  func (t *Table[V]) LookupPrefix(pfx netip.Prefix) (V, bool)
  func (t *Table[V]) LookupPrefixLPM(pfx netip.Prefix) (netip.Prefix, V, bool)

  func (t *Table[V]) Insert(pfx netip.Prefix, val V)
  func (t *Table[V]) Modify(pfx netip.Prefix, func(V, bool) (V, bool)) (V, bool)
  func (t *Table[V]) Delete(pfx netip.Prefix) (V, bool)
  func (t *Table[V]) Get(pfx netip.Prefix) (V, bool)

  func (t *Table[V]) InsertPersist(pfx netip.Prefix, val V) *Table[V]
  func (t *Table[V]) ModifyPersist(pfx netip.Prefix, func(val V, ok bool) (V, bool)) (*Table[V], V, bool)
  func (t *Table[V]) DeletePersist(pfx netip.Prefix) (*Table[V], V, bool)
  func (t *Table[V]) WalkPersist(func(*Table[V], netip.Prefix, V) (*Table[V], bool)) *Table[V]


  func (t *Table[V]) Clone() *Table[V]
  func (t *Table[V]) Union(o *Table[V])
  func (t *Table[V]) UnionPersist(o *Table[V]) *Table[V]

  func (t *Table[V]) OverlapsPrefix(pfx netip.Prefix) bool

  func (t *Table[V]) Overlaps(o *Table[V])  bool
  func (t *Table[V]) Overlaps4(o *Table[V]) bool
  func (t *Table[V]) Overlaps6(o *Table[V]) bool

  func (t *Table[V]) Equal(o *Table[V])  bool

  func (t *Table[V]) Subnets(pfx netip.Prefix)   iter.Seq2[netip.Prefix, V]
  func (t *Table[V]) Supernets(pfx netip.Prefix) iter.Seq2[netip.Prefix, V]

  func (t *Table[V]) All()  iter.Seq2[netip.Prefix, V]
  func (t *Table[V]) All4() iter.Seq2[netip.Prefix, V]
  func (t *Table[V]) All6() iter.Seq2[netip.Prefix, V]

  func (t *Table[V]) AllSorted()  iter.Seq2[netip.Prefix, V]
  func (t *Table[V]) AllSorted4() iter.Seq2[netip.Prefix, V]
  func (t *Table[V]) AllSorted6() iter.Seq2[netip.Prefix, V]

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

A `bart.Lite` wrapper is also included, this is ideal for simple IP
ACLs (access-control-lists) with plain true/false results and no payload.
Lite is just a convenience wrapper for Table, instantiated with an empty
struct as payload.

Lite wraps or adapts some methods where needed and delegates almost all
other methods unmodified to the underlying Table.
Some delegated methods are pointless without a payload.

```go
   type Lite struct {
     Table[struct{}]
   }

   func (l *Lite) Exists(pfx netip.Prefix) bool
   func (l *Lite) Contains(ip netip.Addr) bool

   func (l *Lite) Insert(pfx netip.Prefix)
   func (l *Lite) Delete(pfx netip.Prefix) bool

   func (l *Lite) InsertPersist(pfx netip.Prefix) *Lite
   func (l *Lite) DeletePersist(pfx netip.Prefix) (*Lite, bool)
   func (l *Lite) WalkPersist(fn func(*Lite, netip.Prefix) (*Lite, bool)) *Lite

   func (l *Lite) Clone() *Lite
   func (l *Lite) Union(o *Lite)
   func (l *Lite) UnionPersist(o *Lite) *Lite

   func (l *Lite) Overlaps(o *Lite) bool
   func (l *Lite) Overlaps4(o *Lite) bool
   func (l *Lite) Overlaps6(o *Lite) bool

   func (l *Lite) Equal(o *Lite) bool
```

## Benchmarks

Please see the extensive [benchmarks](https://github.com/gaissmai/iprbench) comparing `bart` with other IP routing table implementations.

Just a teaser, `Contains` and `Lookup` against the Tier1 full Internet routing table with
random IP address probes:

```
$ GOAMD64=v3 go test -run=xxx -bench=FullM/Contains -cpu=1
goos: linux
goarch: amd64
pkg: github.com/gaissmai/bart
cpu: Intel(R) Core(TM) i5-8250U CPU @ 1.60GHz
BenchmarkFullMatch4/Contains        82013714	        13.59 ns/op
BenchmarkFullMatch6/Contains        64516006	        18.66 ns/op
BenchmarkFullMiss4/Contains         75341578	        15.94 ns/op
BenchmarkFullMiss6/Contains         148116180	         8.122 ns/op

$ GOAMD64=v3 go test -run=xxx -bench=FullM/Lookup -skip=/x -cpu=1
goos: linux
goarch: amd64
pkg: github.com/gaissmai/bart
cpu: Intel(R) Core(TM) i5-8250U CPU @ 1.60GHz
BenchmarkFullMatch4/Lookup         	54616323	        22.02 ns/op
BenchmarkFullMatch6/Lookup         	30073657	        39.98 ns/op
BenchmarkFullMiss4/Lookup          	55132899	        21.90 ns/op
BenchmarkFullMiss6/Lookup          	100000000	        11.12 ns/op
```

## Compatibility Guarantees

The package is currently released as a pre-v1 version, which gives the author the freedom to break
backward compatibility to help improve the API as he learns which initial design decisions would need
to be revisited to better support the use cases that the library solves for.

These occurrences are expected to be rare in frequency and the API is already quite stable.

## Contribution

Please open an issue for discussion before sending a pull request.

## Credit

Standing on the shoulders of giants.

Credits for many inspirations go to

- the clever folks at Tailscale,
- to Daniel Lemire for his inspiring blog,
- to Donald E. Knuth for the Allotment Routing Table (ART) algorithm and
- to Yoichi Hariguchi who deciphered it for us mere mortals

And last but not least to the Go team who do a wonderful job!

## LICENSE

MIT
