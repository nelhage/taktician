package playtak

import (
	"reflect"
	"testing"

	"nelhage.com/tak/tak"
)

func TestParseServer(t *testing.T) {
	cases := []struct {
		in  string
		out tak.Move
	}{
		{
			"P A1",
			tak.Move{
				X: 0, Y: 0, Type: tak.PlaceFlat,
			},
		},
		{
			"P H8 C",
			tak.Move{
				X: 7, Y: 7, Type: tak.PlaceCapstone,
			},
		},
		{
			"P C1 W",
			tak.Move{
				X: 2, Y: 0, Type: tak.PlaceStanding,
			},
		},
		{
			"M C1 C3 4 1",
			tak.Move{
				X: 2, Y: 0, Type: tak.SlideUp,
				Slides: []byte{4, 1},
			},
		},
		{
			"M D2 E2 1",
			tak.Move{
				X: 3, Y: 1, Type: tak.SlideRight,
				Slides: []byte{1},
			},
		},
		{
			"M D4 D1 1 1 1",
			tak.Move{
				X: 3, Y: 3, Type: tak.SlideDown,
				Slides: []byte{1, 1, 1},
			},
		},
		{
			"M D4 A4 3 1 1",
			tak.Move{
				X: 3, Y: 3, Type: tak.SlideLeft,
				Slides: []byte{3, 1, 1},
			},
		},
	}
	for _, tc := range cases {
		m, e := ParseServer(tc.in)
		if e != nil {
			t.Errorf("parse(%s): %v", tc.in, e)
			continue
		}
		if !reflect.DeepEqual(m, tc.out) {
			t.Errorf("parse(%s) = %#v not %#v", tc.in, m, tc.out)
		}
		back := FormatServer(&m)
		if back != tc.in {
			t.Errorf("round-trip(%s) = %s", tc.in, back)
		}
	}
}
