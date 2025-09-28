# package bart

# Node Types Comparison
 
 BART implements three different node types, each optimized for specific use cases:
 
## Memory Footprint (64-bit systems)
 
**Base Components:**
- `BitSet256`: `[4]uint64` = **32 bytes**
- `sparse.Array256[T]`: `BitSet256 + []T` = **56 bytes + n×sizeof(T)**
 
**Child Reference Sizing:**
- `childRef`: 8 bytes (pointer) or 16 bytes (interface value storage)
- The actual size depends on implementation: 8B for `*node` pointers, 16B for `interface{}` values

### bartNode[V] - Dynamic Sparse Node
 ```go
type bartNode[V any] struct {
    prefixes sparse.Array256[V]        // 56 + n×sizeof(V)  
    children sparse.Array256[childRef] // 56 + m×sizeof(childRef)
 }
 ```
**Memory Usage:** **112 bytes + n×sizeof(V) + m×sizeof(childRef)**
 
### liteNode - Dynamic Sparse, prefixes Bitset-Only Node
 ```go
type liteNode struct {
    prefixes bitset.BitSet256           // 32 bytes (presence only)
    children sparse.Array256[childRef]  // 56 + m×sizeof(childRef)
    pfxCount uint16                     // 2 bytes + padding
 }
 ```
**Memory Usage:** **96 bytes + m×sizeof(childRef)** (no value storage)

### fastNode[V] - Fixed Array Node
 ```go
type fastNode[V any] struct {
    prefixes struct {
        bitset.BitSet256
        items [256]*V
    }                                // 2,048 + 32 bytes BitSet256
    children struct {
        bitset.BitSet256
        items [256]*any              // pointer-to-interface for 8‑byte nils
    }                                // 2,048 + 32 bytes BitSet256
    pfxCount uint16
    cldCount uint16                  // + padding
 }
 ```
**Memory Usage:** **4,168 bytes** (fixed, regardless of occupancy)
 
## Real-World Example
**Scenario:** Node with 10 prefixes, 5 children
 
 | Node Type | Base | *Payload | +Children | Total | **Bytes/Prefix** ¹ |
 |-----------|------|----------|----------|-----------|------------------|
 | liteNode | 96 | 0 | 5×16=80 | 176 bytes | **17** |
 | bartNode[int] | 112 | 10×8=80 | 5×16=80 | 272 bytes | **27** |
 | fastNode[int] | 4,168 | 0 | 0 | 4,168 bytes | **417** |
 
¹ Values assume childRef = 16 bytes and pointer to payload = 8 bytes
 
## Lookup Performance Deep Dive
 
 All three node types achieve **O(1) per-level lookup performance**, but must traverse trie levels:
 
### Trie Structure & Performance
- **8-bit strides per level**: Each trie level handles 8 bits of the IP address
- **IPv4 traversal**: Worst case  4 levels (32÷8),  real-world typically 3 levels for /24 routes
- **IPv6 traversal**: Worst case 16 levels (128÷8), real-world typically 6 levels for /48 routes
- **Performance characteristic**: O(trie_depth) not O(number_of_routes)
- **IPv6 vs IPv4**: IPv6 inherently ~2× slower due to deeper tree structure
 
### bartNode[V] & liteNode - Optimized Level Operations
- **Precomputed lookup tables** (`lmp.LookupTbl[idx]`) eliminate search within each level
- **BitSet256 intersections** via `IntersectionTop()` for instant prefix matching
- **Rank-based indirection**: Bitset-to-slice mapping uses precomputed Rank masks
- **Pipeline-friendly**: Only 4 bitset operations (4×uint64) per level, optimized for CPU pipelining
- **No backtracking**: Traditional longest-prefix-match backtracking replaced with direct table lookups
 
### fastNode[V] - Direct Array Access per Level
- **Zero indirection per level**: Direct array indexing `prefixes[idx]` and `children[idx]`
- **Cache-optimal**: Contiguous memory layout within each level
- **Performance advantage**: Still ~40% faster per level despite sparse optimizations
