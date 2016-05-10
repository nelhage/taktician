package ai

import (
	"github.com/nelhage/taktician/bitboard"
	"github.com/nelhage/taktician/tak"
)

const (
	v3WeightFlat       = 200
	v3WeightStanding   = 100
	v3WeightCaptured   = 100
	v3WeightControlled = 300
	v3WeightCapstone   = -150
	v3WeightGroup      = 4
)

func V3evaluate(m *MinimaxAI, p *tak.Position) int64 {
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
	weightGroup := v3WeightGroup
	maxStack := 0
	analysis := p.Analysis()
	for x := 0; x < p.Size(); x++ {
		for y := 0; y < p.Size(); y++ {
			sq := p.At(x, y)
			if len(sq) == 0 {
				continue
			}
			if len(sq) > maxStack {
				maxStack = len(sq)
			}
			if sq[0].Kind() == tak.Standing {
				addw(sq[0].Color(), v3WeightStanding)
			} else {
				addw(sq[0].Color(), v3WeightControlled)
			}
			if sq[0].Kind() == tak.Capstone {
				addw(sq[0].Color(), v3WeightCapstone)
			}

			for i, stone := range sq {
				if i > 0 && i < p.Size() {
					addw(sq[0].Color(), v3WeightCaptured)
				}
				if stone.Kind() == tak.Flat {
					addw(stone.Color(), v3WeightFlat)
				}
			}
		}
	}

	empty := ^(analysis.White | analysis.Black)
	if maxStack >= 3 {
		weightGroup--
	}
	if bitboard.Popcount(empty) < (p.Size()*p.Size())/2 {
		weightGroup--
	}
	addw(tak.White, m.scoreGroups(analysis.WhiteGroups, empty, weightGroup))
	addw(tak.Black, m.scoreGroups(analysis.BlackGroups, empty, weightGroup))

	return int64(mine - theirs)
}

func (ai *MinimaxAI) scoreGroups(gs []uint64, empty uint64, weight int) int {
	sc := 0
	for _, g := range gs {
		w, h := bitboard.Dimensions(&ai.c, g)
		sz := bitboard.Popcount(g)
		libs := bitboard.Popcount(bitboard.Grow(&ai.c, g|empty, g) &^ g)

		sp := w
		if h > sp {
			sp = h
		}
		sc += (sp*sp + (2 * sz) + libs) * weight
	}

	return sc
}
