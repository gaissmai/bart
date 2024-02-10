// Copyright (c) 2024 Karl Gaissmaier
// SPDX-License-Identifier: MIT

package bart

import (
	"net/netip"
	"reflect"
	"testing"
)

func TestStatisticsZero(t *testing.T) {
	tbl := new(Table[any])
	stats := tbl.readTableStats()

	// no prefixes
	wantSize4 := 0
	gotSize4 := stats["/ipv4/size:count"].(int)
	if gotSize4 != wantSize4 {
		t.Errorf("Zero, Size4, want: %d, got: %d", wantSize4, gotSize4)
	}

	wantSize6 := 0
	gotSize6 := stats["/ipv6/size:count"].(int)
	if gotSize6 != wantSize6 {
		t.Errorf("Zero, Size6, want: %d, got: %d", wantSize6, gotSize6)
	}

	// just the ROOT nodes
	wantTypes4 := map[string]int{"ROOT": 1}
	gotTypes4 := stats["/ipv4/types:histogram"].(map[string]int)
	if !reflect.DeepEqual(gotTypes4, wantTypes4) {
		t.Errorf("Zero, Types4, want: %v, got: %v", wantTypes4, gotTypes4)
	}

	wantTypes6 := map[string]int{"ROOT": 1}
	gotTypes6 := stats["/ipv6/types:histogram"].(map[string]int)
	if !reflect.DeepEqual(gotTypes6, wantTypes6) {
		t.Errorf("Zero, Types6, want: %v, got: %v", wantTypes6, gotTypes6)
	}
}

func TestStatisticsOne(t *testing.T) {
	p := netip.MustParsePrefix
	tbl := new(Table[any])
	tbl.Insert(p("10.0.0.0/8"), nil)
	tbl.Insert(p("2000::/8"), nil)

	stats := tbl.readTableStats()

	wantSize4 := 1
	gotSize4 := stats["/ipv4/size:count"].(int)
	if gotSize4 != wantSize4 {
		t.Errorf("One, Size4, want: %d, got: %d", wantSize4, gotSize4)
	}

	wantSize6 := 1
	gotSize6 := stats["/ipv6/size:count"].(int)
	if gotSize6 != wantSize6 {
		t.Errorf("One, Size6, want: %d, got: %d", wantSize6, gotSize6)
	}

	// just LEAF nodes
	wantTypes4 := map[string]int{"LEAF": 1}
	gotTypes4 := stats["/ipv4/types:histogram"].(map[string]int)
	if !reflect.DeepEqual(gotTypes4, wantTypes4) {
		t.Errorf("One, Types4, want: %v, got: %v", wantTypes4, gotTypes4)
	}

	wantTypes6 := map[string]int{"LEAF": 1}
	gotTypes6 := stats["/ipv6/types:histogram"].(map[string]int)
	if !reflect.DeepEqual(gotTypes6, wantTypes6) {
		t.Errorf("One, Types6, want: %v, got: %v", wantTypes6, gotTypes6)
	}
}

func TestStatisticsDuplicate(t *testing.T) {
	p := netip.MustParsePrefix
	tbl := new(Table[any])
	tbl.Insert(p("10.0.0.1/32"), nil)
	tbl.Insert(p("10.0.0.1/32"), nil)
	tbl.Insert(p("2001:db8:beef::/48"), nil)
	tbl.Insert(p("2001:db8:beef::/48"), nil)

	stats := tbl.readTableStats()

	wantSize4 := 1
	gotSize4 := stats["/ipv4/size:count"].(int)
	if gotSize4 != wantSize4 {
		t.Errorf("Duplicate, Size4, want: %d, got: %d", wantSize4, gotSize4)
	}

	wantSize6 := 1
	gotSize6 := stats["/ipv6/size:count"].(int)
	if gotSize6 != wantSize6 {
		t.Errorf("Duplicate, Size6, want: %d, got: %d", wantSize6, gotSize6)
	}

	// child distribution, zero childs = 1 node, 1 child = 3 nodes
	wantChilds4 := map[int]int{0: 1, 1: 3}
	gotChilds4 := stats["/ipv4/childs:histogram"].(map[int]int)
	if !reflect.DeepEqual(gotChilds4, wantChilds4) {
		t.Errorf("Duplicate, Childs4, want: %v, got: %v", wantChilds4, gotChilds4)
	}

	// child distribution, zero childs = 1 node, 1 child = 5 nodes
	wantChilds6 := map[int]int{0: 1, 1: 5}
	gotChilds6 := stats["/ipv6/childs:histogram"].(map[int]int)
	if !reflect.DeepEqual(gotChilds6, wantChilds6) {
		t.Errorf("Duplicate, Childs6, want: %v, got: %v", wantChilds6, gotChilds6)
	}

	// type distribution
	wantTypes4 := map[string]int{"LEAF": 1, "IMED": 3}
	gotTypes4 := stats["/ipv4/types:histogram"].(map[string]int)
	if !reflect.DeepEqual(gotTypes4, wantTypes4) {
		t.Errorf("Duplicate, Types4, want: %v, got: %v", wantTypes4, gotTypes4)
	}

	wantTypes6 := map[string]int{"LEAF": 1, "IMED": 5}
	gotTypes6 := stats["/ipv6/types:histogram"].(map[string]int)
	if !reflect.DeepEqual(gotTypes6, wantTypes6) {
		t.Errorf("Duplicate, Types6, want: %v, got: %v", wantTypes6, gotTypes6)
	}
}

func TestStatisticsDelete(t *testing.T) {
	p := netip.MustParsePrefix
	tbl := new(Table[any])
	tbl.Insert(p("10.0.0.1/32"), nil)
	tbl.Insert(p("2001:db8::/32"), nil)

	tbl.Delete(p("10.0.0.1/32"))
	tbl.Delete(p("2001:db8::/32"))

	stats := tbl.readTableStats()

	// no prefixes left
	want := 0
	got := stats["/ipv4/size:count"].(int)
	if got != want {
		t.Errorf("Delete, Size4, want: %d, got: %d", want, got)
	}

	got = stats["/ipv6/size:count"].(int)
	if got != want {
		t.Errorf("Delete, Size6, want: %d, got: %d", want, got)
	}

	// just the ROOT nodes left
	wantTypes := map[string]int{"ROOT": 1}
	gotTypes := stats["/ipv4/types:histogram"].(map[string]int)

	if !reflect.DeepEqual(gotTypes, wantTypes) {
		t.Errorf("Delete, Types4, want: %v, got: %v", wantTypes, gotTypes)
	}

	gotTypes = stats["/ipv6/types:histogram"].(map[string]int)

	if !reflect.DeepEqual(gotTypes, wantTypes) {
		t.Errorf("Delete, Types6, want: %v, got: %v", wantTypes, gotTypes)
	}
}

func TestStatisticsSome(t *testing.T) {
	p := netip.MustParsePrefix
	tbl := new(Table[any])

	tbl.Insert(p("127.0.0.0/8"), nil)
	tbl.Insert(p("127.0.0.1/32"), nil)
	tbl.Insert(p("169.254.0.0/16"), nil)
	tbl.Insert(p("192.168.1.0/24"), nil)
	tbl.Insert(p("172.16.0.0/12"), nil)
	tbl.Insert(p("10.0.0.0/24"), nil)
	tbl.Insert(p("10.0.1.0/24"), nil)
	tbl.Insert(p("192.168.0.0/16"), nil)
	tbl.Insert(p("10.0.0.0/8"), nil)

	tbl.Insert(p("::/0"), nil)
	tbl.Insert(p("::1/128"), nil)
	tbl.Insert(p("2000::/3"), nil)
	tbl.Insert(p("2001:db8::/32"), nil)
	tbl.Insert(p("fe80::/10"), nil)

	stats := tbl.readTableStats()

	wantSize4 := 9
	gotSize4 := stats["/ipv4/size:count"].(int)

	if gotSize4 != wantSize4 {
		t.Errorf("Some, Size4, want: %d, got: %d", wantSize4, gotSize4)
	}

	wantSize6 := 5
	gotSize6 := stats["/ipv6/size:count"].(int)
	if gotSize6 != wantSize6 {
		t.Errorf("Some, Size6, want: %d, got: %d", wantSize6, gotSize6)
	}

	wantDepth4 := map[int]int{0: 1, 1: 5, 2: 3, 3: 1}
	gotDepth4 := stats["/ipv4/depth:histogram"].(map[int]int)
	if !reflect.DeepEqual(gotDepth4, wantDepth4) {
		t.Errorf("Some, Depth4, want:\n%v\ngot:\n%d", wantDepth4, gotDepth4)
	}

	wantDepth6 := map[int]int{
		0: 1, 1: 3, 2: 2, 3: 2, 4: 1, 5: 1, 6: 1, 7: 1,
		8: 1, 9: 1, 10: 1, 11: 1, 12: 1, 13: 1, 14: 1, 15: 1,
	}
	gotDepth6 := stats["/ipv6/depth:histogram"].(map[int]int)
	if !reflect.DeepEqual(gotDepth6, wantDepth6) {
		t.Errorf("Some, Depth6, want:\n%v\ngot:\n%d", wantDepth6, gotDepth6)
	}

	// child distribution
	wantChilds4 := map[int]int{0: 5, 1: 4, 5: 1}
	gotChilds4 := stats["/ipv4/childs:histogram"].(map[int]int)
	if !reflect.DeepEqual(gotChilds4, wantChilds4) {
		t.Errorf("Some, Childs4, want: %v, got: %v", wantChilds4, gotChilds4)
	}

	wantChilds6 := map[int]int{0: 3, 1: 16, 3: 1}
	gotChilds6 := stats["/ipv6/childs:histogram"].(map[int]int)
	if !reflect.DeepEqual(gotChilds6, wantChilds6) {
		t.Errorf("Some, Childs6, want: %v, got: %v", wantChilds6, gotChilds6)
	}

	// types distribution
	wantTypes4 := map[string]int{"FULL": 2, "LEAF": 5, "IMED": 3}
	gotTypes4 := stats["/ipv4/types:histogram"].(map[string]int)
	if !reflect.DeepEqual(gotTypes4, wantTypes4) {
		t.Errorf("Some, Types4, want:\n%v\ngot:\n%v", wantTypes4, gotTypes4)
	}

	wantTypes6 := map[string]int{"FULL": 1, "LEAF": 3, "IMED": 16}
	gotTypes6 := stats["/ipv6/types:histogram"].(map[string]int)
	if !reflect.DeepEqual(gotTypes6, wantTypes6) {
		t.Errorf("Some, Types6, want:\n%v\ngot:\n%v", wantTypes6, gotTypes6)
	}

	// prefixLen distribution
	wantPfxLen4 := map[int]int{8: 2, 12: 1, 16: 2, 24: 3, 32: 1}
	gotPfxLen4 := stats["/ipv4/prefixlen:histogram"].(map[int]int)
	if !reflect.DeepEqual(gotPfxLen4, wantPfxLen4) {
		t.Errorf("Some, Prefixlen4, want: %v, got: %v", wantPfxLen4, gotPfxLen4)
	}

	wantPfxLen6 := map[int]int{0: 1, 3: 1, 10: 1, 32: 1, 128: 1}
	gotPfxLen6 := stats["/ipv6/prefixlen:histogram"].(map[int]int)
	if !reflect.DeepEqual(gotPfxLen6, wantPfxLen6) {
		t.Errorf("Some, Prefixlen6, want: %v, got: %v", wantPfxLen6, gotPfxLen6)
	}
}
