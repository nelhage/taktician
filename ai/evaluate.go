package ai

import (
	"github.com/nelhage/taktician/bitboard"
	"github.com/nelhage/taktician/tak"
)

const (
	weightFlat       = 200
	weightCaptured   = 100
	weightControlled = 500
	weightCapstone   = -150
	weightThreat     = 150
	weightAdvantage  = 50
)

func (m *MinimaxAI) evaluate(p *tak.Position) int64 {
	if over, winner := p.GameOver(); over {
		switch winner {
		case tak.NoColor:
			return 0
		case p.ToMove():
			return maxEval - int64(p.MoveNumber())
		default:
			return minEval + int64(p.MoveNumber())
		}
	}
	mine, theirs := 0, 0
	me := p.ToMove()
	addw := func(c tak.Color, w int) {
		if c == me {
			mine += w
		} else {
			theirs += w
		}
	}
	analysis := p.Analysis()
	for x := 0; x < p.Size(); x++ {
		for y := 0; y < p.Size(); y++ {
			sq := p.At(x, y)
			if len(sq) == 0 {
				continue
			}
			addw(sq[0].Color(), weightControlled)
			if sq[0].Kind() == tak.Capstone {
				addw(sq[0].Color(), weightCapstone)
			}

			for i, stone := range sq {
				if i > 0 && i < p.Size() {
					addw(sq[0].Color(), weightCaptured)
				}
				if stone.Kind() == tak.Flat {
					addw(stone.Color(), weightFlat)
				}
			}
		}
	}
	for _, r := range m.regions {
		w := bitboard.Popcount(analysis.White & r)
		b := bitboard.Popcount(analysis.Black & r)
		if w > b {
			addw(tak.White, (w-b)*weightAdvantage)
		} else {
			addw(tak.Black, (b-w)*weightAdvantage)
		}
	}
	o := analysis.White | analysis.Black
	addw(tak.White, weightThreat*m.threats(analysis.WhiteGroups, o))
	addw(tak.Black, weightThreat*m.threats(analysis.BlackGroups, o))

	return int64(mine - theirs)
}

func (m *MinimaxAI) threats(groups []uint64, filled uint64) int {
	count := 0
	empty := ^filled
	s := uint(m.cfg.Size)
	for _, g := range groups {
		if g&m.c.L != 0 {
			if g&(m.c.R<<1) != 0 && empty&m.c.R != 0 {
				count++
			}
		}
		if g&m.c.R != 0 {
			if g&(m.c.L>>1) != 0 && empty&m.c.L != 0 {
				count++
			}
		}
		if g&m.c.B != 0 {
			if g&(m.c.T>>s) != 0 && empty&m.c.T != 0 {
				count++
			}
		}
		if g&m.c.T != 0 {
			if g&(m.c.B<<s) != 0 && empty&m.c.B != 0 {
				count++
			}
		}
	}
	return count
}
