package ai

import (
	"github.com/nelhage/taktician/bitboard"
	"github.com/nelhage/taktician/tak"
)

type Weights struct {
	TopFlat  int
	Standing int
	Capstone int

	Flat     int
	Captured int

	Concentration int

	Liberties int

	Groups [8]int
}

var DefaultWeights = Weights{
	TopFlat:  300,
	Standing: 200,
	Capstone: 300,

	Flat:      100,
	Liberties: 25,

	Captured: 25,

	Concentration: 50,

	Groups: [8]int{
		0,   // 0
		0,   // 1
		0,   // 2
		100, // 3
		200, // 4
	},
}

func MakeEvaluator(w *Weights) EvaluationFunc {
	return func(m *MinimaxAI, p *tak.Position) int64 {
		return evaluate(w, m, p)
	}
}

var DefaultEvaluate = MakeEvaluator(&DefaultWeights)

func evaluate(w *Weights, m *MinimaxAI, p *tak.Position) int64 {
	if over, winner := p.GameOver(); over {
		var pieces int64
		if winner == tak.White {
			pieces = int64(p.WhiteStones())
		} else {
			pieces = int64(p.BlackStones())
		}
		switch winner {
		case tak.NoColor:
			return 0
		case p.ToMove():
			return maxEval - int64(p.MoveNumber()) + pieces
		default:
			return minEval + int64(p.MoveNumber()) - pieces
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
			switch sq[0].Kind() {
			case tak.Standing:
				addw(sq[0].Color(), w.Standing)
			case tak.Flat:
				addw(sq[0].Color(), w.TopFlat)
			case tak.Capstone:
				addw(sq[0].Color(), w.Capstone)
			}

			for i, stone := range sq {
				if i > 0 && i < p.Size() {
					addw(sq[0].Color(), w.Captured)
				}
				if stone.Kind() == tak.Flat {
					addw(stone.Color(), w.Flat)
				}
			}
		}
	}

	empty := ^(analysis.White | analysis.Black)
	addw(tak.White, m.scoreGroups(analysis.WhiteGroups, empty, w))
	addw(tak.Black, m.scoreGroups(analysis.BlackGroups, empty, w))

	for _, r := range m.regions {
		wc := bitboard.Popcount(analysis.White & r)
		bc := bitboard.Popcount(analysis.Black & r)
		if wc > bc {
			addw(tak.White, (wc-bc)*w.Concentration)
		} else {
			addw(tak.Black, (bc-wc)*w.Concentration)
		}
	}

	wl := bitboard.Popcount(bitboard.Grow(&m.c, ^analysis.Black, analysis.White) &^ analysis.White)
	bl := bitboard.Popcount(bitboard.Grow(&m.c, ^analysis.White, analysis.Black) &^ analysis.Black)
	addw(tak.White, w.Liberties*wl)
	addw(tak.Black, w.Liberties*bl)

	return int64(mine - theirs)
}

func (ai *MinimaxAI) scoreGroups(gs []uint64, empty uint64, ws *Weights) int {
	sc := 0
	for _, g := range gs {
		w, h := bitboard.Dimensions(&ai.c, g)

		sp := w
		if h > sp {
			sp = h
		}
		sc += ws.Groups[sp]
	}

	return sc
}
