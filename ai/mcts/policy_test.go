package mcts

import (
	"log"
	"testing"

	"github.com/nelhage/taktician/bitboard"
	"github.com/nelhage/taktician/ptn"
	"github.com/nelhage/taktician/tak"
)

func TestFindPlaceWins(t *testing.T) {
	cases := []struct {
		tps  string
		x, y int8
	}{
		{"x4/x2,1,2/x,2,1,1/2,x2,1 1 4", 2, 3},
	}
	for n, tc := range cases {
		board, err := ptn.ParseTPS(tc.tps)
		if err != nil {
			t.Errorf("%d: %v", n, err)
			continue
		}
		log.Printf("to move=%s", board.ToMove())
		c := bitboard.Precompute(uint(board.Size()))
		mv := placeWinMove(&c, board)
		if mv.Type != tak.PlaceFlat {
			t.Errorf("%d: bad move: type=%s", n, mv.Type)
			continue
		}
		if mv.X != tc.x || mv.Y != tc.y {
			t.Errorf("%d: bad move: (%d, %d) != (%d, %d)",
				n, mv.X, mv.Y, tc.x, tc.y)
		}

	}
}
