package bart

import "testing"

func TestMy(t *testing.T) {
	pfx1 := mpp("10.0.0.0/7")
	pfx2 := mpp("20.0.0.0/8")
	pfx3 := mpp("30.30.0.0/15")

	t1 := new(Table[string])

	t1.Insert(pfx1, "as prefix")
	t1.Insert(pfx2, "as fringe")
	t1.Insert(pfx3, "as leaf")

	t.Log(t1.dumpString())

	// prefix
	t1ValBefore, t1OkBefore := t1.Get(pfx1)
	_ = t1.InsertPersist(pfx1, "override prefix")
	t1ValAfter, t1OkAfter := t1.Get(pfx1)

	if t1ValBefore != t1ValAfter || t1OkBefore != t1OkAfter {
		t.Errorf("InsertPersist changed underlying table for prefix: %s", pfx1)
	}

	// fringe
	t1ValBefore, t1OkBefore = t1.Get(pfx2)
	_ = t1.InsertPersist(pfx1, "override fringe")
	t1ValAfter, t1OkAfter = t1.Get(pfx2)

	if t1ValBefore != t1ValAfter || t1OkBefore != t1OkAfter {
		t.Errorf("InsertPersist changed underlying table for prefix: %s", pfx2)
	}

	// leaf
	t1ValBefore, t1OkBefore = t1.Get(pfx3)
	_ = t1.InsertPersist(pfx1, "override leaf")
	t1ValAfter, t1OkAfter = t1.Get(pfx3)

	if t1ValBefore != t1ValAfter || t1OkBefore != t1OkAfter {
		t.Errorf("InsertPersist changed underlying table for prefix: %s", pfx3)
	}

	t.Log(t1.dumpString())
}
