package ai

import (
	"github.com/nelhage/taktician/bitboard"
	"github.com/nelhage/taktician/tak"
)

const (
	weightFlat       = 200
	weightCaptured   = 100
	weightControlled = 300
	weightCapstone   = -150
	weightGroup      = 4
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
			if sq[0].Kind() != tak.Standing {
				addw(sq[0].Color(), weightControlled)
			}
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

	empty := ^(analysis.White | analysis.Black)
	addw(tak.White, m.scoreGroups(analysis.WhiteGroups, empty))
	addw(tak.Black, m.scoreGroups(analysis.BlackGroups, empty))

	for _, r := range m.regions {
		w := bitboard.Popcount(analysis.White & r)
		b := bitboard.Popcount(analysis.Black & r)
		if w > b {
			addw(tak.White, (w-b)*weightAdvantage)
		} else {
			addw(tak.Black, (b-w)*weightAdvantage)
		}
	}

	return int64(mine - theirs)
}

func (ai *MinimaxAI) scoreGroups(gs []uint64, empty uint64) int {
	sc := 0
	for _, g := range gs {
		w, h := bitboard.Dimensions(&ai.c, g)
		sz := bitboard.Popcount(g)
		libs := bitboard.Popcount(bitboard.Grow(&ai.c, g|empty, g) &^ g)

		sp := w
		if h > sp {
			sp = h
		}
		sc += (sp*sp + (2 * sz) + libs) * weightGroup
	}

	return sc
}
