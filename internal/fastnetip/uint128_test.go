package fastnetip

import "testing"

func TestUint128IsZero(t *testing.T) {
	tests := []struct {
		name string
		u    uint128
		want bool
	}{
		{
			name: "both zero",
			u:    uint128{hi: 0, lo: 0},
			want: true,
		},
		{
			name: "hi non-zero",
			u:    uint128{hi: 1, lo: 0},
			want: false,
		},
		{
			name: "lo non-zero",
			u:    uint128{hi: 0, lo: 1},
			want: false,
		},
		{
			name: "both max uint64",
			u:    uint128{hi: ^uint64(0), lo: ^uint64(0)},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.u.isZero(); got != tt.want {
				t.Errorf("uint128.isZero() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestUint128And(t *testing.T) {
	tests := []struct {
		name string
		u    uint128
		v    uint128
		want uint128
	}{
		{
			name: "zero and zero",
			u:    uint128{hi: 0, lo: 0},
			v:    uint128{hi: 0, lo: 0},
			want: uint128{hi: 0, lo: 0},
		},
		{
			name: "all bits set and zero",
			u:    uint128{hi: ^uint64(0), lo: ^uint64(0)},
			v:    uint128{hi: 0, lo: 0},
			want: uint128{hi: 0, lo: 0},
		},
		{
			name: "pattern bitwise AND",
			u:    uint128{hi: 0xF0F0_F0F0_F0F0_F0F0, lo: 0x0F0F_0F0F_0F0F_0F0F},
			v:    uint128{hi: 0xFFFF_0000_FFFF_0000, lo: 0x0000_FFFF_0000_FFFF},
			want: uint128{hi: 0xF0F0_0000_F0F0_0000, lo: 0x0000_0F0F_0000_0F0F},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.u.and(tt.v); got != tt.want {
				t.Errorf("uint128.and() = %#v, want %#v", got, tt.want)
			}
		})
	}
}

func TestUint128Xor(t *testing.T) {
	tests := []struct {
		name string
		u    uint128
		v    uint128
		want uint128
	}{
		{
			name: "same values return zero",
			u:    uint128{hi: 0x1234, lo: 0x5678},
			v:    uint128{hi: 0x1234, lo: 0x5678},
			want: uint128{hi: 0, lo: 0},
		},
		{
			name: "complementary bits set all bits",
			u:    uint128{hi: 0xAAAA_AAAA_AAAA_AAAA, lo: 0x5555_5555_5555_5555},
			v:    uint128{hi: 0x5555_5555_5555_5555, lo: 0xAAAA_AAAA_AAAA_AAAA},
			want: uint128{hi: ^uint64(0), lo: ^uint64(0)},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.u.xor(tt.v); got != tt.want {
				t.Errorf("uint128.xor() = %#v, want %#v", got, tt.want)
			}
		})
	}
}

func TestMask6(t *testing.T) {
	tests := []struct {
		name string
		n    int
		want uint128
	}{
		{
			name: "prefix length 0",
			n:    0,
			want: uint128{hi: 0x0, lo: 0x0},
		},
		{
			name: "prefix length 1",
			n:    1,
			want: uint128{hi: 0x8000_0000_0000_0000, lo: 0x0},
		},
		{
			name: "prefix length 32 (/32 IPv4-in-IPv6 or subnets)",
			n:    32,
			want: uint128{hi: 0xFFFF_FFFF_0000_0000, lo: 0x0},
		},
		{
			name: "prefix length 64 (standard IPv6 subnet boundary)",
			n:    64,
			want: uint128{hi: 0xFFFF_FFFF_FFFF_FFFF, lo: 0x0},
		},
		{
			name: "prefix length 65",
			n:    65,
			want: uint128{hi: 0xFFFF_FFFF_FFFF_FFFF, lo: 0x8000_0000_0000_0000},
		},
		{
			name: "prefix length 128 (host mask)",
			n:    128,
			want: uint128{hi: 0xFFFF_FFFF_FFFF_FFFF, lo: 0xFFFF_FFFF_FFFF_FFFF},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := mask6(tt.n); got != tt.want {
				t.Errorf("mask6(%d) = uint128{hi: %#016x, lo: %#016x}, want uint128{hi: %#016x, lo: %#016x}",
					tt.n, got.hi, got.lo, tt.want.hi, tt.want.lo)
			}
		})
	}
}
