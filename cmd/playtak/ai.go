package main

import (
	"math/rand"

	"nelhage.com/tak/game"
)

type randomAI struct {
	r *rand.Rand
}

func (r *randomAI) GetMove(p *game.Position) *game.Move {
	moves := p.AllMoves()
	i := r.r.Int31n(int32(len(moves)))
	return &moves[i]
}
