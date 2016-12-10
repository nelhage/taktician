package canonicalize

import (
	"strings"
	"testing"

	"github.com/nelhage/taktician/ptn"
	"github.com/nelhage/taktician/tak"
)

func TestCanonical(t *testing.T) {
	cases := []struct {
		size    int
		in, out string
	}{
		{5, "a1", "a1"},
		{5, "a5", "a1"},
		{5, "e5", "a1"},
		{5, "e1", "a1"},

		{5, "a1 a5", "a1 e1"},
		{5, "a5 e5", "a1 e1"},
		{5, "e5 e1", "a1 e1"},
		{5, "e1 a1", "a1 e1"},

		{5, "a5 a1", "a1 e1"},
		{5, "a1 e1", "a1 e1"},
		{5, "e1 e5", "a1 e1"},
		{5, "e5 a5", "a1 e1"},

		{5, "e5 a1", "a1 e5"},
		{5, "a1 e5", "a1 e5"},

		{5, "a5 e1", "a1 e5"},
		{5, "e1 a5", "a1 e5"},

		{5, "a1 e5 b4", "a1 e5 d2"},

		{5, "a1 a5 e5 e1 c4 b4", "a1 e1 e5 a5 d3 d2"},

		{5, "b1 a1", "b1 a1"},
		{5, "a2 a1", "b1 a1"},
		{5, "d1 e1", "b1 a1"},
		{5, "e2 e1", "b1 a1"},
		{5, "d5 e5", "b1 a1"},
		{5, "e4 e5", "b1 a1"},
		{5, "a4 a5", "b1 a1"},
		{5, "b5 a5", "b1 a1"},

		{6,
			"a2 b2 e2 f2 e5 f5 b5 a5 c4 b6 d4 e6 d3 e1 c3 b1",
			"b1 b2 b5 b6 e2 e1 e5 e6 c3 a2 c4 a5 d3 f2 d4 f5",
		},

		{6,
			"a2 b2 e2 f2 e5 f5 b5 a5 c4 b6 d4 e6 d3 e1 c3 b1 c1",
			"b1 b2 b5 b6 e2 e1 e5 e6 c3 a2 c4 a5 d3 f2 d4 f5 c1",
		},
		{6,
			"a2 b2 e2 f2 e5 f5 b5 a5 c4 b6 d4 e6 d3 e1 c3 b1 d1",
			"b1 b2 b5 b6 e2 e1 e5 e6 c3 a2 c4 a5 d3 f2 d4 f5 c1",
		},
		{6,
			"a2 b2 e2 f2 e5 f5 b5 a5 c4 b6 d4 e6 d3 e1 c3 b1 f3",
			"b1 b2 b5 b6 e2 e1 e5 e6 c3 a2 c4 a5 d3 f2 d4 f5 c1",
		},
		{6,
			"a2 b2 e2 f2 e5 f5 b5 a5 c4 b6 d4 e6 d3 e1 c3 b1 f4",
			"b1 b2 b5 b6 e2 e1 e5 e6 c3 a2 c4 a5 d3 f2 d4 f5 c1",
		},
		{6,
			"a2 b2 e2 f2 e5 f5 b5 a5 c4 b6 d4 e6 d3 e1 c3 b1 c6",
			"b1 b2 b5 b6 e2 e1 e5 e6 c3 a2 c4 a5 d3 f2 d4 f5 c1",
		},
		{6,
			"a2 b2 e2 f2 e5 f5 b5 a5 c4 b6 d4 e6 d3 e1 c3 b1 d6",
			"b1 b2 b5 b6 e2 e1 e5 e6 c3 a2 c4 a5 d3 f2 d4 f5 c1",
		},
		{6,
			"a2 b2 e2 f2 e5 f5 b5 a5 c4 b6 d4 e6 d3 e1 c3 b1 a3",
			"b1 b2 b5 b6 e2 e1 e5 e6 c3 a2 c4 a5 d3 f2 d4 f5 c1",
		},
		{6,
			"a2 b2 e2 f2 e5 f5 b5 a5 c4 b6 d4 e6 d3 e1 c3 b1 a4",
			"b1 b2 b5 b6 e2 e1 e5 e6 c3 a2 c4 a5 d3 f2 d4 f5 c1",
		},
	}
	for _, tc := range cases {
		tc := tc
		t.Run(tc.in, func(t *testing.T) {
			bits := strings.Split(tc.in, " ")
			var ms []tak.Move
			for _, b := range bits {
				m, e := ptn.ParseMove(b)
				if e != nil {
					t.Fatalf("Parse %s: %v", b, e)
				}
				ms = append(ms, m)
			}
			out, _ := Canonical(tc.size, ms)
			bits = nil
			for _, o := range out {
				bits = append(bits, ptn.FormatMove(o))
			}
			got := strings.Join(bits, " ")
			if got != tc.out {
				t.Fatalf("Canonical(%q) = %q != %q",
					tc.in, got, tc.out,
				)
			}
		})
	}
}

func TestRotations(t *testing.T) {
	p := tak.New(tak.Config{Size: 6})
	ss, e := Symmetries(p)
	if e != nil {
		t.Fatal(e)
	}
	if len(ss) != 1 {
		t.Fatal("bad symmetries ", len(ss))
	}

	p, _ = p.Move(tak.Move{Type: tak.PlaceFlat, X: 0, Y: 0})
	ss, e = Symmetries(p)
	if e != nil {
		t.Fatal(e)
	}
	if len(ss) != 4 {
		t.Error("bad symmetries n=", len(ss))
	}
}
