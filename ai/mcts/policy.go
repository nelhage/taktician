package mcts

import (
	"context"
	"fmt"

	"github.com/nelhage/taktician/ai"
	"github.com/nelhage/taktician/tak"
)

type builder func(*MCTSConfig) Policy

var policyMap map[string]builder

func init() {
	policyMap = make(map[string]builder)
	policyMap[""] = buildUniform
	policyMap["uniform"] = buildUniform
}

func (mc *MonteCarloAI) buildPolicy() Policy {
	builder := policyMap[mc.cfg.Policy]
	if builder == nil {
		panic(fmt.Sprintf("no such policy: %s", mc.cfg.Policy))
	}
	return builder(&mc.cfg)
}

type UniformRandom struct {
	alloc *tak.Position
}

func buildUniform(cfg *MCTSConfig) Policy {
	return &UniformRandom{
		alloc: tak.Alloc(cfg.Size),
	}
}

func (u *UniformRandom) Select(ctx context.Context, m *MonteCarloAI, p *tak.Position) *tak.Position {
	moves := p.AllMoves(nil)
	var next *tak.Position
	for {
		r := m.r.Int31n(int32(len(moves)))
		m := moves[r]
		var e error
		if next, e = p.MovePreallocated(m, u.alloc); e == nil {
			break
		}
		moves[0], moves[r] = moves[r], moves[0]
		moves = moves[1:]
	}
	u.alloc = p
	return next
}

func EvalWeightedPolicy(ctx context.Context,
	mc *MonteCarloAI,
	p *tak.Position, alloc *tak.Position) *tak.Position {
	var buf [500]tak.Move
	moves := p.AllMoves(buf[:])
	var best tak.Move
	var sum int64
	base := mc.eval(&mc.c, p) - 500
	for _, m := range moves {
		child, e := p.MovePreallocated(m, alloc)
		if e != nil {
			continue
		}
		w := mc.eval(&mc.c, child)
		if w > ai.WinThreshold {
			return child
		}
		w -= base
		if w <= 0 {
			w = 1
		}
		sum += w
		if mc.r.Int63n(sum) < w {
			best = m
		}
	}
	next, _ := p.MovePreallocated(best, alloc)
	return next
}
