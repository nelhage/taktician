package ptn

import (
	"reflect"
	"testing"

	"nelhage.com/tak/tak"
)

func TestParseMove(t *testing.T) {
	cases := []struct {
		in  string
		out tak.Move
		str string
	}{
		{
			"a1",
			tak.Move{X: 0, Y: 0, Type: tak.PlaceFlat},
			"a1",
		},
		{
			"Sa4",
			tak.Move{X: 0, Y: 3, Type: tak.PlaceStanding},
			"Sa4",
		},
		{
			"Ch7",
			tak.Move{X: 7, Y: 6, Type: tak.PlaceCapstone},
			"Ch7",
		},
		{
			"Fh7",
			tak.Move{X: 7, Y: 6, Type: tak.PlaceFlat},
			"h7",
		},
		{
			"a1>",
			tak.Move{X: 0, Y: 0, Type: tak.SlideRight, Slides: []byte{1}},
			"a1>",
		},
		{
			"2a2<",
			tak.Move{X: 0, Y: 1, Type: tak.SlideLeft, Slides: []byte{2}},
			"2a2<",
		},
		{
			"3a1+111",
			tak.Move{X: 0, Y: 0, Type: tak.SlideUp, Slides: []byte{1, 1, 1}},
			"3a1+111",
		},
		{
			"5d4-22",
			tak.Move{X: 3, Y: 3, Type: tak.SlideDown, Slides: []byte{2, 2, 1}},
			"5d4-221",
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
		rt := FormatMove(&tc.out)
		if rt != tc.str {
			t.Errorf("FormatMove(%s)=%s not %s", tc.in, rt, tc.str)
		}
	}
}
