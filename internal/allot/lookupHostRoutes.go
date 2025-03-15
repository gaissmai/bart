// Copyright (c) 2025 Karl Gaissmaier
// SPDX-License-Identifier: MIT

package allot

import "github.com/gaissmai/bart/internal/bitset"

// HostRoutesTbl, as precalculated bitsets,
// map the baseIndex to bitset with precomputed complete binary tree.
//
//	  // 1 <= idx <= 255
//		func allotRec(aTbl *bitset.BitSet, idx uint) {
//			aTbl = aTbl.Set(idx)
//			if idx >= 255 {
//				return
//			}
//			allotRec(aTbl, idx<<1)
//			allotRec(aTbl, idx<<1+1)
//		}
//
// Only used for fast bitset intersections instead of
// range loops in table overlaps methods.
//
// Please read the ART paper ./doc/artlookup.pdf to understand the allotment algorithm.

// HostRoutesTbl, the second 256 Bits, see also the PrefixRoutesTbl for the first 256 Bits
var HostRoutesTbl = [256]*bitset.BitSet256{
	/* idx:   0 */ {0x0, 0x0, 0x0, 0x0}, // invalid
	/* idx:   1 */ {0xffffffffffffffff, 0xffffffffffffffff, 0xffffffffffffffff, 0xffffffffffffffff}, // [0 1 2 3 4 5 6 7 8 9 10 11 12 ...
	/* idx:   2 */ {0xffffffffffffffff, 0xffffffffffffffff, 0x0, 0x0}, // [0 1 2 3 4 5 6 7 8 9 10 11 12 13 14 15 16 17 18 19 20 21 22 ...
	/* idx:   3 */ {0x0, 0x0, 0xffffffffffffffff, 0xffffffffffffffff}, // [128 129 130 131 132 133 134 135 136 137 138 139 140 141 142 ...
	/* idx:   4 */ {0xffffffffffffffff, 0x0, 0x0, 0x0}, // [0 1 2 3 4 5 6 7 8 9 10 11 12 13 14 15 16 17 18 19 20 21 22 23 24 25 26 27 ...
	/* idx:   5 */ {0x0, 0xffffffffffffffff, 0x0, 0x0}, // [64 65 66 67 68 69 70 71 72 73 74 75 76 77 78 79 80 81 82 83 84 85 86 87 88 ...
	/* idx:   6 */ {0x0, 0x0, 0xffffffffffffffff, 0x0}, // [128 129 130 131 132 133 134 135 136 137 138 139 140 141 142 143 144 145 146 ...
	/* idx:   7 */ {0x0, 0x0, 0x0, 0xffffffffffffffff}, // [192 193 194 195 196 197 198 199 200 201 202 203 204 205 206 207 208 209 210 ...
	/* idx:   8 */ {0xffffffff, 0x0, 0x0, 0x0}, // [0 1 2 3 4 5 6 7 8 9 10 11 12 13 14 15 16 17 18 19 20 21 22 23 24 25 26 27 28 29 30 31]
	/* idx:   9 */ {0xffffffff00000000, 0x0, 0x0, 0x0}, // [32 33 34 35 36 37 38 39 40 41 42 43 44 45 46 47 48 49 50 51 52 53 54 55 56 57 ...
	/* idx:  10 */ {0x0, 0xffffffff, 0x0, 0x0}, // [64 65 66 67 68 69 70 71 72 73 74 75 76 77 78 79 80 81 82 83 84 85 86 87 88 89 90 91 ...
	/* idx:  11 */ {0x0, 0xffffffff00000000, 0x0, 0x0}, // [96 97 98 99 100 101 102 103 104 105 106 107 108 109 110 111 112 113 114 115 ...
	/* idx:  12 */ {0x0, 0x0, 0xffffffff, 0x0}, // [128 129 130 131 132 133 134 135 136 137 138 139 140 141 142 143 144 145 146 147 148 ...
	/* idx:  13 */ {0x0, 0x0, 0xffffffff00000000, 0x0}, // [160 161 162 163 164 165 166 167 168 169 170 171 172 173 174 175 176 177 178 ...
	/* idx:  14 */ {0x0, 0x0, 0x0, 0xffffffff}, // [192 193 194 195 196 197 198 199 200 201 202 203 204 205 206 207 208 209 210 211 212 ...
	/* idx:  15 */ {0x0, 0x0, 0x0, 0xffffffff00000000}, // [224 225 226 227 228 229 230 231 232 233 234 235 236 237 238 239 240 241 242 ...
	/* idx:  16 */ {0xffff, 0x0, 0x0, 0x0}, // [0 1 2 3 4 5 6 7 8 9 10 11 12 13 14 15]
	/* idx:  17 */ {0xffff0000, 0x0, 0x0, 0x0}, // [16 17 18 19 20 21 22 23 24 25 26 27 28 29 30 31]
	/* idx:  18 */ {0xffff00000000, 0x0, 0x0, 0x0}, // [32 33 34 35 36 37 38 39 40 41 42 43 44 45 46 47]
	/* idx:  19 */ {0xffff000000000000, 0x0, 0x0, 0x0}, // [48 49 50 51 52 53 54 55 56 57 58 59 60 61 62 63]
	/* idx:  20 */ {0x0, 0xffff, 0x0, 0x0}, // [64 65 66 67 68 69 70 71 72 73 74 75 76 77 78 79]
	/* idx:  21 */ {0x0, 0xffff0000, 0x0, 0x0}, // [80 81 82 83 84 85 86 87 88 89 90 91 92 93 94 95]
	/* idx:  22 */ {0x0, 0xffff00000000, 0x0, 0x0}, // [96 97 98 99 100 101 102 103 104 105 106 107 108 109 110 111]
	/* idx:  23 */ {0x0, 0xffff000000000000, 0x0, 0x0}, // [112 113 114 115 116 117 118 119 120 121 122 123 124 125 126 127]
	/* idx:  24 */ {0x0, 0x0, 0xffff, 0x0}, // [128 129 130 131 132 133 134 135 136 137 138 139 140 141 142 143]
	/* idx:  25 */ {0x0, 0x0, 0xffff0000, 0x0}, // [144 145 146 147 148 149 150 151 152 153 154 155 156 157 158 159]
	/* idx:  26 */ {0x0, 0x0, 0xffff00000000, 0x0}, // [160 161 162 163 164 165 166 167 168 169 170 171 172 173 174 175]
	/* idx:  27 */ {0x0, 0x0, 0xffff000000000000, 0x0}, // [176 177 178 179 180 181 182 183 184 185 186 187 188 189 190 191]
	/* idx:  28 */ {0x0, 0x0, 0x0, 0xffff}, // [192 193 194 195 196 197 198 199 200 201 202 203 204 205 206 207]
	/* idx:  29 */ {0x0, 0x0, 0x0, 0xffff0000}, // [208 209 210 211 212 213 214 215 216 217 218 219 220 221 222 223]
	/* idx:  30 */ {0x0, 0x0, 0x0, 0xffff00000000}, // [224 225 226 227 228 229 230 231 232 233 234 235 236 237 238 239]
	/* idx:  31 */ {0x0, 0x0, 0x0, 0xffff000000000000}, // [240 241 242 243 244 245 246 247 248 249 250 251 252 253 254 255]
	/* idx:  32 */ {0xff, 0x0, 0x0, 0x0}, // [0 1 2 3 4 5 6 7]
	/* idx:  33 */ {0xff00, 0x0, 0x0, 0x0}, // [8 9 10 11 12 13 14 15]
	/* idx:  34 */ {0xff0000, 0x0, 0x0, 0x0}, // [16 17 18 19 20 21 22 23]
	/* idx:  35 */ {0xff000000, 0x0, 0x0, 0x0}, // [24 25 26 27 28 29 30 31]
	/* idx:  36 */ {0xff00000000, 0x0, 0x0, 0x0}, // [32 33 34 35 36 37 38 39]
	/* idx:  37 */ {0xff0000000000, 0x0, 0x0, 0x0}, // [40 41 42 43 44 45 46 47]
	/* idx:  38 */ {0xff000000000000, 0x0, 0x0, 0x0}, // [48 49 50 51 52 53 54 55]
	/* idx:  39 */ {0xff00000000000000, 0x0, 0x0, 0x0}, // [56 57 58 59 60 61 62 63]
	/* idx:  40 */ {0x0, 0xff, 0x0, 0x0}, // [64 65 66 67 68 69 70 71]
	/* idx:  41 */ {0x0, 0xff00, 0x0, 0x0}, // [72 73 74 75 76 77 78 79]
	/* idx:  42 */ {0x0, 0xff0000, 0x0, 0x0}, // [80 81 82 83 84 85 86 87]
	/* idx:  43 */ {0x0, 0xff000000, 0x0, 0x0}, // [88 89 90 91 92 93 94 95]
	/* idx:  44 */ {0x0, 0xff00000000, 0x0, 0x0}, // [96 97 98 99 100 101 102 103]
	/* idx:  45 */ {0x0, 0xff0000000000, 0x0, 0x0}, // [104 105 106 107 108 109 110 111]
	/* idx:  46 */ {0x0, 0xff000000000000, 0x0, 0x0}, // [112 113 114 115 116 117 118 119]
	/* idx:  47 */ {0x0, 0xff00000000000000, 0x0, 0x0}, // [120 121 122 123 124 125 126 127]
	/* idx:  48 */ {0x0, 0x0, 0xff, 0x0}, // [128 129 130 131 132 133 134 135]
	/* idx:  49 */ {0x0, 0x0, 0xff00, 0x0}, // [136 137 138 139 140 141 142 143]
	/* idx:  50 */ {0x0, 0x0, 0xff0000, 0x0}, // [144 145 146 147 148 149 150 151]
	/* idx:  51 */ {0x0, 0x0, 0xff000000, 0x0}, // [152 153 154 155 156 157 158 159]
	/* idx:  52 */ {0x0, 0x0, 0xff00000000, 0x0}, // [160 161 162 163 164 165 166 167]
	/* idx:  53 */ {0x0, 0x0, 0xff0000000000, 0x0}, // [168 169 170 171 172 173 174 175]
	/* idx:  54 */ {0x0, 0x0, 0xff000000000000, 0x0}, // [176 177 178 179 180 181 182 183]
	/* idx:  55 */ {0x0, 0x0, 0xff00000000000000, 0x0}, // [184 185 186 187 188 189 190 191]
	/* idx:  56 */ {0x0, 0x0, 0x0, 0xff}, // [192 193 194 195 196 197 198 199]
	/* idx:  57 */ {0x0, 0x0, 0x0, 0xff00}, // [200 201 202 203 204 205 206 207]
	/* idx:  58 */ {0x0, 0x0, 0x0, 0xff0000}, // [208 209 210 211 212 213 214 215]
	/* idx:  59 */ {0x0, 0x0, 0x0, 0xff000000}, // [216 217 218 219 220 221 222 223]
	/* idx:  60 */ {0x0, 0x0, 0x0, 0xff00000000}, // [224 225 226 227 228 229 230 231]
	/* idx:  61 */ {0x0, 0x0, 0x0, 0xff0000000000}, // [232 233 234 235 236 237 238 239]
	/* idx:  62 */ {0x0, 0x0, 0x0, 0xff000000000000}, // [240 241 242 243 244 245 246 247]
	/* idx:  63 */ {0x0, 0x0, 0x0, 0xff00000000000000}, // [248 249 250 251 252 253 254 255]
	/* idx:  64 */ {0xf, 0x0, 0x0, 0x0}, // [0 1 2 3]
	/* idx:  65 */ {0xf0, 0x0, 0x0, 0x0}, // [4 5 6 7]
	/* idx:  66 */ {0xf00, 0x0, 0x0, 0x0}, // [8 9 10 11]
	/* idx:  67 */ {0xf000, 0x0, 0x0, 0x0}, // [12 13 14 15]
	/* idx:  68 */ {0xf0000, 0x0, 0x0, 0x0}, // [16 17 18 19]
	/* idx:  69 */ {0xf00000, 0x0, 0x0, 0x0}, // [20 21 22 23]
	/* idx:  70 */ {0xf000000, 0x0, 0x0, 0x0}, // [24 25 26 27]
	/* idx:  71 */ {0xf0000000, 0x0, 0x0, 0x0}, // [28 29 30 31]
	/* idx:  72 */ {0xf00000000, 0x0, 0x0, 0x0}, // [32 33 34 35]
	/* idx:  73 */ {0xf000000000, 0x0, 0x0, 0x0}, // [36 37 38 39]
	/* idx:  74 */ {0xf0000000000, 0x0, 0x0, 0x0}, // [40 41 42 43]
	/* idx:  75 */ {0xf00000000000, 0x0, 0x0, 0x0}, // [44 45 46 47]
	/* idx:  76 */ {0xf000000000000, 0x0, 0x0, 0x0}, // [48 49 50 51]
	/* idx:  77 */ {0xf0000000000000, 0x0, 0x0, 0x0}, // [52 53 54 55]
	/* idx:  78 */ {0xf00000000000000, 0x0, 0x0, 0x0}, // [56 57 58 59]
	/* idx:  79 */ {0xf000000000000000, 0x0, 0x0, 0x0}, // [60 61 62 63]
	/* idx:  80 */ {0x0, 0xf, 0x0, 0x0}, // [64 65 66 67]
	/* idx:  81 */ {0x0, 0xf0, 0x0, 0x0}, // [68 69 70 71]
	/* idx:  82 */ {0x0, 0xf00, 0x0, 0x0}, // [72 73 74 75]
	/* idx:  83 */ {0x0, 0xf000, 0x0, 0x0}, // [76 77 78 79]
	/* idx:  84 */ {0x0, 0xf0000, 0x0, 0x0}, // [80 81 82 83]
	/* idx:  85 */ {0x0, 0xf00000, 0x0, 0x0}, // [84 85 86 87]
	/* idx:  86 */ {0x0, 0xf000000, 0x0, 0x0}, // [88 89 90 91]
	/* idx:  87 */ {0x0, 0xf0000000, 0x0, 0x0}, // [92 93 94 95]
	/* idx:  88 */ {0x0, 0xf00000000, 0x0, 0x0}, // [96 97 98 99]
	/* idx:  89 */ {0x0, 0xf000000000, 0x0, 0x0}, // [100 101 102 103]
	/* idx:  90 */ {0x0, 0xf0000000000, 0x0, 0x0}, // [104 105 106 107]
	/* idx:  91 */ {0x0, 0xf00000000000, 0x0, 0x0}, // [108 109 110 111]
	/* idx:  92 */ {0x0, 0xf000000000000, 0x0, 0x0}, // [112 113 114 115]
	/* idx:  93 */ {0x0, 0xf0000000000000, 0x0, 0x0}, // [116 117 118 119]
	/* idx:  94 */ {0x0, 0xf00000000000000, 0x0, 0x0}, // [120 121 122 123]
	/* idx:  95 */ {0x0, 0xf000000000000000, 0x0, 0x0}, // [124 125 126 127]
	/* idx:  96 */ {0x0, 0x0, 0xf, 0x0}, // [128 129 130 131]
	/* idx:  97 */ {0x0, 0x0, 0xf0, 0x0}, // [132 133 134 135]
	/* idx:  98 */ {0x0, 0x0, 0xf00, 0x0}, // [136 137 138 139]
	/* idx:  99 */ {0x0, 0x0, 0xf000, 0x0}, // [140 141 142 143]
	/* idx: 100 */ {0x0, 0x0, 0xf0000, 0x0}, // [144 145 146 147]
	/* idx: 101 */ {0x0, 0x0, 0xf00000, 0x0}, // [148 149 150 151]
	/* idx: 102 */ {0x0, 0x0, 0xf000000, 0x0}, // [152 153 154 155]
	/* idx: 103 */ {0x0, 0x0, 0xf0000000, 0x0}, // [156 157 158 159]
	/* idx: 104 */ {0x0, 0x0, 0xf00000000, 0x0}, // [160 161 162 163]
	/* idx: 105 */ {0x0, 0x0, 0xf000000000, 0x0}, // [164 165 166 167]
	/* idx: 106 */ {0x0, 0x0, 0xf0000000000, 0x0}, // [168 169 170 171]
	/* idx: 107 */ {0x0, 0x0, 0xf00000000000, 0x0}, // [172 173 174 175]
	/* idx: 108 */ {0x0, 0x0, 0xf000000000000, 0x0}, // [176 177 178 179]
	/* idx: 109 */ {0x0, 0x0, 0xf0000000000000, 0x0}, // [180 181 182 183]
	/* idx: 110 */ {0x0, 0x0, 0xf00000000000000, 0x0}, // [184 185 186 187]
	/* idx: 111 */ {0x0, 0x0, 0xf000000000000000, 0x0}, // [188 189 190 191]
	/* idx: 112 */ {0x0, 0x0, 0x0, 0xf}, // [192 193 194 195]
	/* idx: 113 */ {0x0, 0x0, 0x0, 0xf0}, // [196 197 198 199]
	/* idx: 114 */ {0x0, 0x0, 0x0, 0xf00}, // [200 201 202 203]
	/* idx: 115 */ {0x0, 0x0, 0x0, 0xf000}, // [204 205 206 207]
	/* idx: 116 */ {0x0, 0x0, 0x0, 0xf0000}, // [208 209 210 211]
	/* idx: 117 */ {0x0, 0x0, 0x0, 0xf00000}, // [212 213 214 215]
	/* idx: 118 */ {0x0, 0x0, 0x0, 0xf000000}, // [216 217 218 219]
	/* idx: 119 */ {0x0, 0x0, 0x0, 0xf0000000}, // [220 221 222 223]
	/* idx: 120 */ {0x0, 0x0, 0x0, 0xf00000000}, // [224 225 226 227]
	/* idx: 121 */ {0x0, 0x0, 0x0, 0xf000000000}, // [228 229 230 231]
	/* idx: 122 */ {0x0, 0x0, 0x0, 0xf0000000000}, // [232 233 234 235]
	/* idx: 123 */ {0x0, 0x0, 0x0, 0xf00000000000}, // [236 237 238 239]
	/* idx: 124 */ {0x0, 0x0, 0x0, 0xf000000000000}, // [240 241 242 243]
	/* idx: 125 */ {0x0, 0x0, 0x0, 0xf0000000000000}, // [244 245 246 247]
	/* idx: 126 */ {0x0, 0x0, 0x0, 0xf00000000000000}, // [248 249 250 251]
	/* idx: 127 */ {0x0, 0x0, 0x0, 0xf000000000000000}, // [252 253 254 255]
	/* idx: 128 */ {0x3, 0x0, 0x0, 0x0}, // [0 1]
	/* idx: 129 */ {0xc, 0x0, 0x0, 0x0}, // [2 3]
	/* idx: 130 */ {0x30, 0x0, 0x0, 0x0}, // [4 5]
	/* idx: 131 */ {0xc0, 0x0, 0x0, 0x0}, // [6 7]
	/* idx: 132 */ {0x300, 0x0, 0x0, 0x0}, // [8 9]
	/* idx: 133 */ {0xc00, 0x0, 0x0, 0x0}, // [10 11]
	/* idx: 134 */ {0x3000, 0x0, 0x0, 0x0}, // [12 13]
	/* idx: 135 */ {0xc000, 0x0, 0x0, 0x0}, // [14 15]
	/* idx: 136 */ {0x30000, 0x0, 0x0, 0x0}, // [16 17]
	/* idx: 137 */ {0xc0000, 0x0, 0x0, 0x0}, // [18 19]
	/* idx: 138 */ {0x300000, 0x0, 0x0, 0x0}, // [20 21]
	/* idx: 139 */ {0xc00000, 0x0, 0x0, 0x0}, // [22 23]
	/* idx: 140 */ {0x3000000, 0x0, 0x0, 0x0}, // [24 25]
	/* idx: 141 */ {0xc000000, 0x0, 0x0, 0x0}, // [26 27]
	/* idx: 142 */ {0x30000000, 0x0, 0x0, 0x0}, // [28 29]
	/* idx: 143 */ {0xc0000000, 0x0, 0x0, 0x0}, // [30 31]
	/* idx: 144 */ {0x300000000, 0x0, 0x0, 0x0}, // [32 33]
	/* idx: 145 */ {0xc00000000, 0x0, 0x0, 0x0}, // [34 35]
	/* idx: 146 */ {0x3000000000, 0x0, 0x0, 0x0}, // [36 37]
	/* idx: 147 */ {0xc000000000, 0x0, 0x0, 0x0}, // [38 39]
	/* idx: 148 */ {0x30000000000, 0x0, 0x0, 0x0}, // [40 41]
	/* idx: 149 */ {0xc0000000000, 0x0, 0x0, 0x0}, // [42 43]
	/* idx: 150 */ {0x300000000000, 0x0, 0x0, 0x0}, // [44 45]
	/* idx: 151 */ {0xc00000000000, 0x0, 0x0, 0x0}, // [46 47]
	/* idx: 152 */ {0x3000000000000, 0x0, 0x0, 0x0}, // [48 49]
	/* idx: 153 */ {0xc000000000000, 0x0, 0x0, 0x0}, // [50 51]
	/* idx: 154 */ {0x30000000000000, 0x0, 0x0, 0x0}, // [52 53]
	/* idx: 155 */ {0xc0000000000000, 0x0, 0x0, 0x0}, // [54 55]
	/* idx: 156 */ {0x300000000000000, 0x0, 0x0, 0x0}, // [56 57]
	/* idx: 157 */ {0xc00000000000000, 0x0, 0x0, 0x0}, // [58 59]
	/* idx: 158 */ {0x3000000000000000, 0x0, 0x0, 0x0}, // [60 61]
	/* idx: 159 */ {0xc000000000000000, 0x0, 0x0, 0x0}, // [62 63]
	/* idx: 160 */ {0x0, 0x3, 0x0, 0x0}, // [64 65]
	/* idx: 161 */ {0x0, 0xc, 0x0, 0x0}, // [66 67]
	/* idx: 162 */ {0x0, 0x30, 0x0, 0x0}, // [68 69]
	/* idx: 163 */ {0x0, 0xc0, 0x0, 0x0}, // [70 71]
	/* idx: 164 */ {0x0, 0x300, 0x0, 0x0}, // [72 73]
	/* idx: 165 */ {0x0, 0xc00, 0x0, 0x0}, // [74 75]
	/* idx: 166 */ {0x0, 0x3000, 0x0, 0x0}, // [76 77]
	/* idx: 167 */ {0x0, 0xc000, 0x0, 0x0}, // [78 79]
	/* idx: 168 */ {0x0, 0x30000, 0x0, 0x0}, // [80 81]
	/* idx: 169 */ {0x0, 0xc0000, 0x0, 0x0}, // [82 83]
	/* idx: 170 */ {0x0, 0x300000, 0x0, 0x0}, // [84 85]
	/* idx: 171 */ {0x0, 0xc00000, 0x0, 0x0}, // [86 87]
	/* idx: 172 */ {0x0, 0x3000000, 0x0, 0x0}, // [88 89]
	/* idx: 173 */ {0x0, 0xc000000, 0x0, 0x0}, // [90 91]
	/* idx: 174 */ {0x0, 0x30000000, 0x0, 0x0}, // [92 93]
	/* idx: 175 */ {0x0, 0xc0000000, 0x0, 0x0}, // [94 95]
	/* idx: 176 */ {0x0, 0x300000000, 0x0, 0x0}, // [96 97]
	/* idx: 177 */ {0x0, 0xc00000000, 0x0, 0x0}, // [98 99]
	/* idx: 178 */ {0x0, 0x3000000000, 0x0, 0x0}, // [100 101]
	/* idx: 179 */ {0x0, 0xc000000000, 0x0, 0x0}, // [102 103]
	/* idx: 180 */ {0x0, 0x30000000000, 0x0, 0x0}, // [104 105]
	/* idx: 181 */ {0x0, 0xc0000000000, 0x0, 0x0}, // [106 107]
	/* idx: 182 */ {0x0, 0x300000000000, 0x0, 0x0}, // [108 109]
	/* idx: 183 */ {0x0, 0xc00000000000, 0x0, 0x0}, // [110 111]
	/* idx: 184 */ {0x0, 0x3000000000000, 0x0, 0x0}, // [112 113]
	/* idx: 185 */ {0x0, 0xc000000000000, 0x0, 0x0}, // [114 115]
	/* idx: 186 */ {0x0, 0x30000000000000, 0x0, 0x0}, // [116 117]
	/* idx: 187 */ {0x0, 0xc0000000000000, 0x0, 0x0}, // [118 119]
	/* idx: 188 */ {0x0, 0x300000000000000, 0x0, 0x0}, // [120 121]
	/* idx: 189 */ {0x0, 0xc00000000000000, 0x0, 0x0}, // [122 123]
	/* idx: 190 */ {0x0, 0x3000000000000000, 0x0, 0x0}, // [124 125]
	/* idx: 191 */ {0x0, 0xc000000000000000, 0x0, 0x0}, // [126 127]
	/* idx: 192 */ {0x0, 0x0, 0x3, 0x0}, // [128 129]
	/* idx: 193 */ {0x0, 0x0, 0xc, 0x0}, // [130 131]
	/* idx: 194 */ {0x0, 0x0, 0x30, 0x0}, // [132 133]
	/* idx: 195 */ {0x0, 0x0, 0xc0, 0x0}, // [134 135]
	/* idx: 196 */ {0x0, 0x0, 0x300, 0x0}, // [136 137]
	/* idx: 197 */ {0x0, 0x0, 0xc00, 0x0}, // [138 139]
	/* idx: 198 */ {0x0, 0x0, 0x3000, 0x0}, // [140 141]
	/* idx: 199 */ {0x0, 0x0, 0xc000, 0x0}, // [142 143]
	/* idx: 200 */ {0x0, 0x0, 0x30000, 0x0}, // [144 145]
	/* idx: 201 */ {0x0, 0x0, 0xc0000, 0x0}, // [146 147]
	/* idx: 202 */ {0x0, 0x0, 0x300000, 0x0}, // [148 149]
	/* idx: 203 */ {0x0, 0x0, 0xc00000, 0x0}, // [150 151]
	/* idx: 204 */ {0x0, 0x0, 0x3000000, 0x0}, // [152 153]
	/* idx: 205 */ {0x0, 0x0, 0xc000000, 0x0}, // [154 155]
	/* idx: 206 */ {0x0, 0x0, 0x30000000, 0x0}, // [156 157]
	/* idx: 207 */ {0x0, 0x0, 0xc0000000, 0x0}, // [158 159]
	/* idx: 208 */ {0x0, 0x0, 0x300000000, 0x0}, // [160 161]
	/* idx: 209 */ {0x0, 0x0, 0xc00000000, 0x0}, // [162 163]
	/* idx: 210 */ {0x0, 0x0, 0x3000000000, 0x0}, // [164 165]
	/* idx: 211 */ {0x0, 0x0, 0xc000000000, 0x0}, // [166 167]
	/* idx: 212 */ {0x0, 0x0, 0x30000000000, 0x0}, // [168 169]
	/* idx: 213 */ {0x0, 0x0, 0xc0000000000, 0x0}, // [170 171]
	/* idx: 214 */ {0x0, 0x0, 0x300000000000, 0x0}, // [172 173]
	/* idx: 215 */ {0x0, 0x0, 0xc00000000000, 0x0}, // [174 175]
	/* idx: 216 */ {0x0, 0x0, 0x3000000000000, 0x0}, // [176 177]
	/* idx: 217 */ {0x0, 0x0, 0xc000000000000, 0x0}, // [178 179]
	/* idx: 218 */ {0x0, 0x0, 0x30000000000000, 0x0}, // [180 181]
	/* idx: 219 */ {0x0, 0x0, 0xc0000000000000, 0x0}, // [182 183]
	/* idx: 220 */ {0x0, 0x0, 0x300000000000000, 0x0}, // [184 185]
	/* idx: 221 */ {0x0, 0x0, 0xc00000000000000, 0x0}, // [186 187]
	/* idx: 222 */ {0x0, 0x0, 0x3000000000000000, 0x0}, // [188 189]
	/* idx: 223 */ {0x0, 0x0, 0xc000000000000000, 0x0}, // [190 191]
	/* idx: 224 */ {0x0, 0x0, 0x0, 0x3}, // [192 193]
	/* idx: 225 */ {0x0, 0x0, 0x0, 0xc}, // [194 195]
	/* idx: 226 */ {0x0, 0x0, 0x0, 0x30}, // [196 197]
	/* idx: 227 */ {0x0, 0x0, 0x0, 0xc0}, // [198 199]
	/* idx: 228 */ {0x0, 0x0, 0x0, 0x300}, // [200 201]
	/* idx: 229 */ {0x0, 0x0, 0x0, 0xc00}, // [202 203]
	/* idx: 230 */ {0x0, 0x0, 0x0, 0x3000}, // [204 205]
	/* idx: 231 */ {0x0, 0x0, 0x0, 0xc000}, // [206 207]
	/* idx: 232 */ {0x0, 0x0, 0x0, 0x30000}, // [208 209]
	/* idx: 233 */ {0x0, 0x0, 0x0, 0xc0000}, // [210 211]
	/* idx: 234 */ {0x0, 0x0, 0x0, 0x300000}, // [212 213]
	/* idx: 235 */ {0x0, 0x0, 0x0, 0xc00000}, // [214 215]
	/* idx: 236 */ {0x0, 0x0, 0x0, 0x3000000}, // [216 217]
	/* idx: 237 */ {0x0, 0x0, 0x0, 0xc000000}, // [218 219]
	/* idx: 238 */ {0x0, 0x0, 0x0, 0x30000000}, // [220 221]
	/* idx: 239 */ {0x0, 0x0, 0x0, 0xc0000000}, // [222 223]
	/* idx: 240 */ {0x0, 0x0, 0x0, 0x300000000}, // [224 225]
	/* idx: 241 */ {0x0, 0x0, 0x0, 0xc00000000}, // [226 227]
	/* idx: 242 */ {0x0, 0x0, 0x0, 0x3000000000}, // [228 229]
	/* idx: 243 */ {0x0, 0x0, 0x0, 0xc000000000}, // [230 231]
	/* idx: 244 */ {0x0, 0x0, 0x0, 0x30000000000}, // [232 233]
	/* idx: 245 */ {0x0, 0x0, 0x0, 0xc0000000000}, // [234 235]
	/* idx: 246 */ {0x0, 0x0, 0x0, 0x300000000000}, // [236 237]
	/* idx: 247 */ {0x0, 0x0, 0x0, 0xc00000000000}, // [238 239]
	/* idx: 248 */ {0x0, 0x0, 0x0, 0x3000000000000}, // [240 241]
	/* idx: 249 */ {0x0, 0x0, 0x0, 0xc000000000000}, // [242 243]
	/* idx: 250 */ {0x0, 0x0, 0x0, 0x30000000000000}, // [244 245]
	/* idx: 251 */ {0x0, 0x0, 0x0, 0xc0000000000000}, // [246 247]
	/* idx: 252 */ {0x0, 0x0, 0x0, 0x300000000000000}, // [248 249]
	/* idx: 253 */ {0x0, 0x0, 0x0, 0xc00000000000000}, // [250 251]
	/* idx: 254 */ {0x0, 0x0, 0x0, 0x3000000000000000}, // [252 253]
	/* idx: 255 */ {0x0, 0x0, 0x0, 0xc000000000000000}, // [254 255]
}
