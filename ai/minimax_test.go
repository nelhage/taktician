package ai

import (
	"flag"
	"testing"

	"github.com/nelhage/taktician/ptn"
	"github.com/nelhage/taktician/tak"
)

var size = flag.Int("size", 5, "board size to benchmark")
var depth = flag.Int("depth", 4, "minimax search depth")

func BenchmarkMinimax(b *testing.B) {
	var cfg = tak.Config{Size: *size}
	p := tak.New(cfg)
	p, _ = p.Move(&tak.Move{X: 0, Y: 0, Type: tak.PlaceFlat})
	p, _ = p.Move(&tak.Move{X: *size - 1, Y: *size - 1, Type: tak.PlaceFlat})
	ai := NewMinimax(*size, *depth)

	for i := 0; i < b.N; i++ {
		var e error
		m := ai.GetMove(p)
		p, e = p.Move(&m)
		if e != nil {
			b.Fatal("bad move", e)
		}
		if over, _ := p.GameOver(); over {
			p = tak.New(cfg)
			p, _ = p.Move(&tak.Move{X: 0, Y: 0, Type: tak.PlaceFlat})
			p, _ = p.Move(&tak.Move{X: *size - 1, Y: *size - 1, Type: tak.PlaceFlat})
		}
	}
}

func TestRegression(t *testing.T) {
	game, err := ptn.ParseTPS(
		`2,x4/x2,2,x2/x,2,2,x2/x2,12,2,1/1,1,21,2,1 1 9`,
	)
	if err != nil {
		panic(err)
	}
	ai := NewMinimax(game.Size(), 3)
	m := ai.GetMove(game)
	_, e := game.Move(&m)
	if e != nil {
		t.Fatalf("ai returned illegal move: %s: %s", ptn.FormatMove(&m), e)
	}
}
