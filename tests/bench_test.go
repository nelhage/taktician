package tests

import (
	"testing"

	"github.com/nelhage/taktician/ptn"
	"github.com/nelhage/taktician/tak"
)

func BenchmarkMoveEmpty(b *testing.B) {
	p := tak.New(tak.Config{Size: 5})
	ms := p.AllMoves()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		for {
			_, e := p.Move(&ms[i%len(ms)])
			if e == nil {
				break
			}
		}
	}
}

func BenchmarkMoveComplex(b *testing.B) {
	p, e := ptn.ParseTPS("112S,12,1112S,x2/x2,121C,12S,x/1,21,2,2,2/x,2,1,1,1/2,x3,21 2 24")
	if e != nil {
		panic("bad tps")
	}
	ms := p.AllMoves()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		j := i
		for {
			_, e := p.Move(&ms[j%len(ms)])
			if e == nil {
				break
			}
			j++
		}
	}
}