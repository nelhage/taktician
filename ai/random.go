package ai

import (
	"math/rand"

	"nelhage.com/tak/game"
)

type RandomAI struct {
	r *rand.Rand
}

func (r *RandomAI) GetMove(p *game.Position) *game.Move {
	moves := p.AllMoves()
	i := r.r.Int31n(int32(len(moves)))
	return &moves[i]
}

func NewRandom(seed int64) TakPlayer {
	return &RandomAI{
		r: rand.New(rand.NewSource(seed)),
	}
}
