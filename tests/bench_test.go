package tests

import (
	"flag"
	"testing"

	"github.com/nelhage/taktician/ai"
	"github.com/nelhage/taktician/ptn"
	"github.com/nelhage/taktician/tak"
)

var seed = flag.Int64("seed", 4, "random seed")

func BenchmarkMoveEmpty(b *testing.B) {
	p := tak.New(tak.Config{Size: 5})
	n := tak.New(tak.Config{Size: 5})
	ms := p.AllMoves(nil)
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		for {
			_, e := p.MovePreallocated(&ms[i%len(ms)], n)
			if e == nil {
				break
			}
		}
	}
}

func BenchmarkMoveComplex(b *testing.B) {
	p, e := ptn.ParseTPS("112S,12,1112S,x2/x2,121C,12S,x/1,21,2,2,2/x,2,1,1,1/2,x3,21 2 24")
	n := tak.New(tak.Config{Size: 5})
	if e != nil {
		panic("bad tps")
	}
	ms := p.AllMoves(nil)
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		j := i
		for {
			_, e := p.MovePreallocated(&ms[j%len(ms)], n)
			if e == nil {
				break
			}
			j++
		}
	}
}

func BenchmarkPuzzle1(b *testing.B) {
	p, e := ptn.ParseTPS("2,x2,121C,1/x2,2,12,1/x2,2,12S,2/x3,1,1/x4,1 1 2")
	if e != nil {
		panic("bad tps")
	}

	mm := ai.NewMinimax(ai.MinimaxConfig{
		Depth: 7,
		Seed:  *seed,
		Size:  p.Size(),
	})

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		mm.GetMove(p, 0)
	}
}
