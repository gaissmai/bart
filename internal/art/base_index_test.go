package art

import "testing"

func TestHostIdx(t *testing.T) {
	testCases := []struct {
		octet uint
		want  uint
	}{
		{
			octet: 0,
			want:  256,
		},
		{
			octet: 255,
			want:  511,
		},
	}

	for _, tc := range testCases {
		got := HostIdx(tc.octet)
		if got != tc.want {
			t.Errorf("HostIdx(%d), want: %d, got: %d", tc.octet, tc.want, got)
		}
	}
}

func TestPfxLen(t *testing.T) {
	testCases := []struct {
		depth int
		idx   uint
		want  int
	}{
		{
			depth: 0,
			idx:   0,  // invalid
			want:  -1, // invalid
		},
		{
			depth: 0,
			idx:   1,
			want:  0,
		},
		{
			depth: 0,
			idx:   19,
			want:  4,
		},
		{
			depth: 15,
			idx:   19,
			want:  124,
		},
		{
			depth: 0,
			idx:   511,
			want:  8,
		},
		{
			depth: 3,
			idx:   511,
			want:  32,
		},
		{
			depth: 15,
			idx:   511,
			want:  128,
		},
	}

	for _, tc := range testCases {
		got := PfxLen(tc.depth, tc.idx)
		if got != tc.want {
			t.Errorf("PfxLen(%d, %d), want: %d, got: %d", tc.depth, tc.idx, tc.want, got)
		}
	}
}

func TestPfxToIdx(t *testing.T) {
	testCases := []struct {
		octet  uint8
		pfxLen int
		want   uint
	}{
		{
			octet:  0,
			pfxLen: 0,
			want:   1,
		},
		{
			octet:  0,
			pfxLen: 1,
			want:   2,
		},
		{
			octet:  128,
			pfxLen: 1,
			want:   3,
		},
		{
			octet:  80,
			pfxLen: 4,
			want:   21,
		},
		{
			octet:  255,
			pfxLen: 8,
			want:   511,
		},
	}

	for _, tc := range testCases {
		got := PfxToIdx(tc.octet, tc.pfxLen)
		if got != tc.want {
			t.Errorf("PfxToIdx(%d, %d), want: %d, got: %d", tc.octet, tc.pfxLen, tc.want, got)
		}
	}
}

func TestIdxToPfx(t *testing.T) {
	testCases := []struct {
		idx        uint
		wantOctet  uint8
		wantPfxLen int
	}{
		{
			idx:        0,  // invalid
			wantOctet:  0,  // invalid
			wantPfxLen: -1, // invalid
		},
		{
			idx:        1,
			wantOctet:  0,
			wantPfxLen: 0,
		},
		{
			idx:        15,
			wantOctet:  224,
			wantPfxLen: 3,
		},
		{
			idx:        256,
			wantOctet:  0,
			wantPfxLen: 8,
		},
		{
			idx:        511,
			wantOctet:  255,
			wantPfxLen: 8,
		},
	}

	for _, tc := range testCases {
		gotOctet, gotPfxLen := IdxToPfx(tc.idx)
		if gotOctet != tc.wantOctet || gotPfxLen != tc.wantPfxLen {
			t.Errorf("IdxToPfx(%d), want: (%d, %d), got: (%d, %d)", tc.idx, tc.wantOctet, tc.wantPfxLen, gotOctet, gotPfxLen)
		}
	}
}

func TestIdxToRange(t *testing.T) {
	testCases := []struct {
		idx       uint
		wantFirst uint8
		wantLast  uint8
	}{
		{
			idx:       0, // invalid, but not catched
			wantFirst: 0,
			wantLast:  255,
		},
		{
			idx:       1,
			wantFirst: 0,
			wantLast:  255,
		},
		{
			idx:       2,
			wantFirst: 0,
			wantLast:  127,
		},
		{
			idx:       3,
			wantFirst: 128,
			wantLast:  255,
		},
		{
			idx:       4,
			wantFirst: 0,
			wantLast:  63,
		},
		{
			idx:       8,
			wantFirst: 0,
			wantLast:  31,
		},
		{
			idx:       81,
			wantFirst: 68,
			wantLast:  71,
		},
		{
			idx:       510,
			wantFirst: 254,
			wantLast:  254,
		},
		{
			idx:       511,
			wantFirst: 255,
			wantLast:  255,
		},
	}

	for _, tc := range testCases {
		gotFirst, gotLast := IdxToRange(tc.idx)
		if gotFirst != tc.wantFirst || gotLast != tc.wantLast {
			t.Errorf("IdxToRange(%d), want: (%d, %d), got: (%d, %d)",
				tc.idx, tc.wantFirst, tc.wantLast, gotFirst, gotLast)
		}
	}
}
