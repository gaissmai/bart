// Copyright (c) 2025 Karl Gaissmaier
// SPDX-License-Identifier: MIT

// Package bart provides high-performance Balanced Routing Tables (BART)
// for fastest IP-to-CIDR lookups on IPv4 and IPv6 addresses.
//
// BART offers three table variants optimized for different use cases:
//
//   - Lite:  Memory-optimized with popcount-compressed sparse arrays
//   - Table: Full-featured with popcount-compressed sparse arrays
//   - Fast:  Speed-optimized with fixed-size 256-element arrays
//
// The implementation is based on Knuth's ART algorithm with novel
// optimizations for memory efficiency and lookup speed.
//
// `Table` and `Lite` use popcount compression for memory efficiency, while
// `Fast` trades memory for maximum lookup speed with uncompressed arrays.
//
// BART excels at efficient set operations on routing tables including Union,
// Overlaps, Equal, Subnets, and Supernets with optimal complexity, making it
// ideal for large-scale IP prefix matching in ACLs, RIBs, FIBs, firewalls,
// and routers.
//
// All variants also support copy-on-write persistence.
package bart
