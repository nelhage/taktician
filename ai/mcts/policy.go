package mcts

import (
	"golang.org/x/net/context"

	"github.com/nelhage/taktician/ai"
	"github.com/nelhage/taktician/tak"
)

func UniformRandomPolicy(ctx context.Context,
	m *MonteCarloAI,
	p *tak.Position, alloc *tak.Position) *tak.Position {
	moves := p.AllMoves(nil)
	var next *tak.Position
	for {
		r := m.r.Int31n(int32(len(moves)))
		m := moves[r]
		var e error
		if next, e = p.MovePreallocated(&m, alloc); e == nil {
			break
		}
		moves[0], moves[r] = moves[r], moves[0]
		moves = moves[1:]
	}
	return next
}

func NewMinimaxPolicy(cfg *MCTSConfig, depth int) PolicyFunc {
	mm := ai.NewMinimax(ai.MinimaxConfig{
		Size:    cfg.Size,
		NoTable: true,
		Depth:   depth,
		Seed:    cfg.Seed,
	})
	return func(ctx context.Context,
		m *MonteCarloAI,
		p *tak.Position, next *tak.Position) *tak.Position {
		move := mm.GetMove(ctx, p)
		next, _ = p.MovePreallocated(&move, next)
		return next
	}
}

func EvalWeightedPolicy(ctx context.Context,
	mc *MonteCarloAI,
	p *tak.Position, alloc *tak.Position) *tak.Position {
	var buf [500]tak.Move
	moves := p.AllMoves(buf[:])
	var best tak.Move
	var sum int64
	for _, m := range moves {
		child, e := p.MovePreallocated(&m, alloc)
		if e != nil {
			continue
		}
		w := mc.eval(mc.mm, child)
		if w > ai.WinThreshold {
			return child
		}
		w += 1000
		if w <= 0 {
			w = 1
		}
		sum += w
		if mc.r.Int63n(sum) < w {
			best = m
		}
	}
	next, _ := p.MovePreallocated(&best, alloc)
	return next
}
