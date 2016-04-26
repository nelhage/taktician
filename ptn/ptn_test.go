package ptn

import (
	"bytes"
	"reflect"
	"strings"
	"testing"
)

func TestParsePTN(t *testing.T) {
	text := `
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
	ptn, err := ParsePTN(bytes.NewBufferString(text))
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
			m.Move = *mm
		}
		if !reflect.DeepEqual(ops[i], ptn.Ops[i]) {
			t.Errorf("[%d] %#v != %#v", i, ptn.Ops[i], ops[i])
		}
	}
}
