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
- **`bart.Table[T]`**: Stores associated payloads alongside routing prefixes.

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

type Table[T any] struct {
  // Has unexported fields.
}

func (t *Table[T]) Contains(netip.Addr) bool
func (t *Table[T]) Lookup(netip.Addr) (T, bool)

func (t *Table[T]) LookupPrefix(netip.Prefix) (T, bool)
func (t *Table[T]) LookupPrefixLPM(netip.Prefix) (netip.Prefix, T, bool)

func (t *Table[T]) Get(netip.Prefix) (T, bool)
func (t *Table[T]) Insert(netip.Prefix, T)
func (t *Table[T]) Delete(netip.Prefix)
func (t *Table[T]) Modify(netip.Prefix, cb func(T, bool) (T, bool))

func (t *Table[T]) InsertPersist(netip.Prefix, T) *Table[T]
func (t *Table[T]) DeletePersist(netip.Prefix) *Table[T]
func (t *Table[T]) ModifyPersist(netip.Prefix, cb func(T, bool) (T, bool)) *Table[T]

func (t *Table[T]) Clone() *Table[T]
func (t *Table[T]) Union(o *Table[T])
func (t *Table[T]) UnionPersist(o *Table[T]) *Table[T]

func (t *Table[T]) OverlapsPrefix(netip.Prefix) bool

func (t *Table[T]) Overlaps(o *Table[T]) bool
func (t *Table[T]) Overlaps4(o *Table[T]) bool
func (t *Table[T]) Overlaps6(o *Table[T]) bool

func (t *Table[T]) Equal(o *Table[T]) bool

func (t *Table[T]) Subnets(netip.Prefix) iter.Seq2[netip.Prefix, T]
func (t *Table[T]) Supernets(netip.Prefix) iter.Seq2[netip.Prefix, T]

func (t *Table[T]) All() iter.Seq2[netip.Prefix, T]
func (t *Table[T]) All4() iter.Seq2[netip.Prefix, T]
func (t *Table[T]) All6() iter.Seq2[netip.Prefix, T]

func (t *Table[T]) AllSorted() iter.Seq2[netip.Prefix, T]
func (t *Table[T]) AllSorted4() iter.Seq2[netip.Prefix, T]
func (t *Table[T]) AllSorted6() iter.Seq2[netip.Prefix, T]

func (t *Table[T]) Size() int
func (t *Table[T]) Size4() int
func (t *Table[T]) Size6() int

func (t *Table[T]) Fprint(w io.Writer) error
func (t *Table[T]) MarshalText() ([]byte, error)
func (t *Table[T]) MarshalJSON() ([]byte, error)

func (t *Table[T]) DumpList4() []DumpListNode[T]
func (t *Table[T]) DumpList6() []DumpListNode[T]
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
type Fast[T any] = Table[T]
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
