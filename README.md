![GitHub release (latest SemVer)](https://img.shields.io/github/v/release/gaissmai/bart)
[![Go Reference](https://pkg.go.dev/badge/github.com/gaissmai/bart.svg)](https://pkg.go.dev/github.com/gaissmai/bart#section-documentation)
[![CodeRabbit Pull Request Reviews](https://img.shields.io/coderabbit/prs/github/gaissmai/bart)](https://coderabbit.ai)
[![Mentioned in Awesome Go](https://awesome.re/mentioned-badge-flat.svg)](https://github.com/avelino/awesome-go#networking)
[![CI](https://github.com/gaissmai/bart/actions/workflows/go.yml/badge.svg)](https://github.com/gaissmai/bart/actions/workflows/go.yml)
[![Coverage Status](https://coveralls.io/repos/github/gaissmai/bart/badge.svg)](https://coveralls.io/github/gaissmai/bart)
[![License: MIT](https://img.shields.io/badge/License-MIT-green.svg)](https://opensource.org/licenses/MIT)
[![Stand With Ukraine](https://raw.githubusercontent.com/vshymanskyy/StandWithUkraine/main/badges/StandWithUkraine.svg)](https://stand-with-ukraine.pp.ua)

# package bart

The `bart` package provides **Balanced Routing Tables (BART)** for high-performance IP-to-CIDR lookups and prefix management:

- **ACL** (access control lists): quickly decide whether an IP address matches any of millions of CIDR rules.
- **RIB** (routing information bases): handle very large routing tables with low memory overhead and fast lookups.

BART is designed for workloads where speed and memory efficiency matter most, which makes it well suited for firewalls, routers, and network telemetry systems.

## Overview

BART is a multibit trie with a fixed stride of 8 bits. A fast mapping function, derived from Donald E. Knuth’s **Allotment Routing Table** (ART) algorithm, maps the prefixes of each trie level into a complete binary tree.

BART offers two table variants for different use cases:
- **`bart.Lite`**: a payload-free variant with a minimal memory footprint.
- **`bart.Table[T]`**: stores a payload of type `T` with each prefix.

Both variants share the same optimized core. Popcount-compressed sparse arrays provide **level compression**, and a **novel path and fringe compression** reduces the memory consumption further. Compared to classical ART, the memory consumption drops by nearly two orders of magnitude, while the lookup times remain comparable.

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

  // Check some IPs: Contains reports whether any prefix covers the address
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

## API

BART offers a rich API for `Table[T]` and `Lite`:

- **Prefix aggregation**: fast compaction of ACLs with `Aggregate` (`Lite` only).
- **Containment queries**: fast membership tests for ACLs with `Contains`.
- **Lookups**: fast longest-prefix matching for routing tables with `Lookup` and related methods.
- **Modification**: in-place updates (`Insert`, `Delete`, ...) and persistent copy-on-write variants (`InsertPersist`, `DeletePersist`, ...).
- **Overlap detection**: fast checks with `Overlaps` and related methods.
- **Iteration**: iterators over all entries, sorted or unsorted (`All`, `AllSorted`, ...).
- **Subnets and supernets**: iterators over the subnets in CIDR order (`Subnets`) and over the supernets from the most to the least specific (`Supernets`).
- **Serialization**: a hierarchical tree diagram and a JSON dump of the ordered CIDRs.

## Concurrency Model

Readers can use immutable table snapshots without any locking. Writers serialize their updates with a mutex: a writer loads the current snapshot, builds a modified copy with the `Persist` methods, and publishes the returned table through an atomic pointer. In-place updates such as `Insert` and `Delete` require a lock that is shared by readers and writers.

See the concurrent examples:
- [ExampleLite](example_lite_concurrent_test.go)
- [ExampleTable](example_table_concurrent_test.go)

## Bitset Efficiency

BART is built on fixed-size (256-bit) bit vectors and precomputed lookup tables. A lookup consists mainly of bitmask operations, which the CPU executes with instructions like `POPCNT`, `LZCNT`, and `TZCNT`.

For best performance, tell the compiler the feature level of your CPU at build time:

```bash
# Example for AMD64, choose v2/v3/v4 to match your CPU features.
GOAMD64=v3 go build

# On ARM64, the default target is v8.0. Set GOARM64 for newer targets, for example:
GOARM64=v8.2,lse go build
```

In hot-path operations on these fixed-size bitsets (`[4]uint64`), loops are unrolled by hand where this still allows the functions to be inlined, which keeps the CPU pipeline busy.

## Benchmarks

The [benchmarks](https://github.com/gaissmai/iprbench) compare `bart` extensively with other IP routing table implementations.

Query performance of `Lite.Contains` against the Tier-1 full Internet routing table, probed with random IP addresses:

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

The package is pre-v1. Breaking changes are possible if they improve the API, but they are rare.

### Deprecated: `bart.Fast`

`bart.Fast` is deprecated and is now a type alias for `bart.Table`:

```go
// Deprecated: use Table.
type Fast[T any] = Table[T]
```

With Go 1.26 or later, `go fix` migrates your code automatically:

```text
go get github.com/gaissmai/bart@latest
go fix -diff ./...    # preview changes
go fix ./...          # apply changes
```

The alias will be removed in a future release.

## Contribution

Please open an issue to discuss your idea before sending a pull request.

## Credit

Standing on the shoulders of giants. Credits go to:
- the engineering team at Tailscale,
- Daniel Lemire for his inspiring blog,
- Donald E. Knuth for the Allotment Routing Table ([ART](https://github.com/hariguchi/art/tree/master/orig)) algorithm, and
- Yoichi Hariguchi, who [deciphered it](https://github.com/hariguchi/art/blob/master/docs/art.pdf).

## License

MIT
