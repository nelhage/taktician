package bitboard

import (
	"strconv"
	"testing"
)

func TestPrecompute(t *testing.T) {
	c := Precompute(5)
	if c.B != (1<<5)-1 {
		t.Error("c.b(5):", strconv.FormatUint(c.B, 2))
	}
	if c.T != ((1<<5)-1)<<(4*5) {
		t.Error("c.t(5):", strconv.FormatUint(c.T, 2))
	}
	if c.R != 0x0108421 {
		t.Error("c.r(5):", strconv.FormatUint(c.R, 2))
	}
	if c.L != 0x1084210 {
		t.Error("c.l(5):", strconv.FormatUint(c.L, 2))
	}
	if c.Mask != 0x1ffffff {
		t.Error("c.mask(5):", strconv.FormatUint(c.Mask, 2))
	}

	c = Precompute(8)
	if c.B != (1<<8)-1 {
		t.Error("c.b(8):", strconv.FormatUint(c.B, 2))
	}
	if c.T != ((1<<8)-1)<<(7*8) {
		t.Error("c.t(8):", strconv.FormatUint(c.T, 2))
	}
	if c.R != 0x101010101010101 {
		t.Error("c.r(8):", strconv.FormatUint(c.R, 2))
	}
	if c.L != 0x8080808080808080 {
		t.Error("c.l(8):", strconv.FormatUint(c.L, 2))
	}
	if c.Mask != ^uint64(0) {
		t.Error("c.mask(8):", strconv.FormatUint(c.Mask, 2))
	}
}

func TestFlood(t *testing.T) {
	cases := []struct {
		size  uint
		bound uint64
		seed  uint64
		out   uint64
	}{
		{
			5,
			0x108423c,
			0x4,
			0x108421c,
		},
	}
	for _, tc := range cases {
		c := Precompute(tc.size)
		got := Flood(&c, tc.bound, tc.seed)
		if got != tc.out {
			t.Errorf("Flood[%d](%s, %s)=%s !=%s",
				tc.size,
				strconv.FormatUint(tc.bound, 2),
				strconv.FormatUint(tc.seed, 2),
				strconv.FormatUint(got, 2),
				strconv.FormatUint(tc.out, 2))
		}
	}
}

func TestDimensions(t *testing.T) {
	cases := []struct {
		size uint
		bits uint64
		w    int
		h    int
	}{
		{5, 0x108421c, 3, 5},
		{5, 0, 0, 0},
		{5, 0x843800, 3, 3},
		{5, 0x08000, 1, 1},
	}
	for _, tc := range cases {
		c := Precompute(tc.size)
		w, h := Dimensions(&c, tc.bits)
		if w != tc.w || h != tc.h {
			t.Errorf("Dimensions(%d, %x) = (%d,%d) != (%d,%d)",
				tc.size, tc.bits, w, h, tc.w, tc.h,
			)
		}
	}

}
