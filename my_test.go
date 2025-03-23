package bart

import "testing"

/*
   overlaps_test.go:89: Overlaps(...) = true, want false
       Table1:
       ▼
       ├─ 54.102.29.44/30 (8788480759311552956)
       ├─ 160.0.0.0/5 (2090130847577195184)
       └─ 238.51.122.208/28 (5781180826861556415)
       ▼
       ├─ 2ac:9d75:f1a9::/48 (6680848805017572815)
       ├─ f300::/10 (3513045615083109800)
       └─ f380::/9 (7263590773109019170)

       Table:
       ▼
       ├─ 20.0.0.0/6 (5116958309710543169)
       ├─ 64.0.0.0/2 (9179990863465231815)
       └─ 216.180.200.72/31 (3589150318812813147)
       ▼
       ├─ 1e33:e607:fea:154e:b28e:5ee0::/91 (7151115446578232980)
       ├─ 3c7d:7bf2:d165:878a:8828:24f9:3bec:0/115 (8560986066539745469)
       └─ f352:bc29:8d4d:42c0::/58 (3301376361915948623)
*/

func TestMy(t *testing.T) {
	pfxs1 := []string{
		"54.102.29.44/30",
		"54.102.29.44/32",
		"160.0.0.0/5",
		"238.51.122.208/28",
		"2ac:9d75:f1a9::/48",
		"f380::/9",
		"f300::/10",
	}
	pfxs2 := []string{
		"20.0.0.0/6",
		"64.0.0.0/2",
		"216.180.200.72/31",
		"1e33:e607:fea:154e:b28e:5ee0::/91",
		"3c7d:7bf2:d165:878a:8828:24f9:3bec:0/115",
		"f352:bc29:8d4d:42c0::/58",
	}

	tbl1 := new(Lite)
	tbl2 := new(Lite)

	for _, pfxs := range pfxs1 {
		tbl1.Insert(mpp(pfxs))
	}
	for _, pfxs := range pfxs2 {
		tbl2.Insert(mpp(pfxs))
	}

	if tbl1.Overlaps4(tbl2) {
		t.Errorf("tables overlap, want false!")
	}
	if tbl1.Overlaps6(tbl2) {
		t.Errorf("tables overlap, want false!")
	}
	if tbl1.Overlaps(tbl2) {
		t.Errorf("tables overlap, want false!")
	}

	tbl1.Union(tbl2)
	if !tbl1.Overlaps4(tbl2) {
		t.Errorf("tables don't overlap, want true!")
	}
	if !tbl1.Overlaps6(tbl2) {
		t.Errorf("tables don't overlap, want true!")
	}
	if !tbl1.Overlaps(tbl2) {
		t.Errorf("tables don't overlap, want true!")
	}

	tbl3 := tbl2.Clone()
	if !tbl3.Overlaps4(tbl1) {
		t.Errorf("tables don't overlap, want true!")
	}
	if !tbl3.Overlaps6(tbl1) {
		t.Errorf("tables don't overlap, want true!")
	}
	if !tbl3.Overlaps(tbl1) {
		t.Errorf("tables don't overlap, want true!")
	}

	t.Log(tbl3.Subnets(mpp("0.0.0.0/0")))
}
