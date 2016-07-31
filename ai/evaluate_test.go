package ai

import (
	"testing"

	"github.com/nelhage/taktician/bitboard"
	"github.com/nelhage/taktician/ptn"
)

func TestEvaluateWinner(t *testing.T) {
	cases := []struct {
		tps      string
		min, max int64
	}{
		{
			`x3,2,x/2S,1,2121,2221C,1/1,1,x,111112C,x/1,x,2,2,x/1,1,22111112S,1,2221 2 35`,
			MinEval, -WinThreshold,
		},
		{
			`x4,1/x4,1/x3,2,1/x3,2,1/2,x3,1 1 6`,
			WinThreshold, MaxEval,
		},
		{
			`x4,1/x4,1/x3,2,1/x3,2,x/2,x4 1 4`,
			0, 0,
		},
	}
	c := bitboard.Precompute(5)
	for i, tc := range cases {
		p, e := ptn.ParseTPS(tc.tps)
		if e != nil {
			t.Errorf("%d: tps: %v", i, e)
			continue
		}
		eval := EvaluateWinner(&c, p)
		if eval < tc.min || eval > tc.max {
			t.Errorf("%d: eval=%d (not in [%d,%d])", i, eval, tc.min, tc.max)
		}
	}
}

func benchmarkEval(b *testing.B, tps string) {
	p, e := ptn.ParseTPS(tps)
	if e != nil {
		b.Fatal("tps:", e)
	}
	c := bitboard.Precompute(uint(p.Size()))
	eval := MakeEvaluator(p.Size(), &DefaultWeights[p.Size()])
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		eval(&c, p)
	}
}

func BenchmarkEvalEarlyGame(b *testing.B) {
	benchmarkEval(b, `x5/x3,2,x/x2,1C,1,2/x2,2,1,1/2,x2,2C,1 1 6`)
}

func BenchmarkEvalMidGame(b *testing.B) {
	benchmarkEval(b, `x3,2,x/x4,12/1,1,x,1,21C/x,1,x,12111112C,2/2,x,22121,x,2 2 20`)
}
