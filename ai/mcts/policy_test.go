package mcts

import (
	"log"
	"testing"

	"github.com/nelhage/taktician/bitboard"
	"github.com/nelhage/taktician/tak"
	"github.com/nelhage/taktician/taktest"
)

func TestFindPlaceWins(t *testing.T) {
	cases := []struct {
		board string
		x, y  int8
	}{
		{`
. . . .
. . W B
. B W W
B . . W
`, 2, 0},
		{`
W B B .
. W W B
. B W W
B . . W
`, 0, 1},
		{`
W B B .
W B B B
W . W .
B . W .
`, 1, 2},
	}
	for n, tc := range cases {
		board, err := taktest.Board(tc.board, tak.White)
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
