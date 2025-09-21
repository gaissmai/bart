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
    prefixes [256]*V                // 2,048 bytes
    children [256]*childRef         // 2,048 bytes (8 B pointers to childRef)
    prefixesBitSet bitset.BitSet256 // 32 bytes
    childrenBitSet bitset.BitSet256 // 32 bytes
    pfxCount uint16                 // 2 bytes + padding
 }
 ```
**Memory Usage:** **4,160 bytes** (fixed, regardless of occupancy)
 
## Real-World Example
**Scenario:** Node with 10 prefixes, 5 children
 
 | Node Type | Base | *Payload | +Children | Total | **Bytes/Prefix** ¹ |
 |-----------|------|----------|----------|-----------|------------------|
 | liteNode | 96 | 0 | 5×16=80 | 176 bytes | **17** |
 | bartNode[int] | 112 | 10×8=80 | 5×16=80 | 272 bytes | **27** |
 | fastNode[int] | 4,160 | 0 | 0 | 4,160 bytes | **416** |
 
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
 
## Performance Comparison
 
 | Aspect | bartNode[V] | liteNode | fastNode[V] |
 |--------|-------------|-------------|-------------|
 | **Per-level Speed** | ⚡ **O(1)** | ⚡ **O(1)** | 🚀 **O(1), ~40% faster per level** |
 | **Overall Lookup** | O(trie_depth) | O(trie_depth) | O(trie_depth) |
 | **IPv4 Performance** | ~3 level traversals | ~3 level traversals | ~3 level traversals |
 | **IPv6 Performance** | ~6 level traversals | ~6 level traversals | ~6 level traversals |
 | **IPv6 vs IPv4** | ~2× slower | ~2× slower | ~2× slower |
 
## When to Use Each Type
 
### 🎯 **bartNode[V]** - The Balanced Choice
- **Recommended** for most routing table use cases (use `Table[V]`)
- Near-optimal per-level performance with excellent memory efficiency
- Perfect balance for both IPv4 and IPv6 routing tables (use it for RIB)
 
### 🪶 **liteNode** - The Minimalist
- **Specialized** for prefix-only operations, no payload (use `Lite`)
- Same per-level performance as bartNode but 35% less memory
- Ideal for IPv4/IPv6 allowlists and set-based operations (use it for ACL)
 
### 🚀 **fastNode[V]** - The Performance Champion
- **40% faster per-level** when memory constraints allow (use `Fast[V]`)
- Best choice for lookup-intensive applications (use it for FIB)
- Consider memory cost vs benefit for IPv6 (6+ level traversals)
 
The elimination of within-level-backtracking and search overhead makes BART extremely fast,
though the fundamental constraint of IP address depth (IPv4: ~3 levels, IPv6: ~6 levels)
remains a physical limitation that affects all implementations equally.
