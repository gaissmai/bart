//go:build !big_endian

package bart

// allotLookupTbl, as precalculated bitsets,
// map the baseIndex to bitset with precomputed complete binary tree.
//
//	  // 1 <= idx <= 511
//		func allotRec(aTbl *bitset.BitSet, idx uint) {
//			aTbl = aTbl.Set(idx)
//			if idx >= 256 {
//				return
//			}
//			allotRec(aTbl, idx<<1)
//			allotRec(aTbl, idx<<1+1)
//		}
//
// Used for bitset intersections instead of range loops in overlaps tests.
var allotLookupTbl = [512][8]uint64{
	/* idx == 0   */ {0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0},
	/* idx == 1   */ {0xfffffffffffffffe, 0xffffffffffffffff, 0xffffffffffffffff, 0xffffffffffffffff, 0xffffffffffffffff, 0xffffffffffffffff, 0xffffffffffffffff, 0xffffffffffffffff},
	/* idx == 2   */ {0xffff00ff0f34, 0xffffffff, 0xffffffffffffffff, 0x0, 0xffffffffffffffff, 0xffffffffffffffff, 0x0, 0x0},
	/* idx == 3   */ {0xffff0000ff00f0c8, 0xffffffff00000000, 0x0, 0xffffffffffffffff, 0x0, 0x0, 0xffffffffffffffff, 0xffffffffffffffff},
	/* idx == 4   */ {0xff000f0310, 0xffff, 0xffffffff, 0x0, 0xffffffffffffffff, 0x0, 0x0, 0x0},
	/* idx == 5   */ {0xff0000f00c20, 0xffff0000, 0xffffffff00000000, 0x0, 0x0, 0xffffffffffffffff, 0x0, 0x0},
	/* idx == 6   */ {0xff00000f003040, 0xffff00000000, 0x0, 0xffffffff, 0x0, 0x0, 0xffffffffffffffff, 0x0},
	/* idx == 7   */ {0xff000000f000c080, 0xffff000000000000, 0x0, 0xffffffff00000000, 0x0, 0x0, 0x0, 0xffffffffffffffff},
	/* idx == 8   */ {0xf00030100, 0xff, 0xffff, 0x0, 0xffffffff, 0x0, 0x0, 0x0},
	/* idx == 9   */ {0xf0000c0200, 0xff00, 0xffff0000, 0x0, 0xffffffff00000000, 0x0, 0x0, 0x0},
	/* idx == 10  */ {0xf0000300400, 0xff0000, 0xffff00000000, 0x0, 0x0, 0xffffffff, 0x0, 0x0},
	/* idx == 11  */ {0xf00000c00800, 0xff000000, 0xffff000000000000, 0x0, 0x0, 0xffffffff00000000, 0x0, 0x0},
	/* idx == 12  */ {0xf000003001000, 0xff00000000, 0x0, 0xffff, 0x0, 0x0, 0xffffffff, 0x0},
	/* idx == 13  */ {0xf000000c002000, 0xff0000000000, 0x0, 0xffff0000, 0x0, 0x0, 0xffffffff00000000, 0x0},
	/* idx == 14  */ {0xf00000030004000, 0xff000000000000, 0x0, 0xffff00000000, 0x0, 0x0, 0x0, 0xffffffff},
	/* idx == 15  */ {0xf0000000c0008000, 0xff00000000000000, 0x0, 0xffff000000000000, 0x0, 0x0, 0x0, 0xffffffff00000000},
	/* idx == 16  */ {0x300010000, 0xf, 0xff, 0x0, 0xffff, 0x0, 0x0, 0x0},
	/* idx == 17  */ {0xc00020000, 0xf0, 0xff00, 0x0, 0xffff0000, 0x0, 0x0, 0x0},
	/* idx == 18  */ {0x3000040000, 0xf00, 0xff0000, 0x0, 0xffff00000000, 0x0, 0x0, 0x0},
	/* idx == 19  */ {0xc000080000, 0xf000, 0xff000000, 0x0, 0xffff000000000000, 0x0, 0x0, 0x0},
	/* idx == 20  */ {0x30000100000, 0xf0000, 0xff00000000, 0x0, 0x0, 0xffff, 0x0, 0x0},
	/* idx == 21  */ {0xc0000200000, 0xf00000, 0xff0000000000, 0x0, 0x0, 0xffff0000, 0x0, 0x0},
	/* idx == 22  */ {0x300000400000, 0xf000000, 0xff000000000000, 0x0, 0x0, 0xffff00000000, 0x0, 0x0},
	/* idx == 23  */ {0xc00000800000, 0xf0000000, 0xff00000000000000, 0x0, 0x0, 0xffff000000000000, 0x0, 0x0},
	/* idx == 24  */ {0x3000001000000, 0xf00000000, 0x0, 0xff, 0x0, 0x0, 0xffff, 0x0},
	/* idx == 25  */ {0xc000002000000, 0xf000000000, 0x0, 0xff00, 0x0, 0x0, 0xffff0000, 0x0},
	/* idx == 26  */ {0x30000004000000, 0xf0000000000, 0x0, 0xff0000, 0x0, 0x0, 0xffff00000000, 0x0},
	/* idx == 27  */ {0xc0000008000000, 0xf00000000000, 0x0, 0xff000000, 0x0, 0x0, 0xffff000000000000, 0x0},
	/* idx == 28  */ {0x300000010000000, 0xf000000000000, 0x0, 0xff00000000, 0x0, 0x0, 0x0, 0xffff},
	/* idx == 29  */ {0xc00000020000000, 0xf0000000000000, 0x0, 0xff0000000000, 0x0, 0x0, 0x0, 0xffff0000},
	/* idx == 30  */ {0x3000000040000000, 0xf00000000000000, 0x0, 0xff000000000000, 0x0, 0x0, 0x0, 0xffff00000000},
	/* idx == 31  */ {0xc000000080000000, 0xf000000000000000, 0x0, 0xff00000000000000, 0x0, 0x0, 0x0, 0xffff000000000000},
	/* idx == 32  */ {0x100000000, 0x3, 0xf, 0x0, 0xff, 0x0, 0x0, 0x0},
	/* idx == 33  */ {0x200000000, 0xc, 0xf0, 0x0, 0xff00, 0x0, 0x0, 0x0},
	/* idx == 34  */ {0x400000000, 0x30, 0xf00, 0x0, 0xff0000, 0x0, 0x0, 0x0},
	/* idx == 35  */ {0x800000000, 0xc0, 0xf000, 0x0, 0xff000000, 0x0, 0x0, 0x0},
	/* idx == 36  */ {0x1000000000, 0x300, 0xf0000, 0x0, 0xff00000000, 0x0, 0x0, 0x0},
	/* idx == 37  */ {0x2000000000, 0xc00, 0xf00000, 0x0, 0xff0000000000, 0x0, 0x0, 0x0},
	/* idx == 38  */ {0x4000000000, 0x3000, 0xf000000, 0x0, 0xff000000000000, 0x0, 0x0, 0x0},
	/* idx == 39  */ {0x8000000000, 0xc000, 0xf0000000, 0x0, 0xff00000000000000, 0x0, 0x0, 0x0},
	/* idx == 40  */ {0x10000000000, 0x30000, 0xf00000000, 0x0, 0x0, 0xff, 0x0, 0x0},
	/* idx == 41  */ {0x20000000000, 0xc0000, 0xf000000000, 0x0, 0x0, 0xff00, 0x0, 0x0},
	/* idx == 42  */ {0x40000000000, 0x300000, 0xf0000000000, 0x0, 0x0, 0xff0000, 0x0, 0x0},
	/* idx == 43  */ {0x80000000000, 0xc00000, 0xf00000000000, 0x0, 0x0, 0xff000000, 0x0, 0x0},
	/* idx == 44  */ {0x100000000000, 0x3000000, 0xf000000000000, 0x0, 0x0, 0xff00000000, 0x0, 0x0},
	/* idx == 45  */ {0x200000000000, 0xc000000, 0xf0000000000000, 0x0, 0x0, 0xff0000000000, 0x0, 0x0},
	/* idx == 46  */ {0x400000000000, 0x30000000, 0xf00000000000000, 0x0, 0x0, 0xff000000000000, 0x0, 0x0},
	/* idx == 47  */ {0x800000000000, 0xc0000000, 0xf000000000000000, 0x0, 0x0, 0xff00000000000000, 0x0, 0x0},
	/* idx == 48  */ {0x1000000000000, 0x300000000, 0x0, 0xf, 0x0, 0x0, 0xff, 0x0},
	/* idx == 49  */ {0x2000000000000, 0xc00000000, 0x0, 0xf0, 0x0, 0x0, 0xff00, 0x0},
	/* idx == 50  */ {0x4000000000000, 0x3000000000, 0x0, 0xf00, 0x0, 0x0, 0xff0000, 0x0},
	/* idx == 51  */ {0x8000000000000, 0xc000000000, 0x0, 0xf000, 0x0, 0x0, 0xff000000, 0x0},
	/* idx == 52  */ {0x10000000000000, 0x30000000000, 0x0, 0xf0000, 0x0, 0x0, 0xff00000000, 0x0},
	/* idx == 53  */ {0x20000000000000, 0xc0000000000, 0x0, 0xf00000, 0x0, 0x0, 0xff0000000000, 0x0},
	/* idx == 54  */ {0x40000000000000, 0x300000000000, 0x0, 0xf000000, 0x0, 0x0, 0xff000000000000, 0x0},
	/* idx == 55  */ {0x80000000000000, 0xc00000000000, 0x0, 0xf0000000, 0x0, 0x0, 0xff00000000000000, 0x0},
	/* idx == 56  */ {0x100000000000000, 0x3000000000000, 0x0, 0xf00000000, 0x0, 0x0, 0x0, 0xff},
	/* idx == 57  */ {0x200000000000000, 0xc000000000000, 0x0, 0xf000000000, 0x0, 0x0, 0x0, 0xff00},
	/* idx == 58  */ {0x400000000000000, 0x30000000000000, 0x0, 0xf0000000000, 0x0, 0x0, 0x0, 0xff0000},
	/* idx == 59  */ {0x800000000000000, 0xc0000000000000, 0x0, 0xf00000000000, 0x0, 0x0, 0x0, 0xff000000},
	/* idx == 60  */ {0x1000000000000000, 0x300000000000000, 0x0, 0xf000000000000, 0x0, 0x0, 0x0, 0xff00000000},
	/* idx == 61  */ {0x2000000000000000, 0xc00000000000000, 0x0, 0xf0000000000000, 0x0, 0x0, 0x0, 0xff0000000000},
	/* idx == 62  */ {0x4000000000000000, 0x3000000000000000, 0x0, 0xf00000000000000, 0x0, 0x0, 0x0, 0xff000000000000},
	/* idx == 63  */ {0x8000000000000000, 0xc000000000000000, 0x0, 0xf000000000000000, 0x0, 0x0, 0x0, 0xff00000000000000},
	/* idx == 64  */ {0x0, 0x1, 0x3, 0x0, 0xf, 0x0, 0x0, 0x0},
	/* idx == 65  */ {0x0, 0x2, 0xc, 0x0, 0xf0, 0x0, 0x0, 0x0},
	/* idx == 66  */ {0x0, 0x4, 0x30, 0x0, 0xf00, 0x0, 0x0, 0x0},
	/* idx == 67  */ {0x0, 0x8, 0xc0, 0x0, 0xf000, 0x0, 0x0, 0x0},
	/* idx == 68  */ {0x0, 0x10, 0x300, 0x0, 0xf0000, 0x0, 0x0, 0x0},
	/* idx == 69  */ {0x0, 0x20, 0xc00, 0x0, 0xf00000, 0x0, 0x0, 0x0},
	/* idx == 70  */ {0x0, 0x40, 0x3000, 0x0, 0xf000000, 0x0, 0x0, 0x0},
	/* idx == 71  */ {0x0, 0x80, 0xc000, 0x0, 0xf0000000, 0x0, 0x0, 0x0},
	/* idx == 72  */ {0x0, 0x100, 0x30000, 0x0, 0xf00000000, 0x0, 0x0, 0x0},
	/* idx == 73  */ {0x0, 0x200, 0xc0000, 0x0, 0xf000000000, 0x0, 0x0, 0x0},
	/* idx == 74  */ {0x0, 0x400, 0x300000, 0x0, 0xf0000000000, 0x0, 0x0, 0x0},
	/* idx == 75  */ {0x0, 0x800, 0xc00000, 0x0, 0xf00000000000, 0x0, 0x0, 0x0},
	/* idx == 76  */ {0x0, 0x1000, 0x3000000, 0x0, 0xf000000000000, 0x0, 0x0, 0x0},
	/* idx == 77  */ {0x0, 0x2000, 0xc000000, 0x0, 0xf0000000000000, 0x0, 0x0, 0x0},
	/* idx == 78  */ {0x0, 0x4000, 0x30000000, 0x0, 0xf00000000000000, 0x0, 0x0, 0x0},
	/* idx == 79  */ {0x0, 0x8000, 0xc0000000, 0x0, 0xf000000000000000, 0x0, 0x0, 0x0},
	/* idx == 80  */ {0x0, 0x10000, 0x300000000, 0x0, 0x0, 0xf, 0x0, 0x0},
	/* idx == 81  */ {0x0, 0x20000, 0xc00000000, 0x0, 0x0, 0xf0, 0x0, 0x0},
	/* idx == 82  */ {0x0, 0x40000, 0x3000000000, 0x0, 0x0, 0xf00, 0x0, 0x0},
	/* idx == 83  */ {0x0, 0x80000, 0xc000000000, 0x0, 0x0, 0xf000, 0x0, 0x0},
	/* idx == 84  */ {0x0, 0x100000, 0x30000000000, 0x0, 0x0, 0xf0000, 0x0, 0x0},
	/* idx == 85  */ {0x0, 0x200000, 0xc0000000000, 0x0, 0x0, 0xf00000, 0x0, 0x0},
	/* idx == 86  */ {0x0, 0x400000, 0x300000000000, 0x0, 0x0, 0xf000000, 0x0, 0x0},
	/* idx == 87  */ {0x0, 0x800000, 0xc00000000000, 0x0, 0x0, 0xf0000000, 0x0, 0x0},
	/* idx == 88  */ {0x0, 0x1000000, 0x3000000000000, 0x0, 0x0, 0xf00000000, 0x0, 0x0},
	/* idx == 89  */ {0x0, 0x2000000, 0xc000000000000, 0x0, 0x0, 0xf000000000, 0x0, 0x0},
	/* idx == 90  */ {0x0, 0x4000000, 0x30000000000000, 0x0, 0x0, 0xf0000000000, 0x0, 0x0},
	/* idx == 91  */ {0x0, 0x8000000, 0xc0000000000000, 0x0, 0x0, 0xf00000000000, 0x0, 0x0},
	/* idx == 92  */ {0x0, 0x10000000, 0x300000000000000, 0x0, 0x0, 0xf000000000000, 0x0, 0x0},
	/* idx == 93  */ {0x0, 0x20000000, 0xc00000000000000, 0x0, 0x0, 0xf0000000000000, 0x0, 0x0},
	/* idx == 94  */ {0x0, 0x40000000, 0x3000000000000000, 0x0, 0x0, 0xf00000000000000, 0x0, 0x0},
	/* idx == 95  */ {0x0, 0x80000000, 0xc000000000000000, 0x0, 0x0, 0xf000000000000000, 0x0, 0x0},
	/* idx == 96  */ {0x0, 0x100000000, 0x0, 0x3, 0x0, 0x0, 0xf, 0x0},
	/* idx == 97  */ {0x0, 0x200000000, 0x0, 0xc, 0x0, 0x0, 0xf0, 0x0},
	/* idx == 98  */ {0x0, 0x400000000, 0x0, 0x30, 0x0, 0x0, 0xf00, 0x0},
	/* idx == 99  */ {0x0, 0x800000000, 0x0, 0xc0, 0x0, 0x0, 0xf000, 0x0},
	/* idx == 100 */ {0x0, 0x1000000000, 0x0, 0x300, 0x0, 0x0, 0xf0000, 0x0},
	/* idx == 101 */ {0x0, 0x2000000000, 0x0, 0xc00, 0x0, 0x0, 0xf00000, 0x0},
	/* idx == 102 */ {0x0, 0x4000000000, 0x0, 0x3000, 0x0, 0x0, 0xf000000, 0x0},
	/* idx == 103 */ {0x0, 0x8000000000, 0x0, 0xc000, 0x0, 0x0, 0xf0000000, 0x0},
	/* idx == 104 */ {0x0, 0x10000000000, 0x0, 0x30000, 0x0, 0x0, 0xf00000000, 0x0},
	/* idx == 105 */ {0x0, 0x20000000000, 0x0, 0xc0000, 0x0, 0x0, 0xf000000000, 0x0},
	/* idx == 106 */ {0x0, 0x40000000000, 0x0, 0x300000, 0x0, 0x0, 0xf0000000000, 0x0},
	/* idx == 107 */ {0x0, 0x80000000000, 0x0, 0xc00000, 0x0, 0x0, 0xf00000000000, 0x0},
	/* idx == 108 */ {0x0, 0x100000000000, 0x0, 0x3000000, 0x0, 0x0, 0xf000000000000, 0x0},
	/* idx == 109 */ {0x0, 0x200000000000, 0x0, 0xc000000, 0x0, 0x0, 0xf0000000000000, 0x0},
	/* idx == 110 */ {0x0, 0x400000000000, 0x0, 0x30000000, 0x0, 0x0, 0xf00000000000000, 0x0},
	/* idx == 111 */ {0x0, 0x800000000000, 0x0, 0xc0000000, 0x0, 0x0, 0xf000000000000000, 0x0},
	/* idx == 112 */ {0x0, 0x1000000000000, 0x0, 0x300000000, 0x0, 0x0, 0x0, 0xf},
	/* idx == 113 */ {0x0, 0x2000000000000, 0x0, 0xc00000000, 0x0, 0x0, 0x0, 0xf0},
	/* idx == 114 */ {0x0, 0x4000000000000, 0x0, 0x3000000000, 0x0, 0x0, 0x0, 0xf00},
	/* idx == 115 */ {0x0, 0x8000000000000, 0x0, 0xc000000000, 0x0, 0x0, 0x0, 0xf000},
	/* idx == 116 */ {0x0, 0x10000000000000, 0x0, 0x30000000000, 0x0, 0x0, 0x0, 0xf0000},
	/* idx == 117 */ {0x0, 0x20000000000000, 0x0, 0xc0000000000, 0x0, 0x0, 0x0, 0xf00000},
	/* idx == 118 */ {0x0, 0x40000000000000, 0x0, 0x300000000000, 0x0, 0x0, 0x0, 0xf000000},
	/* idx == 119 */ {0x0, 0x80000000000000, 0x0, 0xc00000000000, 0x0, 0x0, 0x0, 0xf0000000},
	/* idx == 120 */ {0x0, 0x100000000000000, 0x0, 0x3000000000000, 0x0, 0x0, 0x0, 0xf00000000},
	/* idx == 121 */ {0x0, 0x200000000000000, 0x0, 0xc000000000000, 0x0, 0x0, 0x0, 0xf000000000},
	/* idx == 122 */ {0x0, 0x400000000000000, 0x0, 0x30000000000000, 0x0, 0x0, 0x0, 0xf0000000000},
	/* idx == 123 */ {0x0, 0x800000000000000, 0x0, 0xc0000000000000, 0x0, 0x0, 0x0, 0xf00000000000},
	/* idx == 124 */ {0x0, 0x1000000000000000, 0x0, 0x300000000000000, 0x0, 0x0, 0x0, 0xf000000000000},
	/* idx == 125 */ {0x0, 0x2000000000000000, 0x0, 0xc00000000000000, 0x0, 0x0, 0x0, 0xf0000000000000},
	/* idx == 126 */ {0x0, 0x4000000000000000, 0x0, 0x3000000000000000, 0x0, 0x0, 0x0, 0xf00000000000000},
	/* idx == 127 */ {0x0, 0x8000000000000000, 0x0, 0xc000000000000000, 0x0, 0x0, 0x0, 0xf000000000000000},
	/* idx == 128 */ {0x0, 0x0, 0x1, 0x0, 0x3, 0x0, 0x0, 0x0},
	/* idx == 129 */ {0x0, 0x0, 0x2, 0x0, 0xc, 0x0, 0x0, 0x0},
	/* idx == 130 */ {0x0, 0x0, 0x4, 0x0, 0x30, 0x0, 0x0, 0x0},
	/* idx == 131 */ {0x0, 0x0, 0x8, 0x0, 0xc0, 0x0, 0x0, 0x0},
	/* idx == 132 */ {0x0, 0x0, 0x10, 0x0, 0x300, 0x0, 0x0, 0x0},
	/* idx == 133 */ {0x0, 0x0, 0x20, 0x0, 0xc00, 0x0, 0x0, 0x0},
	/* idx == 134 */ {0x0, 0x0, 0x40, 0x0, 0x3000, 0x0, 0x0, 0x0},
	/* idx == 135 */ {0x0, 0x0, 0x80, 0x0, 0xc000, 0x0, 0x0, 0x0},
	/* idx == 136 */ {0x0, 0x0, 0x100, 0x0, 0x30000, 0x0, 0x0, 0x0},
	/* idx == 137 */ {0x0, 0x0, 0x200, 0x0, 0xc0000, 0x0, 0x0, 0x0},
	/* idx == 138 */ {0x0, 0x0, 0x400, 0x0, 0x300000, 0x0, 0x0, 0x0},
	/* idx == 139 */ {0x0, 0x0, 0x800, 0x0, 0xc00000, 0x0, 0x0, 0x0},
	/* idx == 140 */ {0x0, 0x0, 0x1000, 0x0, 0x3000000, 0x0, 0x0, 0x0},
	/* idx == 141 */ {0x0, 0x0, 0x2000, 0x0, 0xc000000, 0x0, 0x0, 0x0},
	/* idx == 142 */ {0x0, 0x0, 0x4000, 0x0, 0x30000000, 0x0, 0x0, 0x0},
	/* idx == 143 */ {0x0, 0x0, 0x8000, 0x0, 0xc0000000, 0x0, 0x0, 0x0},
	/* idx == 144 */ {0x0, 0x0, 0x10000, 0x0, 0x300000000, 0x0, 0x0, 0x0},
	/* idx == 145 */ {0x0, 0x0, 0x20000, 0x0, 0xc00000000, 0x0, 0x0, 0x0},
	/* idx == 146 */ {0x0, 0x0, 0x40000, 0x0, 0x3000000000, 0x0, 0x0, 0x0},
	/* idx == 147 */ {0x0, 0x0, 0x80000, 0x0, 0xc000000000, 0x0, 0x0, 0x0},
	/* idx == 148 */ {0x0, 0x0, 0x100000, 0x0, 0x30000000000, 0x0, 0x0, 0x0},
	/* idx == 149 */ {0x0, 0x0, 0x200000, 0x0, 0xc0000000000, 0x0, 0x0, 0x0},
	/* idx == 150 */ {0x0, 0x0, 0x400000, 0x0, 0x300000000000, 0x0, 0x0, 0x0},
	/* idx == 151 */ {0x0, 0x0, 0x800000, 0x0, 0xc00000000000, 0x0, 0x0, 0x0},
	/* idx == 152 */ {0x0, 0x0, 0x1000000, 0x0, 0x3000000000000, 0x0, 0x0, 0x0},
	/* idx == 153 */ {0x0, 0x0, 0x2000000, 0x0, 0xc000000000000, 0x0, 0x0, 0x0},
	/* idx == 154 */ {0x0, 0x0, 0x4000000, 0x0, 0x30000000000000, 0x0, 0x0, 0x0},
	/* idx == 155 */ {0x0, 0x0, 0x8000000, 0x0, 0xc0000000000000, 0x0, 0x0, 0x0},
	/* idx == 156 */ {0x0, 0x0, 0x10000000, 0x0, 0x300000000000000, 0x0, 0x0, 0x0},
	/* idx == 157 */ {0x0, 0x0, 0x20000000, 0x0, 0xc00000000000000, 0x0, 0x0, 0x0},
	/* idx == 158 */ {0x0, 0x0, 0x40000000, 0x0, 0x3000000000000000, 0x0, 0x0, 0x0},
	/* idx == 159 */ {0x0, 0x0, 0x80000000, 0x0, 0xc000000000000000, 0x0, 0x0, 0x0},
	/* idx == 160 */ {0x0, 0x0, 0x100000000, 0x0, 0x0, 0x3, 0x0, 0x0},
	/* idx == 161 */ {0x0, 0x0, 0x200000000, 0x0, 0x0, 0xc, 0x0, 0x0},
	/* idx == 162 */ {0x0, 0x0, 0x400000000, 0x0, 0x0, 0x30, 0x0, 0x0},
	/* idx == 163 */ {0x0, 0x0, 0x800000000, 0x0, 0x0, 0xc0, 0x0, 0x0},
	/* idx == 164 */ {0x0, 0x0, 0x1000000000, 0x0, 0x0, 0x300, 0x0, 0x0},
	/* idx == 165 */ {0x0, 0x0, 0x2000000000, 0x0, 0x0, 0xc00, 0x0, 0x0},
	/* idx == 166 */ {0x0, 0x0, 0x4000000000, 0x0, 0x0, 0x3000, 0x0, 0x0},
	/* idx == 167 */ {0x0, 0x0, 0x8000000000, 0x0, 0x0, 0xc000, 0x0, 0x0},
	/* idx == 168 */ {0x0, 0x0, 0x10000000000, 0x0, 0x0, 0x30000, 0x0, 0x0},
	/* idx == 169 */ {0x0, 0x0, 0x20000000000, 0x0, 0x0, 0xc0000, 0x0, 0x0},
	/* idx == 170 */ {0x0, 0x0, 0x40000000000, 0x0, 0x0, 0x300000, 0x0, 0x0},
	/* idx == 171 */ {0x0, 0x0, 0x80000000000, 0x0, 0x0, 0xc00000, 0x0, 0x0},
	/* idx == 172 */ {0x0, 0x0, 0x100000000000, 0x0, 0x0, 0x3000000, 0x0, 0x0},
	/* idx == 173 */ {0x0, 0x0, 0x200000000000, 0x0, 0x0, 0xc000000, 0x0, 0x0},
	/* idx == 174 */ {0x0, 0x0, 0x400000000000, 0x0, 0x0, 0x30000000, 0x0, 0x0},
	/* idx == 175 */ {0x0, 0x0, 0x800000000000, 0x0, 0x0, 0xc0000000, 0x0, 0x0},
	/* idx == 176 */ {0x0, 0x0, 0x1000000000000, 0x0, 0x0, 0x300000000, 0x0, 0x0},
	/* idx == 177 */ {0x0, 0x0, 0x2000000000000, 0x0, 0x0, 0xc00000000, 0x0, 0x0},
	/* idx == 178 */ {0x0, 0x0, 0x4000000000000, 0x0, 0x0, 0x3000000000, 0x0, 0x0},
	/* idx == 179 */ {0x0, 0x0, 0x8000000000000, 0x0, 0x0, 0xc000000000, 0x0, 0x0},
	/* idx == 180 */ {0x0, 0x0, 0x10000000000000, 0x0, 0x0, 0x30000000000, 0x0, 0x0},
	/* idx == 181 */ {0x0, 0x0, 0x20000000000000, 0x0, 0x0, 0xc0000000000, 0x0, 0x0},
	/* idx == 182 */ {0x0, 0x0, 0x40000000000000, 0x0, 0x0, 0x300000000000, 0x0, 0x0},
	/* idx == 183 */ {0x0, 0x0, 0x80000000000000, 0x0, 0x0, 0xc00000000000, 0x0, 0x0},
	/* idx == 184 */ {0x0, 0x0, 0x100000000000000, 0x0, 0x0, 0x3000000000000, 0x0, 0x0},
	/* idx == 185 */ {0x0, 0x0, 0x200000000000000, 0x0, 0x0, 0xc000000000000, 0x0, 0x0},
	/* idx == 186 */ {0x0, 0x0, 0x400000000000000, 0x0, 0x0, 0x30000000000000, 0x0, 0x0},
	/* idx == 187 */ {0x0, 0x0, 0x800000000000000, 0x0, 0x0, 0xc0000000000000, 0x0, 0x0},
	/* idx == 188 */ {0x0, 0x0, 0x1000000000000000, 0x0, 0x0, 0x300000000000000, 0x0, 0x0},
	/* idx == 189 */ {0x0, 0x0, 0x2000000000000000, 0x0, 0x0, 0xc00000000000000, 0x0, 0x0},
	/* idx == 190 */ {0x0, 0x0, 0x4000000000000000, 0x0, 0x0, 0x3000000000000000, 0x0, 0x0},
	/* idx == 191 */ {0x0, 0x0, 0x8000000000000000, 0x0, 0x0, 0xc000000000000000, 0x0, 0x0},
	/* idx == 192 */ {0x0, 0x0, 0x0, 0x1, 0x0, 0x0, 0x3, 0x0},
	/* idx == 193 */ {0x0, 0x0, 0x0, 0x2, 0x0, 0x0, 0xc, 0x0},
	/* idx == 194 */ {0x0, 0x0, 0x0, 0x4, 0x0, 0x0, 0x30, 0x0},
	/* idx == 195 */ {0x0, 0x0, 0x0, 0x8, 0x0, 0x0, 0xc0, 0x0},
	/* idx == 196 */ {0x0, 0x0, 0x0, 0x10, 0x0, 0x0, 0x300, 0x0},
	/* idx == 197 */ {0x0, 0x0, 0x0, 0x20, 0x0, 0x0, 0xc00, 0x0},
	/* idx == 198 */ {0x0, 0x0, 0x0, 0x40, 0x0, 0x0, 0x3000, 0x0},
	/* idx == 199 */ {0x0, 0x0, 0x0, 0x80, 0x0, 0x0, 0xc000, 0x0},
	/* idx == 200 */ {0x0, 0x0, 0x0, 0x100, 0x0, 0x0, 0x30000, 0x0},
	/* idx == 201 */ {0x0, 0x0, 0x0, 0x200, 0x0, 0x0, 0xc0000, 0x0},
	/* idx == 202 */ {0x0, 0x0, 0x0, 0x400, 0x0, 0x0, 0x300000, 0x0},
	/* idx == 203 */ {0x0, 0x0, 0x0, 0x800, 0x0, 0x0, 0xc00000, 0x0},
	/* idx == 204 */ {0x0, 0x0, 0x0, 0x1000, 0x0, 0x0, 0x3000000, 0x0},
	/* idx == 205 */ {0x0, 0x0, 0x0, 0x2000, 0x0, 0x0, 0xc000000, 0x0},
	/* idx == 206 */ {0x0, 0x0, 0x0, 0x4000, 0x0, 0x0, 0x30000000, 0x0},
	/* idx == 207 */ {0x0, 0x0, 0x0, 0x8000, 0x0, 0x0, 0xc0000000, 0x0},
	/* idx == 208 */ {0x0, 0x0, 0x0, 0x10000, 0x0, 0x0, 0x300000000, 0x0},
	/* idx == 209 */ {0x0, 0x0, 0x0, 0x20000, 0x0, 0x0, 0xc00000000, 0x0},
	/* idx == 210 */ {0x0, 0x0, 0x0, 0x40000, 0x0, 0x0, 0x3000000000, 0x0},
	/* idx == 211 */ {0x0, 0x0, 0x0, 0x80000, 0x0, 0x0, 0xc000000000, 0x0},
	/* idx == 212 */ {0x0, 0x0, 0x0, 0x100000, 0x0, 0x0, 0x30000000000, 0x0},
	/* idx == 213 */ {0x0, 0x0, 0x0, 0x200000, 0x0, 0x0, 0xc0000000000, 0x0},
	/* idx == 214 */ {0x0, 0x0, 0x0, 0x400000, 0x0, 0x0, 0x300000000000, 0x0},
	/* idx == 215 */ {0x0, 0x0, 0x0, 0x800000, 0x0, 0x0, 0xc00000000000, 0x0},
	/* idx == 216 */ {0x0, 0x0, 0x0, 0x1000000, 0x0, 0x0, 0x3000000000000, 0x0},
	/* idx == 217 */ {0x0, 0x0, 0x0, 0x2000000, 0x0, 0x0, 0xc000000000000, 0x0},
	/* idx == 218 */ {0x0, 0x0, 0x0, 0x4000000, 0x0, 0x0, 0x30000000000000, 0x0},
	/* idx == 219 */ {0x0, 0x0, 0x0, 0x8000000, 0x0, 0x0, 0xc0000000000000, 0x0},
	/* idx == 220 */ {0x0, 0x0, 0x0, 0x10000000, 0x0, 0x0, 0x300000000000000, 0x0},
	/* idx == 221 */ {0x0, 0x0, 0x0, 0x20000000, 0x0, 0x0, 0xc00000000000000, 0x0},
	/* idx == 222 */ {0x0, 0x0, 0x0, 0x40000000, 0x0, 0x0, 0x3000000000000000, 0x0},
	/* idx == 223 */ {0x0, 0x0, 0x0, 0x80000000, 0x0, 0x0, 0xc000000000000000, 0x0},
	/* idx == 224 */ {0x0, 0x0, 0x0, 0x100000000, 0x0, 0x0, 0x0, 0x3},
	/* idx == 225 */ {0x0, 0x0, 0x0, 0x200000000, 0x0, 0x0, 0x0, 0xc},
	/* idx == 226 */ {0x0, 0x0, 0x0, 0x400000000, 0x0, 0x0, 0x0, 0x30},
	/* idx == 227 */ {0x0, 0x0, 0x0, 0x800000000, 0x0, 0x0, 0x0, 0xc0},
	/* idx == 228 */ {0x0, 0x0, 0x0, 0x1000000000, 0x0, 0x0, 0x0, 0x300},
	/* idx == 229 */ {0x0, 0x0, 0x0, 0x2000000000, 0x0, 0x0, 0x0, 0xc00},
	/* idx == 230 */ {0x0, 0x0, 0x0, 0x4000000000, 0x0, 0x0, 0x0, 0x3000},
	/* idx == 231 */ {0x0, 0x0, 0x0, 0x8000000000, 0x0, 0x0, 0x0, 0xc000},
	/* idx == 232 */ {0x0, 0x0, 0x0, 0x10000000000, 0x0, 0x0, 0x0, 0x30000},
	/* idx == 233 */ {0x0, 0x0, 0x0, 0x20000000000, 0x0, 0x0, 0x0, 0xc0000},
	/* idx == 234 */ {0x0, 0x0, 0x0, 0x40000000000, 0x0, 0x0, 0x0, 0x300000},
	/* idx == 235 */ {0x0, 0x0, 0x0, 0x80000000000, 0x0, 0x0, 0x0, 0xc00000},
	/* idx == 236 */ {0x0, 0x0, 0x0, 0x100000000000, 0x0, 0x0, 0x0, 0x3000000},
	/* idx == 237 */ {0x0, 0x0, 0x0, 0x200000000000, 0x0, 0x0, 0x0, 0xc000000},
	/* idx == 238 */ {0x0, 0x0, 0x0, 0x400000000000, 0x0, 0x0, 0x0, 0x30000000},
	/* idx == 239 */ {0x0, 0x0, 0x0, 0x800000000000, 0x0, 0x0, 0x0, 0xc0000000},
	/* idx == 240 */ {0x0, 0x0, 0x0, 0x1000000000000, 0x0, 0x0, 0x0, 0x300000000},
	/* idx == 241 */ {0x0, 0x0, 0x0, 0x2000000000000, 0x0, 0x0, 0x0, 0xc00000000},
	/* idx == 242 */ {0x0, 0x0, 0x0, 0x4000000000000, 0x0, 0x0, 0x0, 0x3000000000},
	/* idx == 243 */ {0x0, 0x0, 0x0, 0x8000000000000, 0x0, 0x0, 0x0, 0xc000000000},
	/* idx == 244 */ {0x0, 0x0, 0x0, 0x10000000000000, 0x0, 0x0, 0x0, 0x30000000000},
	/* idx == 245 */ {0x0, 0x0, 0x0, 0x20000000000000, 0x0, 0x0, 0x0, 0xc0000000000},
	/* idx == 246 */ {0x0, 0x0, 0x0, 0x40000000000000, 0x0, 0x0, 0x0, 0x300000000000},
	/* idx == 247 */ {0x0, 0x0, 0x0, 0x80000000000000, 0x0, 0x0, 0x0, 0xc00000000000},
	/* idx == 248 */ {0x0, 0x0, 0x0, 0x100000000000000, 0x0, 0x0, 0x0, 0x3000000000000},
	/* idx == 249 */ {0x0, 0x0, 0x0, 0x200000000000000, 0x0, 0x0, 0x0, 0xc000000000000},
	/* idx == 250 */ {0x0, 0x0, 0x0, 0x400000000000000, 0x0, 0x0, 0x0, 0x30000000000000},
	/* idx == 251 */ {0x0, 0x0, 0x0, 0x800000000000000, 0x0, 0x0, 0x0, 0xc0000000000000},
	/* idx == 252 */ {0x0, 0x0, 0x0, 0x1000000000000000, 0x0, 0x0, 0x0, 0x300000000000000},
	/* idx == 253 */ {0x0, 0x0, 0x0, 0x2000000000000000, 0x0, 0x0, 0x0, 0xc00000000000000},
	/* idx == 254 */ {0x0, 0x0, 0x0, 0x4000000000000000, 0x0, 0x0, 0x0, 0x3000000000000000},
	/* idx == 255 */ {0x0, 0x0, 0x0, 0x8000000000000000, 0x0, 0x0, 0x0, 0xc000000000000000},
	//
	// START of HOST ROUTES, pfxLen == 8
	// [:4] are all 0 and only one bit is set in [4:]
	// calculate it at runtime, save 16KB in lookup table
	//
	// idx == 256 {0, 0, 0, 0, 0x1, 0x0, 0x0, 0x0},
	// idx == 257 {0, 0, 0, 0, 0x2, 0x0, 0x0, 0x0},
	// idx == 258 {0, 0, 0, 0, 0x4, 0x0, 0x0, 0x0},
	// idx == 259 {0, 0, 0, 0, 0x8, 0x0, 0x0, 0x0},
	// idx == 260 {0, 0, 0, 0, 0x10, 0x0, 0x0, 0x0},
	// idx == 261 {0, 0, 0, 0, 0x20, 0x0, 0x0, 0x0},
	// idx == 262 {0, 0, 0, 0, 0x40, 0x0, 0x0, 0x0},
	// idx == 263 {0, 0, 0, 0, 0x80, 0x0, 0x0, 0x0},
	// idx == 264 {0, 0, 0, 0, 0x100, 0x0, 0x0, 0x0},
	// idx == 265 {0, 0, 0, 0, 0x200, 0x0, 0x0, 0x0},
	// idx == 266 {0, 0, 0, 0, 0x400, 0x0, 0x0, 0x0},
	// idx == 267 {0, 0, 0, 0, 0x800, 0x0, 0x0, 0x0},
	// idx == 268 {0, 0, 0, 0, 0x1000, 0x0, 0x0, 0x0},
	// idx == 269 {0, 0, 0, 0, 0x2000, 0x0, 0x0, 0x0},
	// idx == 270 {0, 0, 0, 0, 0x4000, 0x0, 0x0, 0x0},
	// idx == 271 {0, 0, 0, 0, 0x8000, 0x0, 0x0, 0x0},
	// idx == 272 {0, 0, 0, 0, 0x10000, 0x0, 0x0, 0x0},
	// idx == 273 {0, 0, 0, 0, 0x20000, 0x0, 0x0, 0x0},
	// idx == 274 {0, 0, 0, 0, 0x40000, 0x0, 0x0, 0x0},
	// idx == 275 {0, 0, 0, 0, 0x80000, 0x0, 0x0, 0x0},
	// idx == 276 {0, 0, 0, 0, 0x100000, 0x0, 0x0, 0x0},
	// idx == 277 {0, 0, 0, 0, 0x200000, 0x0, 0x0, 0x0},
	// idx == 278 {0, 0, 0, 0, 0x400000, 0x0, 0x0, 0x0},
	// idx == 279 {0, 0, 0, 0, 0x800000, 0x0, 0x0, 0x0},
	// idx == 280 {0, 0, 0, 0, 0x1000000, 0x0, 0x0, 0x0},
	// idx == 281 {0, 0, 0, 0, 0x2000000, 0x0, 0x0, 0x0},
	// idx == 282 {0, 0, 0, 0, 0x4000000, 0x0, 0x0, 0x0},
	// idx == 283 {0, 0, 0, 0, 0x8000000, 0x0, 0x0, 0x0},
	// idx == 284 {0, 0, 0, 0, 0x10000000, 0x0, 0x0, 0x0},
	// idx == 285 {0, 0, 0, 0, 0x20000000, 0x0, 0x0, 0x0},
	// idx == 286 {0, 0, 0, 0, 0x40000000, 0x0, 0x0, 0x0},
	// idx == 287 {0, 0, 0, 0, 0x80000000, 0x0, 0x0, 0x0},
	// idx == 288 {0, 0, 0, 0, 0x100000000, 0x0, 0x0, 0x0},
	// idx == 289 {0, 0, 0, 0, 0x200000000, 0x0, 0x0, 0x0},
	// idx == 290 {0, 0, 0, 0, 0x400000000, 0x0, 0x0, 0x0},
	// idx == 291 {0, 0, 0, 0, 0x800000000, 0x0, 0x0, 0x0},
	// idx == 292 {0, 0, 0, 0, 0x1000000000, 0x0, 0x0, 0x0},
	// idx == 293 {0, 0, 0, 0, 0x2000000000, 0x0, 0x0, 0x0},
	// idx == 294 {0, 0, 0, 0, 0x4000000000, 0x0, 0x0, 0x0},
	// idx == 295 {0, 0, 0, 0, 0x8000000000, 0x0, 0x0, 0x0},
	// idx == 296 {0, 0, 0, 0, 0x10000000000, 0x0, 0x0, 0x0},
	// idx == 297 {0, 0, 0, 0, 0x20000000000, 0x0, 0x0, 0x0},
	// idx == 298 {0, 0, 0, 0, 0x40000000000, 0x0, 0x0, 0x0},
	// idx == 299 {0, 0, 0, 0, 0x80000000000, 0x0, 0x0, 0x0},
	// idx == 300 {0, 0, 0, 0, 0x100000000000, 0x0, 0x0, 0x0},
	// idx == 301 {0, 0, 0, 0, 0x200000000000, 0x0, 0x0, 0x0},
	// idx == 302 {0, 0, 0, 0, 0x400000000000, 0x0, 0x0, 0x0},
	// idx == 303 {0, 0, 0, 0, 0x800000000000, 0x0, 0x0, 0x0},
	// idx == 304 {0, 0, 0, 0, 0x1000000000000, 0x0, 0x0, 0x0},
	// idx == 305 {0, 0, 0, 0, 0x2000000000000, 0x0, 0x0, 0x0},
	// idx == 306 {0, 0, 0, 0, 0x4000000000000, 0x0, 0x0, 0x0},
	// idx == 307 {0, 0, 0, 0, 0x8000000000000, 0x0, 0x0, 0x0},
	// idx == 308 {0, 0, 0, 0, 0x10000000000000, 0x0, 0x0, 0x0},
	// idx == 309 {0, 0, 0, 0, 0x20000000000000, 0x0, 0x0, 0x0},
	// idx == 310 {0, 0, 0, 0, 0x40000000000000, 0x0, 0x0, 0x0},
	// idx == 311 {0, 0, 0, 0, 0x80000000000000, 0x0, 0x0, 0x0},
	// idx == 312 {0, 0, 0, 0, 0x100000000000000, 0x0, 0x0, 0x0},
	// idx == 313 {0, 0, 0, 0, 0x200000000000000, 0x0, 0x0, 0x0},
	// idx == 314 {0, 0, 0, 0, 0x400000000000000, 0x0, 0x0, 0x0},
	// idx == 315 {0, 0, 0, 0, 0x800000000000000, 0x0, 0x0, 0x0},
	// idx == 316 {0, 0, 0, 0, 0x1000000000000000, 0x0, 0x0, 0x0},
	// idx == 317 {0, 0, 0, 0, 0x2000000000000000, 0x0, 0x0, 0x0},
	// idx == 318 {0, 0, 0, 0, 0x4000000000000000, 0x0, 0x0, 0x0},
	// idx == 319 {0, 0, 0, 0, 0x8000000000000000, 0x0, 0x0, 0x0},
	// idx == 320 {0, 0, 0, 0, 0x0, 0x1, 0x0, 0x0},
	// idx == 321 {0, 0, 0, 0, 0x0, 0x2, 0x0, 0x0},
	// idx == 322 {0, 0, 0, 0, 0x0, 0x4, 0x0, 0x0},
	// idx == 323 {0, 0, 0, 0, 0x0, 0x8, 0x0, 0x0},
	// idx == 324 {0, 0, 0, 0, 0x0, 0x10, 0x0, 0x0},
	// idx == 325 {0, 0, 0, 0, 0x0, 0x20, 0x0, 0x0},
	// idx == 326 {0, 0, 0, 0, 0x0, 0x40, 0x0, 0x0},
	// idx == 327 {0, 0, 0, 0, 0x0, 0x80, 0x0, 0x0},
	// idx == 328 {0, 0, 0, 0, 0x0, 0x100, 0x0, 0x0},
	// idx == 329 {0, 0, 0, 0, 0x0, 0x200, 0x0, 0x0},
	// idx == 330 {0, 0, 0, 0, 0x0, 0x400, 0x0, 0x0},
	// idx == 331 {0, 0, 0, 0, 0x0, 0x800, 0x0, 0x0},
	// idx == 332 {0, 0, 0, 0, 0x0, 0x1000, 0x0, 0x0},
	// idx == 333 {0, 0, 0, 0, 0x0, 0x2000, 0x0, 0x0},
	// idx == 334 {0, 0, 0, 0, 0x0, 0x4000, 0x0, 0x0},
	// idx == 335 {0, 0, 0, 0, 0x0, 0x8000, 0x0, 0x0},
	// idx == 336 {0, 0, 0, 0, 0x0, 0x10000, 0x0, 0x0},
	// idx == 337 {0, 0, 0, 0, 0x0, 0x20000, 0x0, 0x0},
	// idx == 338 {0, 0, 0, 0, 0x0, 0x40000, 0x0, 0x0},
	// idx == 339 {0, 0, 0, 0, 0x0, 0x80000, 0x0, 0x0},
	// idx == 340 {0, 0, 0, 0, 0x0, 0x100000, 0x0, 0x0},
	// idx == 341 {0, 0, 0, 0, 0x0, 0x200000, 0x0, 0x0},
	// idx == 342 {0, 0, 0, 0, 0x0, 0x400000, 0x0, 0x0},
	// idx == 343 {0, 0, 0, 0, 0x0, 0x800000, 0x0, 0x0},
	// idx == 344 {0, 0, 0, 0, 0x0, 0x1000000, 0x0, 0x0},
	// idx == 345 {0, 0, 0, 0, 0x0, 0x2000000, 0x0, 0x0},
	// idx == 346 {0, 0, 0, 0, 0x0, 0x4000000, 0x0, 0x0},
	// idx == 347 {0, 0, 0, 0, 0x0, 0x8000000, 0x0, 0x0},
	// idx == 348 {0, 0, 0, 0, 0x0, 0x10000000, 0x0, 0x0},
	// idx == 349 {0, 0, 0, 0, 0x0, 0x20000000, 0x0, 0x0},
	// idx == 350 {0, 0, 0, 0, 0x0, 0x40000000, 0x0, 0x0},
	// idx == 351 {0, 0, 0, 0, 0x0, 0x80000000, 0x0, 0x0},
	// idx == 352 {0, 0, 0, 0, 0x0, 0x100000000, 0x0, 0x0},
	// idx == 353 {0, 0, 0, 0, 0x0, 0x200000000, 0x0, 0x0},
	// idx == 354 {0, 0, 0, 0, 0x0, 0x400000000, 0x0, 0x0},
	// idx == 355 {0, 0, 0, 0, 0x0, 0x800000000, 0x0, 0x0},
	// idx == 356 {0, 0, 0, 0, 0x0, 0x1000000000, 0x0, 0x0},
	// idx == 357 {0, 0, 0, 0, 0x0, 0x2000000000, 0x0, 0x0},
	// idx == 358 {0, 0, 0, 0, 0x0, 0x4000000000, 0x0, 0x0},
	// idx == 359 {0, 0, 0, 0, 0x0, 0x8000000000, 0x0, 0x0},
	// idx == 360 {0, 0, 0, 0, 0x0, 0x10000000000, 0x0, 0x0},
	// idx == 361 {0, 0, 0, 0, 0x0, 0x20000000000, 0x0, 0x0},
	// idx == 362 {0, 0, 0, 0, 0x0, 0x40000000000, 0x0, 0x0},
	// idx == 363 {0, 0, 0, 0, 0x0, 0x80000000000, 0x0, 0x0},
	// idx == 364 {0, 0, 0, 0, 0x0, 0x100000000000, 0x0, 0x0},
	// idx == 365 {0, 0, 0, 0, 0x0, 0x200000000000, 0x0, 0x0},
	// idx == 366 {0, 0, 0, 0, 0x0, 0x400000000000, 0x0, 0x0},
	// idx == 367 {0, 0, 0, 0, 0x0, 0x800000000000, 0x0, 0x0},
	// idx == 368 {0, 0, 0, 0, 0x0, 0x1000000000000, 0x0, 0x0},
	// idx == 369 {0, 0, 0, 0, 0x0, 0x2000000000000, 0x0, 0x0},
	// idx == 370 {0, 0, 0, 0, 0x0, 0x4000000000000, 0x0, 0x0},
	// idx == 371 {0, 0, 0, 0, 0x0, 0x8000000000000, 0x0, 0x0},
	// idx == 372 {0, 0, 0, 0, 0x0, 0x10000000000000, 0x0, 0x0},
	// idx == 373 {0, 0, 0, 0, 0x0, 0x20000000000000, 0x0, 0x0},
	// idx == 374 {0, 0, 0, 0, 0x0, 0x40000000000000, 0x0, 0x0},
	// idx == 375 {0, 0, 0, 0, 0x0, 0x80000000000000, 0x0, 0x0},
	// idx == 376 {0, 0, 0, 0, 0x0, 0x100000000000000, 0x0, 0x0},
	// idx == 377 {0, 0, 0, 0, 0x0, 0x200000000000000, 0x0, 0x0},
	// idx == 378 {0, 0, 0, 0, 0x0, 0x400000000000000, 0x0, 0x0},
	// idx == 379 {0, 0, 0, 0, 0x0, 0x800000000000000, 0x0, 0x0},
	// idx == 380 {0, 0, 0, 0, 0x0, 0x1000000000000000, 0x0, 0x0},
	// idx == 381 {0, 0, 0, 0, 0x0, 0x2000000000000000, 0x0, 0x0},
	// idx == 382 {0, 0, 0, 0, 0x0, 0x4000000000000000, 0x0, 0x0},
	// idx == 383 {0, 0, 0, 0, 0x0, 0x8000000000000000, 0x0, 0x0},
	// idx == 384 {0, 0, 0, 0, 0x0, 0x0, 0x1, 0x0},
	// idx == 385 {0, 0, 0, 0, 0x0, 0x0, 0x2, 0x0},
	// idx == 386 {0, 0, 0, 0, 0x0, 0x0, 0x4, 0x0},
	// idx == 387 {0, 0, 0, 0, 0x0, 0x0, 0x8, 0x0},
	// idx == 388 {0, 0, 0, 0, 0x0, 0x0, 0x10, 0x0},
	// idx == 389 {0, 0, 0, 0, 0x0, 0x0, 0x20, 0x0},
	// idx == 390 {0, 0, 0, 0, 0x0, 0x0, 0x40, 0x0},
	// idx == 391 {0, 0, 0, 0, 0x0, 0x0, 0x80, 0x0},
	// idx == 392 {0, 0, 0, 0, 0x0, 0x0, 0x100, 0x0},
	// idx == 393 {0, 0, 0, 0, 0x0, 0x0, 0x200, 0x0},
	// idx == 394 {0, 0, 0, 0, 0x0, 0x0, 0x400, 0x0},
	// idx == 395 {0, 0, 0, 0, 0x0, 0x0, 0x800, 0x0},
	// idx == 396 {0, 0, 0, 0, 0x0, 0x0, 0x1000, 0x0},
	// idx == 397 {0, 0, 0, 0, 0x0, 0x0, 0x2000, 0x0},
	// idx == 398 {0, 0, 0, 0, 0x0, 0x0, 0x4000, 0x0},
	// idx == 399 {0, 0, 0, 0, 0x0, 0x0, 0x8000, 0x0},
	// idx == 400 {0, 0, 0, 0, 0x0, 0x0, 0x10000, 0x0},
	// idx == 401 {0, 0, 0, 0, 0x0, 0x0, 0x20000, 0x0},
	// idx == 402 {0, 0, 0, 0, 0x0, 0x0, 0x40000, 0x0},
	// idx == 403 {0, 0, 0, 0, 0x0, 0x0, 0x80000, 0x0},
	// idx == 404 {0, 0, 0, 0, 0x0, 0x0, 0x100000, 0x0},
	// idx == 405 {0, 0, 0, 0, 0x0, 0x0, 0x200000, 0x0},
	// idx == 406 {0, 0, 0, 0, 0x0, 0x0, 0x400000, 0x0},
	// idx == 407 {0, 0, 0, 0, 0x0, 0x0, 0x800000, 0x0},
	// idx == 408 {0, 0, 0, 0, 0x0, 0x0, 0x1000000, 0x0},
	// idx == 409 {0, 0, 0, 0, 0x0, 0x0, 0x2000000, 0x0},
	// idx == 410 {0, 0, 0, 0, 0x0, 0x0, 0x4000000, 0x0},
	// idx == 411 {0, 0, 0, 0, 0x0, 0x0, 0x8000000, 0x0},
	// idx == 412 {0, 0, 0, 0, 0x0, 0x0, 0x10000000, 0x0},
	// idx == 413 {0, 0, 0, 0, 0x0, 0x0, 0x20000000, 0x0},
	// idx == 414 {0, 0, 0, 0, 0x0, 0x0, 0x40000000, 0x0},
	// idx == 415 {0, 0, 0, 0, 0x0, 0x0, 0x80000000, 0x0},
	// idx == 416 {0, 0, 0, 0, 0x0, 0x0, 0x100000000, 0x0},
	// idx == 417 {0, 0, 0, 0, 0x0, 0x0, 0x200000000, 0x0},
	// idx == 418 {0, 0, 0, 0, 0x0, 0x0, 0x400000000, 0x0},
	// idx == 419 {0, 0, 0, 0, 0x0, 0x0, 0x800000000, 0x0},
	// idx == 420 {0, 0, 0, 0, 0x0, 0x0, 0x1000000000, 0x0},
	// idx == 421 {0, 0, 0, 0, 0x0, 0x0, 0x2000000000, 0x0},
	// idx == 422 {0, 0, 0, 0, 0x0, 0x0, 0x4000000000, 0x0},
	// idx == 423 {0, 0, 0, 0, 0x0, 0x0, 0x8000000000, 0x0},
	// idx == 424 {0, 0, 0, 0, 0x0, 0x0, 0x10000000000, 0x0},
	// idx == 425 {0, 0, 0, 0, 0x0, 0x0, 0x20000000000, 0x0},
	// idx == 426 {0, 0, 0, 0, 0x0, 0x0, 0x40000000000, 0x0},
	// idx == 427 {0, 0, 0, 0, 0x0, 0x0, 0x80000000000, 0x0},
	// idx == 428 {0, 0, 0, 0, 0x0, 0x0, 0x100000000000, 0x0},
	// idx == 429 {0, 0, 0, 0, 0x0, 0x0, 0x200000000000, 0x0},
	// idx == 430 {0, 0, 0, 0, 0x0, 0x0, 0x400000000000, 0x0},
	// idx == 431 {0, 0, 0, 0, 0x0, 0x0, 0x800000000000, 0x0},
	// idx == 432 {0, 0, 0, 0, 0x0, 0x0, 0x1000000000000, 0x0},
	// idx == 433 {0, 0, 0, 0, 0x0, 0x0, 0x2000000000000, 0x0},
	// idx == 434 {0, 0, 0, 0, 0x0, 0x0, 0x4000000000000, 0x0},
	// idx == 435 {0, 0, 0, 0, 0x0, 0x0, 0x8000000000000, 0x0},
	// idx == 436 {0, 0, 0, 0, 0x0, 0x0, 0x10000000000000, 0x0},
	// idx == 437 {0, 0, 0, 0, 0x0, 0x0, 0x20000000000000, 0x0},
	// idx == 438 {0, 0, 0, 0, 0x0, 0x0, 0x40000000000000, 0x0},
	// idx == 439 {0, 0, 0, 0, 0x0, 0x0, 0x80000000000000, 0x0},
	// idx == 440 {0, 0, 0, 0, 0x0, 0x0, 0x100000000000000, 0x0},
	// idx == 441 {0, 0, 0, 0, 0x0, 0x0, 0x200000000000000, 0x0},
	// idx == 442 {0, 0, 0, 0, 0x0, 0x0, 0x400000000000000, 0x0},
	// idx == 443 {0, 0, 0, 0, 0x0, 0x0, 0x800000000000000, 0x0},
	// idx == 444 {0, 0, 0, 0, 0x0, 0x0, 0x1000000000000000, 0x0},
	// idx == 445 {0, 0, 0, 0, 0x0, 0x0, 0x2000000000000000, 0x0},
	// idx == 446 {0, 0, 0, 0, 0x0, 0x0, 0x4000000000000000, 0x0},
	// idx == 447 {0, 0, 0, 0, 0x0, 0x0, 0x8000000000000000, 0x0},
	// idx == 448 {0, 0, 0, 0, 0x0, 0x0, 0x0, 0x1},
	// idx == 449 {0, 0, 0, 0, 0x0, 0x0, 0x0, 0x2},
	// idx == 450 {0, 0, 0, 0, 0x0, 0x0, 0x0, 0x4},
	// idx == 451 {0, 0, 0, 0, 0x0, 0x0, 0x0, 0x8},
	// idx == 452 {0, 0, 0, 0, 0x0, 0x0, 0x0, 0x10},
	// idx == 453 {0, 0, 0, 0, 0x0, 0x0, 0x0, 0x20},
	// idx == 454 {0, 0, 0, 0, 0x0, 0x0, 0x0, 0x40},
	// idx == 455 {0, 0, 0, 0, 0x0, 0x0, 0x0, 0x80},
	// idx == 456 {0, 0, 0, 0, 0x0, 0x0, 0x0, 0x100},
	// idx == 457 {0, 0, 0, 0, 0x0, 0x0, 0x0, 0x200},
	// idx == 458 {0, 0, 0, 0, 0x0, 0x0, 0x0, 0x400},
	// idx == 459 {0, 0, 0, 0, 0x0, 0x0, 0x0, 0x800},
	// idx == 460 {0, 0, 0, 0, 0x0, 0x0, 0x0, 0x1000},
	// idx == 461 {0, 0, 0, 0, 0x0, 0x0, 0x0, 0x2000},
	// idx == 462 {0, 0, 0, 0, 0x0, 0x0, 0x0, 0x4000},
	// idx == 463 {0, 0, 0, 0, 0x0, 0x0, 0x0, 0x8000},
	// idx == 464 {0, 0, 0, 0, 0x0, 0x0, 0x0, 0x10000},
	// idx == 465 {0, 0, 0, 0, 0x0, 0x0, 0x0, 0x20000},
	// idx == 466 {0, 0, 0, 0, 0x0, 0x0, 0x0, 0x40000},
	// idx == 467 {0, 0, 0, 0, 0x0, 0x0, 0x0, 0x80000},
	// idx == 468 {0, 0, 0, 0, 0x0, 0x0, 0x0, 0x100000},
	// idx == 469 {0, 0, 0, 0, 0x0, 0x0, 0x0, 0x200000},
	// idx == 470 {0, 0, 0, 0, 0x0, 0x0, 0x0, 0x400000},
	// idx == 471 {0, 0, 0, 0, 0x0, 0x0, 0x0, 0x800000},
	// idx == 472 {0, 0, 0, 0, 0x0, 0x0, 0x0, 0x1000000},
	// idx == 473 {0, 0, 0, 0, 0x0, 0x0, 0x0, 0x2000000},
	// idx == 474 {0, 0, 0, 0, 0x0, 0x0, 0x0, 0x4000000},
	// idx == 475 {0, 0, 0, 0, 0x0, 0x0, 0x0, 0x8000000},
	// idx == 476 {0, 0, 0, 0, 0x0, 0x0, 0x0, 0x10000000},
	// idx == 477 {0, 0, 0, 0, 0x0, 0x0, 0x0, 0x20000000},
	// idx == 478 {0, 0, 0, 0, 0x0, 0x0, 0x0, 0x40000000},
	// idx == 479 {0, 0, 0, 0, 0x0, 0x0, 0x0, 0x80000000},
	// idx == 480 {0, 0, 0, 0, 0x0, 0x0, 0x0, 0x100000000},
	// idx == 481 {0, 0, 0, 0, 0x0, 0x0, 0x0, 0x200000000},
	// idx == 482 {0, 0, 0, 0, 0x0, 0x0, 0x0, 0x400000000},
	// idx == 483 {0, 0, 0, 0, 0x0, 0x0, 0x0, 0x800000000},
	// idx == 484 {0, 0, 0, 0, 0x0, 0x0, 0x0, 0x1000000000},
	// idx == 485 {0, 0, 0, 0, 0x0, 0x0, 0x0, 0x2000000000},
	// idx == 486 {0, 0, 0, 0, 0x0, 0x0, 0x0, 0x4000000000},
	// idx == 487 {0, 0, 0, 0, 0x0, 0x0, 0x0, 0x8000000000},
	// idx == 488 {0, 0, 0, 0, 0x0, 0x0, 0x0, 0x10000000000},
	// idx == 489 {0, 0, 0, 0, 0x0, 0x0, 0x0, 0x20000000000},
	// idx == 490 {0, 0, 0, 0, 0x0, 0x0, 0x0, 0x40000000000},
	// idx == 491 {0, 0, 0, 0, 0x0, 0x0, 0x0, 0x80000000000},
	// idx == 492 {0, 0, 0, 0, 0x0, 0x0, 0x0, 0x100000000000},
	// idx == 493 {0, 0, 0, 0, 0x0, 0x0, 0x0, 0x200000000000},
	// idx == 494 {0, 0, 0, 0, 0x0, 0x0, 0x0, 0x400000000000},
	// idx == 495 {0, 0, 0, 0, 0x0, 0x0, 0x0, 0x800000000000},
	// idx == 496 {0, 0, 0, 0, 0x0, 0x0, 0x0, 0x1000000000000},
	// idx == 497 {0, 0, 0, 0, 0x0, 0x0, 0x0, 0x2000000000000},
	// idx == 498 {0, 0, 0, 0, 0x0, 0x0, 0x0, 0x4000000000000},
	// idx == 499 {0, 0, 0, 0, 0x0, 0x0, 0x0, 0x8000000000000},
	// idx == 500 {0, 0, 0, 0, 0x0, 0x0, 0x0, 0x10000000000000},
	// idx == 501 {0, 0, 0, 0, 0x0, 0x0, 0x0, 0x20000000000000},
	// idx == 502 {0, 0, 0, 0, 0x0, 0x0, 0x0, 0x40000000000000},
	// idx == 503 {0, 0, 0, 0, 0x0, 0x0, 0x0, 0x80000000000000},
	// idx == 504 {0, 0, 0, 0, 0x0, 0x0, 0x0, 0x100000000000000},
	// idx == 505 {0, 0, 0, 0, 0x0, 0x0, 0x0, 0x200000000000000},
	// idx == 506 {0, 0, 0, 0, 0x0, 0x0, 0x0, 0x400000000000000},
	// idx == 507 {0, 0, 0, 0, 0x0, 0x0, 0x0, 0x800000000000000},
	// idx == 508 {0, 0, 0, 0, 0x0, 0x0, 0x0, 0x1000000000000000},
	// idx == 509 {0, 0, 0, 0, 0x0, 0x0, 0x0, 0x2000000000000000},
	// idx == 510 {0, 0, 0, 0, 0x0, 0x0, 0x0, 0x4000000000000000},
	// idx == 511 {0, 0, 0, 0, 0x0, 0x0, 0x0, 0x8000000000000000},
}
