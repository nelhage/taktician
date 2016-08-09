package ptn

import (
	"bytes"
	"testing"

	"github.com/nelhage/taktician/tak"
)

func TestIterator(t *testing.T) {
	type step struct {
		ply     int
		ptnMove int
		color   tak.Color
	}
	cases := []struct {
		ptn   string
		iters []step
	}{
		{
			`
[Size "5"]
[TPS "2,x2,121C,1/x2,2,12,1/x2,2,12S,2/x3,1,1/x4,1 1 2"]

1.
`,
			[]step{
				{2, 1, tak.White},
			},
		},
		{`
[Size "5"]

1. a1 e5
2. e1 a2
`,
			[]step{
				{0, 1, tak.White},
				{1, 1, tak.Black},
				{2, 2, tak.White},
				{3, 2, tak.Black},
				{4, 2, tak.White},
			},
		},
		{`
[Size "5"]

`,
			[]step{
				{0, 0, tak.White},
			},
		},
	}
	for i, tc := range cases {
		ptn, e := ParsePTN(bytes.NewBufferString(tc.ptn))
		if e != nil {
			t.Errorf("[%d] %v", i, e)
			continue
		}
		it := ptn.Iterator()
		ct := 0
		for it.Next() {
			if ct >= len(tc.iters) {
				t.Errorf("[%d] too many results ply=%d",
					i, it.Position().MoveNumber())
				break
			}
			expect := tc.iters[ct]
			ct++
			if c := it.Position().ToMove(); c != expect.color {
				t.Errorf("[%d] .%d: wrong color %s != %s",
					i, ct, c, expect.color,
				)
			}
			if m := it.PTNMove(); m != expect.ptnMove {
				t.Errorf("[%d] .%d: wrong PTN %d != %d",
					i, ct, m, expect.ptnMove,
				)
			}
			if ply := it.Position().MoveNumber(); ply != expect.ply {
				t.Errorf("[%d] .%d: wrong ply %d != %d",
					i, ct, ply, expect.ply,
				)
			}
		}
		if ct < len(tc.iters) {
			t.Errorf("[%d] too few results %d < %d", i, ct, len(tc.iters))
		}
	}
}
