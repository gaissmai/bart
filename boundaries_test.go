package bart

import (
	"net/netip"
	"testing"
)

// ---- Boundary behavior tests for /0, /32, /128 with lastOctetPlusOne ----

func TestBoundaryBehavior_DefaultRoutes(t *testing.T) {
	t.Parallel()

	// Test both IPv4 and IPv6 default routes
	testCases := []struct {
		name   string
		prefix string
		isIPv6 bool
	}{
		{"IPv4_Default", "0.0.0.0/0", false},
		{"IPv6_Default", "::/0", true},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			tbl := new(Table[*routeEntry])
			pfx := netip.MustParsePrefix(tc.prefix)

			// Create appropriate default route
			var nextHop string
			if tc.isIPv6 {
				nextHop = "fe80::1"
			} else {
				nextHop = "192.168.1.1"
			}

			defaultRoute := newRoute(nextHop, "wan0", 1000)

			// Test Insert
			tbl.Insert(pfx, defaultRoute)
			if tbl.Size() != 1 {
				t.Errorf("expected size 1 after default route insert, got %d", tbl.Size())
			}

			// Test Get
			if got, ok := tbl.Get(pfx); !ok {
				t.Error("default route should be retrievable via Get")
			} else if got.attributes["metric"] != 1000 {
				t.Errorf("expected metric 1000, got %d", got.attributes["metric"])
			}

			// Test Modify
			tbl.Modify(pfx, func(old *routeEntry, found bool) (*routeEntry, bool) {
				if !found {
					t.Error("default route should be found in Modify")
				}
				updated := old.Clone()
				updated.attributes["metric"] = 500
				return updated, false
			})

			if got, ok := tbl.Get(pfx); !ok {
				t.Error("default route should exist after Modify")
			} else if got.attributes["metric"] != 500 {
				t.Errorf("expected updated metric 500, got %d", got.attributes["metric"])
			}

			// Test Delete
			if deleted, exists := tbl.Delete(pfx); !exists {
				t.Error("default route should exist for deletion")
			} else if deleted.attributes["metric"] != 500 {
				t.Error("deleted route should have correct metric")
			}

			if tbl.Size() != 0 {
				t.Errorf("expected size 0 after delete, got %d", tbl.Size())
			}
		})
	}
}

func TestBoundaryBehavior_HostRoutes(t *testing.T) {
	t.Parallel()

	// Test both IPv4 /32 and IPv6 /128 host routes
	testCases := []struct {
		name    string
		prefix  string
		nextHop string
		isIPv6  bool
	}{
		{"IPv4_Host", "203.0.113.5/32", "10.0.0.1", false},
		{"IPv6_Host", "2001:db8::42/128", "2001:db8::1", true},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			tbl := new(Table[*routeEntry])
			pfx := netip.MustParsePrefix(tc.prefix)
			hostRoute := newRoute(tc.nextHop, "lo", 1)

			// Test Insert - host routes should use leaf/fringe path
			tbl.Insert(pfx, hostRoute)
			if tbl.Size() != 1 {
				t.Errorf("expected size 1 after host route insert, got %d", tbl.Size())
			}

			// Test Get
			if got, ok := tbl.Get(pfx); !ok {
				t.Error("host route should be retrievable via Get")
			} else if got.exitIF != "lo" {
				t.Errorf("expected exitIF 'lo', got %s", got.exitIF)
			}

			// Test Contains - should find the exact host
			hostAddr := pfx.Addr()
			if !tbl.Contains(hostAddr) {
				t.Error("Contains should find the host address")
			}

			// Test Lookup - should return the host route
			if route, ok := tbl.Lookup(hostAddr); !ok {
				t.Error("Lookup should find the host route")
			} else if route.attributes["metric"] != 1 {
				t.Error("Lookup should return correct host route")
			}

			// Test Modify
			tbl.Modify(pfx, func(old *routeEntry, found bool) (*routeEntry, bool) {
				if !found {
					t.Error("host route should be found in Modify")
				}
				updated := old.Clone()
				updated.exitIF = "host-if"
				return updated, false
			})

			if got, ok := tbl.Get(pfx); !ok {
				t.Error("host route should exist after Modify")
			} else if got.exitIF != "host-if" {
				t.Errorf("expected updated exitIF 'host-if', got %s", got.exitIF)
			}

			// Test Delete
			if deleted, exists := tbl.Delete(pfx); !exists {
				t.Error("host route should exist for deletion")
			} else if deleted.exitIF != "host-if" {
				t.Error("deleted route should have updated exitIF")
			}

			if tbl.Size() != 0 {
				t.Errorf("expected size 0 after delete, got %d", tbl.Size())
			}

			// Verify host is no longer reachable
			if tbl.Contains(hostAddr) {
				t.Error("Contains should not find deleted host")
			}
		})
	}
}

func TestBoundaryBehavior_TransitionPaths(t *testing.T) {
	t.Parallel()

	// Test transitions between node-index and leaf/fringe paths
	t.Run("IPv4_Transitions", func(t *testing.T) {
		t.Parallel()
		tbl := new(Table[*routeEntry])

		// Insert routes with different prefix lengths that exercise boundary conditions
		routes := []struct {
			prefix string
			metric int
			desc   string
		}{
			{"0.0.0.0/0", 1000, "default_route"},    // depth=0, special case
			{"10.0.0.0/8", 100, "class_a"},          // depth=1, node-index path
			{"192.168.1.0/24", 10, "subnet"},        // depth=3, node-index path
			{"192.168.1.5/32", 1, "host"},           // depth=4, leaf/fringe path
			{"203.0.113.0/31", 5, "point_to_point"}, // depth=4, leaf/fringe path
		}

		// Insert all routes
		for _, r := range routes {
			pfx := netip.MustParsePrefix(r.prefix)
			route := newRoute("10.0.0.1", "eth0", r.metric)
			tbl.Insert(pfx, route)
		}

		if tbl.Size() != len(routes) {
			t.Errorf("expected size %d after all inserts, got %d", len(routes), tbl.Size())
		}

		// Verify all routes are accessible and have correct behavior
		for _, r := range routes {
			pfx := netip.MustParsePrefix(r.prefix)

			// Test Get
			if got, ok := tbl.Get(pfx); !ok {
				t.Errorf("route %s should be retrievable", r.desc)
			} else if got.attributes["metric"] != r.metric {
				t.Errorf("route %s: expected metric %d, got %d", r.desc, r.metric, got.attributes["metric"])
			}

			// Test longest-prefix matching behavior
			if r.desc == "host" {
				hostAddr := pfx.Addr()
				if route, ok := tbl.Lookup(hostAddr); !ok {
					t.Errorf("Lookup should find host route for %s", hostAddr)
				} else if route.attributes["metric"] != r.metric {
					t.Errorf("Lookup for host should return correct route")
				}
			}

		}

		// /31 endpoints should resolve to point_to_point route
		for _, addr := range []string{"203.0.113.0", "203.0.113.1"} {
			ip := netip.MustParseAddr(addr)
			if got, ok := tbl.Lookup(ip); !ok || got.attributes["metric"] != 5 {
				t.Errorf("Lookup %s expected p2p metric 5, got ok=%v", addr, ok)
			}
		}

		// Test modification across different path types
		for _, r := range routes {
			pfx := netip.MustParsePrefix(r.prefix)
			newMetric := r.metric + 1000

			tbl.Modify(pfx, func(old *routeEntry, found bool) (*routeEntry, bool) {
				if !found {
					t.Errorf("route %s should exist for modification", r.desc)
				}
				updated := old.Clone()
				updated.attributes["metric"] = newMetric
				return updated, false
			})

			// Verify modification worked
			if got, ok := tbl.Get(pfx); !ok {
				t.Errorf("route %s should exist after modification", r.desc)
			} else if got.attributes["metric"] != newMetric {
				t.Errorf("route %s: expected modified metric %d, got %d", r.desc, newMetric, got.attributes["metric"])
			}
		}

		// Test deletion in reverse order (most specific first)
		for i := len(routes) - 1; i >= 0; i-- {
			r := routes[i]
			pfx := netip.MustParsePrefix(r.prefix)

			if deleted, exists := tbl.Delete(pfx); !exists {
				t.Errorf("route %s should exist for deletion", r.desc)
			} else if deleted.attributes["metric"] != r.metric+1000 {
				t.Errorf("deleted route %s should have modified metric", r.desc)
			}

			expectedSize := i
			if tbl.Size() != expectedSize {
				t.Errorf("expected size %d after deleting %s, got %d", expectedSize, r.desc, tbl.Size())
			}
		}
	})

	t.Run("IPv6_Transitions", func(t *testing.T) {
		t.Parallel()
		tbl := new(Table[*routeEntry])

		// Insert IPv6 routes with different prefix lengths
		routes := []struct {
			prefix string
			metric int
			desc   string
		}{
			{"::/0", 1000, "ipv6_default"},                    // depth=0, special case
			{"2001:db8::/32", 100, "allocation"},              // depth=4, node-index path
			{"2001:db8:1::/48", 50, "site"},                   // depth=6, node-index path
			{"2001:db8:1:2::/64", 10, "subnet"},               // depth=8, node-index path
			{"2001:db8:1:2::5/128", 1, "ipv6_host"},           // depth=16, leaf/fringe path
			{"2001:db8:1:2:3:4:5:6/127", 5, "point_to_point"}, // depth=16, leaf/fringe path
		}

		// Insert all routes
		for _, r := range routes {
			pfx := netip.MustParsePrefix(r.prefix)
			route := &routeEntry{
				nextHop:    netip.MustParseAddr("2001:db8::1"),
				exitIF:     "eth0",
				attributes: map[string]int{"metric": r.metric, "preference": 100},
			}
			tbl.Insert(pfx, route)
		}

		if tbl.Size() != len(routes) {
			t.Errorf("expected size %d after all IPv6 inserts, got %d", len(routes), tbl.Size())
		}

		// Verify all IPv6 routes work correctly
		for _, r := range routes {
			pfx := netip.MustParsePrefix(r.prefix)

			if got, ok := tbl.Get(pfx); !ok {
				t.Errorf("IPv6 route %s should be retrievable", r.desc)
			} else if got.attributes["metric"] != r.metric {
				t.Errorf("IPv6 route %s: expected metric %d, got %d", r.desc, r.metric, got.attributes["metric"])
			}

			// Test IPv6 longest-prefix matching for host routes
			if r.desc == "ipv6_host" {
				hostAddr := pfx.Addr()
				if route, ok := tbl.Lookup(hostAddr); !ok {
					t.Errorf("IPv6 Lookup should find host route for %s", hostAddr)
				} else if route.attributes["metric"] != r.metric {
					t.Errorf("IPv6 Lookup for host should return correct route")
				}
			}
		}
	})
}

func TestBoundaryBehavior_MixedPrefixLengths(t *testing.T) {
	t.Parallel()

	// Test mixed prefix lengths that exercise various boundary conditions
	tbl := new(Table[*routeEntry])

	// Create a comprehensive set of prefixes that test boundary transitions
	prefixes := []struct {
		cidr   string
		metric int
		desc   string
	}{
		// IPv4 boundaries
		{"0.0.0.0/0", 1000, "ipv4_default"},    // depth=0
		{"128.0.0.0/1", 900, "ipv4_half"},      // depth=0, index=128
		{"192.0.0.0/2", 800, "ipv4_quarter"},   // depth=0, index=192
		{"10.0.0.0/8", 700, "class_a"},         // depth=1
		{"172.16.0.0/12", 600, "rfc1918_b"},    // depth=1
		{"192.168.0.0/16", 500, "rfc1918_c"},   // depth=2
		{"203.0.113.0/24", 400, "test_net"},    // depth=3
		{"203.0.113.0/25", 300, "subnet_half"}, // depth=3
		{"203.0.113.5/32", 100, "host_route"},  // depth=4, leaf/fringe

		// IPv6 boundaries
		{"::/0", 2000, "ipv6_default"},                   // depth=0
		{"2000::/3", 1900, "global_unicast"},             // depth=0
		{"2001:db8::/32", 1800, "documentation"},         // depth=4
		{"2001:db8:1::/48", 1700, "site_prefix"},         // depth=6
		{"2001:db8:1:2::/64", 1600, "subnet_prefix"},     // depth=8
		{"2001:db8:1:2:3::/80", 1500, "extended_prefix"}, // depth=10
		{"2001:db8:1:2:3:4::/96", 1400, "almost_host"},   // depth=12
		{"2001:db8:1:2:3:4:5:6/128", 1300, "ipv6_host"},  // depth=16, leaf/fringe
	}

	// Insert all prefixes
	for _, p := range prefixes {
		pfx := netip.MustParsePrefix(p.cidr)
		var nextHop netip.Addr
		if pfx.Addr().Is6() {
			nextHop = netip.MustParseAddr("2001:db8::1")
		} else {
			nextHop = netip.MustParseAddr("10.0.0.1")
		}

		route := &routeEntry{
			nextHop:    nextHop,
			exitIF:     "eth0",
			attributes: map[string]int{"metric": p.metric, "preference": 100},
		}
		tbl.Insert(pfx, route)
	}

	expectedSize := len(prefixes)
	if tbl.Size() != expectedSize {
		t.Errorf("expected size %d after mixed inserts, got %d", expectedSize, tbl.Size())
	}

	// Test that all operations work across the different boundary conditions
	for _, p := range prefixes {
		pfx := netip.MustParsePrefix(p.cidr)

		// Test Get works for all prefix types
		if got, ok := tbl.Get(pfx); !ok {
			t.Errorf("Get failed for %s (%s)", p.desc, p.cidr)
		} else if got.attributes["metric"] != p.metric {
			t.Errorf("%s: expected metric %d, got %d", p.desc, p.metric, got.attributes["metric"])
		}

		// Test Contains for appropriate addresses
		testAddr := pfx.Addr()
		if !tbl.Contains(testAddr) {
			t.Errorf("Contains should find address for %s", p.desc)
		}

		// Test Lookup returns correct route
		if route, ok := tbl.Lookup(testAddr); !ok {
			t.Errorf("Lookup failed for %s address", p.desc)
		} else {
			// Should get the most specific route (could be this one or more specific)
			if route.attributes["metric"] > p.metric {
				t.Errorf("Lookup returned less specific route than expected for %s", p.desc)
			}
		}
	}

	// Test Modify works across all boundary types
	for _, p := range prefixes {
		pfx := netip.MustParsePrefix(p.cidr)
		newMetric := p.metric + 10000

		tbl.Modify(pfx, func(old *routeEntry, found bool) (*routeEntry, bool) {
			if !found {
				t.Errorf("Modify: route should exist for %s", p.desc)
				return old, false
			}
			updated := old.Clone()
			updated.attributes["metric"] = newMetric
			return updated, false
		})

		// Verify modification
		if got, ok := tbl.Get(pfx); !ok {
			t.Errorf("route should exist after Modify for %s", p.desc)
		} else if got.attributes["metric"] != newMetric {
			t.Errorf("Modify failed for %s: expected %d, got %d", p.desc, newMetric, got.attributes["metric"])
		}
	}

	// Test Delete works for all boundary types
	for _, p := range prefixes {
		pfx := netip.MustParsePrefix(p.cidr)

		if deleted, exists := tbl.Delete(pfx); !exists {
			t.Errorf("Delete: route should exist for %s", p.desc)
		} else if deleted.attributes["metric"] != p.metric+10000 {
			t.Errorf("Delete returned wrong route for %s", p.desc)
		}

		expectedSize--
		if tbl.Size() != expectedSize {
			t.Errorf("expected size %d after deleting %s, got %d", expectedSize, p.desc, tbl.Size())
		}
	}

	// Final verification: table should be empty
	if tbl.Size() != 0 {
		t.Errorf("expected empty table after all deletions, got size %d", tbl.Size())
	}
}
