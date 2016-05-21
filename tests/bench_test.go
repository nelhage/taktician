package tests

import (
	"testing"

	"github.com/nelhage/taktician/ptn"
	"github.com/nelhage/taktician/tak"
)

func BenchmarkMoveEmpty(b *testing.B) {
	p := tak.New(tak.Config{Size: 5})
	n := tak.New(tak.Config{Size: 5})
	ms := p.AllMoves(nil)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		for {
			_, e := p.MoveToAllocated(&ms[i%len(ms)], n)
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
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		j := i
		for {
			_, e := p.MoveToAllocated(&ms[j%len(ms)], n)
			if e == nil {
				break
			}
			j++
		}
	}
}
