package bart

// testVal as a simple value type.
type testVal struct {
	Data int
}

// Clone ensures deep copying for use with ...Persist.
//
// We use *testVal as the generic payload V,
// which is a pointer type, so it must implement bart.Cloner[V]
func (v *testVal) Clone() *testVal {
	if v == nil {
		return nil
	}
	return &testVal{Data: v.Data}
}
