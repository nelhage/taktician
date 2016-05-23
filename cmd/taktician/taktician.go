package main

import (
	"time"

	"github.com/nelhage/taktician/ai"
	"github.com/nelhage/taktician/tak"
)

func timeBound(remaining time.Duration) time.Duration {
	return *limit
}

type Taktician struct {
	ai *ai.MinimaxAI
}

func (t *Taktician) NewGame(g *Game) {
	t.ai = ai.NewMinimax(ai.MinimaxConfig{
		Size:  g.size,
		Depth: *depth,
		Debug: *debug,

		NoSort:  !*sort,
		NoTable: !*table,
	})
}

func (t *Taktician) GetMove(p *tak.Position, mine, theirs time.Duration) tak.Move {
	return t.ai.GetMove(p, timeBound(mine))
}

func (t *Taktician) GameOver()                 {}
func (t *Taktician) HandleChat(string, string) {}
