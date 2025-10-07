package bart

import (
	"bufio"
	"compress/gzip"
	"fmt"
	"iter"
	"log"
	"math/rand/v2"
	"net/netip"
	"os"
	"slices"
	"strings"
	"testing"
)

// this file contains init functions and helpers for test functions

// workLoadN to adjust loops for tests with -short
func workLoadN() int {
	if testing.Short() {
		return 100
	}
	return 1_000
}

// full internet prefix list, gzipped
const prefixFile = "testdata/prefixes.txt.gz"

var (
	routes  []route
	routes4 []route
	routes6 []route

	randRoute4 route
	randRoute6 route

	matchIP4  netip.Addr
	matchIP6  netip.Addr
	matchPfx4 netip.Prefix
	matchPfx6 netip.Prefix

	missIP4  netip.Addr
	missIP6  netip.Addr
	missPfx4 netip.Prefix
	missPfx6 netip.Prefix
)

type route struct {
	CIDR  netip.Prefix
	Value any
}

func init() {
	prng := rand.New(rand.NewPCG(42, 42))
	fillRouteTables()

	if len(routes4) == 0 || len(routes6) == 0 {
		log.Fatal("no routes loaded from " + prefixFile)
	}

	randRoute4 = routes4[prng.IntN(len(routes4))]
	randRoute6 = routes6[prng.IntN(len(routes6))]

	lt := new(Lite)
	for _, route := range routes {
		lt.Insert(route.CIDR)
	}

	// find a random match IP4 and IP6
	for {
		matchIP4 = randomRealWorldPrefixes4(prng, 1)[0].Addr().Next()
		if ok := lt.Contains(matchIP4); ok {
			break
		}
	}
	for {
		matchIP6 = randomRealWorldPrefixes6(prng, 1)[0].Addr().Next()
		if ok := lt.Contains(matchIP6); ok {
			break
		}
	}

	// find a random match Pfx4
	for {
		matchPfx4 = randomRealWorldPrefixes4(prng, 1)[0]
		if ok := lt.LookupPrefix(matchPfx4); ok {
			break
		}
	}
	for {
		matchPfx6 = randomRealWorldPrefixes6(prng, 1)[0]
		if ok := lt.LookupPrefix(matchPfx6); ok {
			break
		}
	}

	for {
		missIP4 = randomRealWorldPrefixes4(prng, 1)[0].Addr().Next()
		if ok := lt.Contains(missIP4); !ok {
			break
		}
	}
	for {
		missIP6 = randomRealWorldPrefixes6(prng, 1)[0].Addr().Next()
		if ok := lt.Contains(missIP6); !ok {
			break
		}
	}

	for {
		missPfx4 = randomRealWorldPrefixes4(prng, 1)[0]
		if ok := lt.LookupPrefix(missPfx4); !ok {
			break
		}
	}
	for {
		missPfx6 = randomRealWorldPrefixes6(prng, 1)[0]
		if ok := lt.LookupPrefix(missPfx6); !ok {
			break
		}
	}
}

func fillRouteTables() {
	file, err := os.Open(prefixFile)
	if err != nil {
		log.Fatal(err)
	}
	defer file.Close()

	rgz, err := gzip.NewReader(file)
	if err != nil {
		log.Fatal(err)
	}
	defer rgz.Close()

	scanner := bufio.NewScanner(rgz)
	for scanner.Scan() {
		line := scanner.Text()
		line = strings.TrimSpace(line)

		cidr := netip.MustParsePrefix(line)
		cidr = cidr.Masked()

		routes = append(routes, route{cidr, cidr})

		if cidr.Addr().Is4() {
			routes4 = append(routes4, route{cidr, cidr})
		} else {
			routes6 = append(routes6, route{cidr, cidr})
		}
	}

	if err = scanner.Err(); err != nil {
		log.Fatalf("reading %s, %v", prefixFile, err)
	}
}

// #########################################################

// tests for deep copies with Cloner interface
type MyInt int

// implement the Cloner interface
func (i *MyInt) Clone() *MyInt {
	a := *i
	return &a
}

// A simple type that implements Equaler for testing.
type stringVal string

func (v stringVal) Equal(other stringVal) bool {
	return v == other
}

// Helper functions
func countDumpListNodes[V any](nodes []DumpListNode[V]) int {
	count := len(nodes)
	for _, node := range nodes {
		count += countDumpListNodes(node.Subnets)
	}
	return count
}

func verifyAllIPv4Nodes[V any](t *testing.T, nodes []DumpListNode[V]) {
	for i, node := range nodes {
		if !node.CIDR.Addr().Is4() {
			t.Errorf("Node %d is not IPv4 prefix: %v", i, node.CIDR)
		}
		// Recursively check subnets
		verifyAllIPv4Nodes(t, node.Subnets)
	}
}

func verifyAllIPv6Nodes[V any](t *testing.T, nodes []DumpListNode[V]) {
	for i, node := range nodes {
		if !node.CIDR.Addr().Is6() {
			t.Errorf("Node %d is not IPv6 prefix: %v", i, node.CIDR)
		}
		// Recursively check subnets
		verifyAllIPv6Nodes(t, node.Subnets)
	}
}

func getsEqual[V comparable](a V, aOK bool, b V, bOK bool) bool {
	if !aOK && !bOK {
		return true
	}
	if aOK != bOK {
		return false
	}
	return a == b
}

var mpa = netip.MustParseAddr

var mpp = func(s string) netip.Prefix {
	pfx := netip.MustParsePrefix(s)
	if pfx == pfx.Masked() {
		return pfx
	}
	panic(fmt.Sprintf("%s is not canonicalized as %s", s, pfx.Masked()))
}

func mustPanic(t *testing.T, name string, fn func()) {
	t.Helper()
	defer func() {
		if recover() == nil {
			t.Fatalf("%s must panic", name)
		}
	}()
	fn()
}

func noPanic(t *testing.T, name string, fn func()) {
	t.Helper()
	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("%s panicked: %v", name, r)
		}
	}()
	fn()
}

func noPanicRangeOverFunc[V any](t *testing.T, name string, fn any) {
	t.Helper()
	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("%s panicked: %v", name, r)
		}
	}()

	switch iterFunc := fn.(type) {
	case func() iter.Seq[netip.Prefix]:
		for range iterFunc() {
		}
	case func() iter.Seq2[netip.Prefix, V]:
		for range iterFunc() {
		}
	case func(netip.Prefix) iter.Seq[netip.Prefix]:
		pfx := mpp("1.2.3.4/32")
		for range iterFunc(pfx) {
		}
	case func(netip.Prefix) iter.Seq2[netip.Prefix, V]:
		pfx := mpp("1.2.3.4/32")
		for range iterFunc(pfx) {
		}
	default:
		t.Fatalf("%s unknown iter function: %T", name, iterFunc)
	}
}

// ##################### helpers ############################

type tableOverlapsTest struct {
	prefix string
	want   bool
}

// checkOverlapsPrefix verifies that the overlaps lookups in tt return the
// expected results on tbl.
func checkOverlapsPrefix(t *testing.T, tblInterface any, tests []tableOverlapsTest) {
	t.Helper()
	tbl := tblInterface.(interface{ OverlapsPrefix(netip.Prefix) bool })
	for _, tt := range tests {
		got := tbl.OverlapsPrefix(mpp(tt.prefix))
		if got != tt.want {
			t.Errorf("OverlapsPrefix(%v) = %v, want %v", mpp(tt.prefix), got, tt.want)
		}
	}
}

// dumpAsGoldTable, just a helper to compare with golden table.
func (t *Table[V]) dumpAsGoldTable() goldTable[V] {
	var gold goldTable[V]

	for p, v := range t.AllSorted() {
		gold = append(gold, goldTableItem[V]{pfx: p, val: v})
	}

	return gold
}

// dumpAsGoldTable, just a helper to compare with golden table.
func (f *Fast[V]) dumpAsGoldTable() goldTable[V] {
	var gold goldTable[V]

	for p, v := range f.AllSorted() {
		gold = append(gold, goldTableItem[V]{pfx: p, val: v})
	}

	return gold
}

// dumpAsGoldTable, just a helper to compare with golden table.
func dumpAsGoldTable[V any](l *Lite) goldTable[V] {
	var zero V
	var gold goldTable[V]

	for p := range l.AllSorted() {
		gold = append(gold, goldTableItem[V]{pfx: p, val: zero})
	}

	return gold
}

// goldTable is a simple and slow route table, implemented as a slice of prefixes
// and values as a golden reference for bart.Table.
type goldTable[V any] []goldTableItem[V]

type goldTableItem[V any] struct {
	pfx netip.Prefix
	val V
}

func (g goldTableItem[V]) String() string {
	return fmt.Sprintf("(%s, %v)", g.pfx, g.val)
}

func (t *goldTable[V]) insert(pfx netip.Prefix, val V) {
	pfx = pfx.Masked()
	for i, item := range *t {
		if item.pfx == pfx {
			(*t)[i].val = val // de-dupe
			return
		}
	}
	*t = append(*t, goldTableItem[V]{pfx, val})
}

func (t *goldTable[V]) delete(pfx netip.Prefix) (exists bool) {
	pfx = pfx.Masked()

	for i, item := range *t {
		if item.pfx == pfx {
			*t = slices.Delete(*t, i, i+1)
			return true
		}
	}
	return false
}

func (t goldTable[V]) get(pfx netip.Prefix) (val V, ok bool) {
	pfx = pfx.Masked()
	for _, item := range t {
		if item.pfx == pfx {
			return item.val, true
		}
	}
	return val, false
}

func (t *goldTable[V]) update(pfx netip.Prefix, cb func(V, bool) V) (val V) {
	pfx = pfx.Masked()
	for i, item := range *t {
		if item.pfx == pfx {
			// update val
			val = cb(item.val, true)
			(*t)[i].val = val
			return val
		}
	}
	// new val
	val = cb(val, false)

	*t = append(*t, goldTableItem[V]{pfx, val})
	return val
}

func (ta *goldTable[V]) union(tb *goldTable[V]) {
	for _, bItem := range *tb {
		var match bool
		for i, aItem := range *ta {
			if aItem.pfx == bItem.pfx {
				(*ta)[i] = bItem
				match = true
				break
			}
		}
		if !match {
			*ta = append(*ta, bItem)
		}
	}
}

func (t goldTable[V]) lookup(addr netip.Addr) (val V, ok bool) {
	bestLen := -1

	for _, item := range t {
		if item.pfx.Contains(addr) && item.pfx.Bits() > bestLen {
			val = item.val
			ok = true
			bestLen = item.pfx.Bits()
		}
	}
	return val, ok
}

func (t goldTable[V]) lookupPfx(pfx netip.Prefix) (val V, ok bool) {
	pfx = pfx.Masked()
	bestLen := -1

	for _, item := range t {
		if item.pfx.Overlaps(pfx) && item.pfx.Bits() <= pfx.Bits() && item.pfx.Bits() > bestLen {
			val = item.val
			ok = true
			bestLen = item.pfx.Bits()
		}
	}
	return val, ok
}

func (t goldTable[V]) lookupPfxLPM(pfx netip.Prefix) (lpm netip.Prefix, val V, ok bool) {
	pfx = pfx.Masked()
	bestLen := -1

	for _, item := range t {
		if item.pfx.Overlaps(pfx) && item.pfx.Bits() <= pfx.Bits() && item.pfx.Bits() > bestLen {
			val = item.val
			lpm = item.pfx
			ok = true
			bestLen = item.pfx.Bits()
		}
	}
	return lpm, val, ok
}

func (t goldTable[V]) allSorted() []netip.Prefix {
	var result []netip.Prefix

	for _, item := range t {
		result = append(result, item.pfx)
	}
	slices.SortFunc(result, cmpPrefix)
	return result
}

func (t goldTable[V]) subnets(pfx netip.Prefix) []netip.Prefix {
	pfx = pfx.Masked()
	var result []netip.Prefix

	for _, item := range t {
		if pfx.Overlaps(item.pfx) && pfx.Bits() <= item.pfx.Bits() {
			result = append(result, item.pfx)
		}
	}
	slices.SortFunc(result, cmpPrefix)
	return result
}

func (t goldTable[V]) supernets(pfx netip.Prefix) []netip.Prefix {
	pfx = pfx.Masked()
	var result []netip.Prefix

	for _, item := range t {
		if item.pfx.Overlaps(pfx) && item.pfx.Bits() <= pfx.Bits() {
			result = append(result, item.pfx)
		}
	}
	slices.SortFunc(result, cmpPrefix)
	slices.Reverse(result)
	return result
}

//nolint:unused
func (t goldTable[V]) overlapsPrefix(pfx netip.Prefix) bool {
	pfx = pfx.Masked()
	for _, p := range t {
		if p.pfx.Overlaps(pfx) {
			return true
		}
	}
	return false
}

//nolint:unused
func (ta *goldTable[V]) overlaps(tb *goldTable[V]) bool {
	for _, aItem := range *ta {
		for _, bItem := range *tb {
			if aItem.pfx.Overlaps(bItem.pfx) {
				return true
			}
		}
	}
	return false
}

// sort, inplace by netip.Prefix, all prefixes are in normalized form
func (t *goldTable[V]) sort() {
	slices.SortFunc(*t, func(a, b goldTableItem[V]) int {
		return cmpPrefix(a.pfx, b.pfx)
	})
}

// #####################################################################

// randomPrefix returns a randomly generated prefix
func randomPrefix(prng *rand.Rand) netip.Prefix {
	if prng.IntN(2) == 1 {
		return randomPrefix4(prng)
	}
	return randomPrefix6(prng)
}

func randomPrefix4(prng *rand.Rand) netip.Prefix {
	bits := prng.IntN(33)
	pfx, err := randomIP4(prng).Prefix(bits)
	if err != nil {
		panic(err)
	}
	return pfx
}

func randomPrefix6(prng *rand.Rand) netip.Prefix {
	bits := prng.IntN(129)
	pfx, err := randomIP6(prng).Prefix(bits)
	if err != nil {
		panic(err)
	}
	return pfx
}

func randomIP4(prng *rand.Rand) netip.Addr {
	var b [4]byte
	for i := range b {
		b[i] = byte(prng.UintN(256))
	}
	return netip.AddrFrom4(b)
}

func randomIP6(prng *rand.Rand) netip.Addr {
	var b [16]byte
	for i := range b {
		b[i] = byte(prng.UintN(256))
	}
	return netip.AddrFrom16(b)
}

func randomAddr(prng *rand.Rand) netip.Addr {
	if prng.IntN(2) == 1 {
		return randomIP4(prng)
	}
	return randomIP6(prng)
}

func randomRealWorldPrefixes4(prng *rand.Rand, n int) []netip.Prefix {
	set := make(map[netip.Prefix]struct{})
	pfxs := make([]netip.Prefix, 0, n)

	for len(set) < n {
		pfx := randomPrefix4(prng)

		// skip too small or too big masks
		if pfx.Bits() < 8 || pfx.Bits() > 28 {
			continue
		}

		// skip reserved/experimental ranges (e.g., 240.0.0.0/8)
		if pfx.Overlaps(mpp("240.0.0.0/8")) {
			continue
		}

		if _, ok := set[pfx]; !ok {
			set[pfx] = struct{}{}
			pfxs = append(pfxs, pfx)
		}
	}
	return pfxs
}

func randomRealWorldPrefixes6(prng *rand.Rand, n int) []netip.Prefix {
	set := make(map[netip.Prefix]struct{})
	pfxs := make([]netip.Prefix, 0, n)

	for len(set) < n {
		pfx := randomPrefix6(prng)

		// skip too small or too big masks
		if pfx.Bits() < 16 || pfx.Bits() > 56 {
			continue
		}

		// skip non global routes seen in the real world
		if !pfx.Overlaps(mpp("2000::/3")) {
			continue
		}
		if pfx.Addr().Compare(mpp("2c0f::/16").Addr()) == 1 {
			continue
		}

		if _, ok := set[pfx]; !ok {
			set[pfx] = struct{}{}
			pfxs = append(pfxs, pfx)
		}
	}
	return pfxs
}

func randomRealWorldPrefixes(prng *rand.Rand, n int) []netip.Prefix {
	pfxs := make([]netip.Prefix, 0, n)
	pfxs = append(pfxs, randomRealWorldPrefixes4(prng, n/2)...)
	pfxs = append(pfxs, randomRealWorldPrefixes6(prng, n-len(pfxs))...)

	prng.Shuffle(len(pfxs), func(i, j int) {
		pfxs[i], pfxs[j] = pfxs[j], pfxs[i]
	})

	return pfxs
}
