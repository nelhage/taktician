package ai

import (
	"flag"
	"testing"

	"nelhage.com/tak/tak"
)

var size = flag.Int("size", 5, "board size to benchmark")
var depth = flag.Int("depth", 4, "minimax search depth")

func BenchmarkMinimax(b *testing.B) {
	var cfg = tak.Config{Size: *size}
	p := tak.New(cfg)
	ai := NewMinimax(*depth)
	for i := 0; i < b.N; i++ {
		var e error
		p, e = p.Move(*ai.GetMove(p))
		if e != nil {
			b.Fatal("bad move", e)
		}
		if over, _ := p.GameOver(); over {
			p = tak.New(cfg)
		}
	}
}
