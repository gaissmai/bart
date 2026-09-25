package nodes

import (
	"bytes"
	"net/netip"
	"slices"
	"strings"
	"testing"
)

// TestFastACLNode_IsDirectlyCoveredBy validates CBT ancestor tracking logic
// for direct containment checks across both IPv4 and IPv6 bitsets.
func TestFastACLNode_IsDirectlyCoveredBy(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		nodeSetup func() *FastACLNode
		idx       uint8
		parentIdx uint8
		want      bool
	}{
		{
			name: "Fast path: ancestor index smaller than parentIdx",
			nodeSetup: func() *FastACLNode {
				return &FastACLNode{}
			},
			idx:       2, // nextIdx = 1
			parentIdx: 3,
			want:      false,
		},
		{
			name: "Direct CBT ancestor matches parentIdx",
			nodeSetup: func() *FastACLNode {
				n := &FastACLNode{}
				n.Prefixes.Set(1) // Root prefix /0 (idx = 1)
				return n
			},
			idx:       2, // Shift right by 1 gives nextIdx = 1 -> LPM returns 1
			parentIdx: 1,
			want:      true,
		},
		{
			name: "Indirect coverage blocked by intermediate prefix",
			nodeSetup: func() *FastACLNode {
				n := &FastACLNode{}
				n.Prefixes.Set(1) // Root /0 (idx = 1)
				n.Prefixes.Set(2) // Intermediate /1 (idx = 2)
				return n
			},
			idx:       4, // Shift right by 1 gives nextIdx = 2 -> LPM returns 2, not 1
			parentIdx: 1,
			want:      false,
		},
		{
			name: "Zero index bounds check",
			nodeSetup: func() *FastACLNode {
				return &FastACLNode{}
			},
			idx:       0, // nextIdx = 0
			parentIdx: 1,
			want:      false,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			node := tt.nodeSetup()
			got := node.IsDirectlyCoveredBy(tt.idx, tt.parentIdx)
			if got != tt.want {
				t.Errorf("IsDirectlyCoveredBy(%d, %d) = %v; want %v", tt.idx, tt.parentIdx, got, tt.want)
			}
		})
	}
}

// TestFastACLNode_DirectItems verifies immediate child collection logic
// for IPv4 and IPv6 Trie nodes using canonical CBT indexes.
func TestFastACLNode_DirectItems(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		nodeSetup func() *FastACLNode
		ptx       PathContext
		wantCount int
		wantCidrs []netip.Prefix
	}{
		{
			name: "Empty node returns nil or empty slice",
			nodeSetup: func() *FastACLNode {
				return &FastACLNode{}
			},
			ptx: PathContext{
				Depth: 0,
				Idx:   0,
				Is4:   true,
			},
			wantCount: 0,
		},
		{
			name: "IPv4 collect local direct prefixes under root (Idx = 1)",
			nodeSetup: func() *FastACLNode {
				n := &FastACLNode{}
				n.InsertPrefix(1) // 0.0.0.0/0 (root)
				n.InsertPrefix(2) // 0.0.0.0/1
				n.InsertPrefix(3) // 128.0.0.0/1
				return n
			},
			ptx: PathContext{
				Depth: 0,
				Idx:   1, // Direct descendants covered by root (CBT index 1)
				Is4:   true,
			},
			wantCount: 2,
			wantCidrs: []netip.Prefix{
				mpp("0.0.0.0/1"),
				mpp("128.0.0.0/1"),
			},
		},
		{
			name: "IPv6 collect local direct prefixes under root (Idx = 1)",
			nodeSetup: func() *FastACLNode {
				n := &FastACLNode{}
				n.InsertPrefix(1) // ::/0 (root)
				n.InsertPrefix(2) // ::/1
				n.InsertPrefix(3) // 8000::/1
				return n
			},
			ptx: PathContext{
				Depth: 0,
				Idx:   1, // Direct descendants covered by root (CBT index 1)
				Is4:   false,
			},
			wantCount: 2,
			wantCidrs: []netip.Prefix{
				mpp("::/1"),
				mpp("8000::/1"),
			},
		},
		{
			name: "IPv6 deep stride traversal path context",
			nodeSetup: func() *FastACLNode {
				n := &FastACLNode{}
				// Under sub-node context, Idx = 0 matches root-level prefix inside sub-node (idx = 1)
				n.InsertPrefix(1)
				return n
			},
			ptx: PathContext{
				Path:  StridePath{0x20, 0x01, 0x0d, 0xb8}, // 2001:db8::
				Depth: 4,
				Idx:   0,
				Is4:   false,
			},
			wantCount: 1,
			wantCidrs: []netip.Prefix{
				mpp("2001:db8::/32"),
			},
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			node := tt.nodeSetup()
			got := node.DirectItems(tt.ptx)

			if len(got) != tt.wantCount {
				t.Fatalf("DirectItems() count = %d; want %d", len(got), tt.wantCount)
			}

			for i, wantPrefix := range tt.wantCidrs {
				if got[i].Cidr != wantPrefix {
					t.Errorf("DirectItems()[%d].Cidr = %s; want %s", i, got[i].Cidr, wantPrefix)
				}
			}
		})
	}
}

// TestFastACLNode_FprintRec verifies the hierarchical ASCII tree rendering
// of FastACLNode instances across IPv4 and IPv6 path contexts.
func TestFastACLNode_FprintRec(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		nodeSetup func() *FastACLNode
		ptx       PathContext
		want      []string
	}{
		{
			name: "Empty node produces no output",
			nodeSetup: func() *FastACLNode {
				return &FastACLNode{}
			},
			ptx: PathContext{
				Depth: 0,
				Idx:   0,
				Is4:   true,
			},
			want: nil,
		},
		{
			name: "IPv4 single level hierarchy rendering",
			nodeSetup: func() *FastACLNode {
				n := &FastACLNode{}
				// Set root /0 prefix and its direct child /1 prefixes
				n.InsertPrefix(1) // 0.0.0.0/0
				n.InsertPrefix(2) // 0.0.0.0/1
				n.InsertPrefix(3) // 128.0.0.0/1
				return n
			},
			ptx: PathContext{
				Depth: 0,
				Idx:   1, // Root scope (0.0.0.0/0)
				Is4:   true,
			},
			want: []string{
				"├─ 0.0.0.0/1",
				"└─ 128.0.0.0/1",
			},
		},
		{
			name: "IPv6 single level hierarchy rendering",
			nodeSetup: func() *FastACLNode {
				n := &FastACLNode{}
				n.InsertPrefix(1) // ::/0
				n.InsertPrefix(2) // ::/1
				n.InsertPrefix(3) // 8000::/1
				return n
			},
			ptx: PathContext{
				Depth: 0,
				Idx:   1, // Root scope (::/0)
				Is4:   false,
			},
			want: []string{
				"├─ ::/1",
				"└─ 8000::/1",
			},
		},
		{
			name: "IPv4 nested multi-stride subtrees and leaves",
			nodeSetup: func() *FastACLNode {
				n := &FastACLNode{}
				n.InsertPrefix(1) // 0.0.0.0/0

				// Insert nested prefixes into trie
				pfx1 := mpp("10.0.0.0/8")
				pfx2 := mpp("10.1.0.0/16")
				n.Insert(pfx1, 0)
				n.Insert(pfx2, 0)

				return n
			},
			ptx: PathContext{
				Depth: 0,
				Idx:   1,
				Is4:   true,
			},
			want: []string{
				"└─ 10.0.0.0/8",
				"   └─ 10.1.0.0/16",
			},
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			node := tt.nodeSetup()
			var buf bytes.Buffer

			err := node.FprintRec(&buf, tt.ptx, "")
			if err != nil {
				t.Fatalf("FprintRec() unexpected error: %v", err)
			}

			var wantStr string
			if len(tt.want) > 0 {
				wantStr = strings.Join(tt.want, "\n") + "\n"
			}

			got := buf.String()
			if got != wantStr {
				t.Errorf("FprintRec() mismatch:\ngot:\n%s\nwant:\n%s", got, wantStr)
			}
		})
	}
}

// TestCIDRLeaf_Fprint checks boundary rendering for terminal leaf structures.
func TestIDRLeaf_Fprint(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		leaf *CIDRLeaf
		pad  string
		want string
	}{
		{
			name: "Nil leaf renders nothing",
			leaf: nil,
			pad:  "  ",
			want: "",
		},
		{
			name: "IPv4 leaf renders correct prefix and padding",
			leaf: &CIDRLeaf{
				Prefix: mpp("10.0.0.0/8"),
			},
			pad:  "│  ",
			want: "│  └─ 10.0.0.0/8\n",
		},
		{
			name: "IPv6 leaf renders correct prefix and padding",
			leaf: &CIDRLeaf{
				Prefix: mpp("2001:db8::/32"),
			},
			pad:  "│  ",
			want: "│  └─ 2001:db8::/32\n",
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			var buf bytes.Buffer
			err := tt.leaf.Fprint(&buf, tt.pad)
			if err != nil {
				t.Fatalf("unexpected error during leaf Fprint: %v", err)
			}

			if got := buf.String(); got != tt.want {
				t.Errorf("Fprint() = %q; want %q", got, tt.want)
			}
		})
	}
}

// TestCIDRLeaf_FprintRec verifies the canonical 3-level hierarchical ASCII tree output
// consisting of a top-level default prefix (0.0.0.0/0 or ::/0), an intermediate stride boundary fringe,
// and a path-compressed CIDRLeaf nested underneath the fringe.
func TestCIDRLeaf_FprintRec(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		nodeSetup func() *FastACLNode
		ptx       PathContext
		want      []string
	}{
		{
			name: "IPv4 3-level hierarchy: default prefix -> /8 fringe -> /17 CIDRLeaf",
			nodeSetup: func() *FastACLNode {
				n := &FastACLNode{}

				pfx := mpp("0.0.0.0/0")
				n.Insert(pfx, 0)

				pfxFringe := mpp("10.0.0.0/8")
				n.Insert(pfxFringe, 0)

				pfxLeaf := mpp("10.0.0.0/17")
				n.Insert(pfxLeaf, 0)

				return n
			},
			ptx: PathContext{
				Depth: 0,
				Idx:   0, // Global root evaluation scope
				Is4:   true,
			},
			want: []string{
				"└─ 0.0.0.0/0",
				"   └─ 10.0.0.0/8",
				"      └─ 10.0.0.0/17",
			},
		},
		{
			name: "IPv6 3-level hierarchy: default prefix -> /16 fringe -> /33 CIDRLeaf",
			nodeSetup: func() *FastACLNode {
				n := &FastACLNode{}

				pfx := mpp("::/0")
				n.Insert(pfx, 0)

				pfxFringe := mpp("2001::/16")
				n.Insert(pfxFringe, 0)

				pfxLeaf := mpp("2001::/33")
				n.Insert(pfxLeaf, 0)

				return n
			},
			ptx: PathContext{
				Depth: 0,
				Idx:   0, // Global root evaluation scope
				Is4:   false,
			},
			want: []string{
				"└─ ::/0",
				"   └─ 2001::/16",
				"      └─ 2001::/33",
			},
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			node := tt.nodeSetup()
			var buf bytes.Buffer

			err := node.FprintRec(&buf, tt.ptx, "")
			if err != nil {
				t.Fatalf("FprintRec() unexpected error: %v", err)
			}

			var wantStr string
			if len(tt.want) > 0 {
				wantStr = strings.Join(tt.want, "\n") + "\n"
			}

			got := buf.String()
			if got != wantStr {
				t.Errorf("FprintRec() mismatch:\ngot:\n%s\nwant:\n%s", got, wantStr)
			}
		})
	}
}

// TestFastACLNode_DumpRec verifies the recursive ASCII dump formatting of FastACLNode
// structures across IPv4 and IPv6 topologies, covering empty nodes, local prefixes,
// fringe entries, CIDR leaves, and nested child subtrees.
func TestFastACLNode_DumpRec(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		nodeSetup func() *FastACLNode
		path      StridePath
		depth     int
		is4       bool
		want      []string
	}{
		{
			name: "Empty node produces no dump output",
			nodeSetup: func() *FastACLNode {
				return &FastACLNode{}
			},
			path:  StridePath{},
			depth: 0,
			is4:   true,
			want:  nil,
		},
		{
			name: "IPv4 single root node with prefixes and fringes",
			nodeSetup: func() *FastACLNode {
				n := &FastACLNode{}

				// Insert 0.0.0.0/0 at root depth 0
				n.Insert(mpp("0.0.0.0/0"), 0)

				// Insert fringe entry for 10.0.0.0/8 at root octet 10
				n.InsertFringe(10)

				return n
			},
			path:  StridePath{},
			depth: 0,
			is4:   true,
			want: []string{
				"",
				"[STOP] depth:  0 path: [] / 0",
				"prefix(#1): [1]➜{0.0.0.0/0}",
				"fringe(#1): [10]➜{10.0.0.0/8}",
			},
		},
		{
			name: "IPv4 node with nested child FastACLNode and CIDRLeaf",
			nodeSetup: func() *FastACLNode {
				parent := &FastACLNode{}
				child := &FastACLNode{}

				// Root prefix 0.0.0.0/0 at depth 0
				parent.Insert(mpp("0.0.0.0/0"), 0)

				// Child prefix 10.0.0.0/8 inserted at depth 1 scope
				child.Insert(mpp("10.0.0.0/8"), 1)
				parent.InsertChild(10, child)

				// Compressed leaf for 192.168.0.0/16 under octet 192
				leaf := &CIDRLeaf{Prefix: mpp("192.168.0.0/16")}
				parent.InsertChild(192, leaf)

				return parent
			},
			path:  StridePath{},
			depth: 0,
			is4:   true,
			want: []string{
				"",
				"[FULL] depth:  0 path: [] / 0",
				"prefix(#1): [1]➜{0.0.0.0/0}",
				" child(#2): [10]↓ [192]➜{192.168.0.0/16}",
				"",
				".[STOP] depth:  1 path: [10] / 8",
				".prefix(#1): [1]➜{10.0.0.0/8}",
			},
		},
		{
			name: "IPv6 deep stride traversal node dump",
			nodeSetup: func() *FastACLNode {
				n := &FastACLNode{}

				// Insert 2001:db8::/32 prefix at depth 4 scope
				n.Insert(mpp("2001:db8::/32"), 4)

				return n
			},
			path:  StridePath{0x20, 0x01, 0x0d, 0xb8},
			depth: 4,
			is4:   false,
			want: []string{
				"",
				"....[STOP] depth:  4 path: [2001:0db8] / 32",
				"....prefix(#1): [1]➜{2001:db8::/32}",
			},
		},
		{
			name: "IPv4 node with multiple prefixes and fringes",
			nodeSetup: func() *FastACLNode {
				n := &FastACLNode{}

				// Insert 0.0.0.0/0 (idx 1) and 128.0.0.0/1 (idx 3 in CBT)
				n.InsertPrefix(1)
				n.InsertPrefix(3)

				// Insert two fringe entries
				n.InsertFringe(10)
				n.InsertFringe(20)

				return n
			},
			path:  StridePath{},
			depth: 0,
			is4:   true,
			want: []string{
				"",
				"[STOP] depth:  0 path: [] / 0",
				"prefix(#2): [1]➜{0.0.0.0/0} [3]➜{128.0.0.0/1}",
				"fringe(#2): [10]➜{10.0.0.0/8} [20]➜{20.0.0.0/8}",
			},
		},
		{
			name: "IPv4 halfNode classification (SubNodes + Leaf, no local prefixes/fringes)",
			nodeSetup: func() *FastACLNode {
				parent := &FastACLNode{}
				child := &FastACLNode{}

				// SubNode under octet 10
				child.Insert(mpp("10.0.0.0/8"), 1)
				parent.InsertChild(10, child)

				// CIDRLeaf under octet 192 (no local prefixes or fringes on parent)
				leaf := &CIDRLeaf{Prefix: mpp("192.168.0.0/16")}
				parent.InsertChild(192, leaf)

				return parent
			},
			path:  StridePath{},
			depth: 0,
			is4:   true,
			want: []string{
				"",
				"[HALF] depth:  0 path: [] / 0",
				" child(#2): [10]↓ [192]➜{192.168.0.0/16}",
				"",
				".[STOP] depth:  1 path: [10] / 8",
				".prefix(#1): [1]➜{10.0.0.0/8}",
			},
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			node := tt.nodeSetup()
			var buf bytes.Buffer

			node.DumpRec(&buf, tt.path, tt.depth, tt.is4)

			var wantStr string
			if len(tt.want) > 0 {
				wantStr = strings.Join(tt.want, "\n") + "\n"
			}

			got := buf.String()
			if got != wantStr {
				t.Errorf("DumpRec() mismatch:\ngot:\n%s\nwant:\n%s", got, wantStr)
			}
		})
	}
}

// TestFastACLNode_Insert verifies prefix insertion mechanics, including direct prefix
// placement, fringe placement, path-compressed CIDRLeaf generation, update semantics,
// structural collision resolution (leaf push-down), and visual tree structure via DumpRec.
func TestFastACLNode_Insert(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		ops       []netip.Prefix
		wantPfx   netip.Prefix
		wantExist bool
		wantDump  []string
		verify    func(t *testing.T, root *FastACLNode)
	}{
		{
			name: "IPv4 insert default route /0 at root",
			ops: []netip.Prefix{
				mpp("0.0.0.0/0"),
			},
			wantPfx:   mpp("0.0.0.0/0"),
			wantExist: false,
			wantDump: []string{
				"",
				"[STOP] depth:  0 path: [] / 0",
				"prefix(#1): [1]➜{0.0.0.0/0}",
			},
			verify: func(t *testing.T, root *FastACLNode) {
				t.Helper()
				if root.PrefixCount() != 1 {
					t.Errorf("got PrefixCount %d, want 1", root.PrefixCount())
				}
			},
		},
		{
			name: "IPv4 insert update existing prefix returns true",
			ops: []netip.Prefix{
				mpp("10.0.0.0/8"),
				mpp("10.0.0.0/8"),
			},
			wantPfx:   mpp("10.0.0.0/8"),
			wantExist: true,
			wantDump: []string{
				"",
				"[STOP] depth:  0 path: [] / 0",
				"fringe(#1): [10]➜{10.0.0.0/8}",
			},
			verify: func(t *testing.T, root *FastACLNode) {
				t.Helper()
				if root.FringeCount() != 1 {
					t.Errorf("got FringeCount %d, want 1", root.FringeCount())
				}
			},
		},
		{
			name: "IPv4 insert path compressed leaf into empty child slot",
			ops: []netip.Prefix{
				mpp("192.168.1.0/24"),
			},
			wantPfx:   mpp("192.168.1.0/24"),
			wantExist: false,
			wantDump: []string{
				"",
				"[STOP] depth:  0 path: [] / 0",
				" child(#1): [192]➜{192.168.1.0/24}",
			},
			verify: func(t *testing.T, root *FastACLNode) {
				t.Helper()
				child, ok := root.GetChild(192)
				if !ok {
					t.Fatal("expected child at octet 192")
				}
				leaf, ok := child.(*CIDRLeaf)
				if !ok {
					t.Fatalf("expected *CIDRLeaf, got %T", child)
				}
				if leaf.Prefix != mpp("192.168.1.0/24") {
					t.Errorf("got leaf prefix %s, want 192.168.1.0/24", leaf.Prefix)
				}
			},
		},
		{
			name: "IPv4 insert collision resolution pushes existing leaf down",
			ops: []netip.Prefix{
				mpp("10.1.1.0/24"),
				mpp("10.1.2.0/24"),
			},
			wantPfx:   mpp("10.1.2.0/24"),
			wantExist: false,
			wantDump: []string{
				"",
				"[PATH] depth:  0 path: [] / 0",
				" child(#1): [10]↓",
				"",
				".[PATH] depth:  1 path: [10] / 8",
				". child(#1): [1]↓",
				"",
				"..[STOP] depth:  2 path: [10.1] / 16",
				"..fringe(#2): [1]➜{10.1.1.0/24} [2]➜{10.1.2.0/24}",
			},
			verify: func(t *testing.T, root *FastACLNode) {
				t.Helper()
				child, ok := root.GetChild(10)
				if !ok {
					t.Fatal("expected child at octet 10")
				}
				subNode, ok := child.(*FastACLNode)
				if !ok {
					t.Fatalf("expected intermediate *FastACLNode, got %T", child)
				}
				if !subNode.Children.Test(1) {
					t.Error("expected subnode child at octet 1 under root[10]")
				}
			},
		},
		{
			name: "IPv6 insert deep stride traversal compressed leaf",
			ops: []netip.Prefix{
				mpp("2001:db8:85a3::/48"),
			},
			wantPfx:   mpp("2001:db8:85a3::/48"),
			wantExist: false,
			wantDump: []string{
				"",
				"[STOP] depth:  0 path: [] / 0",
				" child(#1): [0x20]➜{2001:db8:85a3::/48}",
			},
			verify: func(t *testing.T, root *FastACLNode) {
				t.Helper()
				child, ok := root.GetChild(0x20)
				if !ok {
					t.Fatal("expected child at octet 0x20")
				}
				if _, ok := child.(*CIDRLeaf); !ok {
					t.Fatalf("expected *CIDRLeaf at root, got %T", child)
				}
			},
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			root := &FastACLNode{}
			var lastExists bool

			for _, pfx := range tt.ops {
				lastExists = root.Insert(pfx, 0)
			}

			if lastExists != tt.wantExist {
				t.Errorf("Insert(%s) exist flag = %v, want %v", tt.wantPfx, lastExists, tt.wantExist)
			}

			if tt.verify != nil {
				tt.verify(t, root)
			}

			if len(tt.wantDump) > 0 {
				var buf bytes.Buffer
				is4 := tt.wantPfx.Addr().Is4()
				root.DumpRec(&buf, StridePath{}, 0, is4)

				wantStr := strings.Join(tt.wantDump, "\n") + "\n"
				if got := buf.String(); got != wantStr {
					t.Errorf("DumpRec() mismatch after Insert:\ngot:\n%s\nwant:\n%s", got, wantStr)
				}
			}
		})
	}
}

// TestFastACLNode_Delete verifies single and multi-prefix deletion operations across all
// internal node types (Prefixes, Fringes, CIDRLeaves) and verifies bottom-up path
// re-compression (PurgeAndCompress) using DumpRec output matching.
func TestFastACLNode_Delete(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		initial   []netip.Prefix
		deletePfx []netip.Prefix
		wantExist bool
		wantDump  []string
		verify    func(t *testing.T, root *FastACLNode)
	}{
		{
			name: "IPv4 delete root default prefix /0",
			initial: []netip.Prefix{
				mpp("0.0.0.0/0"),
			},
			deletePfx: []netip.Prefix{
				mpp("0.0.0.0/0"),
			},
			wantExist: true,
			wantDump:  nil,
			verify: func(t *testing.T, root *FastACLNode) {
				t.Helper()
				if !root.IsEmpty() {
					t.Error("expected root node to be empty after deletion")
				}
			},
		},
		{
			name: "IPv4 delete non-existent prefix returns false",
			initial: []netip.Prefix{
				mpp("10.0.0.0/8"),
			},
			deletePfx: []netip.Prefix{
				mpp("192.168.0.0/16"),
			},
			wantExist: false,
			wantDump: []string{
				"",
				"[STOP] depth:  0 path: [] / 0",
				"fringe(#1): [10]➜{10.0.0.0/8}",
			},
			verify: func(t *testing.T, root *FastACLNode) {
				t.Helper()
				if root.IsEmpty() {
					t.Error("root node should not be empty")
				}
			},
		},
		{
			name: "IPv4 delete fringe prefix /8",
			initial: []netip.Prefix{
				mpp("10.0.0.0/8"),
			},
			deletePfx: []netip.Prefix{
				mpp("10.0.0.0/8"),
			},
			wantExist: true,
			wantDump:  nil,
			verify: func(t *testing.T, root *FastACLNode) {
				t.Helper()
				if root.FringeCount() != 0 {
					t.Errorf("got FringeCount %d, want 0", root.FringeCount())
				}
			},
		},
		{
			name: "IPv4 delete compressed leaf",
			initial: []netip.Prefix{
				mpp("192.168.1.0/24"),
			},
			deletePfx: []netip.Prefix{
				mpp("192.168.1.0/24"),
			},
			wantExist: true,
			wantDump:  nil,
			verify: func(t *testing.T, root *FastACLNode) {
				t.Helper()
				if root.ChildCount() != 0 {
					t.Errorf("got ChildCount %d, want 0", root.ChildCount())
				}
			},
		},
		{
			name: "IPv4 delete multiple prefixes sequentially",
			initial: []netip.Prefix{
				mpp("10.0.0.0/8"),
				mpp("192.168.1.0/24"),
				mpp("172.16.0.0/12"),
			},
			deletePfx: []netip.Prefix{
				mpp("10.0.0.0/8"),
				mpp("192.168.1.0/24"),
			},
			wantExist: true,
			wantDump: []string{
				"",
				"[STOP] depth:  0 path: [] / 0",
				" child(#1): [172]➜{172.16.0.0/12}",
			},
			verify: func(t *testing.T, root *FastACLNode) {
				t.Helper()
				if root.FringeCount() != 0 {
					t.Errorf("got FringeCount %d, want 0", root.FringeCount())
				}
				if root.ChildCount() != 1 {
					t.Errorf("got ChildCount %d, want 1", root.ChildCount())
				}
			},
		},
		{
			name: "IPv4 delete triggers bottom-up path re-compression",
			initial: []netip.Prefix{
				mpp("10.1.1.0/24"),
				mpp("10.1.2.0/24"),
			},
			deletePfx: []netip.Prefix{
				mpp("10.1.2.0/24"),
			},
			wantExist: true,
			wantDump: []string{
				"",
				"[STOP] depth:  0 path: [] / 0",
				" child(#1): [10]➜{10.1.1.0/24}",
			},
			verify: func(t *testing.T, root *FastACLNode) {
				t.Helper()
				child, ok := root.GetChild(10)
				if !ok {
					t.Fatal("expected compressed leaf child at octet 10")
				}
				leaf, ok := child.(*CIDRLeaf)
				if !ok {
					t.Fatalf("expected intermediate node to be re-compressed to *CIDRLeaf, got %T", child)
				}
				if leaf.Prefix != mpp("10.1.1.0/24") {
					t.Errorf("got elevated leaf prefix %s, want 10.1.1.0/24", leaf.Prefix)
				}
			},
		},
		{
			name: "IPv6 delete compressed leaf with path purging",
			initial: []netip.Prefix{
				mpp("2001:db8::1/128"),
			},
			deletePfx: []netip.Prefix{
				mpp("2001:db8::1/128"),
			},
			wantExist: true,
			wantDump:  nil,
			verify: func(t *testing.T, root *FastACLNode) {
				t.Helper()
				if !root.IsEmpty() {
					t.Error("expected root node to be completely purged after deleting single IPv6 entry")
				}
			},
		},
		{
			name: "IPv6 delete adjacent /128 collapses deep intermediate path to single leaf",
			initial: []netip.Prefix{
				mpp("2001:db8::1/128"),
				mpp("2001:db8::2/128"),
			},
			deletePfx: []netip.Prefix{
				mpp("2001:db8::2/128"),
			},
			wantExist: true,
			wantDump: []string{
				"",
				"[STOP] depth:  0 path: [] / 0",
				" child(#1): [0x20]➜{2001:db8::1/128}",
			},
			verify: func(t *testing.T, root *FastACLNode) {
				t.Helper()

				// Root must contain exactly one child at octet 0x20 (first octet of 2001:db8::1).
				if root.ChildCount() != 1 {
					t.Fatalf("got root ChildCount %d, want 1", root.ChildCount())
				}

				child, ok := root.GetChild(0x20)
				if !ok {
					t.Fatal("expected child at octet 0x20")
				}

				// Verify all intermediate nodes were purged and elevated to a single path-compressed *CIDRLeaf.
				leaf, ok := child.(*CIDRLeaf)
				if !ok {
					t.Fatalf("expected path to collapse back into *CIDRLeaf, got %T", child)
				}

				if leaf.Prefix != mpp("2001:db8::1/128") {
					t.Errorf("got elevated leaf prefix %s, want 2001:db8::1/128", leaf.Prefix)
				}
			},
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			root := &FastACLNode{}
			for _, pfx := range tt.initial {
				root.Insert(pfx, 0)
			}

			var anyExisted bool
			for _, pfx := range tt.deletePfx {
				if root.Delete(pfx) {
					anyExisted = true
				}
			}

			if anyExisted != tt.wantExist {
				t.Errorf("Delete(%v) exist flag = %v, want %v", tt.deletePfx, anyExisted, tt.wantExist)
			}

			if tt.verify != nil {
				tt.verify(t, root)
			}

			if len(tt.wantDump) > 0 || tt.wantDump == nil {
				var buf bytes.Buffer
				is4 := true
				if len(tt.deletePfx) > 0 {
					is4 = tt.deletePfx[0].Addr().Is4()
				}
				root.DumpRec(&buf, StridePath{}, 0, is4)

				var wantStr string
				if len(tt.wantDump) > 0 {
					wantStr = strings.Join(tt.wantDump, "\n") + "\n"
				}

				if got := buf.String(); got != wantStr {
					t.Errorf("DumpRec() mismatch after Delete:\ngot:\n%s\nwant:\n%s", got, wantStr)
				}
			}
		})
	}
}

// TestFastACLNode_AllRec verifies recursive trie traversal and prefix reconstruction
// across direct node prefixes, fringe entries, and path-compressed child leaves,
// ensuring accurate early termination propagation via the yield callback.
func TestFastACLNode_AllRec(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		nodeSetup func() *FastACLNode
		ptx       PathContext
		stopAfter int // stop yield after N elements (-1 for no early stop)
		want      []string
	}{
		{
			name: "Empty node produces no yields",
			nodeSetup: func() *FastACLNode {
				return &FastACLNode{}
			},
			ptx:       PathContext{Is4: true},
			stopAfter: -1,
			want:      nil,
		},
		{
			name: "Node with prefixes, fringes, and compressed leaves (IPv4)",
			nodeSetup: func() *FastACLNode {
				n := &FastACLNode{}

				// Insert direct prefix 0.0.0.0/0
				n.Insert(mpp("0.0.0.0/0"), 0)

				// Insert fringe route 10.0.0.0/8
				n.Insert(mpp("10.0.0.0/8"), 0)

				// Insert path-compressed leaf 192.168.1.0/24
				n.Insert(mpp("192.168.1.0/24"), 0)

				return n
			},
			ptx:       PathContext{Is4: true},
			stopAfter: -1,
			want: []string{
				"0.0.0.0/0",
				"10.0.0.0/8",
				"192.168.1.0/24",
			},
		},
		{
			name: "Nested child subtrees with depth-based path reconstruction (IPv6)",
			nodeSetup: func() *FastACLNode {
				root := &FastACLNode{}
				// Insert divergent IPv6 prefixes sharing the 2001:db8::/32 prefix path
				root.Insert(mpp("2001:db8:1::/48"), 0)
				root.Insert(mpp("2001:db8:2::/48"), 0)
				return root
			},
			ptx:       PathContext{Is4: false},
			stopAfter: -1,
			want: []string{
				"2001:db8:1::/48",
				"2001:db8:2::/48",
			},
		},
		{
			name: "Early termination at direct prefix stage halts traversal",
			nodeSetup: func() *FastACLNode {
				n := &FastACLNode{}
				n.Insert(mpp("10.0.0.0/8"), 0)
				n.Insert(mpp("192.168.0.0/16"), 0)
				return n
			},
			ptx:       PathContext{Is4: true},
			stopAfter: 1,
			want: []string{
				"10.0.0.0/8",
			},
		},
		{
			name: "Early termination during child subtree descent propagates upward",
			nodeSetup: func() *FastACLNode {
				parent := &FastACLNode{}

				// Root prefix
				parent.Insert(mpp("0.0.0.0/0"), 0)

				// Two nested prefixes forcing intermediate subnode creation
				parent.Insert(mpp("10.1.1.0/24"), 0)
				parent.Insert(mpp("10.1.2.0/24"), 0)

				return parent
			},
			ptx:       PathContext{Is4: true},
			stopAfter: 2, // stop after root prefix + 1st nested leaf
			want: []string{
				"0.0.0.0/0",
				"10.1.1.0/24",
			},
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			node := tt.nodeSetup()

			var got []string
			count := 0

			completed := node.AllRec(tt.ptx, func(pfx netip.Prefix) bool {
				got = append(got, pfx.String())
				count++
				if tt.stopAfter > 0 && count == tt.stopAfter {
					return false
				}
				return true
			})

			if tt.stopAfter > 0 && completed {
				t.Error("AllRec() returned true, want false (early termination)")
			}
			if tt.stopAfter < 0 && !completed {
				t.Error("AllRec() returned false, want true (complete traversal)")
			}

			// Sort results to ensure deterministic assertion regardless of map/bitset iteration order
			slices.Sort(got)
			want := slices.Clone(tt.want)
			slices.Sort(want)

			if !slices.Equal(got, want) {
				t.Errorf("AllRec() prefix mismatch:\ngot:  %v\nwant: %v", got, want)
			}
		})
	}
}
