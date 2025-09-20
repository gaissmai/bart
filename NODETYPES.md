# package bart

## Node Types Comparison

BART implements three different node types, each optimized for specific use cases:

### Memory Footprint (64-bit systems)

**Base Components:**
- `BitSet256`: `[4]uint64` = **32 bytes**
- `sparse.Array256[T]`: `BitSet256 + []T` = **56 bytes + n×sizeof(T)**

#### bartNode[V] - Dynamic Sparse Node
```go
type bartNode[V any] struct {
    prefixes sparse.Array256[V]    // 56 + n×sizeof(V)  
    children sparse.Array256[any]  // 56 + m×16 bytes
}
```
**Memory Usage:** **112 bytes + n×sizeof(V) + m×16 bytes**

#### liteNode[V] - Bitset-Only Node
```go
type liteNode[V any] struct {
    prefixes bitset.BitSet256      // 32 bytes (presence only)
    children sparse.Array256[any]  // 56 + m×16 bytes
    pfxCount uint16                // 2 bytes + padding
}
```
**Memory Usage:** **96 bytes + m×16 bytes** (no value storage)

#### fastNode[V] - Fixed Array Node
```go
type fastNode[V any] struct {
    prefixes [256]*V                 // 2,048 bytes
    children [256]*any               // 2,048 bytes
    prefixesBitSet bitset.BitSet256  // 32 bytes
    childrenBitSet bitset.BitSet256  // 32 bytes
    pfxCount uint16                  // 2 bytes + padding
}
```
**Memory Usage:** **4,168 bytes** (fixed, regardless of occupancy)

### Lookup Performance Deep Dive

All three node types achieve **O(1) per-level lookup performance**, but must traverse trie levels:

#### Trie Structure & Performance
- **8-bit strides per level**: Each trie level handles 8 bits of the IP address
- **IPv4 traversal**: Worst case 4 levels (32÷8), real-world typically 3 levels for /24 routes
- **IPv6 traversal**: Worst case 16 levels (128÷8), real-world typically 6 levels for /48 routes
- **Performance characteristic**: O(IP_depth) not O(number_of_routes)
- **IPv6 vs IPv4**: IPv6 inherently ~2× slower due to deeper tree structure

#### bartNode[V] & liteNode[V] - Optimized Level Operations
- **Precomputed lookup tables** (`lmp.LookupTbl[idx]`) eliminate search within each level
- **BitSet256 intersections** via `IntersectionTop()` for instant prefix matching
- **Rank-based indirection**: Bitset-to-slice mapping uses precomputed Rank operations
- **Pipeline-friendly**: Only 4 bitset operations (4×uint64) per level, optimized for CPU pipelining
- **No backtracking**: Traditional longest-prefix-match backtracking replaced with direct table lookups

#### fastNode[V] - Direct Array Access per Level
- **Zero indirection per level**: Direct array indexing `prefixes[idx]` and `children[idx]`
- **Cache-optimal**: Contiguous memory layout within each level
- **Performance advantage**: Still ~40% faster per level despite sparse optimizations

### Performance Comparison

| Aspect | bartNode[V] | liteNode[V] | fastNode[V] |
|--------|-------------|-------------|-------------|
| **Per-level Speed** | ⚡ **O(1)** | ⚡ **O(1)** | 🚀 **O(1) + 40%** |
| **Overall Lookup** | O(IP_depth) | O(IP_depth) | O(IP_depth) |
| **IPv4 Performance** | ~3 level traversals | ~3 level traversals | ~3 level traversals |
| **IPv6 Performance** | ~6 level traversals | ~6 level traversals | ~6 level traversals |
| **IPv6 vs IPv4** | ~2× slower | ~2× slower | ~2× slower |

### Real-World Example
**Scenario:** Node with 10 prefixes, 5 children

| Node Type | Base | Prefixes | Children | **Total** |
|-----------|------|----------|----------|-----------|
| bartNode[int] | 112 | 10×8=80 | 5×16=80 | **272 bytes** |
| liteNode[int] | 96 | 0 | 5×16=80 | **176 bytes** |
| fastNode[int] | 4,168 | 0* | 0* | **4,168 bytes** |

*fastNode pre-allocates full arrays regardless of usage

### When to Use Each Type

#### 🎯 **bartNode[V]** - The Balanced Choice
- **Recommended** for most routing table use cases (use `Table[V]`)
- Near-optimal per-level performance with excellent memory efficiency
- Perfect balance for both IPv4 and IPv6 routing tables

#### 🪶 **liteNode[V]** - The Minimalist
- **Specialized** for prefix-only operations (`Contains`, `Exists`) (use `Lite`)
- Same per-level performance as bartNode but 35% less memory
- Ideal for IPv4/IPv6 allowlists and set-based operations

#### 🚀 **fastNode[V]** - The Performance Champion
- **40% faster per-level** when memory constraints allow (use `Fast[V]`)
- Best choice for lookup-intensive IPv4 applications
- Consider memory cost vs benefit for IPv6 (6+ level traversals)

The elimination of backtracking and within-level search overhead makes BART extremely fast,
though the fundamental constraint of IP address depth (IPv4: ~3 levels, IPv6: ~6 levels)
remains a physical limitation that affects all implementations equally.
