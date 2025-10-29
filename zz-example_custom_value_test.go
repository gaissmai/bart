package bart_test

import (
	"fmt"
	"maps"

	"github.com/gaissmai/bart"
)

type Route struct {
	ASN   int
	Attrs map[string]string
}

func (r Route) Equal(other Route) bool {
	return r.ASN == other.ASN && maps.Equal(r.Attrs, other.Attrs)
}

func (r Route) Clone() Route {
	return Route{
		ASN:   r.ASN,
		Attrs: maps.Clone(r.Attrs),
	}
}

// Example of a custom value type with both equality and cloning
func ExampleTable_customValue() {
	table := new(bart.Table[Route])
	table = table.InsertPersist(mpp("10.0.0.0/8"), Route{ASN: 64512, Attrs: map[string]string{"foo": "bar"}})
	clone := table.Clone()
	fmt.Printf("Cloned tables are equal: %v\n", table.Equal(clone))

	// Output:
	// Cloned tables are equal: true
}
