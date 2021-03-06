package ptn

import (
	"reflect"
	"testing"

	"github.com/nelhage/taktician/tak"
)

func TestParseMove(t *testing.T) {
	cases := []struct {
		in   string
		out  tak.Move
		str  string
		long string
	}{
		{
			"a1",
			tak.Move{X: 0, Y: 0, Type: tak.PlaceFlat},
			"a1",
			"Fa1",
		},
		{
			"Sa4",
			tak.Move{X: 0, Y: 3, Type: tak.PlaceStanding},
			"Sa4",
			"Sa4",
		},
		{
			"Ch7",
			tak.Move{X: 7, Y: 6, Type: tak.PlaceCapstone},
			"Ch7",
			"Ch7",
		},
		{
			"Fh7",
			tak.Move{X: 7, Y: 6, Type: tak.PlaceFlat},
			"h7",
			"Fh7",
		},
		{
			"a1>",
			tak.Move{X: 0, Y: 0, Type: tak.SlideRight, Slides: tak.MkSlides(1)},
			"a1>",
			"1a1>1",
		},
		{
			"2a2<",
			tak.Move{X: 0, Y: 1, Type: tak.SlideLeft, Slides: tak.MkSlides(2)},
			"2a2<",
			"2a2<2",
		},
		{
			"3a1+111",
			tak.Move{X: 0, Y: 0, Type: tak.SlideUp, Slides: tak.MkSlides(1, 1, 1)},
			"3a1+111",
			"3a1+111",
		},
		{
			"5d4-22",
			tak.Move{X: 3, Y: 3, Type: tak.SlideDown, Slides: tak.MkSlides(2, 2, 1)},
			"5d4-221",
			"5d4-221",
		},
		{
			"a1?",
			tak.Move{X: 0, Y: 0, Type: tak.PlaceFlat},
			"a1",
			"Fa1",
		},
		{
			"Ch7!",
			tak.Move{X: 7, Y: 6, Type: tak.PlaceCapstone},
			"Ch7",
			"Ch7",
		},
		{
			"b1>*'",
			tak.Move{X: 1, Y: 0, Type: tak.SlideRight, Slides: tak.MkSlides(1)},
			"b1>",
			"1b1>1",
		},
		{
			"2a2<*",
			tak.Move{X: 0, Y: 1, Type: tak.SlideLeft, Slides: tak.MkSlides(2)},
			"2a2<",
			"2a2<2",
		},
		{
			"3a1+111''!",
			tak.Move{X: 0, Y: 0, Type: tak.SlideUp, Slides: tak.MkSlides(1, 1, 1)},
			"3a1+111",
			"3a1+111",
		},
	}
	for _, tc := range cases {
		get, err := ParseMove(tc.in)
		if err != nil {
			t.Errorf("ParseMove(%s): err=%v", tc.in, err)
			continue
		}
		if !reflect.DeepEqual(get, tc.out) {
			t.Errorf("ParseMove(%s)=%#v not %#v", tc.in, get, tc.out)
		}
		rt := FormatMove(tc.out)
		if rt != tc.str {
			t.Errorf("FormatMove(%s)=%s not %s", tc.in, rt, tc.str)
		}
		long := FormatMoveLong(tc.out)
		if long != tc.long {
			t.Errorf("FormatMoveLong(%s)=%s not %s", tc.in, long, tc.long)
		}
	}
}

func TestParseMoveErrors(t *testing.T) {
	bad := []string{
		"",
		"a11",
		"z3",
		"14c4>",
		"6a1",
		"6a1>2222",
		"a",
		"3a",
	}
	for _, b := range bad {
		_, e := ParseMove(b)
		if e == nil {
			t.Errorf("parse(%q): no error", b)
		}
	}
}

func BenchmarkParseMove(b *testing.B) {
	for i := 0; i < b.N; i++ {
		ParseMove("3a1+111")
	}
}
