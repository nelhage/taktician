package ptn

import (
	"reflect"
	"testing"

	"nelhage.com/tak/game"
)

func TestParseMove(t *testing.T) {
	cases := []struct {
		in  string
		out game.Move
	}{
		{
			"a1",
			game.Move{X: 0, Y: 0, Type: game.PlaceFlat},
		},
		{
			"Sa4",
			game.Move{X: 0, Y: 3, Type: game.PlaceStanding},
		},
		{
			"Ch7",
			game.Move{X: 7, Y: 6, Type: game.PlaceCapstone},
		},
		{
			"Fh7",
			game.Move{X: 7, Y: 6, Type: game.PlaceFlat},
		},
		{
			"a1>",
			game.Move{X: 0, Y: 0, Type: game.SlideRight, Slides: []byte{1}},
		},
		{
			"2a2<",
			game.Move{X: 0, Y: 1, Type: game.SlideLeft, Slides: []byte{2}},
		},
		{
			"3a1+111",
			game.Move{X: 0, Y: 0, Type: game.SlideUp, Slides: []byte{1, 1, 1}},
		},
		{
			"5d4-22",
			game.Move{X: 3, Y: 3, Type: game.SlideDown, Slides: []byte{2, 2, 1}},
		},
	}
	for _, tc := range cases {
		get, err := ParseMove(tc.in)
		if err != nil {
			t.Errorf("ParseMove(%s): err=%v", tc.in, err)
			continue
		}
		if !reflect.DeepEqual(get, &tc.out) {
			t.Errorf("ParseMove(%s)=%#v not %#v", tc.in, get, &tc.out)
		}
	}
}
