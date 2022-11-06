package ai

import (
	"math/rand"

	"context"

	"github.com/nelhage/taktician/tak"
)

type RandomAI struct {
	r *rand.Rand
}

func (r *RandomAI) GetMove(ctx context.Context, p *tak.Position) tak.Move {
	var buffer [100]tak.Move
	moves := p.AllMoves(buffer[:0])
	i := r.r.Int31n(int32(len(moves)))
	return moves[i]
}

func NewRandom(seed int64) TakPlayer {
	return &RandomAI{
		r: rand.New(rand.NewSource(seed)),
	}
}
