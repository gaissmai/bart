package bart

import (
	"net/netip"
	"slices"

	"github.com/gaissmai/bart/internal/nodes"
)

const (
	maxItems     = nodes.MaxItems
	maxTreeDepth = nodes.MaxTreeDepth
	depthMask    = nodes.DepthMask
)

type stridePath = nodes.StridePath

var LastOctetPlusOneAndLastBits = nodes.LastOctetPlusOneAndLastBits

// DumpListNode contains CIDR, Value and Subnets, representing the trie
// in a sorted, recursive representation, especially useful for serialization.
type DumpListNode[V any] struct {
	CIDR    netip.Prefix      `json:"cidr"`
	Value   V                 `json:"value"`
	Subnets []DumpListNode[V] `json:"subnets,omitempty"`
}

// dumpListRec, build the data structure rec-descent with the help
// of directItemsRec.
func dumpListRec[V any](n nodes.NodeReader[V], parentIdx uint8, path stridePath, depth int, is4 bool) []DumpListNode[V] {
	// recursion stop condition
	if n == nil {
		return nil
	}

	directItems := nodes.DirectItemsRec(n, parentIdx, path, depth, is4)

	// sort the items by prefix
	slices.SortFunc(directItems, func(a, b nodes.TrieItem[V]) int {
		return nodes.CmpPrefix(a.Cidr, b.Cidr)
	})

	nodes := make([]DumpListNode[V], 0, len(directItems))

	for _, item := range directItems {
		nodes = append(nodes, DumpListNode[V]{
			CIDR:  item.Cidr,
			Value: item.Val,
			// build it rec-descent
			Subnets: dumpListRec(item.Node, item.Idx, item.Path, item.Depth, is4),
		})
	}

	return nodes
}
