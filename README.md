![GitHub release (latest SemVer)](https://img.shields.io/github/v/release/gaissmai/bart)
[![Go Reference](https://pkg.go.dev/badge/github.com/gaissmai/bart.svg)](https://pkg.go.dev/github.com/gaissmai/bart#section-documentation)
[![CodeRabbit Pull Request Reviews](https://img.shields.io/coderabbit/prs/github/gaissmai/bart)](https://coderabbit.ai)
[![Mentioned in Awesome Go](https://awesome.re/mentioned-badge-flat.svg)](https://github.com/avelino/awesome-go#networking)
[![CI](https://github.com/gaissmai/bart/actions/workflows/go.yml/badge.svg)](https://github.com/avelino/awesome-go)
[![Coverage Status](https://coveralls.io/repos/github/gaissmai/bart/badge.svg)](https://coveralls.io/github/gaissmai/bart)
[![License: MIT](https://img.shields.io/badge/License-MIT-green.svg)](https://opensource.org/licenses/MIT)
[![Stand With Ukraine](https://raw.githubusercontent.com/vshymanskyy/StandWithUkraine/main/badges/StandWithUkraine.svg)](https://stand-with-ukraine.pp.ua)

# package bart

The `bart` package provides **Balanced Routing Tables (BART)** for high-performance IP-to-CIDR lookups and prefix management:

- **ACL**: Determine extremely fast whether an IP address matches any of millions of CIDR rules.
- **RIB**: Handle very large routing tables with low memory overhead while keeping lookups fast.

BART is designed for workloads where speed and memory efficiency matter most, making it ideal for firewalls, routers, and network telemetry systems.

## Overview

BART is implemented as a multibit trie with a fixed stride of 8 bits, using a fast mapping function derived from Donald E. Knuth’s **Allotment Routing Table** (ART) algorithm to map prefixes at each trie level into a complete binary tree.

BART provides two table variants optimized for different use cases:
- **`bart.Lite`**: A payload-free variant optimized for minimal memory footprint.
- **`bart.Table[V]`**: Stores associated payloads alongside routing prefixes.

Both implementations share the same highly optimized core engine, using popcount-compressed sparse arrays for **level compression**. Combined with a **novel path and fringe compression**, this design reduces memory consumption by nearly two orders of magnitude compared to classical ART while maintaining identical, lightning-fast lookup times.

## Getting Started

### Example: Simple ACL with `bart.Lite`

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

The BART algorithm relies on fixed-size (256-bit) bit vectors and precomputed lookup tables. Lookups execute entirely via cache-resident bitmask operations accelerated by CPU instructions like `POPCNT`, `LZCNT`, and `TZCNT`.

For maximum performance, specify your CPU feature set at compile time:

```bash
# Example for AMD64, choose v2/v3/v4 to match your CPU features.
GOAMD64=v3 go build

# On ARM64, Go auto-selects available CPU instructions.
```

Loops over these bitsets are fully unrolled by the compiler, ensuring predictable
memory access and efficient CPU pipeline utilization.

## Concurrency Model

Readers can access the table completely lock-free. Writers synchronize via a mutex so that only one writer modifies the table at a time, avoiding complex Compare-And-Swap (CAS) loops that degrade performance under heavy contention.

See the concurrent test examples for reference:
- [ExampleLite](example_lite_concurrent_test.go)
- [ExampleTable](example_table_concurrent_test.go)

## Advanced Operations & Use Cases

Beyond high-performance prefix matching, BART provides a rich suite of advanced set operations and structural queries designed for routing daemons and firewalls:

- **Overlap Detection**: Instantly check whether routing tables overlap in just a few nanoseconds.
- **Table Unions**: Efficiently merge routing tables.
- **Prefix Aggregation**: Compact and minimize ACL or routing prefix lists in-place.
- **Hierarchical Queries**: Traverse subnets and supernets.
- **Canonical Iteration**: Stream prefixes in sorted order.

## API

BART provides a rich API:

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

Please see the extensive [benchmarks](https://github.com/gaissmai/iprbench) comparing `bart` with other IP routing table implementations.

Query performance of `Lite.Contains` against the Tier-1 full Internet routing table with random IP address probes:

```text
$ GOAMD64=v3 go test -run=^$ -bench=FullLite/Contains$ -cpu=1
goos: linux
goarch: amd64
pkg: github.com/gaissmai/bart
cpu: AMD EPYC 9275F 24-Core Processor
BenchmarkFullLiteMatch4/Contains         134873065             8.933 ns/op
BenchmarkFullLiteMatch6/Contains          82965392             14.14 ns/op
BenchmarkFullLiteMiss4/Contains          120155722             9.998 ns/op
BenchmarkFullLiteMiss6/Contains           93057260             12.56 ns/op
```

## Compatibility Guarantees

The package is pre-v1, allowing breaking changes if necessary to refine API design. Such changes are rare.

### Deprecated: `bart.Fast`

`bart.Fast` is deprecated and is now a type alias for `bart.Table`:

```go
// Deprecated: use Table.
type Fast[V any] = Table[V]
```

Use `go fix` to migrate automatically:

```text
go get github.com/gaissmai/bart@latest
go fix -diff ./...    # preview changes
go fix ./...          # apply changes
```

The alias will be removed in a future release.

## Contribution

Please open an issue for discussion before sending a pull request.

## Credit

Standing on the shoulders of giants. Credits go to:
- the engineering team at Tailscale,
- Daniel Lemire for his inspiring blog,
- Donald E. Knuth for the Allotment Routing Table [ART](https://github.com/hariguchi/art/tree/master/orig) algorithm, and
- Yoichi Hariguchi who [deciphered it](https://github.com/hariguchi/art/blob/master/docs/art.pdf).

## License

MIT
