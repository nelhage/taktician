package ai

import "nelhage.com/tak/tak"

const (
	maxEval int64 = 1 << 30
	minEval       = -maxEval
)

type MinimaxAI struct {
	depth int
}

func (m *MinimaxAI) GetMove(p *tak.Position) *tak.Move {
	var move *tak.Move
	for i := 1; i <= m.depth; i++ {
		move, _ = m.minimax(p, i, move, minEval-1, maxEval+1)
	}
	return move
}

func (ai *MinimaxAI) minimax(
	p *tak.Position,
	depth int,
	pv *tak.Move,
	α, β int64) (*tak.Move, int64) {
	if depth == 0 {
		return nil, ai.evaluate(p)
	}
	var best tak.Move
	max := minEval - 1
	moves := p.AllMoves()
	if pv != nil {
		for i, m := range moves {
			if m.Equal(pv) {
				moves[0], moves[i] = moves[i], moves[0]
			}
		}
	}
	for _, m := range moves {
		child, e := p.Move(m)
		if e != nil {
			continue
		}
		_, v := ai.minimax(child, depth-1, nil, -β, -α)
		v = -v
		if v > max {
			max = v
			best = m
		}
		if v > α {
			α = v
			if α > β {
				break
			}
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
		switch winner {
		case tak.NoColor:
			return 0
		case p.ToMove():
			return maxEval
		default:
			return minEval
		}
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
