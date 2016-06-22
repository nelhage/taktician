package ptn

import (
	"bytes"
	"reflect"
	"strings"
	"testing"

	"github.com/nelhage/taktician/tak"
)

const testGame = `
[Event "PTN Viewer Demo"]
[Site "Here"]
[Date "2015.11.21"]
[Player1 "No One"]
[Player2 "N/A"]
[Round "342"]
[Result "It Works!"]
[Size "5"]
[TPS "x5/x3,2112S,x/x5/x,1221,x3/x5 1 1"]

1. a3 c2
2. c2> {What a nub} a3+
3. d2+ a4>
4. d3- b4-
5. d2< Cc5? {Can you even believe this guy?}
6. c2+ b3>'
7. a5 2c3-2!
`

func TestParsePTN(t *testing.T) {
	ptn, err := ParsePTN(bytes.NewBufferString(testGame))
	if err != nil {
		t.Fatal("parse:", err)
	}
	if ptn == nil {
		t.Fatal("nil ptn")
	}
	if !reflect.DeepEqual(ptn.Tags, []Tag{
		{"Event", "PTN Viewer Demo"},
		{"Site", "Here"},
		{"Date", "2015.11.21"},
		{"Player1", "No One"},
		{"Player2", "N/A"},
		{"Round", "342"},
		{"Result", "It Works!"},
		{"Size", "5"},
		{"TPS", "x5/x3,2112S,x/x5/x,1221,x3/x5 1 1"},
	}) {
		t.Fatal("tags", ptn.Tags)
	}

	ops := []Op{
		&MoveNumber{opCommon: opCommon{src: "1."}, Number: 1},
		&Move{opCommon: opCommon{src: "a3"}},
		&Move{opCommon: opCommon{src: "c2"}},
		&MoveNumber{opCommon: opCommon{src: "2."}, Number: 2},
		&Move{opCommon: opCommon{src: "c2>"}},
		&Comment{opCommon: opCommon{src: "{What a nub}"}, Comment: "What a nub"},
		&Move{opCommon: opCommon{src: "a3+"}},
		&MoveNumber{opCommon: opCommon{src: "3."}, Number: 3},
		&Move{opCommon: opCommon{src: "d2+"}},
		&Move{opCommon: opCommon{src: "a4>"}},
		&MoveNumber{opCommon: opCommon{src: "4."}, Number: 4},
		&Move{opCommon: opCommon{src: "d3-"}},
		&Move{opCommon: opCommon{src: "b4-"}},
		&MoveNumber{opCommon: opCommon{src: "5."}, Number: 5},
		&Move{opCommon: opCommon{src: "d2<"}},
		&Move{opCommon: opCommon{src: "Cc5?"}, Modifiers: "?"},
		&Comment{opCommon: opCommon{src: "{Can you even believe this guy?}"}, Comment: "Can you even believe this guy?"},
		&MoveNumber{opCommon: opCommon{src: "6."}, Number: 6},
		&Move{opCommon: opCommon{src: "c2+"}},
		&Move{opCommon: opCommon{src: "b3>'"}, Modifiers: "'"},
		&MoveNumber{opCommon: opCommon{src: "7."}, Number: 7},
		&Move{opCommon: opCommon{src: "a5"}},
		&Move{opCommon: opCommon{src: "2c3-2!"}, Modifiers: "!"},
	}
	for i, o := range ops {
		if m, ok := o.(*Move); ok {
			mm, e := ParseMove(strings.TrimRight(m.src, "?!'"))
			if e != nil {
				panic(e)
			}
			m.Move = mm
		}
		if !reflect.DeepEqual(ops[i], ptn.Ops[i]) {
			t.Errorf("[%d] %#v != %#v", i, ptn.Ops[i], ops[i])
		}
	}
}

func TestParsePTNBOM(t *testing.T) {
	var buf bytes.Buffer
	buf.WriteRune(0xFEFF)
	buf.WriteString(testGame)
	_, err := ParsePTN(&buf)
	if err != nil {
		t.Fatal("failed to parse PTN with BOM")
	}
}

func TestRoundTripPTN(t *testing.T) {
	ptn, err := ParsePTN(bytes.NewBufferString(testGame))
	if err != nil {
		t.Fatal("parse")
	}
	render := ptn.Render()
	back, err := ParsePTN(bytes.NewBufferString(render))
	if err != nil {
		t.Fatal("parse round-tripped")
	}
	if !reflect.DeepEqual(back.Tags, ptn.Tags) {
		t.Fatal("tags did not round-trip")
	}
	for _, o := range ptn.Ops {
		o.(Op).clearSrc()
	}
	for _, o := range back.Ops {
		o.(Op).clearSrc()
	}
	if !reflect.DeepEqual(ptn.Ops, back.Ops) {
		t.Fatalf("different ops! in=%#v, out=%#v",
			ptn.Ops, back.Ops,
		)
	}
}

const emptyPTN = `
[Size "8"]
[Date "2016-05-02"]
[Player1 "applemonkeyman"]
[Player2 "KingSultan"]
[Result "1-0"]

`

func TestEmpty(t *testing.T) {
	ptn, err := ParsePTN(bytes.NewBufferString(emptyPTN))
	if err != nil {
		t.Fatal("parse empty", err)
	}
	if len(ptn.Ops) != 0 {
		t.Fatal("ops", ptn.Ops)
	}
	if len(ptn.Tags) != 5 {
		t.Fatal("tags", ptn.Tags)
	}
}

func TestPositionAtMove(t *testing.T) {
	src := `[Size "5"]
[Date "2016-05-07"]
[Time "2016-05-07T10:17:03Z"]
[Player1 "Guest369"]
[Player2 "TakticianBot"]
[Result "0-R"]
[Id "1334"]


1. a1 e1
2. c3 b3
3. b4 a3
4. b2 d3
5. b1 c2
6. c3< a3>
7. b4- Sc3
8. b5 c3<
9. Cc3 4b3-22
10. a2 c5
11. a2> 3b1+
12. b1 4b2-
13. b4 a2
14. c3- a5
15. 2c2< a3
16. Sa4 c4
17. Sc3 d1
18. b2- 2b2+11
0-R`
	cases := []struct {
		move  int
		color tak.Color
		tps   string
	}{
		{1, tak.White, "x5/x5/x5/x5/x5 1 1"},
		{1, tak.Black, "x5/x5/x5/x5/2,x4 2 1"},
		{1, tak.NoColor, ""},
		{18, tak.Black, "2,1,2,x2/1S,1,2,x2/2,2,1S,2,x/2,1122,x3/2,111121C,x,2,1 2 18"},
		{0, tak.NoColor, "2,1,2,x2/1S,12,2,x2/2,22,1S,2,x/2,11,x3/2,111121C,x,2,1 1 19"},
	}
	p, e := ParsePTN(bytes.NewBufferString(src))
	if e != nil {
		panic(e)
	}
	for _, tc := range cases {
		pos, e := p.PositionAtMove(tc.move, tc.color)
		if tc.tps == "" {
			if e == nil {
				t.Errorf("AtMove(%d, %s) did not return error", tc.move, tc.color)
			}
			continue
		}

		if e != nil {
			t.Errorf("AtMove(%d, %s): %v", tc.move, tc.color, e)
			continue
		}
		tps := FormatTPS(pos)
		if tps != tc.tps {
			t.Errorf("AtMove(%d, %s) =\n   %s\n!= %s",
				tc.move, tc.color, tps, tc.tps,
			)
		}
	}

}
