package ai

import "nelhage.com/tak/tak"

const (
	maxEval = 1 << 30
	minEval = -maxEval
)

type MinimaxAI struct {
	depth int
}

func (m *MinimaxAI) GetMove(p *tak.Position) *tak.Move {
	move, _ := m.minimax(p, m.depth)
	return move
}

func (ai *MinimaxAI) minimax(p *tak.Position, depth int) (*tak.Move, int64) {
	if depth == 0 {
		return nil, ai.evaluate(p)
	}
	var best tak.Move
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

func imin(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func (m *MinimaxAI) evaluate(p *tak.Position) int64 {
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
			val := 0
			val += imin(x, p.Size()-x-1)
			val += imin(y, p.Size()-y-1)
			if sq[0].Kind() == tak.Flat {
				if sq[0].Color() == p.ToMove() {
					me += val
				} else {
					them += val
				}
			}
		}
	}
	return int64(me - them)
}

func NewMinimax(depth int) TakPlayer {
	return &MinimaxAI{depth}
}
