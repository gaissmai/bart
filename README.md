![GitHub release (latest SemVer)](https://img.shields.io/github/v/release/gaissmai/bart)
[![Go Reference](https://pkg.go.dev/badge/github.com/gaissmai/bart.svg)](https://pkg.go.dev/github.com/gaissmai/bart#section-documentation)
[![Mentioned in Awesome Go](https://awesome.re/mentioned-badge-flat.svg)](https://github.com/avelino/awesome-go)
[![CI](https://github.com/gaissmai/bart/actions/workflows/go.yml/badge.svg)](https://github.com/gaissmai/bart/actions/workflows/go.yml)
[![Coverage Status](https://coveralls.io/repos/github/gaissmai/bart/badge.svg)](https://coveralls.io/github/gaissmai/bart)
[![Go Report Card](https://goreportcard.com/badge/github.com/gaissmai/bart)](https://goreportcard.com/report/github.com/gaissmai/bart)
[![License: MIT](https://img.shields.io/badge/License-MIT-green.svg)](https://opensource.org/licenses/MIT)
[![Stand With Ukraine](https://raw.githubusercontent.com/vshymanskyy/StandWithUkraine/main/badges/StandWithUkraine.svg)](https://stand-with-ukraine.pp.ua)

# package bart

The bart package provides some **Balanced Routing Tables (BART)** for
fastest IP-to-CIDR lookups and related tasks such as:

- **ACL** determine extremely fast whether an IP address matches any of millions of CIDR rules.
- **RIB** handle very large routing tables with low memory overhead, while keeping lookups fast.
- **FIB** high-speed lookups, achieve LPM in constant-time for packet forwarding in the datapath.

BART is designed for workloads where both speed and/or memory efficiency matter,
making it a best fit for firewalls, routers, or any system that needs large-scale
IP prefix matching.

## Overview

BART is implemented as a multibit trie with a fixed stride of 8 bits,
using a fast mapping function derived from Donald E. Knuth’s
**Allotment Routing Table** (ART) algorithm, to map the possible prefixes
at each level into a complete binary tree.

BART implements three different routing tables, each optimized for specific
use cases:
- **bart.Lite**
- **bart.Table**
- **bart.Fast**

For **bart.Table** this binary tree is represented with popcount‑compressed
sparse arrays for **level compression**.
Combined with a **novel path and fringe compression**, this design reduces
memory consumption by nearly two orders of magnitude compared to classical ART.

For **bart.Fast** this binary tree is represented with fixed arrays
without level compression (classical ART), but combined with the same
novel **path and fringe compression** from BART. This design
reduces memory consumption by more than an order of magnitude compared
to classical ART and thus makes ART usable in the first place for large
routing tables.

**bart.Lite** is a special form of **bart.Table**, but without a payload, and therefore
has the lowest memory overhead while maintaining the same lookup times.

## Comparison
 
 | Aspect | Table | Lite | Fast |
 |--------|-------------|-------------|-------------|
 | **Per-level Speed** | ⚡ **O(1)** | ⚡ **O(1)** | 🚀 **O(1), ~40% faster per level** |
 | **Overall Lookup** | O(trie_depth) | O(trie_depth) | O(trie_depth) |
 | **IPv4 Performance** | ~3 level traversals | ~3 level traversals | ~3 level traversals |
 | **IPv6 Performance** | ~6 level traversals | ~6 level traversals | ~6 level traversals |
 | **IPv6 vs IPv4** | ~2× slower | ~2× slower | ~2× slower |
 | **Memory** | efficient | very efficient | inefficient |

A more detailed description can be found [here](NODETYPES.md).

## When to Use Each Type

### 🎯 **bart.Table[V]** - The Balanced Choice                                                                        
- **Recommended** for most routing table use cases
- Near-optimal per-level performance with excellent memory efficiency
- Perfect balance for both IPv4 and IPv6 routing tables (use it for RIB)
 
### 🪶 **bart.Lite** - The Minimalist
- **Specialized** for prefix-only operations, no payload
- Same per-level performance as *bart.Table[V]* but 35% less memory
- Ideal for IPv4/IPv6 allowlists and set-based operations (use it for ACL)
 
### 🚀 **bart.Fast[V]** - The Performance Champion
- **40% faster per-level** when memory constraints allow
- Best choice for lookup-intensive applications (use it for FIB)

## Usage and Compilation

Example: simple ACL with bart.Lite

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


## Bitset Efficiency

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
Critical loops over these fixed-size bitsets can be unrolled for additional speed,
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
- [ExampleFast](example_fast_concurrent_test.go)


## Additional Use Cases

Beyond high-performance prefix matching, BART also excels at detecting overlaps
between two routing tables.
In internal benchmarks the check runs in a few nanoseconds per query with zero
heap allocations on a modern CPU.

## API

BART has a rich API for CRUD, lookup, comparison, iteration,
serialization and persistence. 

**Table** and **Fast** expose the identical API, while **Lite** deviates in
its methods from the common API when it comes to the payload, since *Lite*
has no payload.

```go
import "github.com/gaissmai/bart"

type Table[V any] struct {
	// Has unexported fields.
}

func (t *Table[V]) Contains(netip.Addr) bool
func (t *Table[V]) Lookup(netip.Addr) (V, bool)

func (t *Table[V]) LookupPrefix(netip.Prefix) (V, bool)
func (t *Table[V]) LookupPrefixLPM(netip.Prefix) (netip.Prefix, V, bool)

func (t *Table[V]) Insert(netip.Prefix, V)
func (t *Table[V]) Modify(netip.Prefix, cb func(_ V, bool) (_ V, bool)) (_ V, bool)
func (t *Table[V]) Delete(netip.Prefix) (V, exists bool)
func (t *Table[V]) Get(netip.Prefix) (V, exists bool)

func (t *Table[V]) InsertPersist(netip.Prefix, V) *Table[V]
func (t *Table[V]) ModifyPersist(netip.Prefix, cb func(_ V, bool) (_ V, bool)) (*Table[V], _ V, bool)
func (t *Table[V]) DeletePersist(netip.Prefix) (*Table[V], V, bool)
func (t *Table[V]) WalkPersist(fn func(*Table[V], netip.Prefix, V) (*Table[V], bool)) *Table[V]

func (t *Table[V]) Clone() *Table[V]
func (t *Table[V]) Union(o *Table[V])
func (t *Table[V]) UnionPersist(o *Table[V]) *Table[V]

func (t *Table[V]) OverlapsPrefix(netip.Prefix) bool

func (t *Table[V]) Overlaps(o *Table[V]) bool
func (t *Table[V]) Overlaps4(o *Table[V]) bool
func (t *Table[V]) Overlaps6(o *Table[V]) bool

func (t *Table[V]) Equal(o *Table[V]) bool

func (t *Table[V]) Subnets(netip.Prefix) iter.Seq2[netip.Prefix, V]
func (t *Table[V]) Supernets(netip.Prefix) iter.Seq2[netip.Prefix, V]

func (t *Table[V]) All() iter.Seq2[netip.Prefix, V]
func (t *Table[V]) All4() iter.Seq2[netip.Prefix, V]
func (t *Table[V]) All6() iter.Seq2[netip.Prefix, V]

func (t *Table[V]) AllSorted() iter.Seq2[netip.Prefix, V]
func (t *Table[V]) AllSorted4() iter.Seq2[netip.Prefix, V]
func (t *Table[V]) AllSorted6() iter.Seq2[netip.Prefix, V]

func (t *Table[V]) Size() int
func (t *Table[V]) Size4() int
func (t *Table[V]) Size6() int

func (t *Table[V]) String() string
func (t *Table[V]) Fprint(w io.Writer) error
func (t *Table[V]) MarshalText() ([]byte, error)
func (t *Table[V]) MarshalJSON() ([]byte, error)

func (t *Table[V]) DumpList4() []DumpListNode[V]
func (t *Table[V]) DumpList6() []DumpListNode[V]
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
