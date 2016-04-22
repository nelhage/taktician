package ai

import "nelhage.com/tak/game"

const (
	maxEval = 1 << 30
	minEval = -maxEval
)

type MinimaxAI struct {
	depth int
}

func (m *MinimaxAI) GetMove(p *game.Position) *game.Move {
	move, _ := m.minimax(p, m.depth)
	return move
}

func (ai *MinimaxAI) minimax(p *game.Position, depth int) (*game.Move, int64) {
	if depth == 0 {
		return nil, ai.evaluate(p)
	}
	var best game.Move
	var max int64 = minEval
	moves := p.AllMoves()
	for _, m := range moves {
		child, e := p.Move(m)
		if e != nil {
			continue
		}
		_, v := ai.minimax(child, depth-1)
		v = -v
		if v > max {
			max = v
			best = m
		}
	}
	return &best, max
}

func (m *MinimaxAI) evaluate(p *game.Position) int64 {
	if over, winner := p.GameOver(); over {
		if winner == p.ToMove() {
			return maxEval
		}
		return minEval
	}
	me, them := 0, 0
	for x := 0; x < p.Size(); x++ {
		for y := 0; y < p.Size(); y++ {
			sq := p.At(x, y)
			if len(sq) == 0 {
				continue
			}
			if sq[0].Kind() == game.Flat {
				if sq[0].Color() == p.ToMove() {
					me++
				} else {
					them++
				}
			}
		}
	}
	return int64(me - them)
}

func NewMinimax(depth int) TakPlayer {
	return &MinimaxAI{depth}
}
