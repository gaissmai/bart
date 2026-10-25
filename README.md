![GitHub release (latest SemVer)](https://img.shields.io/github/v/release/gaissmai/bart)
[![Go Reference](https://pkg.go.dev/badge/github.com/gaissmai/bart.svg)](https://pkg.go.dev/github.com/gaissmai/bart#section-documentation)
[![CodeRabbit Pull Request Reviews](https://img.shields.io/coderabbit/prs/github/gaissmai/bart)](https://coderabbit.ai)
[![Mentioned in Awesome Go](https://awesome.re/mentioned-badge-flat.svg)](https://github.com/avelino/awesome-go)
[![CI](https://github.com/gaissmai/bart/actions/workflows/go.yml/badge.svg)](https://github.com/gaissmai/bart/actions/workflows/go.yml)
[![Coverage Status](https://coveralls.io/repos/github/gaissmai/bart/badge.svg)](https://coveralls.io/github/gaissmai/bart)
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

BART implements two different routing tables, each optimized for specific
use cases:
- **bart.Lite**
- **bart.Table**

For **bart.Table** this binary tree is represented with popcount‑compressed
sparse arrays for **level compression**.
Combined with a **novel path and fringe compression**, this design reduces
memory consumption by nearly two orders of magnitude compared to classical ART.

**bart.Lite** is a special form of **bart.Table**, but without a payload, and therefore
has the lowest memory overhead while maintaining the same lookup times.

## Getting Started

### Example: simple ACL with bart.Lite

```go
package main

import (
  "fmt"
  "net/netip"
  "github.com/gaissmai/bart"
)

var a = netip.MustParseAddr
var p = netip.MustParsePrefix

func main() {
  allowList := new(bart.Lite)

  // Add allowed networks
  allowList.Insert(p("192.168.0.0/16"))
  allowList.Insert(p("2001:db8::/32"))
  allowList.Insert(p("10.0.0.0/24"))

  // Test some IPs with Contains (longest-prefix match)
  testIPs := []netip.Addr{
    a("192.168.1.100"), // allowed (matches 192.168.0.0/16)
    a("2001:db8::1"),   // allowed (matches 2001:db8::/32)
    a("172.16.0.1"),    // denied (no match)
  }

  for _, ip := range testIPs {
    if allowList.Contains(ip) {
      fmt.Printf("%s is allowed\n", ip)
    } else {
      fmt.Printf("%s is denied\n", ip)
    }
  }
}
```

## Bitset Efficiency

The BART algorithm is based on a fixed-size (256) bit vector and precomputed lookup tables.
Lookups are executed entirely with fast, cache-resident bitmask operations, which
modern CPUs accelerate using specialized instructions such as POPCNT, LZCNT, and TZCNT.

For maximum performance, specify the CPU feature set when compiling.
See the [Go minimum requirements](https://go.dev/wiki/MinimumRequirements#architectures) for details.

```bash
# Example for AMD64, choose v2/v3/v4 to match your CPU features.
GOAMD64=v3 go build

# On ARM64, Go auto-selects CPU instructions.

```
Loops over these fixed-size bitsets are unrolled for additional speed,
ensuring predictable memory access and efficient use of CPU pipelines.

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
In internal benchmarks the check runs in a few nanoseconds per query with zero
heap allocations on a modern CPU.

## API

BART has a rich API for CRUD, lookup, comparison, iteration,
serialization and persistence.

**Lite** deviates from the common API since it carries no payload.
Additionally, **Lite** provides specialized methods like `Aggregate`
to compact ACL prefixes in-place.

```go
import "github.com/gaissmai/bart"

type Table[V any] struct {
  // Has unexported fields.
}

func (t *Table[V]) Contains(netip.Addr) bool
func (t *Table[V]) Lookup(netip.Addr) (V, bool)

func (t *Table[V]) LookupPrefix(netip.Prefix) (V, bool)
func (t *Table[V]) LookupPrefixLPM(netip.Prefix) (netip.Prefix, V, bool)

func (t *Table[V]) Get(netip.Prefix) (V, bool)
func (t *Table[V]) Insert(netip.Prefix, V)
func (t *Table[V]) Delete(netip.Prefix)
func (t *Table[V]) Modify(netip.Prefix, cb func(V, bool) (V, bool))

func (t *Table[V]) InsertPersist(netip.Prefix, V) *Table[V]
func (t *Table[V]) DeletePersist(netip.Prefix) *Table[V]
func (t *Table[V]) ModifyPersist(netip.Prefix, cb func(V, bool) (V, bool)) *Table[V]

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

func (t *Table[V]) Fprint(w io.Writer) error
func (t *Table[V]) MarshalText() ([]byte, error)
func (t *Table[V]) MarshalJSON() ([]byte, error)

func (t *Table[V]) DumpList4() []DumpListNode[V]
func (t *Table[V]) DumpList6() []DumpListNode[V]
```

## Benchmarks

Please see the extensive [benchmarks](https://github.com/gaissmai/iprbench)
comparing `bart` with other IP routing table implementations.

Just a teaser, `Lite.Contains` against the Tier1 full
Internet routing table with random IP address probes:

```text
$ GOAMD64=v3 go test -run=^$ -bench=FullLite/Contains$ -cpu=1
goos: linux
goarch: amd64
pkg: github.com/gaissmai/bart
cpu: AMD EPYC 9275F 24-Core Processor
BenchmarkFullLiteMatch4/Contains         	134873065	         8.933 ns/op
BenchmarkFullLiteMatch6/Contains         	82965392	        14.14 ns/op
BenchmarkFullLiteMiss4/Contains          	120155722	         9.998 ns/op
BenchmarkFullLiteMiss6/Contains          	93057260	        12.56 ns/op
```

## Compatibility Guarantees

The package is currently released as a pre-v1 version, which gives the author the freedom to break
backward compatibility to help improve the API as he learns which initial design decisions would need
to be revisited to better support the use cases that the library solves for.

These occurrences are expected to be rare in frequency and the API is already quite stable.

### Deprecated: `bart.Fast`

`bart.Fast` is deprecated and no longer a separate table type. Every node now
carries a compact rank cache in its sparse arrays, which makes the additional
256 bytes per node of `bart.Fast` obsolete. `bart.Table` has the identical API.

`bart.Fast` remains as a type alias for `bart.Table`, so existing code keeps
compiling:

```go
// Deprecated: use Table.
type Fast[V any] = Table[V]
```

With Go 1.26 or later, `go fix` migrates your code automatically:

```text
go get github.com/gaissmai/bart@latest
go fix -diff ./...   # preview the changes
go fix ./...         # apply them
```

The alias will be removed in a future release.

## Contribution

Please open an issue for discussion before sending a pull request.

## Credit

Standing on the shoulders of giants.

Credits for many inspirations go to

- the clever folks at Tailscale,
- to Daniel Lemire for his inspiring blog,
- to Donald E. Knuth for the Allotment Routing Table [ART](https://github.com/hariguchi/art/tree/master/orig) algorithm and
- to Yoichi Hariguchi who [deciphered it](https://github.com/hariguchi/art/blob/master/docs/art.pdf) for us mere mortals

And last but not least to the Go team who do a wonderful job!

## LICENSE

MIT
